package application

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/http/server"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
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
		config: config.ConfigFromEnv(),
	}
}

func (a *Application) Run(ctx context.Context) int {
	logger := config.SetupLogger(a.config.Mode)
	defer logger.Sync()
	ctx, cancel := context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
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
	if err = goose.Up(db, "internal/db/migrations"); err != nil {
		logger.Fatalf("Ошибка наката миграции: %v", err)
	}
	o.Recover(ctx, pool, logger)
	shutdownFunc := server.Run(logger, a.config.Addr, o, a.config, pool)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer stop()

	<-c
	cancel()
	err = shutdownFunc(ctx)
	if err != nil {
		logger.Errorf("Ошибка при закрытии сервера: %v", err)
		return 1
	}
	o.Shutdown()
	logger.Info("Сервер закрыт.")
	return 0
}
