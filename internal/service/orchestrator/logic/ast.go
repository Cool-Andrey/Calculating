package logic

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (n *node) restoreFromTask(task models.Task) {
	if n.value == task.Operation {
		n.pending = true
		n.operation = &task
		n.retriesCnt++
	}
}

func (n *node) traverseAndRestore(ctx context.Context, tasks <-chan models.Task) {
	if n == nil {
		return
	}
	select {
	case task := <-tasks:
		n.restoreFromTask(task)
	default:
	}
	n.left.traverseAndRestore(ctx, tasks)
	n.right.traverseAndRestore(ctx, tasks)
}

func (n *node) ToJSON() ([]byte, error) {
	ast := convertToASTNode(n)
	return json.Marshal(ast)
}

func convertToASTNode(n *node) *models.ASTNode {
	if n == nil {
		return nil
	}
	return &models.ASTNode{
		Value:     n.value,
		Left:      convertToASTNode(n.left),
		Right:     convertToASTNode(n.right),
		Result:    n.result,
		Processed: !n.pending,
	}
}

func convertToNode(ast *models.ASTNode) *node {
	if ast == nil {
		return nil
	}
	n := &node{
		value:   ast.Value,
		result:  ast.Result,
		pending: !ast.Processed,
	}
	if ast.Left != nil {
		n.left = convertToNode(ast.Left)
	}
	if ast.Right != nil {
		n.right = convertToNode(ast.Right)
	}
	return n
}

func ParseASTFromJSON(data []byte) (*node, error) {
	var ast models.ASTNode
	if err := json.Unmarshal(data, &ast); err != nil {
		return nil, err
	}
	return convertToNode(&ast), nil
}

func calcLvl(
	ctx context.Context,
	n *node,
	tasks chan models.Task,
	results chan float64,
	id int,
	pool *pgxpool.Pool,
	logger *zap.SugaredLogger,
) (float64, error) {
	//defer func() {
	//	ast, err := n.ToJSON()
	//	if err != nil {
	//		logger.Errorf("Ошибка преобразования дерева в JSON: %v", err)
	//	}
	//	postgres.UpdateAST(ctx, id, ast, pool)
	//}()
	if !n.pending {
		if !calc.IsOperator(n.value) {
			val, err := strconv.ParseFloat(n.value, 64)
			if err != nil {
				return 0, err
			}
			n.result = val
		}
		return n.result, nil
	}
	if !calc.IsOperator(n.value) {
		val, _ := strconv.ParseFloat(n.value, 64)
		n.result = val
		return val, nil
	}
	left, err := calcLvl(ctx, n.left, tasks, results, id, pool, logger)
	if err != nil {
		logger.Errorf("Ошибка вычисления левого дерева: %v", err)
		return 0, err
	}
	right, err := calcLvl(ctx, n.right, tasks, results, id, pool, logger)
	if err != nil {
		logger.Errorf("Ошибка вычисления правого дерева: %v", err)
		return 0, err
	}
	if n.value == "/" && right == 0 {
		return 0, calc.ErrDivByZero
	}
	task := models.Task{
		Operation: n.value,
		Arg1:      left,
		Arg2:      right,
		Id:        int64(id),
	}
	logger.Debugf("Отдал операцию:%.2f%s%.2f", task.Arg1, task.Operation, task.Arg2)
	select {
	case tasks <- task:
		n.pending = true
		n.operation = &task
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	select {
	case result := <-results:
		n.pending = false
		n.operation = nil
		n.result = result
		return result, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func Process(
	ctx context.Context,
	root *node,
	tasks chan models.Task,
	results chan float64,
	resCh chan float64,
	errorsCh chan error,
	done chan int,
	id int,
	pool *pgxpool.Pool,
	logger *zap.SugaredLogger,
) {
	res, err := calcLvl(ctx, root, tasks, results, id, pool, logger)
	if err != nil {
		errorsCh <- err
	} else {
		resCh <- res
	}
	done <- 1
}

func Calc(
	ctx context.Context,
	expression string,
	logger *zap.SugaredLogger,
	tasks chan models.Task,
	results chan float64,
	resCh chan float64,
	errorsCh chan error,
	done chan int,
	id int,
	pool *pgxpool.Pool,
) {
	if !calc.RightString(expression) {
		errorsCh <- calc.ErrInvalidBracket
		done <- 1
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidBracket)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	if calc.IsLetter(expression) {
		errorsCh <- calc.ErrInvalidOperands
		done <- 1
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	if expression == "" || expression == " " {
		errorsCh <- calc.ErrEmptyExpression
		done <- 1
		logger.Errorf("Ошибка вычисления: %v", calc.ErrEmptyExpression)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	expression = strings.ReplaceAll(expression, " ", "")
	tokens := calc.Tokenize(expression)
	tokens = calc.InfixToPostfix(tokens)
	if !calc.CountOp(tokens) {
		errorsCh <- calc.ErrInvalidOperands
		done <- 1
		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
		logger.Debug("Оркестратор завершил работу.")
		return
	}
	ast := buildAST(tokens)
	astData, err := ast.ToJSON()
	if err != nil {
		logger.Errorf("Ошибка парсинга AST в JSON: %v", err)
		errorsCh <- errors.New("Неизвестная ошибка")
	}
	postgres.UpdateAST(ctx, id, astData, pool)
	go Process(ctx, ast, tasks, results, resCh, errorsCh, done, id, pool, logger)
}
