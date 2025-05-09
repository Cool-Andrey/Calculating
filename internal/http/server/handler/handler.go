package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	Expression models.Expressions `json:"expression"`
}

type ExprWr struct {
	Expressions []models.Expressions `json:"expressions"`
}

type ResoponseId struct {
	ID int `json:"id"`
}

type ResultBad struct {
	Err string `json:"error"`
}

func CalcHandler(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestrator, conn *pgxpool.Pool) {
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
		logger.Debugf("Прочитал: %s", request.Expression)
	}
	ctx := r.Context()
	id, err := postgres.SetWithExpression(ctx, models.Expressions{Status: "Подсчёт"}, request.Expression, conn)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка записи выражения в СУБД: %v", err)
		return
	}
	resp := ResoponseId{ID: id}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(201)
		fmt.Fprint(w, string(jsonBytes))
	}
	o.Calculate(ctx, request.Expression, id, logger, conn)
	time.Sleep(1) // Если знаете, как без этого исправить тот факт, что time.duration() в server.go возвращает 0, буду рад! Сам промучался, так и не найдя решение. Буду премного благодарен за совет!
}

func GiveTask(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestrator, delay config.Delay, pool *pgxpool.Pool) {
	switch r.Method {
	case http.MethodGet:
		handleGetTask(w, logger, o, delay)
	case http.MethodPost:
		handlePostResult(w, r, logger, o, pool)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleGetTask(w http.ResponseWriter, logger *zap.SugaredLogger, o *orchestrator.Orchestrator, delay config.Delay) {
	select {
	case task := <-o.Out:
		setOperationTime(&task, delay)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(models.TaskWrapper{Task: task}); err != nil {
			logger.Errorf("Ошибка кодирования в json задачи: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func handlePostResult(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, o *orchestrator.Orchestrator, pool *pgxpool.Pool) {
	taskRes := new(models.TaskWrapper)
	if err := json.NewDecoder(r.Body).Decode(taskRes); err != nil {
		logger.Errorf("Ошибка анмаршалинга json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	task := taskRes.Task
	ctx := r.Context()
	status, err := postgres.GetStatus(ctx, task.Id, pool)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Errorf("Ошибка запроса статуса выражения к СУБД: %v", err)
			return
		}
	}
	if status != "Подсчёт" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	err = postgres.ProcessTaskResult(ctx, task, o.In, pool, logger)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Errorf("Ошибка записи таски в СУБД: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
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

func GetExpression(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, pool *pgxpool.Pool) {
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
	logger.Debugf("Преобразовал ID: %v", id)
	res, err := postgres.Get(r.Context(), id, pool)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Errorf("Ошибка запроса к СУБД: %v", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
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

func GetAllExpressions(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, pool *pgxpool.Pool) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Errorf("Попытка получить выражение не методом GET")
		return
	}
	expressions, err := postgres.GetAll(r.Context(), pool)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка запроса к СУБД: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
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
