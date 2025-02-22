package orchestrator

import (
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	Calc "github.com/Cool-Andrey/Calculating/internal/service/orchestrator/logic"
	"go.uber.org/zap"
)

type Orchestator struct {
	Out      chan logic.Task
	In       chan float64
	ErrorsCh chan error
	Ready    chan int
}

func NewOrchestator() *Orchestator {
	return &Orchestator{
		Out:      make(chan logic.Task, 128),
		In:       make(chan float64, 128),
		ErrorsCh: make(chan error, 128),
		Ready:    make(chan int, 1),
	}
}

func (o *Orchestator) TakeExpression(expression string, logger *zap.SugaredLogger, id int) {
	Calc.Calc(expression, logger, o.Out, o.In, o.ErrorsCh, o.Ready, id)
}
