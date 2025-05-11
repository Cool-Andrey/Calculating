package models

import (
	"time"
)

type Task struct {
	Id            int64   `json:"id"`
	Operation     string  `json:"operation"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Result        float64
	OperationTime time.Duration `json:"operation_time"`
}

type TaskWrapper struct {
	Task Task `json:"task"`
}

type Expressions struct {
	Id     int64  `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type Expression struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Result     string `json:"result"`
	Expression string `json:"expression"`
	ASTData    []byte `json:"ast_data"`
}

type ASTNode struct {
	Value     string   `json:"value"`
	Left      *ASTNode `json:"left,omitempty"`
	Right     *ASTNode `json:"right,omitempty"`
	Result    float64  `json:"result"`
	Processed bool     `json:"processed"`
}
