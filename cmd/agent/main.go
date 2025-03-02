package main

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/agent/transport"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
)

func main() {
	cntGoroutinsStr := os.Getenv("COMPUTING_POWER")
	if cntGoroutinsStr == "" {
		log.Fatal("Укажите количество запускаемых потоков(переменную COMPUTING_POWER)")
	}
	cntGoroutins, err := strconv.Atoi(cntGoroutinsStr)
	if err != nil {
		log.Fatalf("Не удалось преобразовать значение COMPUTING_POWER в число: %v", err)
	}
	agent := transport.NewAgent(cntGoroutins)
	//conf := config.ConfigFromEnv()
	//address := "http://localhost:" + conf.Addr + "/internal/task"
	url := os.Getenv("URL")
	if url == "" {
		url = "http://127.0.0.1:8080"
	}
	address := url + "/internal/task"
	wg := &sync.WaitGroup{}
	for i := 0; i < cntGoroutins; i++ {
		wg.Add(1)
		go logic.Worker(agent.In, agent.Results, wg)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	agent.Start(address, ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	cancel()
	agent.Shutdown()
	os.Exit(0)
}
