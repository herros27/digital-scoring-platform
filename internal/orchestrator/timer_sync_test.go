package orchestrator

import (
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/hub"
)

func TestTimerSyncBroadcastWhileRunning(t *testing.T) {
	old := timerSyncInterval
	timerSyncInterval = 40 * time.Millisecond
	defer func() { timerSyncInterval = old }()

	r := newRig(t)
	r.start(t)
	drain(r.tvCh)
	time.Sleep(250 * time.Millisecond)

	var syncs []map[string]any
	for _, m := range drain(r.tvCh) {
		if m.Type == hub.MsgTimerSync {
			syncs = append(syncs, m.Payload.(map[string]any))
		}
	}
	if len(syncs) < 3 {
		t.Fatalf("timer_sync diterima %d kali, want >= 3", len(syncs))
	}
	first := syncs[0]["timer_ends_at_ms"].(int64)
	for i, s := range syncs {
		if s["timer_running"] != true {
			t.Fatalf("sinkronisasi #%d: timer_running = %v", i, s["timer_running"])
		}
		// tenggat tetap sama selama berjalan: itulah yang membuat semua layar selaras
		if got := s["timer_ends_at_ms"].(int64); got != first {
			t.Fatalf("tenggat berubah selama berjalan: %d -> %d", first, got)
		}
		if i > 0 && s["server_time_ms"].(int64) < syncs[i-1]["server_time_ms"].(int64) {
			t.Fatal("server_time_ms harus naik monoton")
		}
	}

	// setelah dijeda, siaran berkala berhenti dan tenggat hilang
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	st := find(drain(r.tvCh), hub.MsgStateChange)
	if st == nil || st.Payload.(map[string]any)["timer_ends_at_ms"] != int64(0) {
		t.Fatalf("state_change saat jeda tidak boleh memuat tenggat: %+v", st)
	}
	time.Sleep(150 * time.Millisecond)
	for _, m := range drain(r.tvCh) {
		if m.Type == hub.MsgTimerSync {
			t.Fatal("timer_sync tidak boleh dikirim saat timer dijeda")
		}
	}
}

func TestStateChangeAndSnapshotCarryDeadline(t *testing.T) {
	r := newRig(t)
	r.start(t)

	snap := r.o.Snapshot()
	ends, now := snap["timer_ends_at_ms"].(int64), snap["server_time_ms"].(int64)
	if snap["timer_running"] != true || ends-now < 119000 || ends-now > 120100 {
		t.Fatalf("snapshot: ends-now = %d, running=%v", ends-now, snap["timer_running"])
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": -10000})
	st := find(drain(r.tvCh), hub.MsgStateChange).Payload.(map[string]any)
	ends2 := st["timer_ends_at_ms"].(int64)
	if d := ends - ends2; d < 9900 || d > 10100 {
		t.Fatalf("koreksi -10 detik harus menggeser tenggat ~10000ms, bergeser %d", d)
	}
}
