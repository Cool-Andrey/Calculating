package handler

import (
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	"net/http"
)

type Decorator func(http.Handler) http.Handler

type Request struct {
	Expression string `json:"expression"`
}

type ResponseWr struct {
	Expression models.Expressions `json:"expression"`
}

type ExprWr struct {
	Expressions []models.Expressions `json:"expressions"`
}

type ResponseID struct {
	ID int `json:"id"`
}

type ResultBad struct {
	Err string `json:"error"`
}

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
