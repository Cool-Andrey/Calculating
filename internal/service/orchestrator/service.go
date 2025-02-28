package orchestrator

import (
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	Calc "github.com/Cool-Andrey/Calculating/internal/service/orchestrator/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/Cool-Andrey/Calculating/pkg/calc/safeStructures"
	"go.uber.org/zap"
	"strconv"
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

func (o *Orchestator) Calculate(expression string, logger *zap.SugaredLogger, id int, Map *safeStructures.SafeMap) {
	o.TakeExpression(expression, logger, id)
	<-o.Ready
	logger.Debug("Готов")
	var result float64
	var err error
	if len(o.ErrorsCh) > 0 {
		err = <-o.ErrorsCh
	} else {
		result = <-o.In
	}
	if err != nil {
		if _, ok := calc.ErrorMap[err]; ok {
			logger.Errorf("Ошибка счёта: %v", err)
			Map.Set(id, safeStructures.Expressions{Id: id, Status: "Выполнено", Result: err.Error()})
		} else {
			errJ := errors.New("Что-то пошло не так")
			logger.Errorf("Неизвестная ошибка счёта: %v", err)
			Map.Set(id, safeStructures.Expressions{Id: id, Status: "Выполнено", Result: errJ.Error()})
		}
	} else {
		logger.Debugf("Посчитал: %.2f", result)
		resStr := strconv.FormatFloat(result, 'f', 2, 64)
		Map.Set(id, safeStructures.Expressions{Id: id, Status: "Выполнено", Result: resStr})
	}
}

func (o *Orchestator) Shutdown() {
	close(o.Out)
	close(o.In)
	close(o.ErrorsCh)
	close(o.Ready)
}
