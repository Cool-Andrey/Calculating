package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"net/url"
	"os"
	"time"
)

type Delay struct {
	Plus     time.Duration
	Minus    time.Duration
	Multiple time.Duration
	Divide   time.Duration
}

type GRPCConfig struct {
	Host           string
	Port           int
	Ping           time.Duration
	ComputingPower int
}

type Config struct {
	Addr  string
	URLdb string
	Delay Delay
	Mode  Mode
	GRPC  GRPCConfig
}

type envConfig struct {
	URLdb      string `env:"DATABASE_URL"`
	DBHost     string `env:"DATABASE_HOST" env-default:"localhost"`
	DBPort     string `env:"DATABASE_PORT" env-default:"5432"`
	DBUser     string `env:"DATABASE_USER"`
	DBPassword string `env:"DATABASE_PASSWORD"`
	DBName     string `env:"DATABASE_NAME"`
	Delay      struct {
		Plus     int `env:"TIME_ADDITION_MS" env-default:"1000"`
		Minus    int `env:"TIME_SUBTRACTION_MS" env-default:"1000"`
		Multiple int `env:"TIME_MULTIPLICATIONS_MS" env-default:"1000"`
		Divide   int `env:"TIME_DIVISIONS_MS" env-default:"1000"`
	}
	Mode struct {
		Console   string `env:"MODE_CONSOLE" env-default:"Dev"`
		File      string `env:"MODE_FILE" env-default:"Prod"`
		CleanFile string `env:"DEL_FILE" env-default:"False"`
	}
	GRPCConfig struct {
		Host           string `env:"GRPC_HOST" env-default:"0.0.0.0"`
		Port           int    `env:"GRPC_PORT" env-default:"50051"`
		Ping           int    `env:"PING" env-default:"1000"`
		ComputingPower int    `env:"COMPUTING_POWER" env-default:"2"`
	}
}

func ConfigFromEnv(agent bool) *Config {
	var env envConfig
	err := cleanenv.ReadConfig("config.env", &env)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		log.Println("Файл конфигурации не найден, читаю из переменных окружения/ставлю стандартные значения")
		err = cleanenv.ReadEnv(&env)
		if err != nil {
			log.Fatalf("Ошибка чтения переменных окружения: %s", err)
		}
	default:
		log.Printf("Ошибка чтения файла конфигурации: %s\nЧитаю переменные окружения", err)
		err = cleanenv.ReadEnv(&env)
		if err != nil {
			log.Fatalf("Ошибка чтения переменных окружения: %s", err)
		}
	}
	if !agent {
		if env.URLdb == "" {
			if env.DBUser == "" || env.DBPassword == "" || env.DBName == "" {
				log.Fatal("Параметры DATABASE_USER, DATABASE_PASSWORD и DATABASE_NAME должны быть указаны! В ином случае заполните DATABASE_URL!")
			}
			pgURL := url.URL{
				Scheme: "postgres",
				User:   url.UserPassword(env.DBUser, env.DBPassword),
				Host:   fmt.Sprintf("%s:%s", env.DBHost, env.DBPort),
				Path:   env.DBName,
			}
			env.URLdb = pgURL.String()
		}
	}
	return &Config{
		Addr:  "8080",
		URLdb: env.URLdb,
		Delay: Delay{
			Plus:     time.Duration(env.Delay.Plus) * time.Millisecond,
			Minus:    time.Duration(env.Delay.Minus) * time.Millisecond,
			Multiple: time.Duration(env.Delay.Multiple) * time.Millisecond,
			Divide:   time.Duration(env.Delay.Divide) * time.Millisecond,
		},
		Mode: Mode{
			Console:   env.Mode.Console,
			File:      env.Mode.File,
			CleanFile: env.Mode.CleanFile,
		},
		GRPC: GRPCConfig{
			Port:           env.GRPCConfig.Port,
			Ping:           time.Duration(env.GRPCConfig.Ping) * time.Millisecond,
			ComputingPower: env.GRPCConfig.ComputingPower,
			Host:           env.GRPCConfig.Host,
		},
	}
}
