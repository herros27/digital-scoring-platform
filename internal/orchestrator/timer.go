package orchestrator

import (
	"sync"
	"time"
)

type Timer struct {
	mu        sync.Mutex
	remaining time.Duration
	ticker    *time.Ticker
	stop      chan struct{}
	onExpired func()
	running   bool
}

func NewTimer(duration time.Duration, onExpired func()) *Timer {
	return &Timer{
		remaining: duration,
		onExpired: onExpired,
		stop:      make(chan struct{}),
	}
}

func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return
	}
	t.running = true
	t.ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-t.ticker.C:
				t.mu.Lock()
				t.remaining -= time.Second
				expired := t.remaining <= 0
				t.mu.Unlock()
				if expired {
					t.Stop()
					t.onExpired()
					return
				}
			case <-t.stop:
				return
			}
		}
	}()
}

func (t *Timer) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.running = false
	t.ticker.Stop()
	select {
	case t.stop <- struct{}{}:
	default:
	}
}

func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.running = false
	t.ticker.Stop()
	// drain stop channel non-blocking
	select {
	case t.stop <- struct{}{}:
	default:
	}
}

func (t *Timer) Remaining() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.remaining
}

func (t *Timer) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}
