package main

import (
	"context"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/agent/transport"
	config2 "github.com/Cool-Andrey/Calculating/internal/orchestrator/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"os/signal"
	"sync"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	conf := config2.ConfigFromEnv(true)
	logger := config2.SetupLogger(conf.Mode)
	cred := grpc.WithTransportCredentials(insecure.NewCredentials())
	addr := fmt.Sprintf("%s:%d", conf.GRPC.Host, conf.GRPC.Port)
	logger.Debugf("Запускаю стрим на адрес: %s", addr)
	client, err := grpc.NewClient(addr, cred)
	if err != nil {
		logger.Fatalf("Ошибка создания клиента: %v", err)
	}
	agent := transport.NewAgent(conf.GRPC.ComputingPower, logger, conf.GRPC.Ping, conf.GRPC.Port, client)
	wg := &sync.WaitGroup{}
	for i := 0; i < conf.GRPC.ComputingPower; i++ {
		wg.Add(1)
		go logic.Worker(agent.In, agent.Results, wg)
	}
	agent.Run(ctx)
	<-ctx.Done()
}
