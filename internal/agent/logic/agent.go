package logic

import (
	"sync"
	"time"
)

func Worker(tasks <-chan Task, results chan<- Task, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		var res float64
		switch task.Operation {
		case "+":
			res = task.Arg1 + task.Arg2
		case "-":
			res = task.Arg1 - task.Arg2
		case "*":
			res = task.Arg1 * task.Arg2
		case "/":
			res = task.Arg1 / task.Arg2
		}
		task.Result = res
		time.Sleep(task.OperationTime)
		results <- task
	}
}
