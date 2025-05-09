package main

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/agent/transport"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"os/signal"
	"sync"
)

type AgentConfig struct {
	ComputingPower int    `env:"COMPUTING_POWER" env-default:"2"`
	ServerAddr     string `env:"URL" env-default:"http://127.0.0.1:8080"`
	Ping           int    `env:"PING" env-default:"1000"`
}

func NewAgentConfig() *AgentConfig {
	var env AgentConfig
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
	env.ServerAddr += "/internal/task"
	return &env
}

func main() {
	config := NewAgentConfig()
	agent := transport.NewAgent(config.ComputingPower)
	wg := &sync.WaitGroup{}
	for i := 0; i < config.ComputingPower; i++ {
		wg.Add(1)
		go logic.Worker(agent.In, agent.Results, wg)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	agent.Start(config.ServerAddr, config.Ping, ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	cancel()
	agent.Shutdown()
	os.Exit(0)
}
