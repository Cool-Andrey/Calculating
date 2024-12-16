package calc

import (
	"testing"
)

func TestCalc(t *testing.T) {
	tests := []struct {
		name         string
		expression   string
		expected_num float64
		expected_err error
	}{
		{
			name:         "easy",
			expression:   "2 + 2",
			expected_num: 4,
			expected_err: nil,
		},
		{
			name:         "with brackets",
			expression:   "(2+2)*3",
			expected_num: 12,
			expected_err: nil,
		},
		{
			name:         "invalid brackets",
			expression:   "((2+2)*3",
			expected_num: 0,
			expected_err: ErrInvalidBracket,
		},
		{
			name:         "invalid operands",
			expression:   "2**3",
			expected_num: 0,
			expected_err: ErrInvalidOperands,
		},
		{
			name:         "division by zero",
			expression:   "1/0",
			expected_num: 0,
			expected_err: ErrDivByZero,
		},
		{
			name:         "with double digit numbers",
			expression:   "22*3",
			expected_num: 66,
			expected_err: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			val, err := Calc(test.expression)
			if err != test.expected_err {
				t.Errorf("Name: %s\nCalc(%q): expected error %v, got %v", test.name, test.expression, test.expected_err, err)
			}
			if val != test.expected_num {
				t.Errorf("Name: %s\nCalc(%q): expected num %.2f, got %.2f", test.name, test.expression, test.expected_num, val)
			}
		})
	}
}
