package http

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/simple-erp/backend/internal/auth/domain"
)

type LoginUseCase interface {
	Login(context.Context, string, string, string) (domain.PlainTokens, error)
}
type Handler struct {
	login    LoginUseCase
	attempts *attemptLimiter
}

func NewHandler(login LoginUseCase) Handler {
	return Handler{login: login, attempts: &attemptLimiter{items: map[string][]time.Time{}}}
}

type attemptLimiter struct {
	mu    sync.Mutex
	items map[string][]time.Time
}

func (l *attemptLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-15 * time.Minute)
	recent := l.items[key][:0]
	for _, v := range l.items[key] {
		if v.After(cut) {
			recent = append(recent, v)
		}
	}
	l.items[key] = recent
	return len(recent) < 5
}
func (l *attemptLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items[key] = append(l.items[key], time.Now())
}
func (l *attemptLimiter) clear(key string) { l.mu.Lock(); defer l.mu.Unlock(); delete(l.items, key) }
func (h Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/auth/login", h.handleLogin)
	return r
}
func (h Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		write(w, http.StatusUnsupportedMediaType, map[string]string{"message": "Content-Type harus application/json"})
		return
	}
	var in struct{ Email, Password string }
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil {
		write(w, http.StatusBadRequest, map[string]string{"message": "Data masuk tidak valid."})
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	key := strings.ToLower(strings.TrimSpace(in.Email)) + "|" + host
	if !h.attempts.allow(key) {
		write(w, http.StatusTooManyRequests, map[string]string{"message": "Terlalu banyak percobaan. Coba lagi nanti."})
		return
	}
	tokens, err := h.login.Login(r.Context(), in.Email, in.Password, middleware.GetReqID(r.Context()))
	if err != nil {
		h.attempts.fail(key)
		write(w, http.StatusUnauthorized, map[string]string{"message": "Email atau kata sandi salah."})
		return
	}
	h.attempts.clear(key)
	setCookie(w, "__Host-erp_access", tokens.Access, 15*time.Minute)
	setCookie(w, "__Host-erp_refresh", tokens.Refresh, 7*24*time.Hour)
	setCSRFCookie(w, tokens.CSRF)
	write(w, http.StatusOK, map[string]string{"csrfToken": tokens.CSRF})
}
func setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(ttl), MaxAge: int(ttl.Seconds())})
}
func setCSRFCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{Name: "erp_csrf", Value: value, Path: "/", Secure: true, HttpOnly: false, SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(7 * 24 * time.Hour), MaxAge: int((7 * 24 * time.Hour).Seconds())})
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
