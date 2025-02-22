package server

import (
	"bytes"
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/http/server/handler"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc/safeMap"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

func new(logger *zap.SugaredLogger, o *orchestrator.Orchestator, config *config.Config) http.Handler {
	muxHandler := http.NewServeMux()
	Map := safeMap.NewSafeMap()
	muxHandler.HandleFunc("/api/v1/calculate", func(w http.ResponseWriter, r *http.Request) {
		handler.CalcHandler(w, r, logger, o, Map)
	})
	muxHandler.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		handler.GiveTask(w, r, logger, o, config.Delay, Map)
	})
	return handler.Decorate(muxHandler, loggingMiddleware(logger))
}

func Run(logger *zap.SugaredLogger, addr string, o *orchestrator.Orchestator, configVar *config.Config) func(ctx context.Context) error {
	Handler := new(logger, o, configVar)
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

func loggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				logger.Errorf("Ошибка чтения тела из логера: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			logger.Infow("HTTP запрос",
				zap.String("Метод", r.Method),
				zap.String("Путь", r.URL.String()),
				zap.String("Тело", string(bodyBytes)),
				zap.Duration("Время выполнения", duration),
			)
		})
	}
}
