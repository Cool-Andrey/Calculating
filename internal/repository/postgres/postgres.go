package postgres

import (
	"context"
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"strconv"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool}
}

func (r *Repository) Get(ctx context.Context, key int) (models.Expressions, error) {
	res := models.Expressions{}
	q := `SELECT id, status, COALESCE(result, '') FROM expressions WHERE id = $1`
	err := r.pool.QueryRow(ctx, q, key).Scan(&res.Id, &res.Status, &res.Result)
	return res, err
}

func (r *Repository) Set(ctx context.Context, value models.Expressions) (int64, error) {
	if value.Result == "" {
		q := `INSERT INTO expressions(status) VALUES($1) RETURNING id`
		var id int
		err := r.pool.QueryRow(ctx, q, value.Status).Scan(&id)
		if err != nil {
			return 0, err
		}
		return int64(id), nil
	} else {
		q := `UPDATE expressions SET status = $2, result = $3 WHERE id = $1`
		_, err := r.pool.Exec(ctx, q, value.Id, value.Status, value.Result)
		return value.Id, err
	}
}

func (r *Repository) GetAll(ctx context.Context) ([]models.Expressions, error) {
	var res []models.Expressions
	q := `SELECT id, status, result FROM expressions ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return []models.Expressions{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e models.Expressions
		err = rows.Scan(&e.Id, &e.Status, &e.Result)
		if err != nil {
			return []models.Expressions{}, err
		}
		res = append(res, e)
	}
	return res, nil
}

func (r *Repository) SetWithExpression(ctx context.Context, value models.Expressions, expression string) (int, error) {
	q := `INSERT INTO expressions(status, expression) VALUES($1, $2) RETURNING id`
	var id int
	err := r.pool.QueryRow(ctx, q, value.Status, expression).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) UpdateAST(ctx context.Context, id int, ast []byte) error {
	q := `UPDATE expressions SET ast_data = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, q, ast, id)
	return err
}

func (r *Repository) GetProcTasks(ctx context.Context) ([]models.Expression, error) {
	q := `SELECT id, expression, ast_data FROM expressions WHERE status='Подсчёт'`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var expressions []models.Expression
	for rows.Next() {
		var expression models.Expression
		if err := rows.Scan(&expression.ID, &expression.Expression, &expression.ASTData); err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
	}
	return expressions, nil
}

func (r *Repository) GetStatus(ctx context.Context, id int64) (string, error) {
	q := `SELECT status FROM expressions WHERE id = $1`
	var status string
	err := r.pool.QueryRow(ctx, q, id).Scan(&status)
	return status, err
}

func (r *Repository) CreateUser(ctx context.Context, login, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	q := `INSERT INTO users(login, password_hash) VALUES($1, $2)`
	_, err = r.pool.Exec(ctx, q, login, hash)
	return err
}

func (r *Repository) VerifyUser(ctx context.Context, login, password string) (bool, error) {
	var dbHash string
	q := `SELECT password_hash FROM users WHERE login = $1`
	if err := r.pool.QueryRow(ctx, q, login).Scan(&dbHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(password))
	return err == nil, nil
}

func (r *Repository) SaveTask(ctx context.Context, task *models.Task) (int, error) {
	q := `INSERT INTO tasks(operation, arg1, arg2, left_id, right_id, ready) VALUES($1, $2, $3, $4, $5, FALSE) RETURNING id`
	var id int
	err := r.pool.QueryRow(ctx, q, task.Operation, task.Arg1, task.Arg2, task.LeftID, task.RightID).Scan(&id)
	return id, err
}

func (r *Repository) UpdateTask(ctx context.Context, task *models.Task) error {
	q := `UPDATE tasks SET ready = TRUE, result = $1 WHERE id = $2;
		  UPDATE expressions SET status = 'Выполнено', result = $3 WHERE main_task_id = $4;`
	resStr := strconv.FormatFloat(task.Result, 'f', 2, 64)
	_, err := r.pool.Exec(ctx, q, task.Result, task.Id)
	return err
}

func (r *Repository) GetStatusTask(ctx context.Context, id int64) (bool, error) {
	q := `SELECT status FROM tasks WHERE id = $1`
	var ready bool
	err := r.pool.QueryRow(ctx, q, id).Scan(&ready)
	return ready, err
}

func (r *Repository) RemoveTask(ctx context.Context, id int) error {
	q := `DELETE FROM tasks WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}
