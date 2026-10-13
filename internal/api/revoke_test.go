package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
)

// Token akun yang dicabut (dihapus atau diubah perannya) ditolak 401 oleh AuthMiddleware sebelum kedaluwarsa.
func TestAuthMiddleware_RejectsRevokedAccount(t *testing.T) {
	h := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	call := func(token string) int {
		req := httptest.NewRequest("GET", "/api/v1/officials", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	token, err := auth.GenerateToken("petugas-uji-cabut", "", "", "wasit", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got := call(token); got != http.StatusOK {
		t.Fatalf("sebelum dicabut = %d, want 200", got)
	}
	auth.RevokeTokensOf("petugas-uji-cabut")
	if got := call(token); got != http.StatusUnauthorized {
		t.Fatalf("setelah dicabut = %d, want 401", got)
	}
	other, _ := auth.GenerateToken("petugas-lain", "", "", "wasit", time.Hour)
	if got := call(other); got != http.StatusOK {
		t.Fatalf("akun lain tidak boleh ikut dicabut = %d, want 200", got)
	}
}
