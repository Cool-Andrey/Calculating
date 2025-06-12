package ast

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/repository/postgres"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"slices"
	"strconv"
	"strings"
)

type node struct {
	value     string
	left      *node
	right     *node
	result    float64
	operation *models.Task
}

type handleError interface {
	handleError(ctx context.Context, id int, err error)
}

type AST struct {
	r      *postgres.Repository
	logger *zap.SugaredLogger
}

func NewAST(r *postgres.Repository, logger *zap.SugaredLogger) *AST {
	return &AST{r: r, logger: logger}
}

func (a AST) buildAST(tokens []string) *node {
	var stack []*node
	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			right := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			left := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack = append(stack, &node{
				value: token,
				left:  left,
				right: right,
			})
		default:
			stack = append(stack, &node{
				value: token,
			})
		}
	}
	return stack[0]
}

func (a AST) calcLvl(
	ctx context.Context,
	n *node,
) ([]*models.Task, error) {
	if !calc.IsOperator(n.left.value) && !calc.IsOperator(n.right.value) {
		arg1, err := strconv.ParseFloat(n.left.value, 64)
		if err != nil {
			a.logger.Errorf("Ошибка преобразования 1 операнда: %s", err)
			return []*models.Task{}, err
		}
		arg2, err := strconv.ParseFloat(n.right.value, 64)
		if err != nil {
			a.logger.Errorf("Ошибка преобразования 2 операнда: %s", err)
			return []*models.Task{}, err
		}
		id, err := uuid.NewV7()
		if err != nil {
			a.logger.Errorf("Ошибка создания uuid: %v", err)
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
		tasks, err := a.calcLvl(ctx, n.left)
		if err != nil {
			a.logger.Errorf("Ошибка вычесления левого поддерева: %s", err)
			return []*models.Task{}, err
		}
		arg2, err := strconv.ParseFloat(n.right.value, 64)
		if err != nil {
			a.logger.Errorf("Ошибка преобразования 2 операнда: %s", err)
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
		tasks, err := a.calcLvl(ctx, n.right)
		if err != nil {
			a.logger.Errorf("Ошибка вычесления левого поддерева: %s", err)
			return []*models.Task{}, err
		}
		arg1, err := strconv.ParseFloat(n.left.value, 64)
		if err != nil {
			a.logger.Errorf("Ошибка преобразования 1 операнда: %s", err)
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
		tasks1, err := a.calcLvl(ctx, n.left)
		if err != nil {
			return []*models.Task{}, err
		}
		tasks2, err := a.calcLvl(ctx, n.right)
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

func (a AST) Process(
	ctx context.Context,
	root *node,
	id int,
) {
	tasks, err := a.calcLvl(ctx, root)
	if err != nil {
		a.handleError(ctx, id, err)
	}
	err = a.r.SaveTasks(ctx, tasks, id)
	if err != nil {
		a.handleError(ctx, id, err)
	}
}

func (a AST) Calc(
	ctx context.Context,
	expression string,
	id int,
) {
	if !calc.RightString(expression) {
		a.handleError(ctx, id, calc.ErrInvalidBracket)
		a.logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidBracket)
		a.logger.Debug("Оркестратор завершил работу.")
		return
	}
	if calc.IsLetter(expression) {
		a.handleError(ctx, id, calc.ErrInvalidOperands)
		a.logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		a.logger.Debug("Оркестратор завершил работу.")
		return
	}
	if expression == "" || expression == " " {
		a.handleError(ctx, id, calc.ErrEmptyExpression)
		a.logger.Errorf("Ошибка вычисления: %v", calc.ErrEmptyExpression)
		a.logger.Debug("Оркестратор завершил работу.")
		return
	}
	expression = strings.ReplaceAll(expression, " ", "")
	tokens := calc.Tokenize(expression)
	tokens = calc.InfixToPostfix(tokens)
	if !calc.CountOp(tokens) {
		a.handleError(ctx, id, calc.ErrInvalidOperands)
		a.logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		a.logger.Debug("Оркестратор завершил работу.")
		return
	}
	if res, err := strconv.ParseFloat(expression, 64); err == nil {
		resStr := strconv.FormatFloat(res, 'f', 2, 64)
		currentStatus, err := a.r.GetStatus(ctx, int64(id))
		if err != nil || currentStatus != "Подсчёт" {
			a.logger.Warnf("Попытка обновить неактуальную задачу ID %d", id)
			return
		}
		if _, err := a.r.Set(ctx, models.Expressions{
			ID:     int64(id),
			Status: "Выполнено",
			Result: &resStr,
		}); err != nil {
			a.logger.Error("Ошибка сохранения успешного результата",
				zap.Error(err),
				zap.Int("id", id),
				zap.Float64("результат", res))
		} else {
			a.logger.Debug("Успешно сохранено выражение",
				zap.Int("id", id),
				zap.Float64("результат", res))
		}
		return
	}
	ast := a.buildAST(tokens)
	a.Process(ctx, ast, id)
}

func (a AST) handleError(ctx context.Context, id int, err error) {
	var status string
	var result string

	if ok := slices.Contains(calc.Errors, err); ok {
		status = "Ошибка"
		result = err.Error()
	} else {
		status = "Ошибка"
		result = "Что-то пошло не так"
	}
	a.logger.Errorf("Ошибка: %v", err)
	if _, dbErr := a.r.Set(ctx, models.Expressions{
		ID:     int64(id),
		Status: status,
		Result: &result,
	}); dbErr != nil {
		a.logger.Errorf("Ошибка сохранения ошибки :): %v", dbErr)
	}
}
