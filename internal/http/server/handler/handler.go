package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/Cool-Andrey/Calculating/pkg/calc/safeMap"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Decorator func(http.Handler) http.Handler

type Request struct {
	Expression string `json:"expression"`
}

type Result struct {
	Res string `json:"result"`
}

type ResultBad struct {
	Err string `json:"error"`
}

func CalcHandler(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestator, Map *safeMap.SafeMap) {
	request := new(Request)
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil && err != io.EOF {
		w.WriteHeader(422)
		logger.Errorf("Ошибка чтения json: %v", err)
		errj := calc.ErrInvalidJson
		res := ResultBad{Err: errj.Error()}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		errj := calc.ErrEmptyJson
		res := ResultBad{Err: errj.Error()}
		logger.Error("Пустой json!")
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		return
	} else {
		logger.Debugf("Прочитал: %s", request.Expression)
	}
	id := rand.Intn(10000000000000) + 1
	if Map.In(id) {
		for {
			id++
			if !Map.In(id) {
				break
			}
		}
	}
	Map.Set(id, safeMap.Expressions{Id: id, Status: "Подсчёт"})
	o.TakeExpression(request.Expression, logger, id)
	<-o.Ready
	logger.Debug("Готов")
	var result float64
	if len(o.ErrorsCh) > 0 {
		err = <-o.ErrorsCh
	} else {
		result = <-o.In
	}
	var errJ error
	if err != nil {
		if statusCode, ok := calc.ErrorMap[err]; ok {
			w.WriteHeader(statusCode)
			errJ = err
			logger.Errorf("Ошибка счёта: %v", errJ)
			Map.Set(id, safeMap.Expressions{Id: id, Status: "Выполнено", Result: err.Error()})
		} else {
			w.WriteHeader(500)
			errJ = errors.New("Что-то пошло не так")
			logger.Errorf("Неизвестная ошибка счёта: %v", errJ)
			Map.Set(id, safeMap.Expressions{Id: id, Status: "Выполнено", Result: err.Error()})
		}
		res := ResultBad{Err: errJ.Error()}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
	} else {
		w.WriteHeader(http.StatusOK)
		res1 := Result{Res: fmt.Sprintf("%.2f", result)}
		jsonBytes, _ := json.Marshal(res1)
		fmt.Fprint(w, string(jsonBytes))
		logger.Debugf("Посчитал: %.2f", result)
		resStr := strconv.FormatFloat(result, 'f', 2, 64)
		Map.Set(id, safeMap.Expressions{Id: id, Status: "Выполнено", Result: resStr})
	}
	time.Sleep(1) // Если знаете, как без этого исправить тот факт, что time.duration() в server.go возвращает 0, буду рад! Сам промучался, так и не найдя решение. Буду премного благодарен за совет!
}

func GiveTask(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestator, delay config.Delay, safeMap *safeMap.SafeMap) {
	if r.Method == "GET" {
		if len(o.Out) == 0 {
			w.WriteHeader(404)
		} else {
			task := <-o.Out
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
			taskWr := logic.TaskWrapper{Task: task}
			logger.Info(task, taskWr)
			jsonBytes, err := json.Marshal(taskWr)
			if err != nil {
				w.WriteHeader(500)
				logger.Errorf("Ошибка записи json: %v", err)
			}
			fmt.Fprint(w, string(jsonBytes))
		}
	} else {
		taskRes := new(logic.TaskWrapper)
		err := json.NewDecoder(r.Body).Decode(taskRes)
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(500)
		} else {
			task := taskRes.Task
			if !safeMap.In(task.Id) {
				w.WriteHeader(404)
			} else if safeMap.Get(task.Id).Result != "" {
				w.WriteHeader(422)
			} else {
				o.In <- task.Result
				w.WriteHeader(200)
			}
		}
	}
}

func Decorate(next http.Handler, ds ...Decorator) http.Handler {
	res := next
	for d := len(ds) - 1; d >= 0; d-- {
		res = ds[d](res)
	}
	return res
}
