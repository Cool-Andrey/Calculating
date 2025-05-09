package postgres

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"strconv"
)

func Get(ctx context.Context, key int, pool *pgxpool.Pool) (models.Expressions, error) {
	res := models.Expressions{}
	q := `SELECT id, status, COALESCE(result, '') FROM users WHERE id = $1`
	err := pool.QueryRow(ctx, q, key).Scan(&res.Id, &res.Status, &res.Result)
	return res, err
}

func ProcessTaskResult(ctx context.Context, task models.Task, in chan float64, pool *pgxpool.Pool, logger *zap.SugaredLogger) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	resStr := strconv.FormatFloat(task.Result, 'f', -1, 64)
	if _, err := tx.Exec(ctx,
		`UPDATE users 
         SET result = $1, status = 'Выполнено'
         WHERE id = $2`,
		resStr, task.Id); err != nil {
		return err
	}
	select {
	case in <- task.Result:
		logger.Infof("Результат для задачи %d успешно обработан", task.Id)
	case <-ctx.Done():
		return ctx.Err()
	}
	return tx.Commit(ctx)
}

func Set(ctx context.Context, value models.Expressions, pool *pgxpool.Pool) (int, error) {
	if value.Result == "" {
		q := `INSERT INTO users(status) VALUES($1) RETURNING id`
		var id int
		err := pool.QueryRow(ctx, q, value.Status).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	} else {
		q := `UPDATE users SET status = $2, result = $3 WHERE id = $1`
		_, err := pool.Exec(ctx, q, value.Id, value.Status, value.Result)
		return value.Id, err
	}
}

func GetAll(ctx context.Context, pool *pgxpool.Pool) ([]models.Expressions, error) {
	var res []models.Expressions
	q := `SELECT id, status, result FROM users`
	rows, err := pool.Query(ctx, q)
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

func SetWithExpression(ctx context.Context, value models.Expressions, expression string, pool *pgxpool.Pool) (int, error) {
	q := `INSERT INTO users(status, expression) VALUES($1, $2) RETURNING id`
	var id int
	err := pool.QueryRow(ctx, q, value.Status, expression).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func UpdateAST(ctx context.Context, id int, ast []byte, pool *pgxpool.Pool) error {
	q := `UPDATE users SET ast_data = $1 WHERE id = $2`
	_, err := pool.Exec(ctx, q, ast, id)
	return err
}

func GetExpression(ctx context.Context, id int, pool *pgxpool.Pool) (models.Expression, error) {
	var res models.Expression
	q := `SELECT id, status, result, expression, ast_data FROM users WHERE id = $1`
	err := pool.QueryRow(ctx, q, id).Scan(
		&res.ID,
		&res.Status,
		&res.Result,
		&res.Expression,
		&res.ASTData,
	)
	return res, err
}

func GetAllProcessing(ctx context.Context, pool *pgxpool.Pool) ([]models.Expression, error) {
	q := `SELECT id, expression, ast_data FROM users 
         WHERE status = 'Подсчёт'`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exprs []models.Expression
	for rows.Next() {
		var expr models.Expression
		if err := rows.Scan(&expr.ID, &expr.Expression, &expr.ASTData); err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

func GetProcTasks(ctx context.Context, pool *pgxpool.Pool) ([]models.Expression, error) {
	q := `SELECT id, expression, ast_data FROM users WHERE status='Подсчёт'`
	rows, err := pool.Query(ctx, q)
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

func GetStatus(ctx context.Context, id int, pool *pgxpool.Pool) (string, error) {
	q := `SELECT status FROM users WHERE id = $1`
	var status string
	err := pool.QueryRow(ctx, q, id).Scan(&status)
	return status, err
}
