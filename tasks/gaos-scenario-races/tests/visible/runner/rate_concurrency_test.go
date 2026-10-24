package runner

import (
	"sync"
	"sync/atomic"
	"testing"
)

// From the project's fix (pg-search-sync, gaos b41f2d3). Run with -race.
func TestRateIsExactUnderConcurrency(t *testing.T) {
	rate := NewRate(Scenario{Rate: 1})
	var ignored int64
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := rate.Execute(); err != nil {
				atomic.AddInt64(&ignored, 1)
			}
		}()
	}
	wg.Wait()
	if ignored != 500 {
		t.Fatalf("rate 1 over 1000 calls ignored %d, want 500", ignored)
	}
}
