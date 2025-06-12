package ast

import (
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
	"testing"
)

func TestCalc(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("Correct expression", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := "2 + 3 * 4"
		go Calc(expression, logger, tasks, results, errors, done, 1)
		task := <-tasks
		if task.Operation != "*" || task.Arg1 != 3 || task.Arg2 != 4 {
			t.Errorf("Ожидал задачу: * 3 4, получил: %s %.2f %.2f", task.Operation, task.Arg1, task.Arg2)
		}
		results <- 12
		task = <-tasks
		if task.Operation != "+" || task.Arg1 != 2 || task.Arg2 != 12 {
			t.Errorf("Ожидал задачу: + 2 12, получил: %s %.2f %.2f", task.Operation, task.Arg1, task.Arg2)
		}
		results <- 14
		<-done
		select {
		case err := <-errors:
			t.Errorf("Ожидал отсутствие ошибок, получил: %v", err)
		default:
		}
	})

	t.Run("Division by zero", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := "2/0"
		go Calc(expression, logger, tasks, results, errors, done, 2)
		err := <-errors
		if err != calc.ErrDivByZero {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrDivByZero, err)
		}
		<-done
	})

	t.Run("With letters", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := "2 + a"
		go Calc(expression, logger, tasks, results, errors, done, 3)
		err := <-errors
		if err != calc.ErrInvalidOperands {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrInvalidOperands, err)
		}
		<-done
	})

	t.Run("Empty expression", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := ""
		go Calc(expression, logger, tasks, results, errors, done, 4)
		err := <-errors
		if err != calc.ErrEmptyExpression {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrEmptyExpression, err)
		}
		<-done
	})

	t.Run("Invalid brackets", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := "2 + (3 * 4"
		go Calc(expression, logger, tasks, results, errors, done, 5)
		err := <-errors
		if err != calc.ErrInvalidBracket {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrInvalidBracket, err)
		}
		<-done
	})

	t.Run("Invalid operators", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		errors := make(chan error, 1)
		done := make(chan int, 1)
		expression := "2 + + 3"
		go Calc(expression, logger, tasks, results, errors, done, 6)
		err := <-errors
		if err != calc.ErrInvalidOperands {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrInvalidOperands, err)
		}
		<-done
	})
}

func TestCalcLvl(t *testing.T) {
	logger := zap.NewNop().Sugar()
	t.Run("Simple addition", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		n := &node{value: "+", left: &node{value: "2"}, right: &node{value: "3"}}
		results <- 5
		result, err := calcLvl(n, tasks, results, logger, 1)
		if err != nil {
			t.Errorf("Ожидал отсутствие ошибок, получил: %v", err)
		}
		if result != 5 {
			t.Errorf("Ожидал результат: 5, получил: %.2f", result)
		}
	})

	t.Run("Division by zero", func(t *testing.T) {
		t.Parallel()
		tasks := make(chan logic.Task, 1)
		results := make(chan float64, 1)
		n := &node{value: "/", left: &node{value: "2"}, right: &node{value: "0"}}
		_, err := calcLvl(n, tasks, results, logger, 2)
		if err != calc.ErrDivByZero {
			t.Errorf("Ожидал ошибку: %v, получил: %v", calc.ErrDivByZero, err)
		}
	})
}
