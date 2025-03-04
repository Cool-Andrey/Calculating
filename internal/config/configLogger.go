package config

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Mode struct {
	Console   string
	File      string
	CleanFile string
}

func SetupLogger(newConfig Mode) *zap.SugaredLogger {
	var config, configFile zap.Config
	var consoleEncoder, fileEncoder zapcore.Encoder
	if strings.EqualFold(newConfig.Console, "dev") {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // А чо, красиво жить не запретишь)
		consoleEncoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
	} else if strings.EqualFold(newConfig.Console, "prod") {
		config = zap.NewProductionConfig()
		consoleEncoder = zapcore.NewJSONEncoder(config.EncoderConfig)
	} else {
		log.Fatal("Проверьте значение глобальной переменной MODE_CONSOLE. Подробнее в README")
	}
	config.EncoderConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%.3fµs", float64(d)/1000))
	}
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	if strings.EqualFold(os.Getenv("WRITE_FILE"), "false") {
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			config.Level,
		)
		logger := zap.New(consoleCore)
		return logger.Sugar()
	}
	if strings.EqualFold(newConfig.File, "dev") {
		configFile = zap.NewDevelopmentConfig()
		fileEncoder = zapcore.NewConsoleEncoder(configFile.EncoderConfig)
	} else if strings.EqualFold(newConfig.File, "prod") {
		configFile = zap.NewProductionConfig()
		fileEncoder = zapcore.NewJSONEncoder(configFile.EncoderConfig)
	} else {
		log.Fatal("Проверьте значение глобальной переменной MODE_FILE. Подробнее в README")
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
	} else if err != nil {
		log.Fatalf("Не удалось проверить наличие дирректории %v", err)
	}
	var permision int
	if strings.EqualFold(newConfig.CleanFile, "false") {
		permision = os.O_APPEND
	} else if strings.EqualFold(newConfig.CleanFile, "true") {
		permision = os.O_TRUNC
	} else {
		log.Fatal("Проверьте значение глобальной переменной CLEAN_FILE. Подробнее в README")
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|permision, os.ModePerm)
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
