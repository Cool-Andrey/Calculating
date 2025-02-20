package application

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/http/server"
	"os"
	"os/signal"
)

// TODO: НЕ ЗАБУДЬ ПЕРЕПИСАТЬ README
type Config struct {
	Addr string
	Mode config.Mode
}

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
	}
	config.Mode.Console = os.Getenv("MODE_CONSOLE")
	if config.Mode.Console == "" {
		config.Mode.Console = "Dev"
	}
	config.Mode.File = os.Getenv("MODE_FILE")
	if config.Mode.File == "" {
		config.Mode.File = "Prod"
	}
	config.Mode.DelFile = os.Getenv("DEL_FILE")
	if config.Mode.DelFile == "" {
		config.Mode.DelFile = "No"
	}
	return config
}

type Application struct {
	config *Config
}

func New() *Application {
	return &Application{
		config: ConfigFromEnv(),
	}
}

func (a *Application) Run(ctx context.Context) int {
	logger := config.SetupLogger(a.config.Mode)
	defer logger.Sync()
	shutdownFunc := server.Run(logger, a.config.Addr)
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
