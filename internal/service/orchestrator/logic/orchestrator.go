package logic

import (
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type node struct {
	value  string
	left   *node
	right  *node
	result float64
}

func isOperator(s string) bool {
	return s == "+" || s == "-" || s == "*" || s == "/"
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

func calcLvl(node *node, tasks chan logic.Task, results chan float64, logger *zap.SugaredLogger, id int) (float64, error) {
	if !isOperator(node.value) {
		val, err := strconv.ParseFloat(node.value, 64)
		if err != nil {
			logger.Errorf("Не удалось преобразовать строку в число: %v", err)
		}
		return val, nil
	}
	left, err := calcLvl(node.left, tasks, results, logger, id)
	if err != nil {
		return 0, err
	}
	right, err := calcLvl(node.right, tasks, results, logger, id)
	if err != nil {
		return 0, err
	}
	if node.value == "/" && right == 0 {
		return 0.0, calc.ErrDivByZero
	}
	task := logic.Task{Operation: node.value, Arg1: left, Arg2: right, Result: node.result, Id: id}
	tasks <- task
	result := <-results
	logger.Debugf("Получил результат: %.2f", result)
	return result, nil
}

func Calc(expression string, logger *zap.SugaredLogger, tasks chan logic.Task, results chan float64, errors chan error, done chan int, id int) {
	if !calc.Right_string(expression) {
		errors <- calc.ErrInvalidBracket
		done <- 1
	}
	if calc.IsLetter(expression) {
		errors <- calc.ErrInvalidOperands
		done <- 1
	}
	if expression == "" || expression == " " {
		errors <- calc.ErrEmptyExpression
		done <- 1
	}
	expression = strings.ReplaceAll(expression, " ", "")
	tokens := calc.Tokenize(expression)
	tokens = calc.InfixToPostfix(tokens)
	if !calc.CountOp(tokens) {
		errors <- calc.ErrInvalidOperands
		done <- 1
	}
	node := buildAST(tokens)
	result, err := calcLvl(node, tasks, results, logger, id)
	if err != nil {
		errors <- err
	} else {
		results <- result
	}
	done <- 1
	logger.Debug("Оркестратор завершил работу.")
}
