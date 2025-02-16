package application

import (
	"context"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/http/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

type Config struct {
	Addr string
	Stat string
}

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
	}
	config.Stat = os.Getenv("MODE")
	if config.Stat == "" {
		config.Stat = "Dev"
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
	logger := setupLogger(a.config.Stat)
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

func setupLogger(newConfig string) *zap.SugaredLogger {
	var config zap.Config
	if newConfig == "Dev" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // А чо, красиво жить не запретишь)
	} else if newConfig == "Prod" {
		config = zap.NewProductionConfig()
	} else {
		log.Fatal("Проверьте значение глобальной переменной MODE. Читай README")
	}
	config.EncoderConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%.3fµs", float64(d)/1000))
	}
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	configFile := zap.NewProductionConfig()
	configFile.EncoderConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%.3fµs", float64(d)/1000))
	}
	configFile.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	logDir := "log"
	logName := "server.log"
	logPath := filepath.Join(logDir, logName)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0777)
		if err != nil {
			log.Fatalf("Ошибка создания папки для хранения логов: %v", err)
		}
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Проблема с созданием/открытием файла лога: %v", err)
	}
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(os.Stdout),
		config.Level,
	)
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(file),
		configFile.Level,
	)
	core := zapcore.NewTee(consoleCore, fileCore)
	logger := zap.New(core)
	return logger.Sugar()
}
