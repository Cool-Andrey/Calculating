package logic

import (
	"github.com/Cool-Andrey/Calculating/internal/models"
	"sync"
	"time"
)

func Worker(tasks <-chan models.Task, results chan<- models.Task, wg *sync.WaitGroup) {
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
