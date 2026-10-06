package orchestrator

import (
	"sync"
	"time"
)

type Timer struct {
	mu        sync.Mutex
	remaining time.Duration
	ticker    *time.Ticker
	stopCh    chan struct{}
	onExpired func()
	running   bool
}

func NewTimer(duration time.Duration, onExpired func()) *Timer {
	return &Timer{
		remaining: duration,
		onExpired: onExpired,
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
	t.stopCh = make(chan struct{})

	ticker := t.ticker
	stopCh := t.stopCh

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.mu.Lock()
				t.remaining -= time.Second
				expired := t.remaining <= 0
				t.mu.Unlock()

				if expired {
					t.Stop()
					if t.onExpired != nil {
						t.onExpired()
					}
					return
				}
			case <-stopCh:
				return
			}
		}
	}()
}

func (t *Timer) Pause() {
	t.stopInternal()
}

func (t *Timer) Stop() {
	t.stopInternal()
}

func (t *Timer) stopInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.running = false
	if t.ticker != nil {
		t.ticker.Stop()
	}
	if t.stopCh != nil {
		close(t.stopCh)
		t.stopCh = nil
	}
}

func (t *Timer) Remaining() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.remaining
}

// Adjust menambah/mengurangi sisa waktu (delta boleh negatif) dan membatasinya ke [min, max].
// Mengembalikan sisa waktu baru.
func (t *Timer) Adjust(delta, min, max time.Duration) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.remaining += delta
	if t.remaining > max {
		t.remaining = max
	}
	if t.remaining < min {
		t.remaining = min
	}
	return t.remaining
}

// Reset mengatur sisa waktu ke durasi tertentu tanpa mengubah status berjalan/jeda.
func (t *Timer) Reset(d time.Duration) {
	t.mu.Lock()
	t.remaining = d
	t.mu.Unlock()
}

func (t *Timer) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}
