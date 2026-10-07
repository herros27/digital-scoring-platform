package api

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/logger"
	"github.com/herros27/digital-scoring/internal/metrics"
)

// AuthMiddleware — validasi JWT, inject claims ke context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			handler.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing token", nil)
			return
		}
		claims, err := auth.ParseToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			handler.Fail(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
			return
		}
		ctx := auth.ContextWithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole — Role-Based Access Control (RBAC) middleware
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[strings.ToLower(r)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFrom(r)
			if claims == nil || !allowed[strings.ToLower(claims.Role)] {
				handler.Fail(w, http.StatusForbidden, "FORBIDDEN", "insufficient role permissions", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ClaimsFrom(r *http.Request) *auth.Claims {
	return auth.ClaimsFromContext(r.Context())
}

// CORS — izinkan semua origin (untuk PWA dari domain sama / dev)
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// chain — apply middleware kanan ke kiri
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ClientIP extracts client IP address from proxy headers or RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages per-IP token bucket limiters.
// ponytail: in-memory map per instance. Ceiling: single-node memory / no sync across horizontal replicas.
// Upgrade path: Redis-backed token bucket / sliding window when running multiple replicas behind a load balancer.
type IPRateLimiter struct {
	mu        sync.Mutex
	clients   map[string]*clientLimiter
	rate      rate.Limit
	burst     int
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewIPRateLimiter creates an in-memory token bucket rate limiter per client IP.
func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    r,
		burst:   burst,
		stopCh:  make(chan struct{}),
	}

	// ponytail: background cleanup goroutine prevents memory leak from abandoned IPs.
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				limiter.mu.Lock()
				now := time.Now()
				for ip, c := range limiter.clients {
					if now.Sub(c.lastSeen) > 5*time.Minute {
						delete(limiter.clients, ip)
					}
				}
				limiter.mu.Unlock()
			case <-limiter.stopCh:
				return
			}
		}
	}()

	return limiter
}

// Close stops the cleanup goroutine.
func (l *IPRateLimiter) Close() {
	l.closeOnce.Do(func() {
		close(l.stopCh)
	})
}

// Get retrieves or creates a rate.Limiter for the given IP.
func (l *IPRateLimiter) Get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, exists := l.clients[ip]
	if !exists {
		lim := rate.NewLimiter(l.rate, l.burst)
		l.clients[ip] = &clientLimiter{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	c.lastSeen = time.Now()
	return c.limiter
}

// RateLimit returns a middleware that limits requests per IP using token bucket algorithm.
func RateLimit(r rate.Limit, burst int) func(http.Handler) http.Handler {
	ipLimiter := NewIPRateLimiter(r, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := ClientIP(req)
			if !ipLimiter.Get(ip).Allow() {
				w.Header().Set("Retry-After", "60")
				handler.Fail(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests, please slow down", nil)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack meneruskan ke ResponseWriter asli agar upgrade WebSocket tetap bekerja
// melalui middleware ini (gorilla/websocket memeriksa http.Hijacker lewat type assertion).
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying ResponseWriter does not implement http.Hijacker")
	}
	conn, rw, err := hj.Hijack()
	if err == nil {
		r.statusCode = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

// Flush meneruskan ke ResponseWriter asli bila didukung (SSE/streaming).
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap memungkinkan http.NewResponseController menjangkau writer asli.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// MetricsMiddleware tracks HTTP request latency and total requests for Prometheus.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rec, r)
		duration := time.Since(start)

		// ponytail: use r.Pattern if available (Go 1.22+ routing) to avoid high-cardinality path labels with UUIDs.
		pattern := r.Pattern
		if pattern == "" {
			pattern = r.URL.Path
		} else {
			if parts := strings.SplitN(pattern, " ", 2); len(parts) == 2 {
				pattern = parts[1]
			}
		}

		metrics.RecordHTTPRequest(r.Method, pattern, rec.statusCode, duration)

		// Real-time Access Log ke stdout/console
		if logger.Log != nil {
			clientIP := ClientIP(r)
			status := rec.statusCode
			msg := r.Method + " " + r.URL.Path
			if status >= 500 {
				logger.Log.Error(msg, "status", status, "duration", duration.String(), "ip", clientIP)
			} else if status >= 400 {
				logger.Log.Warn(msg, "status", status, "duration", duration.String(), "ip", clientIP)
			} else {
				logger.Log.Info(msg, "status", status, "duration", duration.String(), "ip", clientIP)
			}
		}
	})
}
