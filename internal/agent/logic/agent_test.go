package logic

import (
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	"sync"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		arg1         float64
		arg2         float64
		expected_num float64
	}{
		{
			name:         "plus",
			operation:    "+",
			arg1:         2,
			arg2:         2,
			expected_num: 4,
		},
		{
			name:         "plus with zero #1",
			operation:    "+",
			arg1:         0,
			arg2:         2,
			expected_num: 2,
		},
		{
			name:         "plus with zero #2",
			operation:    "+",
			arg1:         2,
			arg2:         0,
			expected_num: 2,
		},
		{
			name:         "minus",
			operation:    "-",
			arg1:         3,
			arg2:         1,
			expected_num: 2,
		},
		{
			name:         "minus with zero #1",
			operation:    "-",
			arg1:         0,
			arg2:         2,
			expected_num: -2,
		},
		{
			name:         "minus with zero #2",
			operation:    "-",
			arg1:         2,
			arg2:         0,
			expected_num: 2,
		},
		{
			name:         "minus with negative",
			operation:    "-",
			arg1:         -3,
			arg2:         1,
			expected_num: -4,
		},
		{
			name:         "minus minus negative",
			operation:    "-",
			arg1:         3,
			arg2:         -1,
			expected_num: 4,
		},
		{
			name:         "multiply",
			operation:    "*",
			arg1:         3,
			arg2:         1,
			expected_num: 3,
		},
		{
			name:         "multiply with zero #1",
			operation:    "*",
			arg1:         0,
			arg2:         3,
			expected_num: 0,
		},
		{
			name:         "multiply with zero #2",
			operation:    "*",
			arg1:         3,
			arg2:         0,
			expected_num: 0,
		},
		{
			name:         "multiply negative by positive",
			operation:    "*",
			arg1:         -3,
			arg2:         1,
			expected_num: -3,
		},
		{
			name:         "multiply negative by negative",
			operation:    "*",
			arg1:         -3,
			arg2:         -1,
			expected_num: 3,
		},
		{
			name:         "division",
			operation:    "/",
			arg1:         3,
			arg2:         1,
			expected_num: 3,
		},
		{
			name:         "division zero by not-zero",
			operation:    "/",
			arg1:         0,
			arg2:         1,
			expected_num: 0,
		},
		{
			name:         "division negative by positive",
			operation:    "/",
			arg1:         -3,
			arg2:         1,
			expected_num: -3,
		},
		{
			name:         "division positive by negative",
			operation:    "/",
			arg1:         3,
			arg2:         -1,
			expected_num: -3,
		},
		{
			name:         "division negative by negative",
			operation:    "/",
			arg1:         -3,
			arg2:         -1,
			expected_num: 3,
		},
		{
			name:         "division with res fraction",
			operation:    "/",
			arg1:         3,
			arg2:         2,
			expected_num: 1.5,
		},
		{
			name:         "division fraction by int",
			operation:    "/",
			arg1:         3,
			arg2:         1.5,
			expected_num: 2,
		},
	}
	tasks := make(chan models.Task, 5)
	results := make(chan models.Task, 5)
	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go Worker(tasks, results, wg)
	}
	opTime := time.Millisecond
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := models.Task{Operation: test.operation, Arg1: test.arg1, Arg2: test.arg2, OperationTime: opTime}
			tasks <- task
			result := <-results
			if result.Result != test.expected_num {
				t.Errorf("%s: ожидалось %.f, получил %.f", test.name, test.expected_num, result.Result)
			} else if result.OperationTime != opTime || result.Operation != test.operation || result.Arg1 != test.arg1 || result.Arg2 != test.arg2 {
				t.Error("Искажение изначальных данных")
			}
		})
	}
	//wg.Wait()
	close(tasks)
	close(results)
}
