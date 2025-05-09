package orchestrator

import (
	"context"
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	Calc "github.com/Cool-Andrey/Calculating/internal/service/orchestrator/logic"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"strconv"
)

type Orchestrator struct {
	Out      chan models.Task
	In       chan float64
	ErrorsCh chan error
	Ready    chan int
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		Out:      make(chan models.Task, 128),
		In:       make(chan float64, 128),
		ErrorsCh: make(chan error, 128),
		Ready:    make(chan int, 1),
	}
}

func (o *Orchestrator) TakeExpression(ctx context.Context, expression string, logger *zap.SugaredLogger, id int, pool *pgxpool.Pool) {
	Calc.Calc(ctx, expression, logger, o.Out, o.In, o.ErrorsCh, o.Ready, id, pool)
}

// func (o *Orchestrator) Calculate(expression string, logger *zap.SugaredLogger, id int, conn *pgxpool.Pool) { ctx := context.Background()
//
//		o.TakeExpression(expression, logger, id)
//		<-o.Ready
//		logger.Debug("Готов")
//		var result float64
//		var err error
//		if len(o.ErrorsCh) > 0 {
//			err = <-o.ErrorsCh
//		} else {
//			result = <-o.In
//		}
//		if err != nil {
//			if _, ok := calc.ErrorMap[err]; ok {
//				logger.Errorf("Ошибка счёта: %v", err)
//				_, err = postgres.Set(ctx, models.Expressions{Id: id, Status: "Ошибка", Result: err.Error()}, conn)
//				if err != nil {
//					logger.Errorf("Не получилось записать ошибку в СУБД: %v", err)
//				}
//
//			} else {
//				errJ := errors.New("Что-то пошло не так")
//				logger.Errorf("Неизвестная ошибка счёта: %v", err)
//				_, err = postgres.Set(ctx, models.Expressions{Id: id, Status: "Ошибка", Result: errJ.Error()}, conn)
//				if err != nil {
//					logger.Errorf("Не получилось записать ошибку в СУБД: %v", err)
//				}
//			}
//		} else {
//			logger.Debugf("Посчитал: %.2f", result)
//			resStr := strconv.FormatFloat(result, 'f', 2, 64)
//			_, err = postgres.Set(ctx, models.Expressions{Id: id, Status: "Выполнено", Result: resStr}, conn)
//			if err != nil {
//				logger.Errorf("Не получилось записать результат в СУБД: %v", err)
//			}
//		}
//	}
func (o *Orchestrator) Calculate(ctx context.Context, expression string, id int, logger *zap.SugaredLogger, pool *pgxpool.Pool) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	o.TakeExpression(ctx, expression, logger, id, pool)
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
		o.handleError(ctx, id, pool, context.Canceled, logger)
	case err = <-o.ErrorsCh:
		o.handleError(ctx, id, pool, err, logger)
	case res = <-o.In:
		o.handleSuccess(ctx, id, pool, res, logger)
	}
	logger.Debug("Оркестратор завершил работу")
}
func (o *Orchestrator) handleError(ctx context.Context, id int, pool *pgxpool.Pool, err error, logger *zap.SugaredLogger) {
	var status string
	var result string

	if _, ok := calc.ErrorMap[err]; ok {
		status = "Ошибка"
		result = err.Error()
	} else {
		status = "Ошибка"
		result = "Что-то пошло не так"
	}

	if _, dbErr := postgres.Set(ctx, models.Expressions{
		Id:     id,
		Status: status,
		Result: result,
	}, pool); dbErr != nil {
		logger.Error("Ошибка сохранения ошибки :)",
			zap.Error(dbErr),
			zap.Int("id", id),
			zap.String("ошибка: ", err.Error()))
	}
}

func (o *Orchestrator) handleSuccess(ctx context.Context, id int, pool *pgxpool.Pool, result float64, logger *zap.SugaredLogger) {
	resStr := strconv.FormatFloat(result, 'f', 2, 64)
	currentStatus, err := postgres.GetStatus(ctx, id, pool)
	if err != nil || currentStatus != "Подсчёт" {
		logger.Warnf("Попытка обновить неактуальную задачу ID %d", id)
		return
	}
	if _, err := postgres.Set(ctx, models.Expressions{
		Id:     id,
		Status: "Выполнено",
		Result: resStr,
	}, pool); err != nil {
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

func (o *Orchestrator) Recover(ctx context.Context, pool *pgxpool.Pool, logger *zap.SugaredLogger) {
	expressions, err := postgres.GetProcTasks(ctx, pool)
	if err != nil {
		logger.Errorf("Ошибка восстановления: %v", err)
		return
	}

	for _, expr := range expressions {
		logger.Infof("Восстановление выражения ID %d: %s", expr.ID, expr.Expression)
		ast, err := Calc.ParseASTFromJSON(expr.ASTData)
		if err != nil {
			logger.Errorf("Некорректный AST для ID %d: %v", expr.ID, err)
			o.handleError(ctx, expr.ID, pool, errors.New("Что-то пошло не так"), logger)
			continue
		}
		go Calc.Process(ctx, ast, o.Out, o.In, o.ErrorsCh, o.Ready, expr.ID, pool, logger)
	}
}

func (o *Orchestrator) Shutdown() {
	close(o.Out)
	close(o.In)
	close(o.ErrorsCh)
	close(o.Ready)
}
