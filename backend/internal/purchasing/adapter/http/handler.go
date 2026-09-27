package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/purchasing/application"
	"net/http"
)

type Handler struct{ Service application.Service }

func (h Handler) Routes() http.Handler {
	r := chi.NewRouter()
	h.Register(r)
	return r
}
func (h Handler) Register(r chi.Router) {
	r.Get("/purchase-requests", h.list)
	r.Post("/purchase-requests", h.create)
	r.Post("/purchase-requests/{id}/{action}", h.prAction)
	r.Get("/purchase-orders", h.listOrders)
	r.Get("/purchase-orders/{id}", h.getOrder)
	r.Post("/purchase-orders/{id}/issue", h.issue)
	r.Post("/purchase-orders/{id}/cancel", h.cancelOrder)
	r.Post("/purchase-orders/{id}/goods-receipts", h.receipt)
	r.Get("/goods-receipts", h.listReceipts)
	r.Post("/goods-receipts/{id}/post", h.postReceipt)
	r.Post("/goods-receipts/{id}/reverse", h.reverseReceipt)
	r.Post("/goods-receipts/{id}/cancel", h.cancelReceipt)
}
func (h Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.ListOrders(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca purchase order."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func (h Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.GetOrder(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		respond(w, 404, map[string]string{"message": "Purchase order tidak ditemukan."})
		return
	}
	respond(w, 200, v)
}
func (h Handler) listReceipts(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.ListReceipts(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca goods receipt."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func principal(r *http.Request) authapp.Principal {
	p, _ := authapp.PrincipalFrom(r.Context())
	return p
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
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
	v, e := h.Service.ListRequests(r.Context())
	if e != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca purchase request."})
		return
	}
	respond(w, 200, map[string]any{"data": v})
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Purpose string                  `json:"purpose"`
		Lines   []application.LineInput `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CreateRequest(r.Context(), principal(r).UserID, in.Purpose, in.Lines, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, v)
}
func (h Handler) prAction(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	p := principal(r)
	var in struct {
		Version                         int `json:"version"`
		Reason, SupplierID, WarehouseID string
	}
	if !decode(w, r, &in) {
		return
	}
	var (
		v application.Document
		e error
	)
	if action == "purchase-order" {
		v, e = h.Service.ConvertToOrder(r.Context(), chi.URLParam(r, "id"), in.SupplierID, in.WarehouseID, p.UserID, middleware.GetReqID(r.Context()))
	} else if action == "copy" {
		v, e = h.Service.CopyRequest(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context()))
	} else {
		if (action == "approve" || action == "reject") && p.Role != "ADMIN" {
			respond(w, 403, map[string]string{"message": "Akses ditolak."})
			return
		}
		v, e = h.Service.TransitionRequest(r.Context(), chi.URLParam(r, "id"), action, p.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
	}
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	var in struct{ Reason string }
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CancelOrder(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) cancelReceipt(w http.ResponseWriter, r *http.Request) {
	var in struct{ Reason string }
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CancelReceipt(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) issue(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int `json:"version"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.IssueOrder(r.Context(), chi.URLParam(r, "id"), in.Version, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) receipt(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lines []application.ReceiptLineInput `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.CreateReceipt(r.Context(), chi.URLParam(r, "id"), in.Lines, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 201, v)
}
func (h Handler) postReceipt(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int `json:"version"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.PostReceipt(r.Context(), chi.URLParam(r, "id"), in.Version, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
func (h Handler) reverseReceipt(w http.ResponseWriter, r *http.Request) {
	if principal(r).Role != "ADMIN" {
		respond(w, 403, map[string]string{"message": "Akses ditolak."})
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.Service.ReverseReceipt(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
	if e != nil {
		respond(w, 409, map[string]string{"message": e.Error()})
		return
	}
	respond(w, 200, v)
}
