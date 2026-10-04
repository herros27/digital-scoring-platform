package api

import (
	"net/http"
	"strings"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
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