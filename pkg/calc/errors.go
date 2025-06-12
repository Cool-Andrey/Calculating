package calc

import "errors"

var (
	ErrDivByZero       = errors.New("Деление на ноль! Мы не высшая математика, так что иди лесом!")
	ErrInvalidBracket  = errors.New("Товарищ пользователь! Проверьте скобки и точки!")
	ErrInvalidOperands = errors.New("Товарищ пользователь! Проверьте количество операндов(+,-,/,*), их порядок и проверьте что нет буков")
	ErrInvalidJson     = errors.New("Товарищ пользователь! Проверьте правильность написания json'а")
	ErrEmptyJson       = errors.New("Пустой запрос!")
	ErrEmptyExpression = errors.New("Пустое выражение/json!")
	ErrExpJWTToken     = errors.New("Токен протух")
	ErrInvalidJWTToken = errors.New("Невалидный токен")

	Errors = []error{ErrDivByZero, ErrInvalidBracket, ErrInvalidOperands, ErrInvalidJson, ErrEmptyJson, ErrEmptyExpression, ErrExpJWTToken, ErrInvalidJWTToken}
)
