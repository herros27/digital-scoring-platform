package orchestrator

import (
	"time"

	"github.com/herros27/digital-scoring/internal/hub"
)

// timerSyncInterval seberapa sering server menyiarkan koreksi waktu selama timer berjalan
// (variabel agar bisa dipercepat di tes).
var timerSyncInterval = 5 * time.Second

// timerPayload keadaan timer untuk klien. `timer_ends_at_ms` (epoch ms, 0 bila berhenti) adalah
// tenggat absolut: klien menghitung sisa waktu dari jam sendiri terhadap tenggat ini (setelah
// menyelaraskan jam dengan `server_time_ms`), sehingga semua layar menampilkan waktu yang sama
// dan tidak melenceng. Aman dipanggil dengan t == nil.
func timerPayload(t *Timer) map[string]any {
	p := map[string]any{
		"timer_remaining_ms": int64(0),
		"timer_running":      false,
		"timer_ends_at_ms":   int64(0),
		"server_time_ms":     time.Now().UnixMilli(),
	}
	if t == nil {
		return p
	}
	p["timer_remaining_ms"] = t.Remaining().Milliseconds()
	if endsAt, running := t.EndsAt(); running {
		p["timer_running"] = true
		p["timer_ends_at_ms"] = endsAt.UnixMilli()
	}
	return p
}

// newTimer membuat timer yang menyiarkan `timer_sync` berkala selama berjalan, supaya jam klien
// terkoreksi tanpa menunggu jeda atau event lain.
func (o *Orchestrator) newTimer(d time.Duration, onExpired func()) *Timer {
	t := NewTimer(d, onExpired)
	t.SetSync(timerSyncInterval, func() {
		o.h.Broadcast(hub.Message{Type: hub.MsgTimerSync, Payload: timerPayload(t)})
	})
	return t
}
