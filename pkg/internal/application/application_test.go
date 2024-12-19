package application

import (
	"bytes"
	"encoding/json"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
			expected_num:  nil,
			expected_err:  calc.ErrInvalidBracket,
			expected_code: 422,
		},
		{
			name:          "invalid operands",
			expression:    "2++2",
			expected_num:  nil,
			expected_err:  calc.ErrInvalidOperands,
			expected_code: 422,
		},
		{
			name:          "Empty",
			expression:    "",
			expected_num:  nil,
			expected_err:  calc.ErrEmptyJson,
			expected_code: 422,
		},
	}
	w := httptest.NewRecorder()
	for _, test := range tests {
		jsonBytes, _ := json.Marshal(test.expression)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		handler := http.HandlerFunc(CalcHandler)
		handler.ServeHTTP(w, req)
		if w.Code != test.expected_code {
			t.Errorf("Ожидал код: %d Получил код: %d", test.expected_code, w.Code)
		}

	}
}
