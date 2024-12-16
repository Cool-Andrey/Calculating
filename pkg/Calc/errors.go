package Calc

import "errors"

var (
	ErrDivByZero       = errors.New("Деление на ноль! Мы не высшая математика, так что иди лесом!")
	ErrInvalidBracket  = errors.New("Товарищ пользователь! Проверьте скобки!")
	ErrInvalidOperands = errors.New("Товарищ пользователь! Проверьте количество операндов(+,-,/,*)")
)
