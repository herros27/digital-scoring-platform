package auth

import (
	"sync"
	"time"
)

// AttemptLimiter membatasi percobaan login gagal per kunci (mis. IP). Setelah `max` kegagalan
// dalam `window`, kunci dikunci selama `lockout`. Berbeda dari rate limiter biasa, hanya
// percobaan yang GAGAL yang dihitung dan keberhasilan mereset hitungan.
type AttemptLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	lockout time.Duration
	now     func() time.Time
	entries map[string]*attemptEntry
}

type attemptEntry struct {
	fails       int
	windowStart time.Time
	lockedUntil time.Time
}

func NewAttemptLimiter(max int, window, lockout time.Duration) *AttemptLimiter {
	return &AttemptLimiter{
		max:     max,
		window:  window,
		lockout: lockout,
		now:     time.Now,
		entries: make(map[string]*attemptEntry),
	}
}

// Check memberi tahu apakah kunci sedang terkunci dan berapa lama lagi.
func (l *AttemptLimiter) Check(key string) (locked bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return false, 0
	}
	if remaining := e.lockedUntil.Sub(l.now()); remaining > 0 {
		return true, remaining
	}
	return false, 0
}

// Fail mencatat satu kegagalan; mengunci kunci bila batas terlampaui.
func (l *AttemptLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	e, ok := l.entries[key]
	if !ok || now.Sub(e.windowStart) > l.window {
		e = &attemptEntry{windowStart: now}
		l.entries[key] = e
	}
	e.fails++
	if e.fails >= l.max {
		e.lockedUntil = now.Add(l.lockout)
		e.fails = 0
		e.windowStart = now
	}
}

// Reset menghapus catatan kegagalan (dipanggil saat login berhasil).
func (l *AttemptLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// sweep membuang entri kedaluwarsa agar peta tidak tumbuh tanpa batas. Pemanggil memegang l.mu.
func (l *AttemptLimiter) sweep(now time.Time) {
	if len(l.entries) < 1024 {
		return
	}
	for k, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.windowStart) > l.window {
			delete(l.entries, k)
		}
	}
}
