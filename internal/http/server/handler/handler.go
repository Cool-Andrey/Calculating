package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/Cool-Andrey/Calculating/pkg/calc/safeStructures"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Decorator func(http.Handler) http.Handler

type Request struct {
	Expression string `json:"expression"`
}

type ResponseWr struct {
	Expression safeStructures.Expressions `json:"expression"`
}

type ExprWr struct {
	Expressions []safeStructures.Expressions `json:"expressions"`
}

type ResoponseId struct {
	ID int `json:"id"`
}

type ResultBad struct {
	Err string `json:"error"`
}

func CalcHandler(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestator, Map *safeStructures.SafeMap, Id *safeStructures.SafeId) {
	request := new(Request)
	err := json.NewDecoder(r.Body).Decode(&request)
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Errorf("Попытка отдать выражение на обработку методом не POST")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err != nil && err != io.EOF {
		w.WriteHeader(422)
		logger.Errorf("Ошибка чтения json: %v", err)
		errj := calc.ErrInvalidJson
		res := ResultBad{Err: errj.Error()}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		errj := calc.ErrEmptyJson
		res := ResultBad{Err: errj.Error()}
		logger.Error("Пустой запрос!")
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else {
		w.WriteHeader(201)
		logger.Debugf("Прочитал: %s", request.Expression)
	}
	id := Id.Get()
	logger.Debugf("Сгенерировал ID: %d", id)
	Map.Set(id, safeStructures.Expressions{Id: id, Status: "Подсчёт"})
	resp := ResoponseId{ID: id}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
	} else {
		fmt.Fprint(w, string(jsonBytes))
	}
	go o.Calculate(request.Expression, logger, id, Map)
	time.Sleep(1) // Если знаете, как без этого исправить тот факт, что time.duration() в server.go возвращает 0, буду рад! Сам промучался, так и не найдя решение. Буду премного благодарен за совет!
}

func GiveTask(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestator, delay config.Delay, safeMap *safeStructures.SafeMap) {
	if r.Method == "GET" {
		if len(o.Out) == 0 {
			w.WriteHeader(404)
		} else {
			w.Header().Set("Content-Type", "application/json")
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

func GetExpression(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, safeMap *safeStructures.SafeMap) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Errorf("Попытка получить выражение не методом GET.")
		return
	}
	url := r.URL.Path
	idStr := strings.TrimPrefix(url, "/api/v1/expressions/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка преобразования ID: %v", err)
		return
	}
	logger.Debugf("Преобразовал ID: %v", err)
	res := safeMap.Get(id)
	if res.Id == 0 {
		w.WriteHeader(404)
		logger.Debug("Не нашёл выражения")
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		resWr := ResponseWr{Expression: res}
		jsonBytes, err := json.Marshal(resWr)
		if err != nil {
			w.WriteHeader(500)
			logger.Errorf("Ошибка преобразования json: %v", err)
			return
		}
		fmt.Fprint(w, string(jsonBytes))
	}
}

func GetAllExpressions(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, safeMap *safeStructures.SafeMap) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Errorf("Попытка получить выражение не методом GET")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	expressions := safeMap.GetAll()
	res := ExprWr{Expressions: expressions}
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка преобразования в json: %v", err)
	} else {
		fmt.Fprint(w, string(jsonBytes))
	}
}

func Decorate(next http.Handler, ds ...Decorator) http.Handler {
	res := next
	for d := len(ds) - 1; d >= 0; d-- {
		res = ds[d](res)
	}
	return res
}
