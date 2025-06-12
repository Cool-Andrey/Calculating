package transport

import (
	"context"
	"github.com/Cool-Andrey/Calculating/internal/models"
	pb "github.com/Cool-Andrey/Calculating/pkg/api/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"sync"
	"time"
)

type Client struct {
	client  pb.OrchestratorClient
	In      chan models.Task
	Results chan models.Task
	logger  *zap.SugaredLogger
	Ping    time.Duration
	Port    int
}

func NewAgent(cntGoroutines int, logger *zap.SugaredLogger, ping time.Duration, port int, client *grpc.ClientConn) *Client {
	return &Client{
		client:  pb.NewOrchestratorClient(client),
		In:      make(chan models.Task, cntGoroutines),
		Results: make(chan models.Task, cntGoroutines),
		logger:  logger,
		Ping:    ping,
		Port:    port,
	}
}

func (c *Client) getTask(ctx context.Context, stream grpc.BidiStreamingClient[pb.TaskWithResult, pb.Task]) error {
	defer close(c.In)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				c.logger.Info("Стрим закрыт.")
				return nil
			}
			if err != nil {
				c.logger.Errorf("Ошибка получения задачи: %v", err)
				return err
			}
			c.logger.Debugf("Получил задачу: %+v", msg)
			id, err := uuid.Parse(msg.ID)
			if err != nil {
				c.logger.Errorf("Ошибка преобразования string в uuid: %v", err)
				return err
			}
			c.In <- models.Task{
				ID:            id,
				Operation:     msg.Operation,
				Arg1:          msg.Arg1,
				Arg2:          msg.Arg2,
				OperationTime: time.Duration(msg.OperationTime) * time.Millisecond,
			}
		}
	}
}

func (c *Client) sendTask(ctx context.Context, stream grpc.BidiStreamingClient[pb.TaskWithResult, pb.Task]) error {
	defer close(c.Results)
	ticker := time.NewTicker(c.Ping)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			select {
			case task, ok := <-c.Results:
				if !ok {
					return nil
				}
				err := stream.Send(&pb.TaskWithResult{
					ID:        task.ID.String(),
					Operation: task.Operation,
					Arg1:      task.Arg1,
					Arg2:      task.Arg2,
					Result:    task.Result,
				})
				if err != nil {
					c.logger.Errorf("Ошибка отправки задачи: %v", err)
					return err
				}
				c.logger.Debugf("Отправил задачу: %+v", task)
			default:
			}
		}
	}
}

func (c *Client) Run(ctx context.Context) {
	stream, err := c.client.GiveTakeTask(ctx)
	if err != nil {
		c.logger.Fatalf("Ошибка запуска: %v", err)
	}
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	var wg sync.WaitGroup
	errCh := make(chan error)
	go func() {
		wg.Add(1)
		defer wg.Done()
		errCh <- c.getTask(ctx, stream)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		errCh <- c.sendTask(ctx, stream)
	}()
	for err := range errCh {
		if err != nil {
			c.logger.Fatalf("Ошибка gRPC: %v", err)
		}
	}
}
