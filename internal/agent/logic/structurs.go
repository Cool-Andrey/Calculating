package logic

import "time"

type Task struct {
	Id            int     `json:"id"`
	Operation     string  `json:"operation"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Result        float64
	OperationTime time.Duration `json:"operation_time"`
}

type Result struct {
	Id     int     `json:"id"`
	Result float64 `json:"result"`
}
type TaskWrapper struct {
	Task Task `json:"task"`
}
