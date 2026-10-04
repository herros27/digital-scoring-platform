package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(func() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "dev-secret-change-in-prod" // ponytail: ganti via env di production
}())

// SetJWTSecret allows configuring the secret from centralized configuration.
func SetJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}

type Claims struct {
	jwt.RegisteredClaims
	JudgePinID    string `json:"pin_id"`
	CompetitionID string `json:"comp_id"`
	ArenaID       string `json:"arena_id"`
	Role          string `json:"role"`
	SeatNumber    int    `json:"seat_number,omitempty"`
}

// GenerateToken buat JWT untuk judge/referee yang sudah login PIN
func GenerateToken(pinID, competitionID, arenaID, role string, ttl time.Duration) (string, error) {
	return GenerateTokenWithSeat(pinID, competitionID, arenaID, role, 0, ttl)
}

// GenerateTokenWithSeat buat JWT untuk judge/referee dengan nomor kursi
func GenerateTokenWithSeat(pinID, competitionID, arenaID, role string, seatNumber int, ttl time.Duration) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		JudgePinID:    pinID,
		CompetitionID: competitionID,
		ArenaID:       arenaID,
		Role:          role,
		SeatNumber:    seatNumber,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// ParseToken validasi dan extract claims dari token string
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

type contextKey string

const claimsContextKey contextKey = "claims"

// ContextWithClaims attaches claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext extracts claims from context
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsContextKey).(*Claims)
	return c
}
