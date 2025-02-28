package transport

import (
	"context"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func mockServer(task logic.TaskWrapper, t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(task)
		} else if r.Method == http.MethodPost {
			defer r.Body.Close()
			var resTask logic.TaskWrapper
			if err := json.NewDecoder(r.Body).Decode(&resTask); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if resTask != task {
				t.Errorf("Ожидал %v, получил %v", task, resTask)
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
}

func TestTransport_Shutdown(t *testing.T) {
	os.Setenv("PING", "100")
	os.Setenv("COMPUTING_POWER", "2")
	defer os.Unsetenv("PING")
	defer os.Unsetenv("COMPUTING_POWER")
	agent := NewAgent(1)
	ctx, cancel := context.WithCancel(context.Background())
	agent.Start("", ctx)
	time.Sleep(1 * time.Second)
	cancel()
	agent.Shutdown()
	time.Sleep(1 * time.Second)
	select {
	case _, ok := <-agent.In:
		if ok {
			t.Error("Канал In не был закрыт")
		}
	case _, ok := <-agent.Results:
		if ok {
			t.Error("Канал Results не был закрыт")
		}
	}
}

func TestTranport_StartGet(t *testing.T) {
	os.Setenv("PING", "100")
	os.Setenv("COMPUTING_POWER", "2")
	defer os.Unsetenv("PING")
	defer os.Unsetenv("COMPUTING_POWER")
	task := logic.TaskWrapper{
		Task: logic.Task{
			Id:            1,
			Operation:     "+",
			Arg1:          1,
			Arg2:          2,
			OperationTime: 0,
		},
	}
	server := mockServer(task, t)
	time.Sleep(1 * time.Second)
	defer server.Close()
	agent := NewAgent(1)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	agent.Start(server.URL, ctx)
	time.Sleep(1 * time.Second)
	cancel()
	select {
	case resTask := <-agent.In:
		agent.Shutdown()
		if resTask != task.Task {
			t.Errorf("Ожидал %v, получил %v", task.Task, resTask)
		}
	default:
		agent.Shutdown()
		t.Error("Ждал ждал но не дождался результат(")
	}
}

func TestTranport_StartPost(t *testing.T) {
	os.Setenv("PING", "100")
	os.Setenv("COMPUTING_POWER", "2")
	defer os.Unsetenv("PING")
	defer os.Unsetenv("COMPUTING_POWER")
	task := logic.TaskWrapper{
		Task: logic.Task{
			Id:            1,
			Operation:     "+",
			Arg1:          1,
			Arg2:          2,
			OperationTime: 0,
			Result:        3,
		},
	}
	server := mockServer(task, t)
	agent := NewAgent(1)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	agent.Start(server.URL, ctx)
	agent.Results <- task.Task
	time.Sleep(1 * time.Second)
	server.Close()
	cancel()
	agent.Shutdown()
}
