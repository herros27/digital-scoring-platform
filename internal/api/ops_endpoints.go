package api

import (
	"crypto/subtle"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/herros27/digital-scoring/internal/api/handler"
)

// pinger dipenuhi *sql.DB.
type pinger interface{ Ping() error }

// HealthHandler jawaban sehat/tidak sehat tanpa detail. Galat database hanya dicatat di log server,
// tidak dikirim ke klien (bisa memuat alamat host dan nama pengguna basis data).
func HealthHandler(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			log.Printf("[health] database tidak dapat dihubungi: %v", err)
			handler.Fail(w, http.StatusServiceUnavailable, "UNAVAILABLE", "layanan belum siap", nil)
			return
		}
		handler.Ok(w, map[string]string{"status": "ok"})
	}
}

// MetricsAccess membatasi /metrics. Tanpa token: hanya sambungan dari mesin yang sama (loopback). Dengan
// METRICS_TOKEN: wajib header `Authorization: Bearer <token>` dari mana pun (mis. Prometheus di jaringan
// Docker). Alamat sumber memakai RemoteAddr langsung, bukan X-Forwarded-For, agar tidak bisa dipalsukan.
// Pihak yang tidak berhak mendapat 404 sehingga keberadaan endpoint tidak terlihat.
func MetricsAccess(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !metricsAllowed(token, r) {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func metricsAllowed(token string, r *http.Request) bool {
	if token != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
