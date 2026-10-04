package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple takes first",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.2, 198.51.100.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "203.0.113.2",
		},
		{
			name:       "X-Real-IP fallback",
			headers:    map[string]string{"X-Real-IP": "198.51.100.5"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "198.51.100.5",
		},
		{
			name:       "RemoteAddr fallback with port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.50:54321",
			expectedIP: "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			ip := ClientIP(req)
			if ip != tt.expectedIP {
				t.Fatalf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create limiter allowing 2 requests burst, refill very slowly (1 req/minute)
	limiter := NewIPRateLimiter(rate.Every(1*time.Minute), 2)
	defer limiter.Close()

	handler := limiter.Get
	_ = handler

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			if !limiter.Get(ip).Allow() {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]any{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	testHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// IP 1: First 2 requests should succeed
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1111"
		rec := httptest.NewRecorder()
		testHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("req %d from IP 1: expected 200, got %d", i, rec.Code)
		}
	}

	// IP 1: 3rd request must be blocked with 429
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	rec := httptest.NewRecorder()
	testHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("req 3 from IP 1: expected 429 Too Many Requests, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "60" {
		t.Fatalf("expected Retry-After header to be 60, got %s", rec.Header().Get("Retry-After"))
	}

	// IP 2: Should NOT be blocked because it has its own bucket
	req2 := httptest.NewRequest("POST", "/login", nil)
	req2.RemoteAddr = "10.0.0.2:2222"
	rec2 := httptest.NewRecorder()
	testHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("req 1 from IP 2: expected 200, got %d", rec2.Code)
	}
}

func TestRateLimitFunction(t *testing.T) {
	// Test the RateLimit helper function directly
	mw := RateLimit(rate.Every(1*time.Minute), 1)

	testHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1st request succeeds
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	testHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	// 2nd request fails with 429
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	testHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
}
