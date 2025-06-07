package orchestrator

import (
	"context"
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	Calc "github.com/Cool-Andrey/Calculating/internal/service/orchestrator/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
	"strconv"
)

type Orchestrator struct {
	Out      chan models.Task
	In       chan float64
	Res      chan float64
	ErrorsCh chan error
	Ready    chan int
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		Out:      make(chan models.Task, 128),
		In:       make(chan float64, 128),
		Res:      make(chan float64, 128),
		ErrorsCh: make(chan error, 128),
		Ready:    make(chan int, 1),
	}
}

func (o *Orchestrator) TakeExpression(ctx context.Context, expression string, logger *zap.SugaredLogger, id int, r *postgres.Repository) {
	Calc.Calc(ctx, expression, logger, o.Out, o.In, o.Res, o.ErrorsCh, o.Ready, id, r)
}

func (o *Orchestrator) Calculate(ctx context.Context, expression string, id int, logger *zap.SugaredLogger, r *postgres.Repository) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go o.TakeExpression(ctx, expression, logger, id, r)
	select {
	case <-ctx.Done():
		logger.Warn("Преждевременная отмена")
	case <-o.Ready:
		logger.Debugf("Готов: %d", id)
	}
	var res float64
	var err error
	select {
	case <-ctx.Done():
		logger.Warn("Отмена во время записи результата")
		o.handleError(ctx, id, r, context.Canceled, logger)
	case err = <-o.ErrorsCh:
		o.handleError(ctx, id, r, err, logger)
	case res = <-o.Res:
		o.handleSuccess(ctx, id, r, res, logger)
	}
	logger.Debug("Оркестратор завершил работу")
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

	if _, dbErr := r.Set(ctx, models.Expressions{
		Id:     int64(id),
		Status: status,
		Result: result,
	}); dbErr != nil {
		logger.Errorf("Ошибка сохранения ошибки :): %v", dbErr)
	}
}

func (o *Orchestrator) handleSuccess(ctx context.Context, id int, r *postgres.Repository, result float64, logger *zap.SugaredLogger) {
	resStr := strconv.FormatFloat(result, 'f', 2, 64)
	currentStatus, err := r.GetStatus(ctx, int64(id))
	if err != nil || currentStatus != "Подсчёт" {
		logger.Warnf("Попытка обновить неактуальную задачу ID %d", id)
		return
	}
	if _, err := r.Set(ctx, models.Expressions{
		Id:     int64(id),
		Status: "Выполнено",
		Result: resStr,
	}); err != nil {
		logger.Error("Ошибка сохранения успешного результата",
			zap.Error(err),
			zap.Int("id", id),
			zap.Float64("результат", result))
	} else {
		logger.Debug("Успешно сохранено выражение",
			zap.Int("id", id),
			zap.Float64("результат", result))
	}
}

func (o *Orchestrator) Recover(ctx context.Context, r *postgres.Repository, logger *zap.SugaredLogger) {
	expressions, err := r.GetProcTasks(ctx)
	if err != nil {
		logger.Errorf("Ошибка восстановления: %v", err)
		return
	}

	for _, expr := range expressions {
		logger.Infof("Восстановление выражения ID %d: %s", expr.ID, expr.Expression)
		ast, err := Calc.ParseASTFromJSON(expr.ASTData)
		if err != nil {
			logger.Errorf("Некорректный AST для ID %d: %v", expr.ID, err)
			o.handleError(ctx, int(expr.ID), r, errors.New("Что-то пошло не так"), logger)
			continue
		}
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			go Calc.Process(ctx, ast, o.Out, o.In, o.Res, o.ErrorsCh, o.Ready, int(expr.ID), logger)
			select {
			case <-ctx.Done():
				logger.Warn("Преждевременная отмена")
			case <-o.Ready:
				logger.Debugf("Готов: %d", expr.ID)
			}
			var res float64
			var err error
			select {
			case <-ctx.Done():
				logger.Warn("Отмена во время записи результата")
				o.handleError(ctx, int(expr.ID), r, context.Canceled, logger)
			case err = <-o.ErrorsCh:
				o.handleError(ctx, int(expr.ID), r, err, logger)
			case res = <-o.Res:
				o.handleSuccess(ctx, int(expr.ID), r, res, logger)
			}
			logger.Debug("Оркестратор завершил работу")
		}()
	}
}

func (o *Orchestrator) Shutdown() {
	close(o.Out)
	close(o.In)
	close(o.ErrorsCh)
	close(o.Ready)
}
