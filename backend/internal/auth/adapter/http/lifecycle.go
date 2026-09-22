package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/simple-erp/backend/internal/auth/application"
	"net/http"
)

type LifecycleHandler struct{ Service application.LifecycleService }

func (h LifecycleHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/auth/activate-invitation", h.activate)
	r.Post("/auth/reset-password", h.reset)
	return r
}
func (h LifecycleHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/user-invitations", h.invite)
	r.Post("/user-invitations/{id}/mark-shared", h.shared)
	r.Post("/user-invitations/{id}/revoke", h.revoke)
	r.Post("/password-resets/{id}/mark-shared", h.shared)
	r.Post("/password-resets/{id}/revoke", h.revoke)
	r.Post("/users/{id}/password-resets", h.requestReset)
	return r
}
func (h LifecycleHandler) invite(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, DisplayName, Role string }
	if decode(w, r, &in) != nil {
		return
	}
	p, _ := application.PrincipalFrom(r.Context())
	raw, id, e := h.Service.Invite(r.Context(), in.Email, in.DisplayName, in.Role, p.UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	write(w, 201, map[string]string{"id": id, "invitationToken": raw, "message": "Token ditampilkan satu kali. Bagikan melalui kanal aman."})
}
func (h LifecycleHandler) activate(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if decode(w, r, &in) != nil {
		return
	}
	if e := h.Service.Activate(r.Context(), in.Token, in.Password, middleware.GetReqID(r.Context())); e != nil {
		write(w, 400, map[string]string{"message": "Token tidak valid atau kedaluwarsa."})
		return
	}
	w.WriteHeader(204)
}
func (h LifecycleHandler) requestReset(w http.ResponseWriter, r *http.Request) {
	p, _ := application.PrincipalFrom(r.Context())
	raw, id, e := h.Service.RequestReset(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	write(w, 201, map[string]string{"id": id, "resetToken": raw, "message": "Token ditampilkan satu kali. Bagikan melalui kanal aman."})
}
func (h LifecycleHandler) shared(w http.ResponseWriter, r *http.Request) {
	p, _ := application.PrincipalFrom(r.Context())
	if e := h.Service.MarkShared(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context())); e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	w.WriteHeader(204)
}
func (h LifecycleHandler) revoke(w http.ResponseWriter, r *http.Request) {
	p, _ := application.PrincipalFrom(r.Context())
	if e := h.Service.Revoke(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context())); e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	w.WriteHeader(204)
}
func (h LifecycleHandler) reset(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if decode(w, r, &in) != nil {
		return
	}
	if e := h.Service.Reset(r.Context(), in.Token, in.Password, middleware.GetReqID(r.Context())); e != nil {
		write(w, 400, map[string]string{"message": "Token tidak valid atau kedaluwarsa."})
		return
	}
	w.WriteHeader(204)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(v)
	if e != nil {
		write(w, 400, map[string]string{"message": "Data masuk tidak valid."})
	}
	return e
}
