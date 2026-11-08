package gpx

import (
	"runtime"
	"sync"
	"time"
)

// startSampler records the peak live heap while it runs. A goroutine calls
// runtime.ReadMemStats every interval and keeps the largest HeapAlloc it saw;
// the returned stop function ends it and returns that peak. Peak live heap,
// not cumulative allocation, is what a memory budget is about (Pitfall 10), so
// nothing here or in the tests that use it reads a cumulative counter.
func startSampler(interval time.Duration) (stop func() uint64) {
	var (
		mu   sync.Mutex
		peak uint64
		done = make(chan struct{})
		quit = make(chan struct{})
	)
	sample := func() {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		mu.Lock()
		if ms.HeapAlloc > peak {
			peak = ms.HeapAlloc
		}
		mu.Unlock()
	}
	go func() {
		defer close(done)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			sample()
			select {
			case <-quit:
				return
			case <-tick.C:
			}
		}
	}()
	return func() uint64 {
		close(quit)
		<-done
		sample()
		mu.Lock()
		defer mu.Unlock()
		return peak
	}
}
