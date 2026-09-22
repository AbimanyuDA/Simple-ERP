package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/simple-erp/backend/internal/auth/application"
	"net/http"
)

type UsersHandler struct {
	Service   application.UserAdminService
	Lifecycle application.LifecycleService
}

func (h UsersHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/{id}/activate", h.activate)
	r.Post("/{id}/deactivate", h.deactivate)
	r.Post("/{id}/revoke-sessions", h.revoke)
	r.Post("/{id}/password-resets", h.reset)
	return r
}
func (h UsersHandler) list(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.List(r.Context())
	if e != nil {
		write(w, 500, map[string]string{"message": "Tidak dapat membaca pengguna."})
		return
	}
	write(w, 200, map[string]any{"data": v})
}
func (h UsersHandler) activate(w http.ResponseWriter, r *http.Request) { h.status(w, r, "ACTIVE") }
func (h UsersHandler) deactivate(w http.ResponseWriter, r *http.Request) {
	h.status(w, r, "DEACTIVATED")
}
func (h UsersHandler) status(w http.ResponseWriter, r *http.Request, status string) {
	var in struct{ Version int }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		write(w, 400, map[string]string{"message": "Data tidak valid."})
		return
	}
	p, _ := application.PrincipalFrom(r.Context())
	if e := h.Service.SetStatus(r.Context(), chi.URLParam(r, "id"), status, in.Version, p.UserID, middleware.GetReqID(r.Context())); e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	w.WriteHeader(204)
}
func (h UsersHandler) revoke(w http.ResponseWriter, r *http.Request) {
	p, _ := application.PrincipalFrom(r.Context())
	if e := h.Service.Revoke(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context())); e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	w.WriteHeader(204)
}
func (h UsersHandler) reset(w http.ResponseWriter, r *http.Request) {
	p, _ := application.PrincipalFrom(r.Context())
	raw, id, e := h.Lifecycle.RequestReset(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		write(w, 409, map[string]string{"message": e.Error()})
		return
	}
	write(w, 201, map[string]string{"id": id, "resetToken": raw, "message": "Token ditampilkan satu kali. Bagikan melalui kanal aman."})
}
