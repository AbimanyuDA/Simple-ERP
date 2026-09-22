package application

import "context"

type InvoiceLineInput struct {
	ReferenceID string  `json:"referenceId"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
}

type SupplierInvoiceInput struct {
	PurchaseOrderID       string             `json:"purchaseOrderId"`
	SupplierInvoiceNumber string             `json:"supplierInvoiceNumber"`
	InvoiceDate           string             `json:"invoiceDate"`
	DueDate               string             `json:"dueDate"`
	Currency              string             `json:"currency"`
	ReplacesInvoiceID     string             `json:"replacesInvoiceId"`
	Lines                 []InvoiceLineInput `json:"lines"`
}

type SalesInvoiceInput struct {
	SalesOrderID      string `json:"salesOrderId"`
	InvoiceDate       string `json:"invoiceDate"`
	DueDate           string `json:"dueDate"`
	Currency          string `json:"currency"`
	ReplacesInvoiceID string `json:"replacesInvoiceId"`
}

type AllocationInput struct {
	InvoiceID string  `json:"invoiceId"`
	Amount    float64 `json:"amount"`
}

type PaymentInput struct {
	PartyID     string            `json:"partyId"`
	PaymentDate string            `json:"paymentDate"`
	Method      string            `json:"method"`
	Reference   string            `json:"reference"`
	Currency    string            `json:"currency"`
	Amount      float64           `json:"amount"`
	Allocations []AllocationInput `json:"allocations"`
}

type Document struct {
	ID          string  `json:"id"`
	DocumentNo  string  `json:"documentNo"`
	Status      string  `json:"status"`
	PartyID     string  `json:"partyId,omitempty"`
	SourceID    string  `json:"sourceId,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Total       float64 `json:"total,omitempty"`
	Outstanding float64 `json:"outstanding,omitempty"`
	Version     int     `json:"version"`
}

type Store interface {
	CreateSupplierInvoice(context.Context, SupplierInvoiceInput, string, string) (Document, error)
	MatchSupplierInvoice(context.Context, string, string, string) (Document, error)
	ResolveSupplierInvoice(context.Context, string, []InvoiceLineInput, string, string) (Document, error)
	VoidSupplierInvoice(context.Context, string, string, string, string) (Document, error)
	ListSupplierInvoices(context.Context) ([]Document, error)
	CreateSupplierPayment(context.Context, PaymentInput, string, string) (Document, error)
	PostSupplierPayment(context.Context, string, int, string, string) (Document, error)
	VoidSupplierPayment(context.Context, string, string, string, string) (Document, error)
	ListSupplierPayments(context.Context) ([]Document, error)
	CreateSalesInvoice(context.Context, SalesInvoiceInput, string, string) (Document, error)
	VoidSalesInvoice(context.Context, string, string, string, string) (Document, error)
	ListSalesInvoices(context.Context) ([]Document, error)
	CreateCustomerPayment(context.Context, PaymentInput, string, string) (Document, error)
	PostCustomerPayment(context.Context, string, int, string, string) (Document, error)
	VoidCustomerPayment(context.Context, string, string, string, string) (Document, error)
	ListCustomerPayments(context.Context) ([]Document, error)
}

type Service struct{ store Store }

func NewService(store Store) Service { return Service{store: store} }
func (s Service) CreateSupplierInvoice(c context.Context, in SupplierInvoiceInput, actor, request string) (Document, error) {
	return s.store.CreateSupplierInvoice(c, in, actor, request)
}
func (s Service) MatchSupplierInvoice(c context.Context, id, actor, request string) (Document, error) {
	return s.store.MatchSupplierInvoice(c, id, actor, request)
}
func (s Service) ResolveSupplierInvoice(c context.Context, id string, lines []InvoiceLineInput, actor, request string) (Document, error) {
	return s.store.ResolveSupplierInvoice(c, id, lines, actor, request)
}
func (s Service) VoidSupplierInvoice(c context.Context, id, reason, actor, request string) (Document, error) {
	return s.store.VoidSupplierInvoice(c, id, reason, actor, request)
}
func (s Service) ListSupplierInvoices(c context.Context) ([]Document, error) {
	return s.store.ListSupplierInvoices(c)
}
func (s Service) CreateSupplierPayment(c context.Context, in PaymentInput, actor, request string) (Document, error) {
	return s.store.CreateSupplierPayment(c, in, actor, request)
}
func (s Service) PostSupplierPayment(c context.Context, id string, version int, actor, request string) (Document, error) {
	return s.store.PostSupplierPayment(c, id, version, actor, request)
}
func (s Service) VoidSupplierPayment(c context.Context, id, reason, actor, request string) (Document, error) {
	return s.store.VoidSupplierPayment(c, id, reason, actor, request)
}
func (s Service) ListSupplierPayments(c context.Context) ([]Document, error) {
	return s.store.ListSupplierPayments(c)
}
func (s Service) CreateSalesInvoice(c context.Context, in SalesInvoiceInput, actor, request string) (Document, error) {
	return s.store.CreateSalesInvoice(c, in, actor, request)
}
func (s Service) VoidSalesInvoice(c context.Context, id, reason, actor, request string) (Document, error) {
	return s.store.VoidSalesInvoice(c, id, reason, actor, request)
}
func (s Service) ListSalesInvoices(c context.Context) ([]Document, error) {
	return s.store.ListSalesInvoices(c)
}
func (s Service) CreateCustomerPayment(c context.Context, in PaymentInput, actor, request string) (Document, error) {
	return s.store.CreateCustomerPayment(c, in, actor, request)
}
func (s Service) PostCustomerPayment(c context.Context, id string, version int, actor, request string) (Document, error) {
	return s.store.PostCustomerPayment(c, id, version, actor, request)
}
func (s Service) VoidCustomerPayment(c context.Context, id, reason, actor, request string) (Document, error) {
	return s.store.VoidCustomerPayment(c, id, reason, actor, request)
}
func (s Service) ListCustomerPayments(c context.Context) ([]Document, error) {
	return s.store.ListCustomerPayments(c)
}
