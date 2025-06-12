package logic

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type node struct {
	value      string
	left       *node
	right      *node
	result     float64
	pending    bool
	operation  *models.Task
	retriesCnt int
}

type handleError func(ctx context.Context, id int, r *postgres.Repository, err error, logger *zap.SugaredLogger)

func buildAST(tokens []string) *node {
	stack := []*node{}
	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			right := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			left := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack = append(stack, &node{
				value:   token,
				left:    left,
				right:   right,
				pending: true,
			})
		default:
			stack = append(stack, &node{
				value:   token,
				pending: false,
			})
		}
	}
	return stack[0]
}

func calcLvl(
	ctx context.Context,
	n *node,
	logger *zap.SugaredLogger,
) ([]*models.Task, error) {
	if !calc.IsOperator(n.left.value) && !calc.IsOperator(n.right.value) {
		arg1, err := strconv.ParseFloat(n.left.value, 64)
		if err != nil {
			logger.Errorf("Ошибка преобразования 1 операнда: %s", err)
			return []*models.Task{}, err
		}
		arg2, err := strconv.ParseFloat(n.right.value, 64)
		if err != nil {
			logger.Errorf("Ошибка преобразования 2 операнда: %s", err)
			return []*models.Task{}, err
		}
		id, err := uuid.NewV7()
		if err != nil {
			logger.Errorf("Ошибка создания uuid: %v", err)
			return []*models.Task{}, err
		}
		task := &models.Task{
			ID:        id,
			Operation: n.value,
			Arg1:      arg1,
			Arg2:      arg2,
		}
		return []*models.Task{task}, err
	} else if calc.IsOperator(n.left.value) && !calc.IsOperator(n.right.value) {
		tasks, err := calcLvl(ctx, n.left, logger)
		if err != nil {
			logger.Errorf("Ошибка вычесления левого поддерева: %s", err)
			return []*models.Task{}, err
		}
		arg2, err := strconv.ParseFloat(n.right.value, 64)
		if err != nil {
			logger.Errorf("Ошибка преобразования 2 операнда: %s", err)
			return []*models.Task{}, err
		}
		leftID := tasks[len(tasks)-1].ID
		tasks = append(tasks, &models.Task{
			Operation: n.value,
			Arg2:      arg2,
			LeftID:    &leftID,
		})
		return tasks, nil
	} else if !calc.IsOperator(n.left.value) && calc.IsOperator(n.right.value) {
		tasks, err := calcLvl(ctx, n.right, logger)
		if err != nil {
			logger.Errorf("Ошибка вычесления левого поддерева: %s", err)
			return []*models.Task{}, err
		}
		arg1, err := strconv.ParseFloat(n.left.value, 64)
		if err != nil {
			logger.Errorf("Ошибка преобразования 1 операнда: %s", err)
			return []*models.Task{}, err
		}
		rightID := tasks[len(tasks)-1].ID
		tasks = append(tasks, &models.Task{
			Operation: n.value,
			Arg1:      arg1,
			LeftID:    &rightID,
		})
		return tasks, nil
	} else {
		tasks1, err := calcLvl(ctx, n.left, logger)
		if err != nil {
			return []*models.Task{}, err
		}
		tasks2, err := calcLvl(ctx, n.right, logger)
		if err != nil {
			return []*models.Task{}, err
		}
		leftID := tasks1[len(tasks1)-1].ID
		rightID := tasks2[len(tasks2)-1].ID
		tasks := append(tasks1, tasks2...)
		tasks = append(tasks, &models.Task{
			Operation: n.value,
			LeftID:    &leftID,
			RightID:   &rightID,
		})
		return tasks, nil
	}
}

func Process(
	ctx context.Context,
	root *node,
	id int,
	r *postgres.Repository,
	logger *zap.SugaredLogger,
	handleError handleError,
) {
	tasks, err := calcLvl(ctx, root, logger)
	if err != nil {
		handleError(ctx, id, r, err, logger)
	}
	err = r.SaveTasks(ctx, tasks, id)
	if err != nil {
		handleError(ctx, id, r, err, logger)
	}
}

func Calc(
	ctx context.Context,
	expression string,
	id int,
	logger *zap.SugaredLogger,
	r *postgres.Repository,
	handleError handleError,
) {
	if !calc.RightString(expression) {
		handleError(ctx, id, r, calc.ErrInvalidBracket, logger)
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidBracket)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	if calc.IsLetter(expression) {
		handleError(ctx, id, r, calc.ErrInvalidOperands, logger)
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	if expression == "" || expression == " " {
		handleError(ctx, id, r, calc.ErrEmptyExpression, logger)
		logger.Errorf("Ошибка вычисления: %v", calc.ErrEmptyExpression)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	expression = strings.ReplaceAll(expression, " ", "")
	tokens := calc.Tokenize(expression)
	tokens = calc.InfixToPostfix(tokens)
	if !calc.CountOp(tokens) {
		handleError(ctx, id, r, calc.ErrInvalidOperands, logger)
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	if res, err := strconv.ParseFloat(expression, 64); err == nil {
		resStr := strconv.FormatFloat(res, 'f', 2, 64)
		currentStatus, err := r.GetStatus(ctx, int64(id))
		if err != nil || currentStatus != "Подсчёт" {
			logger.Warnf("Попытка обновить неактуальную задачу ID %d", id)
			return
		}
		if _, err := r.Set(ctx, models.Expressions{
			ID:     int64(id),
			Status: "Выполнено",
			Result: &resStr,
		}); err != nil {
			logger.Error("Ошибка сохранения успешного результата",
				zap.Error(err),
				zap.Int("id", id),
				zap.Float64("результат", res))
		} else {
			logger.Debug("Успешно сохранено выражение",
				zap.Int("id", id),
				zap.Float64("результат", res))
		}
		return
	}
	ast := buildAST(tokens)
	Process(ctx, ast, id, r, logger, handleError)
}
