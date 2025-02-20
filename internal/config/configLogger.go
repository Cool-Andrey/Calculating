package config

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Mode struct {
	Console string
	File    string
	DelFile string
}

func SetupLogger(newConfig Mode) *zap.SugaredLogger {
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
