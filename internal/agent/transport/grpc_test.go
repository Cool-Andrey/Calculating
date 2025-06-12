package transport

import (
	"context"
	"errors"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	pb "github.com/Cool-Andrey/Calculating/pkg/api/proto"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
	"io"
	"testing"
	"time"
)

type stubStream struct {
	recvMsgs []*pb.Task
	sendMsgs []*pb.TaskWithResult
	recvErr  error
	sendErr  error
}

func (s *stubStream) Header() (metadata.MD, error) { return nil, nil }
func (s *stubStream) Trailer() metadata.MD         { return nil }
func (s *stubStream) CloseSend() error             { return nil }
func (s *stubStream) Context() context.Context     { return context.Background() }
func (s *stubStream) SendMsg(m interface{}) error  { return nil }
func (s *stubStream) RecvMsg(m interface{}) error  { return nil }

func (s *stubStream) Recv() (*pb.Task, error) {
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	if len(s.recvMsgs) == 0 {
		return nil, io.EOF
	}
	msg := s.recvMsgs[0]
	s.recvMsgs = s.recvMsgs[1:]
	return msg, s.recvErr
}

func (s *stubStream) Send(m *pb.TaskWithResult) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sendMsgs = append(s.sendMsgs, m)
	return nil
}

func TestGetTask_Success(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	client := Client{In: make(chan models.Task, 1), logger: logger}
	stub := &stubStream{
		recvMsgs: []*pb.Task{
			{
				ID:            1,
				Operation:     "+",
				Arg1:          2,
				Arg2:          3,
				OperationTime: 100,
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := client.getTask(ctx, stub)
	if err != nil {
		t.Fatalf("Ошибка getTask: %v", err)
	}
	task, ok := <-client.In
	if !ok {
		t.Fatal("Ожидал задачу в канале, а канал закрыт :( ")
	}
	if task.ID != 1 || task.Operation != "+" {
		t.Errorf("Неизвестная задача: %+v", task)
	}
}

func TestGetTask_Fail(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	client := Client{In: make(chan models.Task, 1), logger: logger}
	stub := &stubStream{recvErr: errors.New("recv error")}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := client.getTask(ctx, stub)
	if err == nil {
		t.Fatal("Ожидал ошибку, а её нема :(")
	}
	if err.Error() != "recv error" {
		t.Fatalf("Ожидал recv error, получил %v", err)
	}
}

func TestSendTask_Success(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	client := Client{Results: make(chan models.Task, 1), logger: logger, Ping: time.Millisecond}
	stub := &stubStream{}
	select {
	case client.Results <- models.Task{
		ID:        1,
		Operation: "+",
		Arg1:      2,
		Arg2:      3,
		Result:    5,
	}:
	default:
		t.Fatal("Не удалось отправить задачу в канал")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		err := client.sendTask(ctx, stub)
		if err != nil {
			t.Fatalf("Ошибка sendTask: %v", err)
		}
	}()
	<-stop
	if len(stub.sendMsgs) != 1 {
		t.Fatalf("Ожидал 1 задачу, получил %d", len(stub.sendMsgs))
	}
	sent := stub.sendMsgs[0]
	if sent.Operation != "+" || sent.Arg1 != 2 || sent.Arg2 != 3 || sent.Result != 5 {
		t.Errorf("Неизвестная задача: %v", sent)
	}
}

func TestSendTask_Fail(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	client := Client{Results: make(chan models.Task, 1), logger: logger, Ping: time.Millisecond}
	stub := &stubStream{sendErr: errors.New("send error")}
	client.Results <- models.Task{
		ID:        3,
		Operation: "-",
		Arg1:      2,
		Arg2:      3,
		Result:    -1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := client.sendTask(ctx, stub)
	if err == nil {
		t.Fatal("Ожижал send error, получил ничего :(")
	}
	if err.Error() != "send error" {
		t.Fatalf("Ожидал send error, получил %v", err)
	}
}
