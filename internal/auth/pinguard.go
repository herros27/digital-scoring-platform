package auth

import "time"

// PinGuardConfig ambang perlindungan tebak-PIN berlapis. Semua hitungan hanya untuk PIN yang SALAH.
type PinGuardConfig struct {
	Window time.Duration

	// Per perangkat (IP + ID perangkat dari tablet): satu tablet yang salah PIN berulang hanya mengunci dirinya
	// sendiri. Ini lapisan "adil" di jaringan yang berbagi satu IP (mis. Docker Desktop, proxy).
	DeviceMax     int
	DeviceLockout time.Duration

	// Per IP: ID perangkat dikirim klien sehingga bisa diganti-ganti penyerang; batas IP (lebih longgar dari
	// gabungan beberapa tablet jujur) menutup celah itu.
	IPMax     int
	IPLockout time.Duration

	// Global: batas atas total tebakan salah dari semua klien (penyerang dengan banyak IP).
	GlobalMax     int
	GlobalLockout time.Duration
}

// DefaultPinGuardConfig nilai bawaan produksi.
func DefaultPinGuardConfig() PinGuardConfig {
	return PinGuardConfig{
		Window:        15 * time.Minute,
		DeviceMax:     5,
		DeviceLockout: 15 * time.Minute,
		IPMax:         25,
		IPLockout:     15 * time.Minute,
		GlobalMax:     200,
		GlobalLockout: 2 * time.Minute,
	}
}

// PinGuard perlindungan tebak-PIN tiga lapis (perangkat, IP, global). Keberhasilan login hanya mereset hitungan
// PERANGKAT: kalau ikut mereset IP/global, penyerang yang punya satu PIN sah bisa menyisipkan login berhasil di antara
// tebakannya untuk menghapus hitungannya.
type PinGuard struct {
	device *AttemptLimiter
	ip     *AttemptLimiter
	global *AttemptLimiter
}

func NewPinGuard(c PinGuardConfig) *PinGuard {
	return &PinGuard{
		device: NewAttemptLimiter(c.DeviceMax, c.Window, c.DeviceLockout),
		ip:     NewAttemptLimiter(c.IPMax, c.Window, c.IPLockout),
		global: NewAttemptLimiter(c.GlobalMax, c.Window, c.GlobalLockout),
	}
}

const globalKey = "*"

func deviceKey(ip, device string) string {
	if device == "" {
		return ip // klien lama tanpa ID perangkat: jatuh ke kunci IP
	}
	return ip + "|" + device
}

// Check memberi tahu apakah login ditolak dan lapisan mana yang menolak ("device", "ip", "global").
func (g *PinGuard) Check(ip, device string) (locked bool, retryAfter time.Duration, scope string) {
	for _, l := range []struct {
		name string
		lim  *AttemptLimiter
		key  string
	}{
		{"device", g.device, deviceKey(ip, device)},
		{"ip", g.ip, ip},
		{"global", g.global, globalKey},
	} {
		if locked, retry := l.lim.Check(l.key); locked {
			return true, retry, l.name
		}
	}
	return false, 0, ""
}

// Fail mencatat satu PIN salah pada ketiga lapisan.
func (g *PinGuard) Fail(ip, device string) {
	g.device.Fail(deviceKey(ip, device))
	g.ip.Fail(ip)
	g.global.Fail(globalKey)
}

// Succeed dipanggil saat login berhasil: hanya hitungan perangkat itu yang dihapus.
func (g *PinGuard) Succeed(ip, device string) {
	g.device.Reset(deviceKey(ip, device))
}
