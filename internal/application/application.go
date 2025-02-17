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

type Mode struct {
	Console string
	File    string
	DelFile string
}

type Config struct {
	Addr string
	Mode Mode
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
	logger := setupLogger(a.config.Mode)
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

func setupLogger(newConfig Mode) *zap.SugaredLogger {
	var config, configFile zap.Config
	var consoleEncoder, fileEncoder zapcore.Encoder
	if newConfig.Console == "Dev" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // А чо, красиво жить не запретишь)
		consoleEncoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
	} else if newConfig.Console == "Prod" {
		config = zap.NewProductionConfig()
		consoleEncoder = zapcore.NewJSONEncoder(config.EncoderConfig)
	} else {
		log.Fatal("Проверьте значение глобальной переменной MODE_CONSOLE. Читай README")
	}
	config.EncoderConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%.3fµs", float64(d)/1000))
	}
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	if newConfig.File == "Dev" {
		configFile = zap.NewDevelopmentConfig()
		fileEncoder = zapcore.NewConsoleEncoder(configFile.EncoderConfig)
	} else if newConfig.File == "Prod" {
		configFile = zap.NewProductionConfig()
		fileEncoder = zapcore.NewJSONEncoder(configFile.EncoderConfig)
	} else {
		log.Fatal("Проверьте значение глобальной переменной MODE_FILE. Чита README")
	}
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
	var permision int
	if newConfig.DelFile == "No" {
		permision = os.O_APPEND
	} else if newConfig.DelFile == "Yes" {
		permision = os.O_TRUNC
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|permision, 0666)
	if err != nil {
		log.Fatalf("Проблема с созданием/открытием файла лога: %v", err)
	}
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		config.Level,
	)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		configFile.Level,
	)
	core := zapcore.NewTee(consoleCore, fileCore)
	logger := zap.New(core)
	return logger.Sugar()
}
