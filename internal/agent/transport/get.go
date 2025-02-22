package transport

import (
	"bytes"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Transport struct {
	in      chan logic.Task
	results chan logic.Task
}

func NewAgent() *Transport {
	return &Transport{
		in:      make(chan logic.Task, 128),
		results: make(chan logic.Task, 128),
	}
}

func (t *Transport) Start() {
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
	cntGoroutinsStr := os.Getenv("COMPUTING_POWER")
	if cntGoroutinsStr == "" {
		logger.Fatal("Укажите количество запускаемых потоков(переменную COMPUTING_POWER)")
	}
	cntGoroutins, err := strconv.Atoi(cntGoroutinsStr)
	if err != nil {
		logger.Fatalf("Не удалось преобразовать значение COMPUTING_POWER в число: %v", err)
	}

	tasksCh := make(chan logic.Task, cntGoroutins)
	resultsCh := make(chan logic.Task, cntGoroutins)
	for i := 0; i < cntGoroutins; i++ {
		go logic.Worker(tasksCh, resultsCh)
	}

	for {
		time.Sleep(time.Duration(ping) * time.Millisecond)
		address := "http://localhost:" + configLog.Addr + "/internal/task"
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
		logger.Debugf("Расшифровал: %+v", task)
		tasksCh <- task.Task
		resTask := <-resultsCh
		task.Task.Result = resTask.Result
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
