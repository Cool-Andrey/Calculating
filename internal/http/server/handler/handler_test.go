package handler

import (
	"bytes"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/internal/agent/logic"
	"github.com/Cool-Andrey/Calculating/internal/config"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/Cool-Andrey/Calculating/pkg/calc/safeStructures"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCalcHandler(t *testing.T) {
	o := orchestrator.NewOrchestator()
	logger := zap.NewNop().Sugar()
	safeMap := safeStructures.NewSafeMap()
	id := safeStructures.NewSafeId()
	url := "http://127.0.0.1:8080/api/v1/calculate"

	t.Run("Invalid method", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		CalcHandler(w, r, logger, o, safeMap, id)
		res := w.Result()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Ожидал код %d получил %d", http.StatusMethodNotAllowed, res.StatusCode)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		expectedBody := http.StatusText(http.StatusMethodNotAllowed) + "\n"
		if string(body) != expectedBody {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

	t.Run("Empty request", func(t *testing.T) {
		text := bytes.NewBufferString("")
		r := httptest.NewRequest(http.MethodPost, url, text)
		w := httptest.NewRecorder()
		CalcHandler(w, r, logger, o, safeMap, id)
		res := w.Result()
		if res.StatusCode != 422 {
			t.Errorf("Ожидал код %d получил %d", 422, res.StatusCode)
		}
		expectedBody, err := json.Marshal(ResultBad{Err: calc.ErrEmptyJson.Error()})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

	t.Run("Invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		text := bytes.NewBufferString(`{"expression":"2+5"`)
		r := httptest.NewRequest(http.MethodPost, url, text)
		CalcHandler(w, r, logger, o, safeMap, id)
		res := w.Result()
		if res.StatusCode != 422 {
			t.Errorf("Ожидал код %d получил %d", 422, res.StatusCode)
		}
		expectedBody, err := json.Marshal(ResultBad{Err: calc.ErrInvalidJson.Error()})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

	t.Run("Valid json and request", func(t *testing.T) {
		w := httptest.NewRecorder()
		text := bytes.NewBufferString(`{"expression":"2+5"}`)
		r := httptest.NewRequest(http.MethodPost, url, text)
		CalcHandler(w, r, logger, o, safeMap, id)
		time.Sleep(1)
		if id.Id != 1 {
			t.Error("Не изменил id")
		}
		if status := safeMap.Get(1).Status; status != "Подсчёт" {
			t.Errorf("Ожидал статус: %s получил %s", "Подсчёт", status)
		}
		res := w.Result()
		if res.StatusCode != 201 {
			t.Errorf("Ожидал код %d получил %d", 201, res.StatusCode)
		}
		expectedBody, err := json.Marshal(ResoponseId{ID: id.Id})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})
}

func TestGiveTaskHandler(t *testing.T) {
	o := orchestrator.NewOrchestator()
	logger := zap.NewNop().Sugar()
	delay := config.Delay{Plus: 1, Minus: 1, Multiple: 1, Divide: 1}
	safeMap := safeStructures.NewSafeMap()
	url := "http://127.0.0.1:8080/api/v1/task"

	t.Run("GET with no tasks", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("Ожидал код %d получил %d", http.StatusNotFound, res.StatusCode)
		}
		defer res.Body.Close()
	})

	t.Run("GET request with tasks", func(t *testing.T) {
		o.Out <- logic.Task{Id: 1, Operation: "+", OperationTime: 1}
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("Ожидал код %d получил %d", http.StatusOK, res.StatusCode)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		expectedBody, err := json.Marshal(logic.TaskWrapper{Task: logic.Task{Id: 1, Operation: "+", OperationTime: 1}})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

	t.Run("POST with valid data", func(t *testing.T) {
		taskRes := logic.TaskWrapper{Task: logic.Task{Id: 1, Result: 7.0}}
		jsonData, err := json.Marshal(taskRes)
		if err != nil {
			t.Fatalf("Ошибка преобразования задачи в json: %v", err)
		}
		safeMap.Set(1, safeStructures.Expressions{Id: 1, Status: "Подсчёт"})
		r := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("Ожидал код %d получил %d", http.StatusOK, res.StatusCode)
		}
	})

	t.Run("POST request with invalid data", func(t *testing.T) {
		invalidJson := `{"task": {"id": 1, "result": "7"}`
		r := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(invalidJson))
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		defer res.Body.Close()
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("Ожидал код %d получил %d", http.StatusInternalServerError, res.StatusCode)
		}
	})

	t.Run("POST request with non-exist ID", func(t *testing.T) {
		taskRes := logic.TaskWrapper{Task: logic.Task{Id: 999, Result: 7.0}}
		jsonData, err := json.Marshal(taskRes)
		if err != nil {
			t.Fatalf("Ошибка преобразования задачи в json: %v", err)
		}
		r := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("Ожидал код %d получил %d", http.StatusNotFound, res.StatusCode)
		}
	})

	t.Run("POST request with already completed task", func(t *testing.T) {
		taskRes := logic.TaskWrapper{Task: logic.Task{Id: 1, Result: 7.0}}
		jsonData, err := json.Marshal(taskRes)
		if err != nil {
			t.Fatalf("Ошибка преобразования задачи в json: %v", err)
		}
		safeMap.Set(1, safeStructures.Expressions{Id: 1, Status: "Завершено", Result: "7"})
		r := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		GiveTask(w, r, logger, o, delay, safeMap)
		res := w.Result()
		defer res.Body.Close()
		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Ожидал код %d получил %d", http.StatusUnprocessableEntity, res.StatusCode)
		}
	})
}

func TestGetAllExpressionsHandler(t *testing.T) {
	logger := zap.NewNop().Sugar()
	safeMap := safeStructures.NewSafeMap()
	url := "http://127.0.0.1:8080/api/v1/expressions"

	t.Run("Invalid method", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, url, nil)
		w := httptest.NewRecorder()
		GetAllExpressions(w, r, logger, safeMap)
		res := w.Result()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Ожидал код %d получил %d", http.StatusMethodNotAllowed, res.StatusCode)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		expectedBody := http.StatusText(http.StatusMethodNotAllowed) + "\n"
		if string(body) != expectedBody {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})
	t.Run("Request with no expressions", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		GetAllExpressions(w, r, logger, safeMap)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("Ожидал код %d получил %d", http.StatusOK, res.StatusCode)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		expectedBody, err := json.Marshal(ExprWr{Expressions: nil})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

	t.Run("Request with expressions in SafeMap", func(t *testing.T) {
		expr1 := safeStructures.Expressions{Id: 1, Status: "Подсчёт", Result: ""}
		expr2 := safeStructures.Expressions{Id: 2, Status: "Завершено", Result: "7"}
		safeMap.Set(1, expr1)
		safeMap.Set(2, expr2)
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		GetAllExpressions(w, r, logger, safeMap)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("Ожидал код %d получил %d", http.StatusOK, res.StatusCode)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Не получилось прочитать ответ %v", err)
		}
		expectedBody, err := json.Marshal(ExprWr{Expressions: []safeStructures.Expressions{expr1, expr2}})
		if err != nil {
			t.Errorf("Ошибка преобразования ожидаемого результата в json: %v", err)
		}
		if string(body) != string(expectedBody) {
			t.Errorf("Ожидал ответ: %s получил %s", expectedBody, string(body))
		}
	})

}
