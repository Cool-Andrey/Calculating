package orchestrator

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	Calc "github.com/Cool-Andrey/Calculating/internal/service/orchestrator/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
)

type Orchestrator struct{}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

func (o *Orchestrator) TakeExpression(ctx context.Context, expression string, logger *zap.SugaredLogger, id int, r *postgres.Repository) {
	Calc.Calc(ctx, expression, id, logger, r, o.handleError)
}

func (o *Orchestrator) Calculate(ctx context.Context, expression string, id int, logger *zap.SugaredLogger, r *postgres.Repository) {
	o.TakeExpression(ctx, expression, logger, id, r)
}

func (o *Orchestrator) handleError(ctx context.Context, id int, r *postgres.Repository, err error, logger *zap.SugaredLogger) {
	var status string
	var result string

	if _, ok := calc.ErrorMap[err]; ok {
		status = "Ошибка"
		result = err.Error()
	} else {
		status = "Ошибка"
		result = "Что-то пошло не так"
	}
	logger.Errorf("Ошибка: %v", err)
	if _, dbErr := r.Set(ctx, models.Expressions{
		ID:     int64(id),
		Status: status,
		Result: &result,
	}); dbErr != nil {
		logger.Errorf("Ошибка сохранения ошибки :): %v", dbErr)
	}
}
