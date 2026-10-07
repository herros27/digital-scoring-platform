package orchestrator

import (
	"sync/atomic"
	"testing"
	"time"
)

func within(t *testing.T, got, want, tol time.Duration, msg string) {
	t.Helper()
	if d := got - want; d < -tol || d > tol {
		t.Fatalf("%s: got %v, want %v ±%v", msg, got, want, tol)
	}
}

func TestTimer_RemainingIsContinuousNotWholeSeconds(t *testing.T) {
	tm := NewTimer(2*time.Second, nil)
	tm.Start()
	defer tm.Stop()
	time.Sleep(250 * time.Millisecond)
	rem := tm.Remaining()
	// timer lama hanya berkurang per 1 detik penuh (akan menjawab tepat 2s); kini mengikuti jam
	within(t, rem, 1750*time.Millisecond, 120*time.Millisecond, "sisa waktu setelah 250ms")
	if rem%time.Second == 0 {
		t.Fatalf("sisa waktu kelipatan detik penuh (%v): masih berbasis tick", rem)
	}
}

func TestTimer_PauseKeepsExactRemainingAndResumeContinues(t *testing.T) {
	tm := NewTimer(time.Second, nil)
	tm.Start()
	time.Sleep(300 * time.Millisecond)
	tm.Pause()
	paused := tm.Remaining()
	within(t, paused, 700*time.Millisecond, 120*time.Millisecond, "sisa waktu saat jeda")
	time.Sleep(200 * time.Millisecond)
	if got := tm.Remaining(); got != paused {
		t.Fatalf("sisa waktu berubah saat dijeda: %v -> %v", paused, got)
	}
	if _, running := tm.EndsAt(); running {
		t.Fatal("EndsAt tidak boleh aktif saat dijeda")
	}
	tm.Start()
	defer tm.Stop()
	endsAt, running := tm.EndsAt()
	if !running {
		t.Fatal("EndsAt harus aktif setelah dilanjutkan")
	}
	within(t, time.Until(endsAt), paused, 50*time.Millisecond, "tenggat setelah dilanjutkan")
}

func TestTimer_ExpiresOnceAtDeadline(t *testing.T) {
	var fired atomic.Int32
	done := make(chan struct{}, 4)
	tm := NewTimer(150*time.Millisecond, func() { fired.Add(1); done <- struct{}{} })
	start := time.Now()
	tm.Start()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("callback kedaluwarsa tidak terpanggil")
	}
	within(t, time.Since(start), 150*time.Millisecond, 80*time.Millisecond, "waktu kedaluwarsa")
	time.Sleep(150 * time.Millisecond)
	if fired.Load() != 1 {
		t.Fatalf("callback terpanggil %d kali, want 1", fired.Load())
	}
	if tm.IsRunning() || tm.Remaining() != 0 {
		t.Fatalf("setelah habis: running=%v remaining=%v", tm.IsRunning(), tm.Remaining())
	}
}

func TestTimer_PausedBeforeDeadlineNeverExpires(t *testing.T) {
	var fired atomic.Int32
	tm := NewTimer(200*time.Millisecond, func() { fired.Add(1) })
	tm.Start()
	time.Sleep(50 * time.Millisecond)
	tm.Pause()
	time.Sleep(300 * time.Millisecond)
	if fired.Load() != 0 {
		t.Fatal("timer yang dijeda tidak boleh kedaluwarsa")
	}
}

func TestTimer_AdjustWhileRunningShiftsDeadline(t *testing.T) {
	var fired atomic.Int32
	tm := NewTimer(time.Second, func() { fired.Add(1) })
	tm.Start()
	defer tm.Stop()

	got := tm.Adjust(-600*time.Millisecond, 10*time.Millisecond, time.Second)
	within(t, got, 400*time.Millisecond, 50*time.Millisecond, "sisa setelah -600ms")
	endsAt, _ := tm.EndsAt()
	within(t, time.Until(endsAt), 400*time.Millisecond, 50*time.Millisecond, "tenggat bergeser")

	time.Sleep(550 * time.Millisecond)
	if fired.Load() != 1 {
		t.Fatalf("harus kedaluwarsa tepat sekali di tenggat baru, terpanggil %d kali", fired.Load())
	}
}

func TestTimer_AdjustClampsAndResetWhilePaused(t *testing.T) {
	tm := NewTimer(time.Second, nil)
	if got := tm.Adjust(time.Hour, time.Millisecond, time.Second); got != time.Second {
		t.Errorf("batas atas: %v", got)
	}
	if got := tm.Adjust(-time.Hour, 100*time.Millisecond, time.Second); got != 100*time.Millisecond {
		t.Errorf("batas bawah: %v", got)
	}
	tm.Reset(750 * time.Millisecond)
	if tm.Remaining() != 750*time.Millisecond || tm.IsRunning() {
		t.Errorf("reset saat jeda: remaining=%v running=%v", tm.Remaining(), tm.IsRunning())
	}
}

func TestTimer_SyncCallbackFiresWhileRunningAndStopsAfterPause(t *testing.T) {
	var syncs atomic.Int32
	tm := NewTimer(5*time.Second, nil)
	tm.SetSync(40*time.Millisecond, func() { syncs.Add(1) })
	tm.Start()
	time.Sleep(220 * time.Millisecond)
	tm.Pause()
	n := syncs.Load()
	if n < 3 {
		t.Fatalf("callback sinkronisasi terpanggil %d kali, want >= 3", n)
	}
	time.Sleep(200 * time.Millisecond)
	if after := syncs.Load(); after > n+1 {
		t.Fatalf("sinkronisasi harus berhenti setelah jeda: %d -> %d", n, after)
	}
}

func TestTimerPayload(t *testing.T) {
	p := timerPayload(nil)
	if p["timer_running"] != false || p["timer_ends_at_ms"] != int64(0) || p["timer_remaining_ms"] != int64(0) {
		t.Fatalf("payload timer nil: %+v", p)
	}
	if _, ok := p["server_time_ms"].(int64); !ok {
		t.Fatal("server_time_ms wajib ada")
	}

	tm := NewTimer(time.Minute, nil)
	tm.Start()
	defer tm.Stop()
	p = timerPayload(tm)
	endsAt := p["timer_ends_at_ms"].(int64)
	now := p["server_time_ms"].(int64)
	if p["timer_running"] != true || endsAt-now < 59000 || endsAt-now > 60100 {
		t.Fatalf("payload berjalan: ends=%d now=%d (selisih %d)", endsAt, now, endsAt-now)
	}
	tm.Pause()
	p = timerPayload(tm)
	if p["timer_running"] != false || p["timer_ends_at_ms"] != int64(0) {
		t.Fatalf("payload jeda tidak boleh memuat tenggat: %+v", p)
	}
}
