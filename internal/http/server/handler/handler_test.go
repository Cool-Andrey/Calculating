package handler

import (
	"bytes"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type good_req struct {
	Res string `json:"result"`
}

type bad_req struct {
	Res string `json:"error"`
}

type reqS struct {
	Req string `json:"expression"`
}

func setupLogger() *zap.SugaredLogger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Да емаё! Логгер рухнул( Вот подробности: %s", err)
	}
	return logger.Sugar()
}

func TestCalcHandler(t *testing.T) {
	tests := []struct {
		name          string
		expression    string
		expected_num  float64
		expected_err  error
		expected_code int
	}{
		{
			name:          "simple",
			expression:    "2+2",
			expected_num:  4,
			expected_err:  nil,
			expected_code: http.StatusOK,
		},
		{
			name:          "with brackets",
			expression:    "(2+3)*2",
			expected_num:  10,
			expected_err:  nil,
			expected_code: http.StatusOK,
		},
		{
			name:          "no brackets",
			expression:    "2+3*2",
			expected_num:  8,
			expected_err:  nil,
			expected_code: http.StatusOK,
		},
		{
			name:          "invalid brackets",
			expression:    "(2+2*3",
			expected_num:  0,
			expected_err:  calc.ErrInvalidBracket,
			expected_code: 422,
		},
		{
			name:          "invalid operands",
			expression:    "2++2",
			expected_num:  0,
			expected_err:  calc.ErrInvalidOperands,
			expected_code: 422,
		},
		{
			name:          "Empty",
			expression:    "",
			expected_num:  0,
			expected_err:  calc.ErrEmptyExpression,
			expected_code: 422,
		},
	}
	for _, test := range tests {
		w := httptest.NewRecorder()
		reqBody := reqS{Req: test.expression}
		jsonBytes, errjson := json.Marshal(reqBody)
		if errjson != nil {
			t.Errorf("Ошибка маршалинга: %s", errjson.Error())
		}
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		logger := setupLogger()
		defer logger.Sync()
		CalcHandler(w, req, logger)
		if w.Code != test.expected_code {
			t.Errorf("Ожидал код: %d Получил код: %d", test.expected_code, w.Code)
		}
		if w.Code == http.StatusOK {
			resJ := new(good_req)
			err := json.Unmarshal(w.Body.Bytes(), &resJ)
			if err != nil {
				t.Errorf("Ошибка преобразования json'а: %s Сам json: %s", err, w.Body.String())
			}
			res, err1 := strconv.ParseFloat(resJ.Res, 64)
			if err1 != nil {
				t.Errorf("Ошибка преобразования string в float: %s", err1)
			}
			if res != test.expected_num {
				t.Errorf("Ожидал результат: %f Получил результат: %f", test.expected_num, res)
			}
		} else {
			resJ := new(bad_req)
			err := json.Unmarshal(w.Body.Bytes(), &resJ)
			if err != nil {
				t.Errorf("Ошибка преобразования json'а: %s", err)
			}
			if resJ.Res != test.expected_err.Error() {
				t.Errorf("Ожидал ошибку: %s Получил ошибку: %s", test.expected_err, resJ.Res)
			}
		}
	}
}
