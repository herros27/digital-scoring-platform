package auth

import (
	"fmt"
	"testing"
	"time"
)

func testGuard() (*PinGuard, *time.Time) {
	g := NewPinGuard(PinGuardConfig{
		Window: time.Minute, DeviceMax: 3, DeviceLockout: 10 * time.Minute,
		IPMax: 10, IPLockout: 10 * time.Minute, GlobalMax: 30, GlobalLockout: 5 * time.Minute,
	})
	clock := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	g.device.now, g.ip.now, g.global.now = now, now, now
	return g, &clock
}

func TestPinGuard_OneTabletDoesNotLockOthersOnTheSameIP(t *testing.T) {
	g, _ := testGuard()
	for i := 0; i < 3; i++ {
		g.Fail("10.0.0.1", "tablet-A-uuid")
	}
	locked, _, scope := g.Check("10.0.0.1", "tablet-A-uuid")
	if !locked || scope != "device" {
		t.Fatalf("tablet A harus terkunci oleh lapisan perangkat, got locked=%v scope=%q", locked, scope)
	}
	if locked, _, scope := g.Check("10.0.0.1", "tablet-B-uuid"); locked {
		t.Fatalf("tablet B di IP yang sama tidak boleh ikut terkunci (scope %q)", scope)
	}
}

func TestPinGuard_RotatingDeviceIDsDoesNotBypassIPLimit(t *testing.T) {
	g, _ := testGuard()
	for i := 0; i < 10; i++ { // penyerang mengganti ID perangkat tiap tebakan
		g.Fail("10.0.0.9", fmt.Sprintf("rotating-device-%02d", i))
	}
	locked, _, scope := g.Check("10.0.0.9", "device-yang-baru-sama-sekali")
	if !locked || scope != "ip" {
		t.Fatalf("ID baru dari IP yang sama harus kena lapisan IP, got locked=%v scope=%q", locked, scope)
	}
	if locked, _, _ := g.Check("10.0.0.10", "device-yang-baru-sama-sekali"); locked {
		t.Fatal("IP lain tidak boleh terdampak")
	}
}

func TestPinGuard_SuccessResetsOnlyTheDeviceNotTheIP(t *testing.T) {
	g, _ := testGuard()
	for i := 0; i < 2; i++ {
		g.Fail("10.0.0.5", "penyerang-device")
	}
	g.Succeed("10.0.0.5", "penyerang-device") // login sah di antara tebakan
	if locked, _, _ := g.Check("10.0.0.5", "penyerang-device"); locked {
		t.Fatal("hitungan perangkat harus kosong setelah login berhasil")
	}
	// hitungan IP tetap 2: delapan tebakan lagi (ID berganti) harus mengunci IP pada total 10
	for i := 0; i < 8; i++ {
		g.Fail("10.0.0.5", fmt.Sprintf("dev-%d", i))
	}
	if locked, _, scope := g.Check("10.0.0.5", "dev-baru"); !locked || scope != "ip" {
		t.Fatalf("login berhasil tidak boleh menghapus hitungan IP, got locked=%v scope=%q", locked, scope)
	}
}

func TestPinGuard_GlobalLimitBoundsDistributedGuessing(t *testing.T) {
	g, _ := testGuard()
	for i := 0; i < 30; i++ { // 30 IP berbeda, masing-masing 1 tebakan: tak ada IP/perangkat yang terkunci
		g.Fail(fmt.Sprintf("10.1.0.%d", i), "d")
	}
	locked, retry, scope := g.Check("10.9.9.9", "klien-lain")
	if !locked || scope != "global" || retry <= 0 {
		t.Fatalf("batas global harus menolak, got locked=%v scope=%q retry=%v", locked, scope, retry)
	}
}

func TestPinGuard_NoDeviceIDFallsBackToPerIPLock(t *testing.T) {
	g, _ := testGuard()
	for i := 0; i < 3; i++ {
		g.Fail("10.0.0.2", "")
	}
	if locked, _, _ := g.Check("10.0.0.2", ""); !locked {
		t.Fatal("klien tanpa ID perangkat harus tetap dikunci per IP")
	}
}

func TestPinGuard_UnlocksAfterLockoutPeriod(t *testing.T) {
	g, clock := testGuard()
	for i := 0; i < 3; i++ {
		g.Fail("10.0.0.3", "tablet")
	}
	if locked, _, _ := g.Check("10.0.0.3", "tablet"); !locked {
		t.Fatal("seharusnya terkunci")
	}
	*clock = clock.Add(11 * time.Minute)
	if locked, _, _ := g.Check("10.0.0.3", "tablet"); locked {
		t.Fatal("kunci harus terbuka setelah masa kunci habis")
	}
}
