package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/sales/application"
	"net/http"
)

type Handler struct{ Service application.Service }

func (h Handler) Register(r chi.Router) {
	r.Get("/sales-orders", h.list)
	r.Post("/sales-orders", h.create)
	r.Get("/sales-orders/{id}", h.get)
	r.Patch("/sales-orders/{id}", h.update)
	r.Post("/sales-orders/{id}/{action}", h.action)
	r.Post("/sales-orders/{id}/deliveries", h.delivery)
	r.Get("/deliveries", h.listDeliveries)
	r.Post("/deliveries/{id}/post", h.post)
	r.Post("/deliveries/{id}/reverse", h.reverse)
	r.Post("/deliveries/{id}/cancel", h.cancelDelivery)
}
func (h Handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.ListDeliveries(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca delivery."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
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
func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.ListOrders(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca sales order."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.GetOrder(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		respond(w, 404, map[string]string{"message": "Sales order tidak ditemukan."})
		return
	}
	respond(w, 200, v)
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID, WarehouseID string
		Lines                   []application.LineInput `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CreateOrder(r.Context(), in.CustomerID, in.WarehouseID, p(r).UserID, in.Lines, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, v)
}
func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int
		Lines   []application.LineInput `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.UpdateOrder(r.Context(), chi.URLParam(r, "id"), in.Version, p(r).UserID, in.Lines, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) action(w http.ResponseWriter, r *http.Request) {
	a := chi.URLParam(r, "action")
	actor := p(r)
	if (a == "approve" || a == "reject") && actor.Role != "ADMIN" {
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
	var v application.Document
	var e error
	if a == `validate-stock` {
		v, e = h.Service.ValidateStock(r.Context(), chi.URLParam(r, `id`), in.Version, actor.UserID, middleware.GetReqID(r.Context()))
		if e != nil {
			respond(w, 409, map[string]string{`message`: e.Error()})
			return
		}
		respond(w, 200, v)
		return
	}
	if a == "copy" {
		v, e = h.Service.CopyOrder(r.Context(), chi.URLParam(r, "id"), actor.UserID, middleware.GetReqID(r.Context()))
	} else {
		v, e = h.Service.TransitionOrder(r.Context(), chi.URLParam(r, "id"), a, actor.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
	}
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) cancelDelivery(w http.ResponseWriter, r *http.Request) {
	var in struct{ Reason string }
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CancelDelivery(r.Context(), chi.URLParam(r, "id"), in.Reason, p(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) delivery(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lines []application.DeliveryLineInput `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CreateDelivery(r.Context(), chi.URLParam(r, "id"), in.Lines, p(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, v)
}
func (h Handler) post(w http.ResponseWriter, r *http.Request) {
	var in struct{ Version int }
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.PostDelivery(r.Context(), chi.URLParam(r, "id"), in.Version, p(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) reverse(w http.ResponseWriter, r *http.Request) {
	if p(r).Role != "ADMIN" {
		respond(w, 403, map[string]string{"message": "Akses ditolak."})
		return
	}
	var in struct{ Reason string }
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.ReverseDelivery(r.Context(), chi.URLParam(r, "id"), in.Reason, p(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
