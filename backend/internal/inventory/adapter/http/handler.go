package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/inventory/application"
	"net/http"
)

type Handler struct{ Service application.Service }

func (h Handler) Register(r chi.Router) {
	r.Get("/inventory/balances", h.balances)
	r.Get("/inventory/movements", h.movements)
	r.Post("/inventory-adjustments", h.create)
	r.Post("/inventory-adjustments/{id}/{action}", h.action)
}
func p(r *http.Request) authapp.Principal { v, _ := authapp.PrincipalFrom(r.Context()); return v }
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		respond(w, 400, map[string]string{"message": "Data tidak valid."})
		return false
	}
	return true
}
func respond(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
func (h Handler) balances(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Balances(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca saldo."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func (h Handler) movements(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Movements(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca ledger."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WarehouseID string
		Lines       []application.AdjustmentLine `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CreateAdjustment(r.Context(), in.WarehouseID, p(r).UserID, in.Lines, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, v)
}
func (h Handler) action(w http.ResponseWriter, r *http.Request) {
	a := chi.URLParam(r, "action")
	actor := p(r)
	if (a == "approve" || a == "reject" || a == "reverse") && actor.Role != "ADMIN" {
		respond(w, 403, map[string]string{"message": "Akses ditolak."})
		return
	}
	var in struct {
		Version int
		Reason  string
	}
	if !decode(w, r, &in) {
		return
	}
	var (
		v application.Document
		e error
	)
	switch a {
	case "post":
		v, e = h.Service.Post(r.Context(), chi.URLParam(r, "id"), in.Version, actor.UserID, middleware.GetReqID(r.Context()))
	case "reverse":
		v, e = h.Service.Reverse(r.Context(), chi.URLParam(r, "id"), in.Reason, actor.UserID, middleware.GetReqID(r.Context()))
	default:
		v, e = h.Service.Transition(r.Context(), chi.URLParam(r, "id"), a, actor.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
	}
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
