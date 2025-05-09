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
			stack = append(stack, &node{value: token, left: left, right: right})
		default:
			stack = append(stack, &node{value: token})
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

func (n *node) saveState(ctx context.Context, id int, pool *pgxpool.Pool, logger *zap.SugaredLogger) {
	ast, err := n.ToJSON()
	if err != nil {
		logger.Errorf("Ошибка маршалинга AST: %v", err)
		return
	}
	if err := postgres.UpdateAST(ctx, id, ast, pool); err != nil {
		logger.Errorf("Ошибка сохранения состояния: %v", err)
	}
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
		Processed: n.result != 0 || (n.left == nil && n.right == nil),
	}
}

func ParseASTFromJSON(data []byte) (*node, error) {
	var ast models.ASTNode
	if err := json.Unmarshal(data, &ast); err != nil {
		return nil, err
	}
	return convertToNode(&ast), nil
}

func convertToNode(ast *models.ASTNode) *node {
	if ast == nil {
		return nil
	}
	n := &node{
		value:  ast.Value,
		result: ast.Result,
	}
	if ast.Left != nil {
		n.left = convertToNode(ast.Left)
	}
	if ast.Right != nil {
		n.right = convertToNode(ast.Right)
	}
	return n
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
	defer func() {
		ast, err := n.ToJSON()
		if err != nil {
			logger.Errorf("Ошибка преобразования дерева в JSON: %v", err)
		}
		postgres.UpdateAST(ctx, id, ast, pool)
	}()
	if n.result != 0 {
		return n.result, nil
	}
	if !calc.IsOperator(n.value) {
		val, _ := strconv.ParseFloat(n.value, 64)
		n.result = val
		return val, nil
	}
	left, err := calcLvl(ctx, n.left, tasks, results, id, pool, logger)
	if err != nil {
		return 0, err
	}
	right, err := calcLvl(ctx, n.right, tasks, results, id, pool, logger)
	if err != nil {
		return 0, err
	}
	if n.value == "/" && right == 0 {
		return 0, calc.ErrDivByZero
	}
	task := models.Task{
		Operation: n.value,
		Arg1:      left,
		Arg2:      right,
		Id:        id,
	}
	select {
	case tasks <- task:
		n.pending = true
		n.operation = &task
		n.saveState(ctx, id, pool, logger)
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	select {
	case result := <-results:
		n.pending = false
		n.operation = nil
		n.saveState(ctx, id, pool, logger)
		n.result = result
		return result, nil
	case <-ctx.Done():
		n.saveState(ctx, id, pool, logger)
		return 0, ctx.Err()
	}
}

func Process(
	ctx context.Context,
	root *node,
	tasks chan models.Task,
	results chan float64,
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
		results <- res
	}
	done <- 1
}

func Calc(
	ctx context.Context,
	expression string,
	logger *zap.SugaredLogger,
	tasks chan models.Task,
	results chan float64,
	errorsCh chan error,
	done chan int,
	id int,
	pool *pgxpool.Pool,
) {
	//recoverExpr, err := postgres.GetExpression(ctx, id, pool)
	//switch {
	//case err != nil:
	//	logger.Errorf("Ошибка восстановления выражения: %v", err)
	//case recoverExpr.ASTData != nil:
	//	root, err := ParseASTFromJSON(recoverExpr.ASTData)
	//	if err == nil {
	//		go Process(ctx, root, tasks, results, errorsCh, done, id, pool, logger)
	//		return
	//	} else {
	//		logger.Errorf("Ошибка парсинга AST: %v", err)
	//	}
	//}
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
	go Process(ctx, ast, tasks, results, errorsCh, done, id, pool, logger)
}

//func calcLvl(node *node, tasks chan models.Task, results chan float64, logger *zap.SugaredLogger, id int) (float64, error) {
//	if !calc.IsOperator(node.value) {
//		val, err := strconv.ParseFloat(node.value, 64)
//		if err != nil {
//			logger.Errorf("Не удалось преобразовать строку в число: %v", err)
//		}
//		return val, nil
//	}
//	left, err := calcLvl(node.left, tasks, results, logger, id)
//	if err != nil {
//		return 0, err
//	}
//	right, err := calcLvl(node.right, tasks, results, logger, id)
//	if err != nil {
//		return 0, err
//	}
//	if node.value == "/" && right == 0 {
//		return 0.0, calc.ErrDivByZero
//	}
//	task := models.Task{Operation: node.value, Arg1: left, Arg2: right, Result: node.result, Id: id}
//	tasks <- task
//	result := <-results
//	logger.Debugf("Получил результат: %.2f", result)
//	return result, nil
//}

//func Calc(expression string,
//	logger *zap.SugaredLogger,
//	tasks chan models.Task,
//	results chan float64,
//	errors chan error,
//	done chan int,
//	id int) {
//	if !calc.RightString(expression) {
//		errors <- calc.ErrInvalidBracket
//		done <- 1
//		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidBracket)
//		logger.Debug("Оркестратор завершил работу.")
//		return
//	}
//	if calc.IsLetter(expression) {
//		errors <- calc.ErrInvalidOperands
//		done <- 1
//		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
//		logger.Debug("Оркестратор завершил работу.")
//		return
//	}
//	if expression == "" || expression == " " {
//		errors <- calc.ErrEmptyExpression
//		done <- 1
//		logger.Errorf("Ошибка вычисления: %v", calc.ErrEmptyExpression)
//		logger.Debug("Оркестратор завершил работу.")
//		return
//	}
//	expression = strings.ReplaceAll(expression, " ", "")
//	tokens := calc.Tokenize(expression)
//	tokens = calc.InfixToPostfix(tokens)
//	if !calc.CountOp(tokens) {
//		errors <- calc.ErrInvalidOperands
//		done <- 1
//		logger.Errorf("Ошибка вычисления: %v", calc.ErrInvalidOperands)
//		logger.Debug("Оркестратор завершил работу.")
//		return
//	}
//	node := buildAST(tokens)
//	result, err := calcLvl(node, tasks, results, logger, id)
//	if err != nil {
//		errors <- err
//	} else {
//		results <- result
//	}
//	done <- 1
//	logger.Debug("Оркестратор завершил работу.")
//}
