package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	pb "github.com/Cool-Andrey/Calculating/pkg/api/proto"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	r      *postgres.Repository
}

func NewServer(logger *zap.SugaredLogger, cfg *config.Config, r *postgres.Repository) *Server {
	grpcSrv := grpc.NewServer()
	srv := &Server{
		server: grpcSrv,
		logger: logger,
		cfg:    cfg.GRPC,
		delay:  cfg.Delay,
		r:      r,
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
			task, err := s.r.GetTask(ctx)
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				s.logger.Errorf("Ошибка получения из СУБД задачи: %v", err)
				return err
			}
			if task.Arg2 == 0 && task.Operation == "/" {
				errEvaluate := calc.ErrDivByZero.Error()
				_, err = s.r.Set(ctx, models.Expressions{
					ID:     int64(task.ExpressionID),
					Status: "Ошибка",
					Result: &errEvaluate,
				})
			}
			setOperationTime(&task, s.delay)
			err = stream.Send(&pb.Task{
				ID:            task.ID.String(),
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
			id, err := uuid.Parse(msg.ID)
			if err != nil {
				s.logger.Errorf("Ошибка преобразования string в uuid: %v", err)
				return err
			}
			task := &models.Task{
				ID:        id,
				Operation: msg.Operation,
				Arg1:      msg.Arg1,
				Arg2:      msg.Arg2,
				Result:    msg.Result,
			}
			err = s.r.UpdateTask(ctx, task)
			if err != nil {
				s.logger.Errorf("Ошибка обновления задачи в СУБД: %v", err)
				return err
			}
			err = s.r.RemoveTask(ctx, id)
			if err != nil {
				s.logger.Errorf("Ошибка удаления задачи в СУБД: %v", err)
			}
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
