package server

import (
	"bytes"
	"fmt"
	"github.com/Cool-Andrey/Calculating/pkg/calc"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"time"
)

func LoggingMiddleware(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				logger.Errorf("Ошибка чтения тела из логера: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			if r.Method == http.MethodGet && r.URL.Path != "/internal/task" {
				logger.Infow("HTTP запрос", zap.String("Метод", r.Method),
					zap.String("Путь", r.URL.String()),
					zap.Duration("Время выполнения", duration),
				)
			} else if r.URL.Path != "/internal/task" && r.Method == http.MethodGet {
				logger.Infow("HTTP запрос", zap.String("Метод", r.Method),
					zap.String("Путь", r.URL.String()),
					zap.String("Тело", string(bodyBytes)),
					zap.Duration("Время выполнения", duration),
				)
			}
		})
	}
}

func JWTAuthMiddleware(logger *zap.SugaredLogger, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/register" && r.URL.Path != "/api/v1/login" {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					logger.Error("Нет токена")
					http.Error(w, "Нет токена", http.StatusUnauthorized)
					return
				}
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					logger.Error("Неправильный формат хедера")
					http.Error(w, "Неправильный формат хедера", http.StatusUnauthorized)
					return
				}
				err := ValidateToken(parts[1], secret)
				if err != nil {
					http.Error(w, "Чёт не то с токеном", http.StatusUnauthorized)
					logger.Errorf("Ошибка авторизации: %v", err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ValidateToken(tokenString, secret string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Не тот метод подписи: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return calc.ErrExpJWTToken
			}
		}
		return nil
	}
	return calc.ErrInvalidJWTToken
}
