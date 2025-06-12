package server

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/ast"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/repository/postgres"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/transport/http/server/handler"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func newHandler(logger *zap.SugaredLogger, a *ast.AST, rep *postgres.Repository, secret string) http.Handler {
	muxHandler := http.NewServeMux()
	muxHandler.HandleFunc("/api/v1/calculate", func(w http.ResponseWriter, r *http.Request) {
		handler.CalcHandler(w, r, logger, a, rep)
	})
	muxHandler.HandleFunc("/api/v1/expressions/", func(w http.ResponseWriter, r *http.Request) {
		handler.GetExpression(w, r, logger, rep)
	})
	muxHandler.HandleFunc("/api/v1/expressions", func(w http.ResponseWriter, r *http.Request) {
		handler.GetAllExpressions(w, r, logger, rep)
	})
	muxHandler.HandleFunc("/api/v1/register", func(w http.ResponseWriter, r *http.Request) {
		handler.Register(w, r, logger, rep)
	})
	muxHandler.HandleFunc("/api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		handler.Login(w, r, logger, rep, secret)
	})
	return handler.Decorate(muxHandler, JWTAuthMiddleware(logger, secret), LoggingMiddleware(logger))
}

func Run(logger *zap.SugaredLogger, addr string, a *ast.AST, r *postgres.Repository, secret string) func(ctx context.Context) error {
	Handler := newHandler(logger, a, r, secret)
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
