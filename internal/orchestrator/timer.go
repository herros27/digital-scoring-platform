package orchestrator

import (
	"sync"
	"time"
)

// Timer hitung mundur berbasis tenggat (deadline) absolut, bukan penghitung yang dikurangi tiap detik.
// Sisa waktu selalu dihitung dari jam (presisi milidetik) sehingga tidak pernah basi atau melenceng,
// dan klien dapat menampilkan waktu yang sama persis dari `ends_at` yang sama.
type Timer struct {
	mu        sync.Mutex
	remaining time.Duration // sisa waktu saat tidak berjalan
	deadline  time.Time     // tenggat saat berjalan
	running   bool
	gen       uint64 // naik setiap start/stop/adjust agar callback lama diabaikan
	timer     *time.Timer
	onExpired func()

	syncEvery time.Duration
	onSync    func()
}

func NewTimer(duration time.Duration, onExpired func()) *Timer {
	return &Timer{remaining: duration, onExpired: onExpired}
}

// SetSync memasang callback yang dipanggil berkala selama timer berjalan (untuk menyiarkan
// koreksi waktu ke klien). Harus dipanggil sebelum Start.
func (t *Timer) SetSync(every time.Duration, fn func()) {
	t.mu.Lock()
	t.syncEvery, t.onSync = every, fn
	t.mu.Unlock()
}

func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return
	}
	t.running = true
	t.scheduleLocked(t.remaining)
}

// scheduleLocked menjadwalkan kedaluwarsa dan loop sinkronisasi untuk generasi baru. Pemanggil memegang t.mu.
func (t *Timer) scheduleLocked(d time.Duration) {
	t.gen++
	gen := t.gen
	t.deadline = time.Now().Add(d)
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(d, func() { t.expire(gen) })
	if t.syncEvery > 0 && t.onSync != nil {
		go t.syncLoop(gen, t.syncEvery, t.onSync)
	}
}

func (t *Timer) expire(gen uint64) {
	t.mu.Lock()
	if !t.running || t.gen != gen {
		t.mu.Unlock()
		return
	}
	t.running = false
	t.remaining = 0
	t.gen++
	cb := t.onExpired
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (t *Timer) syncLoop(gen uint64, every time.Duration, fn func()) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for range ticker.C {
		t.mu.Lock()
		alive := t.running && t.gen == gen
		t.mu.Unlock()
		if !alive {
			return
		}
		fn()
	}
}

func (t *Timer) Pause() { t.stopInternal() }
func (t *Timer) Stop()  { t.stopInternal() }

func (t *Timer) stopInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.remaining = t.remainingLocked()
	t.running = false
	t.gen++
	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *Timer) remainingLocked() time.Duration {
	if !t.running {
		return t.remaining
	}
	if left := time.Until(t.deadline); left > 0 {
		return left
	}
	return 0
}

func (t *Timer) Remaining() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.remainingLocked()
}

func (t *Timer) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// EndsAt tenggat berakhirnya waktu saat timer berjalan; ok=false bila sedang berhenti/jeda.
func (t *Timer) EndsAt() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return time.Time{}, false
	}
	return t.deadline, true
}

// Adjust menambah/mengurangi sisa waktu (delta boleh negatif) dan membatasinya ke [min, max].
// Berlaku juga saat timer berjalan (tenggat digeser). Mengembalikan sisa waktu baru.
func (t *Timer) Adjust(delta, min, max time.Duration) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	next := t.remainingLocked() + delta
	if next > max {
		next = max
	}
	if next < min {
		next = min
	}
	t.setLocked(next)
	return next
}

// Reset mengatur sisa waktu ke durasi tertentu tanpa mengubah status berjalan/jeda.
func (t *Timer) Reset(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.setLocked(d)
}

func (t *Timer) setLocked(d time.Duration) {
	if t.running {
		t.scheduleLocked(d)
		return
	}
	t.remaining = d
}
