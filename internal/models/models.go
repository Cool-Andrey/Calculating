package models

import (
	"github.com/google/uuid"
	"time"
)

type Task struct {
	ID            uuid.UUID `json:"id"`
	Operation     string    `json:"operation"`
	Arg1          float64   `json:"arg1"`
	Arg2          float64   `json:"arg2"`
	Result        float64
	ExpressionID  int
	LeftID        *uuid.UUID
	RightID       *uuid.UUID
	OperationTime time.Duration `json:"operation_time"`
}

type TaskWrapper struct {
	Task Task `json:"task"`
}

type Expressions struct {
	ID     int64   `json:"id"`
	Status string  `json:"status"`
	Result *string `json:"result"`
}
