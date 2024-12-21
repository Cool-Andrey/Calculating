package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"io"
	"log"
	"net/http"
	"os"
)

type Config struct {
	Addr string
}

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
	}
	return config
}

type Application struct {
	config *Config
}

func New() *Application {
	return &Application{
		config: ConfigFromEnv(),
	}
}

type Request struct {
	Expression string `json:"expression"`
}

type Result struct {
	Res string `json:"result"`
}

type ResultBad struct {
	Err string `json:"error"`
}

func CalcHandler(w http.ResponseWriter, r *http.Request) {
	request := new(Request)
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil && err != io.EOF {
		w.WriteHeader(422)
		log.Printf("Ошибка чтения json: %s", err)
		errj := calc.ErrInvalidJson
		res := ResultBad{Err: errj.Error()}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		return
	} else if err == io.EOF {
		w.WriteHeader(422)
		errj := calc.ErrEmptyJson
		res := ResultBad{Err: errj.Error()}
		log.Println("Пустой json!")
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
		return
	} else {
		log.Printf("Прочитал: %s", request.Expression)
	}

	result, err := calc.Calc(request.Expression)
	var errJ error
	if err != nil {
		w.WriteHeader(422)
		if errors.Is(err, calc.ErrInvalidBracket) {
			errJ = calc.ErrInvalidBracket
			log.Printf("Ошибка счёта: %s", calc.ErrInvalidBracket)
		} else if errors.Is(err, calc.ErrInvalidOperands) {
			errJ = calc.ErrInvalidOperands
			log.Printf("Ошибка счёта: %s", calc.ErrInvalidOperands)
		} else if errors.Is(err, calc.ErrDivByZero) {
			errJ = calc.ErrDivByZero
			log.Printf("Ошибка счёта: %s", calc.ErrDivByZero)
		} else if errors.Is(err, calc.ErrEmptyExpression) {
			errJ = calc.ErrEmptyExpression
			log.Printf("Ошибка счёта: %s", calc.ErrEmptyExpression)
		} else {
			w.WriteHeader(500)
			errJ = errors.New("Что-то пошло не так")
			log.Printf("Неизвестная ошибка счёта: %s", err.Error())
		}
		errj := errJ.Error()
		res := ResultBad{Err: errj}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
	} else {
		w.WriteHeader(http.StatusOK)
		res1 := Result{Res: fmt.Sprintf("%.2f", result)}
		jsonBytes, _ := json.Marshal(res1)
		fmt.Fprint(w, string(jsonBytes))
		log.Printf("Посчитал: %.2f", result)
	}
}

func (a *Application) RunServer() error {
	http.HandleFunc("/api/v1/calculate", CalcHandler)
	fmt.Printf("Сервер слушается на порту: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, nil)
}
