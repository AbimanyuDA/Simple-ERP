package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/simple-erp/backend/internal/reporting/application"
)

type Handler struct{ Service application.Service }

func (h Handler) RegisterDashboard(r chi.Router) { r.Get("/dashboard/summary", h.summary) }
func (h Handler) RegisterAdmin(r chi.Router) {
	r.Get("/reports/purchases", h.purchases)
	r.Get("/reports/sales", h.sales)
	r.Get("/reports/inventory", h.inventory)
	r.Get("/reports/payables", h.payables)
	r.Get("/reports/receivables", h.receivables)
	r.Get("/audit-logs", h.audit)
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h Handler) summary(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Summary(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Ringkasan gagal dimuat."})
		return
	}
	respond(w, 200, value)
}
func (h Handler) purchases(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Purchases(r.Context())
	h.rows(w, value, err)
}
func (h Handler) sales(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Sales(r.Context())
	h.rows(w, value, err)
}
func (h Handler) payables(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Payables(r.Context())
	h.rows(w, value, err)
}
func (h Handler) receivables(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Receivables(r.Context())
	h.rows(w, value, err)
}
func (h Handler) rows(w http.ResponseWriter, value []application.ReportRow, err error) {
	if err != nil {
		respond(w, 500, map[string]string{"message": "Laporan gagal dimuat."})
		return
	}
	respond(w, 200, map[string]any{"data": value})
}
func (h Handler) inventory(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Inventory(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Laporan gagal dimuat."})
		return
	}
	respond(w, 200, map[string]any{"data": value})
}
func (h Handler) audit(w http.ResponseWriter, r *http.Request) {
	value, err := h.Service.Audit(r.Context(), r.URL.Query().Get("entity"), r.URL.Query().Get("actor"), r.URL.Query().Get("action"))
	if err != nil {
		respond(w, 500, map[string]string{"message": "Audit gagal dimuat."})
		return
	}
	respond(w, 200, map[string]any{"data": value})
}
