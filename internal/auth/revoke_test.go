package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func claimsIssuedAt(id string, t time.Time) *Claims {
	return &Claims{JudgePinID: id, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(t)}}
}

// Token yang terbit sebelum (atau pada detik yang sama dengan) pencabutan ditolak; yang terbit sesudahnya, atau milik
// akun lain, tetap sah.
func TestRevoked_OnlyTokensIssuedBeforeRevocation(t *testing.T) {
	revokedAt := time.Date(2026, 6, 1, 10, 0, 30, 500_000_000, time.UTC)
	revokeAt("akun-dicabut", revokedAt)

	cases := []struct {
		name string
		c    *Claims
		want bool
	}{
		{"terbit jauh sebelum", claimsIssuedAt("akun-dicabut", revokedAt.Add(-time.Hour)), true},
		{"terbit di detik yang sama", claimsIssuedAt("akun-dicabut", revokedAt), true},
		{"terbit sesudah", claimsIssuedAt("akun-dicabut", revokedAt.Add(2*time.Second)), false},
		{"akun lain", claimsIssuedAt("akun-lain", revokedAt.Add(-time.Hour)), false},
		{"tanpa iat dianggap dicabut", &Claims{JudgePinID: "akun-dicabut"}, true},
	}
	for _, c := range cases {
		if got := Revoked(c.c); got != c.want {
			t.Errorf("%s: Revoked = %v, want %v", c.name, got, c.want)
		}
	}
}
