package safeStructures

import (
	"github.com/Cool-Andrey/Calculating/internal/orchestrator/models"
	"sync"
)

type SafeMap struct {
	m   map[int]models.Expressions
	mux sync.RWMutex
}

//type Expressions models.Expressions

func NewSafeMap() *SafeMap {
	return &SafeMap{m: make(map[int]models.Expressions), mux: sync.RWMutex{}}
}

func (s *SafeMap) Get(key int) models.Expressions {
	s.mux.RLock()
	defer s.mux.RUnlock()
	res, ok := s.m[key]
	if ok {
		return res
	} else {
		return models.Expressions{}
	}
}

func (s *SafeMap) In(key int) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, ok := s.m[key]
	if ok {
		return true
	} else {
		return false
	}
}

func (s *SafeMap) Set(key int, value models.Expressions) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.m[key] = value
}

func (s *SafeMap) GetAll() []models.Expressions {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var res []models.Expressions
	for _, v := range s.m {
		res = append(res, v)
	}
	return res
}
