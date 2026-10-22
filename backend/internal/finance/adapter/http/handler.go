package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
	"github.com/simple-erp/backend/internal/finance/application"
)

type Handler struct{ Service application.Service }

func (h Handler) Register(r chi.Router) {
	h.RegisterSupplier(r)
	h.RegisterSales(r)
}

func (h Handler) RegisterSupplier(r chi.Router) {
	r.Get("/supplier-invoices", h.listSupplierInvoices)
	r.Post("/supplier-invoices", h.createSupplierInvoice)
	r.Post("/supplier-invoices/{id}/match", h.matchSupplierInvoice)
	r.Post("/supplier-invoices/{id}/resolve-match", h.resolveSupplierInvoice)
	r.Post("/supplier-invoices/{id}/void", h.voidSupplierInvoice)
	r.Get("/supplier-payments", h.listSupplierPayments)
	r.Post("/supplier-payments", h.createSupplierPayment)
	r.Post("/supplier-payments/{id}/post", h.postSupplierPayment)
	r.Post("/supplier-payments/{id}/void", h.voidSupplierPayment)
}

func (h Handler) RegisterSales(r chi.Router) {
	r.Get("/sales-invoices", h.listSalesInvoices)
	r.Post("/sales-orders/{id}/sales-invoice", h.createSalesInvoice)
	r.Post("/sales-invoices/{id}/post", h.postSalesInvoice)
	r.Post("/sales-invoices/{id}/void", h.voidSalesInvoice)
	r.Get("/customer-payments", h.listCustomerPayments)
	r.Post("/customer-payments", h.createCustomerPayment)
	r.Post("/customer-payments/{id}/post", h.postCustomerPayment)
	r.Post("/customer-payments/{id}/void", h.voidCustomerPayment)
}

func actor(r *http.Request) string {
	principal, _ := authapp.PrincipalFrom(r.Context())
	return principal.UserID
}
func role(r *http.Request) string {
	principal, _ := authapp.PrincipalFrom(r.Context())
	return principal.Role
}
func requireCreate(w http.ResponseWriter, r *http.Request) bool {
	if authdomain.CanCreate(role(r)) {
		return true
	}
	respond(w, http.StatusForbidden, map[string]string{"message": "Akses ditolak."})
	return false
}
func requireApprove(w http.ResponseWriter, r *http.Request) bool {
	if authdomain.CanApprove(role(r)) {
		return true
	}
	respond(w, http.StatusForbidden, map[string]string{"message": "Akses ditolak."})
	return false
}
func requestID(r *http.Request) string { return middleware.GetReqID(r.Context()) }
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"message": "Data tidak valid."})
		return false
	}
	return true
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func result(w http.ResponseWriter, document application.Document, err error, status int) {
	if err != nil {
		respond(w, http.StatusConflict, map[string]string{"message": err.Error()})
		return
	}
	respond(w, status, document)
}
func reason(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return "", false
	}
	return input.Reason, true
}
func version(w http.ResponseWriter, r *http.Request) (int, bool) {
	var input struct {
		Version int `json:"version"`
	}
	if !decode(w, r, &input) {
		return 0, false
	}
	return input.Version, true
}

func (h Handler) listSupplierInvoices(w http.ResponseWriter, r *http.Request) {
	documents, err := h.Service.ListSupplierInvoices(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca invoice."})
		return
	}
	respond(w, 200, map[string]any{"data": documents})
}
func (h Handler) createSupplierInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireCreate(w, r) {
		return
	}
	var input application.SupplierInvoiceInput
	if !decode(w, r, &input) {
		return
	}
	document, err := h.Service.CreateSupplierInvoice(r.Context(), input, actor(r), requestID(r))
	result(w, document, err, 201)
}
func (h Handler) matchSupplierInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	document, err := h.Service.MatchSupplierInvoice(r.Context(), chi.URLParam(r, "id"), actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) resolveSupplierInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireCreate(w, r) {
		return
	}
	var input struct {
		Lines []application.InvoiceLineInput `json:"lines"`
	}
	if !decode(w, r, &input) {
		return
	}
	document, err := h.Service.ResolveSupplierInvoice(r.Context(), chi.URLParam(r, "id"), input.Lines, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) voidSupplierInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := reason(w, r)
	if !ok {
		return
	}
	document, err := h.Service.VoidSupplierInvoice(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) listSupplierPayments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.Service.ListSupplierPayments(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca pembayaran."})
		return
	}
	respond(w, 200, map[string]any{"data": documents})
}
func (h Handler) createSupplierPayment(w http.ResponseWriter, r *http.Request) {
	if !requireCreate(w, r) {
		return
	}
	var input application.PaymentInput
	if !decode(w, r, &input) {
		return
	}
	document, err := h.Service.CreateSupplierPayment(r.Context(), input, actor(r), requestID(r))
	result(w, document, err, 201)
}
func (h Handler) postSupplierPayment(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := version(w, r)
	if !ok {
		return
	}
	document, err := h.Service.PostSupplierPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) voidSupplierPayment(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := reason(w, r)
	if !ok {
		return
	}
	document, err := h.Service.VoidSupplierPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) listSalesInvoices(w http.ResponseWriter, r *http.Request) {
	documents, err := h.Service.ListSalesInvoices(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca invoice."})
		return
	}
	respond(w, 200, map[string]any{"data": documents})
}
func (h Handler) createSalesInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireCreate(w, r) {
		return
	}
	var input application.SalesInvoiceInput
	if !decode(w, r, &input) {
		return
	}
	input.SalesOrderID = chi.URLParam(r, "id")
	document, err := h.Service.CreateSalesInvoice(r.Context(), input, actor(r), requestID(r))
	result(w, document, err, 201)
}
func (h Handler) postSalesInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := version(w, r)
	if !ok {
		return
	}
	document, err := h.Service.PostSalesInvoice(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) voidSalesInvoice(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := reason(w, r)
	if !ok {
		return
	}
	document, err := h.Service.VoidSalesInvoice(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) listCustomerPayments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.Service.ListCustomerPayments(r.Context())
	if err != nil {
		respond(w, 500, map[string]string{"message": "Gagal membaca pembayaran."})
		return
	}
	respond(w, 200, map[string]any{"data": documents})
}
func (h Handler) createCustomerPayment(w http.ResponseWriter, r *http.Request) {
	if !requireCreate(w, r) {
		return
	}
	var input application.PaymentInput
	if !decode(w, r, &input) {
		return
	}
	document, err := h.Service.CreateCustomerPayment(r.Context(), input, actor(r), requestID(r))
	result(w, document, err, 201)
}
func (h Handler) postCustomerPayment(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := version(w, r)
	if !ok {
		return
	}
	document, err := h.Service.PostCustomerPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
func (h Handler) voidCustomerPayment(w http.ResponseWriter, r *http.Request) {
	if !requireApprove(w, r) {
		return
	}
	value, ok := reason(w, r)
	if !ok {
		return
	}
	document, err := h.Service.VoidCustomerPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
	result(w, document, err, 200)
}
