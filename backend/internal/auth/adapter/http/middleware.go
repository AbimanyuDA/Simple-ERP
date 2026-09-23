package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/simple-erp/backend/internal/auth/application"
)

type Principal = application.Principal
type Authenticator interface {
	Authenticate(context.Context, string) (Principal, error)
	VerifyCSRF(string, string) bool
}
type Middleware struct {
	auth          Authenticator
	trustedOrigin string
	rateLimit     *authenticatedLimiter
}

func NewMiddleware(auth Authenticator, trustedOrigin string) Middleware {
	return Middleware{auth: auth, trustedOrigin: trustedOrigin, rateLimit: &authenticatedLimiter{items: map[string][]time.Time{}}}
}

type authenticatedLimiter struct {
	mu    sync.Mutex
	items map[string][]time.Time
}

func (l *authenticatedLimiter) allow(userID string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-time.Minute)
	recent := l.items[userID][:0]
	for _, observed := range l.items[userID] {
		if observed.After(cutoff) {
			recent = append(recent, observed)
		}
	}
	if len(recent) >= 120 {
		l.items[userID] = recent
		return false
	}
	l.items[userID] = append(recent, now)
	return true
}
func (m Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("__Host-erp_access")
		if err != nil {
			write(w, http.StatusUnauthorized, map[string]string{"message": "Sesi diperlukan."})
			return
		}
		p, err := m.auth.Authenticate(r.Context(), c.Value)
		if err != nil {
			write(w, http.StatusUnauthorized, map[string]string{"message": "Sesi tidak valid."})
			return
		}
		if !m.rateLimit.allow(p.UserID, time.Now()) {
			w.Header().Set("Retry-After", "60")
			write(w, http.StatusTooManyRequests, map[string]string{"message": "Terlalu banyak permintaan. Coba lagi nanti."})
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if origin := r.Header.Get("Origin"); origin != "" && origin != m.trustedOrigin {
				write(w, http.StatusForbidden, map[string]string{"message": "Origin ditolak."})
				return
			}
			if !m.auth.VerifyCSRF(r.Header.Get("X-CSRF-Token"), p.CSRFHash) {
				write(w, http.StatusForbidden, map[string]string{"message": "CSRF token tidak valid."})
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(application.WithPrincipal(r.Context(), p)))
	})
}
func (m Middleware) RequireRole(role string, next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := application.PrincipalFrom(r.Context())
		if p.Role != role {
			write(w, http.StatusForbidden, map[string]string{"message": "Akses ditolak."})
			return
		}
		next.ServeHTTP(w, r)
	}))
}
func PrincipalFrom(ctx context.Context) (Principal, error) { return application.PrincipalFrom(ctx) }
