package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
)

func TestFailRegistrationConflictIs409(t *testing.T) {
	rec := httptest.NewRecorder()
	failRegistration(rec, fmt.Errorf("%w: atlet sudah terdaftar pada kejuaraan ini", domain.ErrConflict),
		http.StatusBadRequest, "BAD_REQUEST")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string }
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "CONFLICT" {
		t.Fatalf("code = %q, want CONFLICT", body.Error.Code)
	}
	if body.Error.Message != "atlet sudah terdaftar pada kejuaraan ini" {
		t.Fatalf("pesan tidak boleh memuat prefix sentinel, got %q", body.Error.Message)
	}
}

func TestFailRegistrationUsesFallback(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{http.StatusBadRequest, "BAD_REQUEST"}, {http.StatusInternalServerError, "INTERNAL_ERROR"}} {
		rec := httptest.NewRecorder()
		failRegistration(rec, fmt.Errorf("lainnya"), tc.status, tc.code)
		if rec.Code != tc.status {
			t.Fatalf("status = %d, want %d", rec.Code, tc.status)
		}
	}
}
