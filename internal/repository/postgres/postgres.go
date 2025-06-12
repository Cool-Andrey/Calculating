package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
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
	err := r.pool.QueryRow(ctx, q, key).Scan(&res.ID, &res.Status, &res.Result)
	return res, err
}

func (r *Repository) Set(ctx context.Context, value models.Expressions) (int64, error) {
	if value.Result == nil {
		q := `INSERT INTO expressions(status) VALUES($1) RETURNING id`
		var id int
		err := r.pool.QueryRow(ctx, q, value.Status).Scan(&id)
		if err != nil {
			return 0, err
		}
		return int64(id), nil
	} else {
		q := `UPDATE expressions SET status = $2, result = $3, main_task_id=NULL WHERE id = $1`
		_, err := r.pool.Exec(ctx, q, value.ID, value.Status, value.Result)
		return value.ID, err
	}
}

func (r *Repository) GetAll(ctx context.Context) ([]models.Expressions, error) {
	var res []models.Expressions
	q := `SELECT id, status, COALESCE(result, '') FROM expressions ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return []models.Expressions{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e models.Expressions
		err = rows.Scan(&e.ID, &e.Status, &e.Result)
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

func createRequest(tasks []*models.Task, id int) (string, []any) {
	q := &strings.Builder{}
	q.WriteString("INSERT INTO tasks(id, expression_id, operation, arg1, arg2, left_id, right_id) VALUES")
	const requiredFields = 7
	args := make([]any, 0, len(tasks)*requiredFields)
	listLen := len(tasks)
	for i, task := range tasks {
		args = append(args, task.ID, id, task.Operation, task.Arg1, task.Arg2, task.LeftID, task.RightID)
		base := i * requiredFields
		fmt.Fprintf(q, "($%d,$%d,$%d,$%d,$%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		if i < listLen-1 {
			fmt.Fprint(q, ", ")
		}
	}
	return q.String(), args
}

func (r *Repository) SaveTasks(ctx context.Context, tasks []*models.Task, id int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q, args := createRequest(tasks, id)
	_, err = tx.Exec(ctx, q, args...)
	if err != nil {
		return err
	}
	q = "UPDATE expressions SET main_task_id=$1 WHERE id = $2"
	_, err = tx.Exec(ctx, q, tasks[len(tasks)-1].ID, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdateTask(ctx context.Context, task *models.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	resStr := strconv.FormatFloat(task.Result, 'f', 2, 64)
	q := `UPDATE expressions SET status = 'Выполнено', result = $2 WHERE main_task_id = $1`
	_, err = tx.Exec(ctx, q, task.ID, resStr)
	if err != nil {
		return err
	}
	q = `UPDATE tasks SET arg1 = $2 WHERE left_id = $1`
	_, err = tx.Exec(ctx, q, task.ID, task.Result)
	if err != nil {
		return err
	}
	q = `UPDATE tasks SET arg2 = $2 WHERE right_id = $1`
	_, err = tx.Exec(ctx, q, task.ID, task.Result)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetStatusTask(ctx context.Context, id int64) (bool, error) {
	q := `SELECT status FROM tasks WHERE id = $1`
	var ready bool
	err := r.pool.QueryRow(ctx, q, id).Scan(&ready)
	return ready, err
}

func (r *Repository) GetTask(ctx context.Context) (models.Task, error) {
	q := `SELECT id, expression_id, operation, arg1, arg2 FROM tasks WHERE left_id IS NULL AND right_id IS NULL`
	var task models.Task
	err := r.pool.QueryRow(ctx, q).Scan(&task.ID, &task.ExpressionID, &task.Operation, &task.Arg1, &task.Arg2)
	return task, err
}

func (r *Repository) RemoveTask(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM tasks WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}
