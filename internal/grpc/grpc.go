package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	pb "github.com/Cool-Andrey/Calculating/pkg/api/proto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	pb.UnimplementedOrchestratorServer
	server *grpc.Server
	logger *zap.SugaredLogger
	cfg    config.GRPCConfig
	delay  config.Delay
	get    <-chan models.Task
	insert chan<- float64
	pool   *pgxpool.Pool
}

func NewServer(logger *zap.SugaredLogger, cfg *config.Config, get <-chan models.Task, insert chan<- float64, pool *pgxpool.Pool) *Server {
	grpcSrv := grpc.NewServer()
	srv := &Server{
		server: grpcSrv,
		logger: logger,
		cfg:    cfg.GRPC,
		delay:  cfg.Delay,
		get:    get,
		insert: insert,
		pool:   pool,
	}
	pb.RegisterOrchestratorServer(grpcSrv, srv)
	return srv
}

func setOperationTime(task *models.Task, delay config.Delay) {
	switch task.Operation {
	case "+":
		task.OperationTime = delay.Plus
	case "-":
		task.OperationTime = delay.Minus
	case "*":
		task.OperationTime = delay.Multiple
	case "/":
		task.OperationTime = delay.Divide
	}
}

func (s *Server) sendTask(ctx context.Context, stream grpc.BidiStreamingServer[pb.TaskWithResult, pb.Task]) error {
	ticker := time.NewTicker(s.cfg.Ping)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			task := <-s.get
			setOperationTime(&task, s.delay)
			err := stream.Send(&pb.Task{
				ID:            task.Id,
				Operation:     task.Operation,
				Arg1:          task.Arg1,
				Arg2:          task.Arg2,
				OperationTime: task.OperationTime.Milliseconds(),
			})
			if err != nil {
				s.logger.Errorf("Ошибка отправки задачи: %v", err)
				return err
			}
		}
	}
}

func (s *Server) getTask(ctx context.Context, stream grpc.BidiStreamingServer[pb.TaskWithResult, pb.Task]) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				s.logger.Info("Стрим закрыт.")
				return nil
			}
			if err != nil {
				s.logger.Errorf("Ошибка получения задачи: %v", err)
				return err
			}
			task := &models.Task{
				Id:        msg.ID,
				Operation: msg.Operation,
				Arg1:      msg.Arg1,
				Arg2:      msg.Arg2,
				Result:    msg.Result,
			}
			status, err := postgres.GetStatus(ctx, task.Id, s.pool)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					s.logger.Warn("Пришла задача для несуществующего выражения")
				} else {
					s.logger.Errorf("Ошибка запроса статуса выражения к СУБД: %v", err)
					return err
				}
			}
			if status != "Подсчёт" {
				s.logger.Warnf("Пришла задача для выполненного выражения ID: %d", task.Id)
			}
			s.insert <- task.Result
		}
	}
}

func (s *Server) GiveTakeTask(stream grpc.BidiStreamingServer[pb.TaskWithResult, pb.Task]) error {
	ctx := stream.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		errCh <- s.sendTask(ctx, stream)
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()
		errCh <- s.getTask(ctx, stream)
	}()
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Run() {
	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Fatalf("Не смог занять канал: %v", err)
	}
	if err := s.server.Serve(l); err != nil {
		s.logger.Fatalf("Ошибка gRPC: %v", err)
	}

}

func (s *Server) Shutdown() {
	s.server.GracefulStop()
}
