package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/auth"
)

func TestArenaAllowed(t *testing.T) {
	cases := []struct {
		name   string
		claims *auth.Claims
		arena  string
		want   bool
	}{
		{"tanpa klaim (ditolak di tempat lain oleh middleware)", nil, "a1", true},
		{"login email admin/dewan/wasit tidak terikat gelanggang", &auth.Claims{Role: "wasit"}, "a1", true},
		{"PIN wasit di gelanggangnya sendiri", &auth.Claims{Role: "wasit", ArenaID: "a1"}, "a1", true},
		{"PIN wasit di gelanggang lain", &auth.Claims{Role: "wasit", ArenaID: "a2"}, "a1", false},
		{"PIN juri di gelanggang lain", &auth.Claims{Role: "judge", ArenaID: "a2"}, "a1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/x", nil)
			if tc.claims != nil {
				req = req.WithContext(auth.ContextWithClaims(context.Background(), tc.claims))
			}
			rec := httptest.NewRecorder()
			if got := arenaAllowed(rec, req, tc.arena); got != tc.want {
				t.Fatalf("arenaAllowed = %v, want %v", got, tc.want)
			}
			if !tc.want && rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", rec.Code)
			}
			if tc.want && rec.Body.Len() != 0 {
				t.Fatalf("tidak boleh menulis respons saat diizinkan: %s", rec.Body.String())
			}
		})
	}
}
