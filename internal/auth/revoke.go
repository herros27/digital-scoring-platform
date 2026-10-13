package auth

import (
	"sync"
	"time"
)

// revoked: id akun -> waktu pencabutan. Token akun itu yang terbit pada atau sebelum detik pencabutan ditolak.
// shortcut: disimpan di memori proses ini dan hilang saat restart; cukup untuk satu server dan token 24 jam. Beberapa
// server, atau pencabutan yang harus bertahan lewat restart, butuh tabel database.
var revoked sync.Map

// RevokeTokensOf mencabut semua token yang sudah terbit untuk akun ini (dipanggil saat akun dihapus atau perannya
// diubah). Login berikutnya, paling cepat detik berikutnya, mendapat token baru yang sah.
func RevokeTokensOf(accountID string) { revokeAt(accountID, time.Now()) }

func revokeAt(accountID string, t time.Time) {
	if accountID != "" {
		revoked.Store(accountID, t)
	}
}

// Revoked true bila token ini terbit sebelum akunnya dicabut. IssuedAt hanya berpresisi detik, jadi token yang terbit
// pada detik yang sama dengan pencabutan ikut dicabut (lebih aman daripada meloloskannya).
func Revoked(c *Claims) bool {
	v, ok := revoked.Load(c.JudgePinID)
	if !ok {
		return false
	}
	if c.IssuedAt == nil {
		return true
	}
	return !c.IssuedAt.Time.After(v.(time.Time).Truncate(time.Second))
}
