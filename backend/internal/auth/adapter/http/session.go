package http

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/simple-erp/backend/internal/auth/application"
	"net/http"
)

type SessionHandler struct{ Service application.SessionService }

func (h SessionHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, e := r.Cookie("__Host-erp_refresh")
	if e != nil {
		write(w, 401, map[string]string{"message": "Refresh token tidak valid."})
		return
	}
	tokens, e := h.Service.Refresh(r.Context(), c.Value, middleware.GetReqID(r.Context()))
	if e != nil {
		clearCookies(w)
		write(w, 401, map[string]string{"message": "Refresh token tidak valid."})
		return
	}
	setCookie(w, "__Host-erp_access", tokens.Access, 15*60*1e9)
	setCookie(w, "__Host-erp_refresh", tokens.Refresh, 7*24*60*60*1e9)
	setCSRFCookie(w, tokens.CSRF)
	write(w, 200, map[string]string{"csrfToken": tokens.CSRF})
}
func (h SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	c, e := r.Cookie("__Host-erp_access")
	if e == nil {
		e = h.Service.Logout(r.Context(), c.Value, middleware.GetReqID(r.Context()))
	}
	clearCookies(w)
	if e != nil {
		write(w, 401, map[string]string{"message": "Sesi tidak valid."})
		return
	}
	w.WriteHeader(204)
}
func clearCookies(w http.ResponseWriter) {
	for _, n := range []string{"__Host-erp_access", "__Host-erp_refresh", "erp_csrf"} {
		http.SetCookie(w, &http.Cookie{Name: n, Value: "", Path: "/", Secure: true, HttpOnly: n != "erp_csrf", SameSite: http.SameSiteLaxMode, MaxAge: -1})
	}
}
