package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Transport struct {
	In      chan logic.Task
	Results chan logic.Task
}

func NewAgent(cntGoroutins int) *Transport {
	return &Transport{
		In:      make(chan logic.Task, cntGoroutins),
		Results: make(chan logic.Task, cntGoroutins),
	}
}

func (t *Transport) Start(address string, ctx context.Context) {
	configLog := config.ConfigFromEnv()
	logger := config.SetupLogger(configLog.Mode)
	pingStr := os.Getenv("PING")
	if pingStr == "" {
		pingStr = "1000"
	}
	ping, err := strconv.Atoi(pingStr)
	if err != nil {
		logger.Fatalf("Ошибка преобразования значение переменной ping %v", err)
	}
	//cntGoroutinsStr := os.Getenv("COMPUTING_POWER")
	//if cntGoroutinsStr == "" {
	//	logger.Fatal("Укажите количество запускаемых потоков(переменную COMPUTING_POWER)")
	//}
	//cntGoroutins, err := strconv.Atoi(cntGoroutinsStr)
	//if err != nil {
	//	logger.Fatalf("Не удалось преобразовать значение COMPUTING_POWER в число: %v", err)
	//}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Duration(ping) * time.Millisecond)
				req, err := http.Get(address)
				//logger.Debug("Запрос задачи")
				if err != nil {
					logger.Errorf("Ошибка запроса: %v", err)
					continue
				}
				//logger.Debugf("Возращён код: %d", req.StatusCode)
				defer req.Body.Close()
				if req.StatusCode == 404 || req.StatusCode == 500 {
					continue
				}
				//logger.Debugf("Принял %s", req.Body)
				task := logic.TaskWrapper{}
				err = json.NewDecoder(req.Body).Decode(&task)
				t.In <- task.Task
				logger.Debugf("Расшифровал: %+v", task)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resTask := <-t.Results
				task := logic.TaskWrapper{Task: resTask}
				jsonBytes, err := json.Marshal(task)
				if err != nil {
					logger.Errorf("Ошибка кодирования в json: %v", err)
				}
				_, err = http.Post(address, "application/json", bytes.NewBuffer(jsonBytes))
				if err != nil {
					logger.Errorf("Ошибка отправки результата: %v", err)
				}
			}
		}
	}()
}

func (t *Transport) Shutdown() {
	close(t.In)
	close(t.Results)
}
