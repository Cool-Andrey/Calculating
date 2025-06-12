package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/ast"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/repository/postgres"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func CalcHandler(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, a *ast.AST, rep *postgres.Repository) {
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
		res := ResultBad{Err: calc.ErrInvalidJson.Error()}
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		res := ResultBad{Err: calc.ErrEmptyJson.Error()}
		logger.Error("Пустой запрос!")
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else {
		logger.Debugf("Прочитал: %s", request.Expression)
	}
	ctx := r.Context()
	id, err := rep.SetWithExpression(ctx, models.Expressions{Status: "Подсчёт"}, request.Expression)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка записи выражения в СУБД: %v", err)
		return
	}
	resp := ResponseID{ID: id}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(201)
		_, _ = fmt.Fprint(w, string(jsonBytes))
	}
	a.Calc(ctx, request.Expression, id)
}

func GetExpression(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, rep *postgres.Repository) {
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
	res, err := rep.Get(r.Context(), id)
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
		_, _ = fmt.Fprint(w, string(jsonBytes))
	}
}

func GetAllExpressions(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, rep *postgres.Repository) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logger.Errorf("Попытка получить выражение не методом GET")
		return
	}
	expressions, err := rep.GetAll(r.Context())
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
		_, _ = fmt.Fprint(w, string(jsonBytes))
	}
}

func Register(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, rep *postgres.Repository) {
	if r.Method != http.MethodPost {
		logger.Error("Попытка зарегистрироваться не методом POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil && err != io.EOF {
		w.WriteHeader(422)
		logger.Errorf("Ошибка чтения json: %v", err)
		res := ResultBad{Err: calc.ErrInvalidJson.Error()}
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		res := ResultBad{Err: calc.ErrEmptyJson.Error()}
		logger.Error("Пустой запрос!")
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	}
	ctx := r.Context()
	err = rep.CreateUser(ctx, user.Login, user.Password)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка записи в СУБД нового юзера: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func Login(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, rep *postgres.Repository, secret string) {
	if r.Method != http.MethodPost {
		logger.Error("Попытка войти не методом POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil && err != io.EOF {
		w.WriteHeader(422)
		logger.Errorf("Ошибка чтения json: %v", err)
		res := ResultBad{Err: calc.ErrInvalidJson.Error()}
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		res := ResultBad{Err: calc.ErrEmptyJson.Error()}
		logger.Error("Пустой запрос!")
		jsonBytes, _ := json.Marshal(res)
		_, _ = fmt.Fprint(w, string(jsonBytes))
		time.Sleep(1)
		return
	}
	ctx := r.Context()
	ok, err := rep.VerifyUser(ctx, user.Login, user.Password)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка проверки учетной записи в СУБД: %v", err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		logger.Debug("Учётную запись не нашли в СУБД")
		return
	}
	jwtToken, err := GenerateJWT(secret)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка формирования jwt: %v", err)
	}
	response := map[string]string{"token": jwtToken}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка маршалинга jwt: %v", err)
	}
}

func GenerateJWT(secret string) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 1).Unix(),
	})
	return claims.SignedString([]byte(secret))
}

func Decorate(next http.Handler, ds ...Decorator) http.Handler {
	res := next
	for d := len(ds) - 1; d >= 0; d-- {
		res = ds[d](res)
	}
	return res
}
