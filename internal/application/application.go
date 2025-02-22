package application

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/http/server"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
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
	o := orchestrator.NewOrchestator()
	shutdownFunc := server.Run(logger, a.config.Addr, o, a.config)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	<-c
	cancel()
	err := shutdownFunc(ctx)
	if err != nil {
		logger.Errorf("Ошибка при закрытии сервера: %v", err)
		return 1
	}
	logger.Info("Сервер закрыт.")
	return 0
}
