package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"io"
	"log"
	"net/http"
)

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
		if statusCode, ok := calc.ErrorMap[err]; ok {
			w.WriteHeader(statusCode)
			errJ = err
			log.Printf("Ошибка счёта: %v", err)
		} else {
			w.WriteHeader(500)
			errJ = errors.New("Что-то пошло не так")
			log.Printf("Неизвестная ошибка счёта: %v", err)
		}
		res := ResultBad{Err: errJ.Error()}
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
