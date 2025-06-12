package application

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/ast"
	config2 "github.com/Cool-Andrey/Calculating/internal/orchestrator/config"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/repository/postgres"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/transport/grpc"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/transport/http/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"os"
	"os/signal"
)

type Application struct {
	config *config2.Config
}

func New() *Application {
	return &Application{
		config: config2.ConfigFromEnv(false),
	}
}

func (a *Application) Run(ctx context.Context) int {
	logger := config2.SetupLogger(a.config.Mode)
	defer logger.Sync()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	pool, err := pgxpool.New(context.Background(), a.config.URLdb)
	if err != nil {
		logger.Fatalf("Ошибка подключения к СУБД: %v", err)
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	if err = goose.SetDialect("postgres"); err != nil {
		logger.Fatalf("Ошибка постановки диалекта Postgres: %v", err)
	}
	if err = goose.Up(db, "db/migrations/"); err != nil {
		logger.Fatalf("Ошибка наката миграции: %v", err)
	}
	r := postgres.NewRepository(pool)
	AST := ast.NewAST(r, logger)
	g := grpc.NewServer(logger, a.config, r)
	go g.Run()
	logger.Info("Запуск gRPC сервера")
	shutdownFunc := server.Run(logger, a.config.Addr, AST, r, a.config.JWTSecret)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	err = shutdownFunc(ctx)
	if err != nil {
		logger.Errorf("Ошибка при закрытии сервера: %v", err)
		return 1
	}
	logger.Info("Сервер закрыт.")
	return 0
}
