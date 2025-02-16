package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
	"io"
	"net/http"
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

func CalcHandler(w http.ResponseWriter, r *http.Request, logger *zap.SugaredLogger) {
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

	result, err := calc.Calc(request.Expression)
	var errJ error
	if err != nil {
		if statusCode, ok := calc.ErrorMap[err]; ok {
			w.WriteHeader(statusCode)
			errJ = err
			logger.Errorf("Ошибка счёта: %v", errJ)
		} else {
			w.WriteHeader(500)
			errJ = errors.New("Что-то пошло не так")
			logger.Errorf("Неизвестная ошибка счёта: %v", errJ)
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
	}
	time.Sleep(1)
}

func Decorate(next http.Handler, ds ...Decorator) http.Handler {
	res := next
	for d := len(ds) - 1; d >= 0; d-- {
		res = ds[d](res)
	}
	return res
}
