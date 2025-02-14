package application

import (
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/http/server/handler"
	"net/http"
	"os"
)

type Config struct {
	Addr string
}

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
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

func (a *Application) RunServer() error {
	http.HandleFunc("/api/v1/calculate", handler.CalcHandler)
	fmt.Printf("Сервер слушается на порту: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, nil)
}
