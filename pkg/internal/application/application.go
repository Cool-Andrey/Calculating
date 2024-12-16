package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Ошибка чтения json: %s", err)
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
		} else {
			w.WriteHeader(500)
			errJ = errors.New("Internal server error")
			log.Printf("Неизвестная ошибка счёта")
		}
		errj := errJ.Error()
		res := ResultBad{Err: errj}
		jsonBytes, _ := json.Marshal(res)
		fmt.Fprint(w, string(jsonBytes))
	} else {
		w.WriteHeader(http.StatusOK)
		res1 := Result{Res: fmt.Sprintf("%.2f", result)}
		log.Println(res1)
		jsonBytes, err := json.Marshal(res1)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(jsonBytes))
		fmt.Fprint(w, string(jsonBytes))
		log.Printf("Посчитал: %.2f", result)
	}
}

func (a *Application) RunServer() error {
	http.HandleFunc("/api/v1/calculate", CalcHandler)
	return http.ListenAndServe(":"+a.config.Addr, nil)
}
