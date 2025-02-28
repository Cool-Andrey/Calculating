package safeStructures

import (
	"sync"
	"testing"
)

func TestSafeId(t *testing.T) {
	t.Run("One goroutine", func(t *testing.T) {
		id := NewSafeId()
		if res := id.Id; res != 0 {
			t.Errorf("Ожидал начальное значение 0, получил: %d", res)
		}
		for i := 1; i < 10; i++ {
			res := id.Get()
			if res != i {
				t.Errorf("Ожидал ID: %d, получил: %d", i, res)
			}
		}
	})

	t.Run("100 goroutines", func(t *testing.T) {
		id := NewSafeId()
		if res := id.Id; res != 0 {
			t.Errorf("Ожидал начальное значение 0, получил: %d", res)
		}
		var wg sync.WaitGroup
		IDs := make(chan int, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				IDs <- id.Get()
			}()
		}
		wg.Wait()
		close(IDs)
		allId := make(map[int]bool)
		for id := range IDs {
			if _, ok := allId[id]; ok {
				t.Errorf("Есть дубликаты ID: %d", id)
			}
			allId[id] = true
		}
		if len(allId) != 100 {
			t.Errorf("Ожидал 100 уникальных ID, получил: %d", len(allId))
		}
	})
}
