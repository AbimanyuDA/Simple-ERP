package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	mdapp "github.com/simple-erp/backend/internal/masterdata/application"
	"github.com/simple-erp/backend/internal/masterdata/domain"
	"net/http"
	"strings"
)

type Handler struct{ Service mdapp.Service }

func (h Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/{kind}", h.list)
	r.Post("/{kind}", h.create)
	r.Patch("/{kind}/{id}/status", h.status)
	return r
}
func kind(v string) domain.Kind {
	switch strings.ToLower(v) {
	case "products":
		return domain.Product
	case "suppliers":
		return domain.Supplier
	case "customers":
		return domain.Customer
	case "units":
		return domain.Unit
	case "warehouses":
		return domain.Warehouse
	}
	return domain.Kind("")
}
func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	items, e := h.Service.List(r.Context(), kind(chi.URLParam(r, "kind")))
	if e != nil {
		respond(w, 400, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, map[string]any{"data": items})
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct{ Code, Name string }
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&in) != nil {
		respond(w, 400, map[string]string{"message": "Data tidak valid."})
		return
	}
	p, e := authapp.PrincipalFrom(r.Context())
	if e != nil {
		respond(w, 401, map[string]string{"message": "Sesi diperlukan."})
		return
	}
	if p.Role != "ADMIN" {
		respond(w, 403, map[string]string{"message": "Akses ditolak."})
		return
	}
	item, e := h.Service.Create(r.Context(), kind(chi.URLParam(r, "kind")), in.Code, in.Name, p.UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, item)
}
func (h Handler) status(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active  bool
		Version int
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		respond(w, 400, map[string]string{"message": "Data tidak valid."})
		return
	}
	p, e := authapp.PrincipalFrom(r.Context())
	if e != nil {
		respond(w, 401, map[string]string{"message": "Sesi diperlukan."})
		return
	}
	if p.Role != "ADMIN" {
		respond(w, 403, map[string]string{"message": "Akses ditolak."})
		return
	}
	e = h.Service.SetActive(r.Context(), kind(chi.URLParam(r, "kind")), chi.URLParam(r, "id"), in.Active, in.Version, p.UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	w.WriteHeader(204)
}
func respond(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
