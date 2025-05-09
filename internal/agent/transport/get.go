package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"net/http"
	"time"
)

type Transport struct {
	In      chan models.Task
	Results chan models.Task
}

func NewAgent(cntGoroutins int) *Transport {
	return &Transport{
		In:      make(chan models.Task, cntGoroutins),
		Results: make(chan models.Task, cntGoroutins),
	}
}

func (t *Transport) Start(address string, ping int, ctx context.Context) {
	configLog := config.ConfigFromEnv()
	logger := config.SetupLogger(configLog.Mode)
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
				task := models.TaskWrapper{}
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
				task := models.TaskWrapper{Task: resTask}
				jsonBytes, err := json.Marshal(task)
				if err != nil {
					logger.Errorf("Ошибка кодирования в json: %v", err)
				}
				_, err = http.Post(address, "application/json", bytes.NewBuffer(jsonBytes))
				logger.Debugf("Отправил: %+v", task)
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
