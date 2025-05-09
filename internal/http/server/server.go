package server

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/http/server/handler"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func newHandler(logger *zap.SugaredLogger, o *orchestrator.Orchestrator, config *config.Config, pool *pgxpool.Pool) http.Handler {
	muxHandler := http.NewServeMux()
	muxHandler.HandleFunc("/api/v1/calculate", func(w http.ResponseWriter, r *http.Request) {
		handler.CalcHandler(w, r, logger, o, pool)
	})
	muxHandler.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		handler.GiveTask(w, r, logger, o, config.Delay, pool)
	})
	muxHandler.HandleFunc("/api/v1/expressions/", func(w http.ResponseWriter, r *http.Request) {
		handler.GetExpression(w, r, logger, pool)
	})
	muxHandler.HandleFunc("/api/v1/expressions", func(w http.ResponseWriter, r *http.Request) {
		handler.GetAllExpressions(w, r, logger, pool)
	})
	return handler.Decorate(muxHandler, LoggingMiddleware(logger))
}

func Run(logger *zap.SugaredLogger, addr string, o *orchestrator.Orchestrator, configVar *config.Config, pool *pgxpool.Pool) func(ctx context.Context) error {
	Handler := newHandler(logger, o, configVar, pool)
	server := &http.Server{Addr: ":" + addr, Handler: Handler}
	ch := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			ch <- err
		}
	}()
	select {
	case err := <-ch:
		if err != nil {
			logger.Error("Ошибка запуска сервера ",
				zap.String("Ошибочка", err.Error()))
		}
	case <-time.After(100 * time.Millisecond):
		logger.Infof("Сервер слушается на порту: %s", server.Addr[1:])
	}
	return server.Shutdown
}
