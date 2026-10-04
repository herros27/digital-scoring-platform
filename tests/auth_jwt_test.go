package tests

import (
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
)

func TestAuth_JWTGenerationAndParsing(t *testing.T) {
	auth.SetJWTSecret("test-secret-key-12345")

	pinID := "pin-uuid-1"
	compID := "comp-uuid-1"
	arenaID := "arena-uuid-1"
	role := "judge_1"
	ttl := 2 * time.Hour

	tokenStr, err := auth.GenerateToken(pinID, compID, arenaID, role, ttl)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatalf("expected non-empty token string")
	}

	claims, err := auth.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.JudgePinID != pinID {
		t.Errorf("expected PinID %s, got %s", pinID, claims.JudgePinID)
	}
	if claims.CompetitionID != compID {
		t.Errorf("expected CompetitionID %s, got %s", compID, claims.CompetitionID)
	}
	if claims.ArenaID != arenaID {
		t.Errorf("expected ArenaID %s, got %s", arenaID, claims.ArenaID)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}
}

func TestAuth_ExpiredTokenRejected(t *testing.T) {
	auth.SetJWTSecret("test-secret-key-12345")

	// Expired immediately (-1 second TTL)
	tokenStr, err := auth.GenerateToken("p1", "c1", "a1", "referee", -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, errParse := auth.ParseToken(tokenStr)
	if errParse == nil {
		t.Fatalf("expected error parsing expired token, got nil")
	}
}

func TestAuth_InvalidSignatureRejected(t *testing.T) {
	auth.SetJWTSecret("secret-A")
	tokenStr, _ := auth.GenerateToken("p1", "c1", "a1", "judge_2", time.Hour)

	// Change secret key on verifier
	auth.SetJWTSecret("secret-B")
	_, err := auth.ParseToken(tokenStr)
	if err == nil {
		t.Fatalf("expected error verifying token with different secret, got nil")
	}
}

func TestAuth_HashAndVerifyPIN(t *testing.T) {
	pin := "123456"
	hash, err := auth.HashPIN(pin)
	if err != nil {
		t.Fatalf("HashPIN failed: %v", err)
	}

	if !auth.VerifyPIN(pin, hash) {
		t.Errorf("VerifyPIN should return true for correct PIN")
	}

	if auth.VerifyPIN("654321", hash) {
		t.Errorf("VerifyPIN should return false for incorrect PIN")
	}
}
