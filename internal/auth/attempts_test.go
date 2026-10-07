package auth

import (
	"testing"
	"time"
)

func newTestLimiter() (*AttemptLimiter, *time.Time) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	l := NewAttemptLimiter(3, time.Minute, 10*time.Minute)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestAttemptLimiter_LocksAfterMaxFailures(t *testing.T) {
	l, _ := newTestLimiter()
	for i := 0; i < 2; i++ {
		l.Fail("ip")
		if locked, _ := l.Check("ip"); locked {
			t.Fatalf("belum boleh terkunci setelah %d kegagalan", i+1)
		}
	}
	l.Fail("ip")
	locked, retry := l.Check("ip")
	if !locked || retry <= 0 || retry > 10*time.Minute {
		t.Fatalf("harus terkunci ~10 menit, locked=%v retry=%v", locked, retry)
	}
}

func TestAttemptLimiter_UnlocksAfterLockout(t *testing.T) {
	l, now := newTestLimiter()
	for i := 0; i < 3; i++ {
		l.Fail("ip")
	}
	*now = now.Add(10*time.Minute + time.Second)
	if locked, _ := l.Check("ip"); locked {
		t.Fatal("kunci harus berakhir setelah masa lockout")
	}
	// hitungan mulai dari nol lagi
	l.Fail("ip")
	if locked, _ := l.Check("ip"); locked {
		t.Fatal("satu kegagalan baru tidak boleh langsung mengunci")
	}
}

func TestAttemptLimiter_FailuresOutsideWindowDoNotAccumulate(t *testing.T) {
	l, now := newTestLimiter()
	l.Fail("ip")
	l.Fail("ip")
	*now = now.Add(2 * time.Minute) // di luar window 1 menit
	l.Fail("ip")
	if locked, _ := l.Check("ip"); locked {
		t.Fatal("kegagalan di luar window tidak boleh dijumlahkan")
	}
}

func TestAttemptLimiter_ResetAndIsolation(t *testing.T) {
	l, _ := newTestLimiter()
	l.Fail("a")
	l.Fail("a")
	l.Reset("a")
	l.Fail("a")
	if locked, _ := l.Check("a"); locked {
		t.Fatal("Reset harus menghapus hitungan")
	}
	for i := 0; i < 3; i++ {
		l.Fail("b")
	}
	if locked, _ := l.Check("a"); locked {
		t.Fatal("kunci milik b tidak boleh mempengaruhi a")
	}
	if locked, _ := l.Check("b"); !locked {
		t.Fatal("b harus terkunci")
	}
}
