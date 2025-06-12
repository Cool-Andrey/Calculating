package application

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/internal/transport/grpc"
	"github.com/Cool-Andrey/Calculating/internal/transport/http/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"os"
	"os/signal"
)

type Application struct {
	config *config.Config
}

func New() *Application {
	return &Application{
		config: config.ConfigFromEnv(false),
	}
}

func (a *Application) Run(ctx context.Context) int {
	logger := config.SetupLogger(a.config.Mode)
	defer logger.Sync()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	o := orchestrator.NewOrchestrator()
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
	g := grpc.NewServer(logger, a.config, r)
	go g.Run()
	logger.Info("Запуск gRPC сервера")
	shutdownFunc := server.Run(logger, a.config.Addr, o, r, a.config.JWTSecret)
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
