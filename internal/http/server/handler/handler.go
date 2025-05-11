package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/internal/models"
	"github.com/Cool-Andrey/Calculating/internal/repository/postgres"
	"github.com/Cool-Andrey/Calculating/internal/service/orchestrator"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/golang-jwt/jwt/v4"
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

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
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

func Register(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, pool *pgxpool.Pool) {
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
	}
	ctx := r.Context()
	err = postgres.CreateUser(ctx, user.Login, user.Password, pool)
	if err != nil {
		w.WriteHeader(500)
		logger.Errorf("Ошибка записи в СУБД нового юзера: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func Login(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger, pool *pgxpool.Pool, secret string) {
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
	}
	ctx := r.Context()
	ok, err := postgres.VerifyUser(ctx, user.Login, user.Password, pool)
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
