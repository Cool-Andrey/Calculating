package main

import (
	"context"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/agent/transport"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"os/signal"
	"sync"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	conf := config.ConfigFromEnv()
	logger := config.SetupLogger(conf.Mode)
	cred := grpc.WithTransportCredentials(insecure.NewCredentials())
	addr := fmt.Sprintf("0.0.0.0:%d", conf.GRPC.Port)
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
