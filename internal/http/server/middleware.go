package server

import (
	"bytes"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

func LoggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
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
			if r.Method == http.MethodGet && r.URL.Path != "/internal/task" {
				logger.Infow("HTTP запрос", zap.String("Метод", r.Method),
					zap.String("Путь", r.URL.String()),
					zap.Duration("Время выполнения", duration),
				)
			} else if r.URL.Path != "/internal/task" && r.Method == http.MethodGet {
				logger.Infow("HTTP запрос", zap.String("Метод", r.Method),
					zap.String("Путь", r.URL.String()),
					zap.String("Тело", string(bodyBytes)),
					zap.Duration("Время выполнения", duration),
				)
			}
		})
	}
}
