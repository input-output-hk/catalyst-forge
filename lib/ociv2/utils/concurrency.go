package utils

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// safeCounter provides thread-safe counting
type safeCounter struct {
	value int64
}

// Inc increments the counter
func (sc *safeCounter) Inc() int64 {
	return atomic.AddInt64(&sc.value, 1)
}

// Dec decrements the counter
func (sc *safeCounter) Dec() int64 {
	return atomic.AddInt64(&sc.value, -1)
}

// Get returns the current value
func (sc *safeCounter) Get() int64 {
	return atomic.LoadInt64(&sc.value)
}

// Set sets the counter value
func (sc *safeCounter) Set(v int64) {
	atomic.StoreInt64(&sc.value, v)
}

// safeMap provides a thread-safe map
type safeMap struct {
	m  map[string]interface{}
	mu sync.RWMutex
}

// newSafeMap creates a new thread-safe map
func newSafeMap() *safeMap {
	return &safeMap{
		m: make(map[string]interface{}),
	}
}

// Set sets a value in the map
func (sm *safeMap) Set(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

// Get retrieves a value from the map
func (sm *safeMap) Get(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.m[key]
	return v, ok
}

// Delete removes a value from the map
func (sm *safeMap) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.m, key)
}

// Len returns the number of items in the map
func (sm *safeMap) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.m)
}

// workPool manages a pool of workers for parallel execution
type workPool struct {
	workers   int
	queue     chan func()
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// newWorkPool creates a new work pool
func newWorkPool(ctx context.Context, workers int) *workPool {
	poolCtx, cancel := context.WithCancel(ctx)
	
	wp := &workPool{
		workers: workers,
		queue:   make(chan func(), workers*2),
		ctx:     poolCtx,
		cancel:  cancel,
	}
	
	// Start workers
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
	
	return wp
}

// worker processes tasks from the queue
func (wp *workPool) worker() {
	defer wp.wg.Done()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case task, ok := <-wp.queue:
			if !ok {
				return
			}
			if task != nil {
				task()
			}
		}
	}
}

// Submit submits a task to the work pool
func (wp *workPool) Submit(task func()) error {
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	case wp.queue <- task:
		return nil
	}
}

// Close shuts down the work pool
func (wp *workPool) Close() {
	wp.closeOnce.Do(func() {
		wp.cancel()
		close(wp.queue)
		wp.wg.Wait()
	})
}

// rateLimiter provides thread-safe rate limiting
type rateLimiter struct {
	tokens    chan struct{}
	ticker    *time.Ticker
	closeOnce sync.Once
	done      chan struct{}
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(ratePerSecond int) *rateLimiter {
	rl := &rateLimiter{
		tokens: make(chan struct{}, ratePerSecond),
		ticker: time.NewTicker(time.Second / time.Duration(ratePerSecond)),
		done:   make(chan struct{}),
	}
	
	// Fill initial tokens
	for i := 0; i < ratePerSecond; i++ {
		rl.tokens <- struct{}{}
	}
	
	// Start token refill goroutine
	go rl.refill()
	
	return rl
}

// refill adds tokens at the configured rate
func (rl *rateLimiter) refill() {
	for {
		select {
		case <-rl.done:
			return
		case <-rl.ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full
			}
		}
	}
}

// Wait blocks until a token is available
func (rl *rateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	}
}

// Close stops the rate limiter
func (rl *rateLimiter) Close() {
	rl.closeOnce.Do(func() {
		rl.ticker.Stop()
		close(rl.done)
	})
}