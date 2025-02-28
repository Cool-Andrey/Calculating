package safeStructures

import (
	"sync"
	"testing"
)

func TestSafeMap(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		t.Parallel()
		safeMap := NewSafeMap()
		expression := Expressions{Id: 1, Status: "Подсчёт"}
		safeMap.Set(expression.Id, expression)
		result := safeMap.Get(expression.Id)
		if result != expression {
			t.Errorf("Ожидал %v получил %v", expression, result)
		}
		result = safeMap.Get(2)
		if result != (Expressions{}) {
			t.Errorf("Ожидал пустую структуру, получил %v", result)
		}
	})

	t.Run("In", func(t *testing.T) {
		t.Parallel()
		safeMap := NewSafeMap()
		expression := Expressions{Id: 1, Status: "Подсчёт"}
		safeMap.Set(1, expression)
		if !safeMap.In(expression.Id) {
			t.Errorf("Ожидал, что ключ %d существует", expression.Id)
		}
		if safeMap.In(expression.Id + 1) {
			t.Errorf("Ожидал, что ключ %d отсутствует", expression.Id)
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		t.Parallel()
		safeMap := NewSafeMap()
		expression1 := Expressions{Id: 1, Status: "Подсчёт"}
		expression2 := Expressions{Id: 2, Status: "Выполнено", Result: "7.0"}
		safeMap.Set(expression1.Id, expression1)
		safeMap.Set(expression2.Id, expression2)
		allExpressions := safeMap.GetAll()
		if res := len(allExpressions); res != 2 {
			t.Errorf("Ожидал 2 элемента, получил %d", res)
		}
		found1, found2 := false, false
		for _, expression := range allExpressions {
			if expression == expression1 {
				found1 = true
			} else if expression == expression2 {
				found2 = true
			}
		}
		if !found1 {
			t.Error("Не нашёл 1 элемент в мапе")
		} else if !found2 {
			t.Error("Не нашёл 2 элемент в мапе")
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		t.Parallel()
		safeMap := NewSafeMap()
		var wg sync.WaitGroup
		for i := 1; i <= 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				expression := Expressions{Id: i, Status: "Выполнено", Result: "А поч не добавить текст?:)"}
				safeMap.Set(expression.Id, expression)
			}()
		}
		wg.Wait()
		for i := 1; i <= 100; i++ {
			if !safeMap.In(i) {
				t.Errorf("Не нашёл элемент с ID: %d", i)
			}
		}
		allExpressions := safeMap.GetAll()
		if res := len(allExpressions); res != 100 {
			t.Errorf("Ожидал 100 элементов, получил %d", res)
		}
	})
}
