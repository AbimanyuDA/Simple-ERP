package application

import "context"

type LineInput struct {
	ProductID string  `json:"productId"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}
type ReceiptLineInput struct {
	PurchaseOrderLineID string  `json:"purchaseOrderLineId"`
	ProductID           string  `json:"productId"`
	Quantity            float64 `json:"quantity"`
}
type Document struct {
	ID         string `json:"id"`
	DocumentNo string `json:"documentNo"`
	Status     string `json:"status"`
	CreatorID  string `json:"creatorId"`
	Version    int    `json:"version"`
}
type Store interface {
	CreateRequest(context.Context, string, []LineInput, string) (Document, error)
	ListRequests(context.Context) ([]Document, error)
	TransitionRequest(context.Context, string, string, string, string, int, string) (Document, error)
	ConvertToOrder(context.Context, string, string, string, string, string) (Document, error)
	IssueOrder(context.Context, string, int, string, string) (Document, error)
	CreateReceipt(context.Context, string, []ReceiptLineInput, string, string) (Document, error)
	PostReceipt(context.Context, string, int, string, string) (Document, error)
	ReverseReceipt(context.Context, string, string, string, string) (Document, error)
	CopyRequest(context.Context, string, string, string) (Document, error)
	CancelOrder(context.Context, string, string, string, string) (Document, error)
	CancelReceipt(context.Context, string, string, string, string) (Document, error)
}
type Service struct{ store Store }

func NewService(s Store) Service { return Service{s} }
func (s Service) CreateRequest(ctx context.Context, actor string, l []LineInput, req string) (Document, error) {
	return s.store.CreateRequest(ctx, actor, l, req)
}
func (s Service) ListRequests(ctx context.Context) ([]Document, error) {
	return s.store.ListRequests(ctx)
}
func (s Service) TransitionRequest(ctx context.Context, id, action, actor, reason string, version int, request string) (Document, error) {
	return s.store.TransitionRequest(ctx, id, action, actor, reason, version, request)
}
func (s Service) ConvertToOrder(ctx context.Context, id, supplier, warehouse, actor, request string) (Document, error) {
	return s.store.ConvertToOrder(ctx, id, supplier, warehouse, actor, request)
}
func (s Service) IssueOrder(ctx context.Context, id string, version int, actor, request string) (Document, error) {
	return s.store.IssueOrder(ctx, id, version, actor, request)
}
func (s Service) CreateReceipt(ctx context.Context, po string, l []ReceiptLineInput, actor, request string) (Document, error) {
	return s.store.CreateReceipt(ctx, po, l, actor, request)
}
func (s Service) PostReceipt(ctx context.Context, id string, version int, actor, request string) (Document, error) {
	return s.store.PostReceipt(ctx, id, version, actor, request)
}
func (s Service) ReverseReceipt(ctx context.Context, id, reason, actor, request string) (Document, error) {
	return s.store.ReverseReceipt(ctx, id, reason, actor, request)
}
func (s Service) CopyRequest(c context.Context, id, actor, r string) (Document, error) {
	return s.store.CopyRequest(c, id, actor, r)
}
func (s Service) CancelOrder(c context.Context, id, reason, actor, r string) (Document, error) {
	return s.store.CancelOrder(c, id, reason, actor, r)
}
func (s Service) CancelReceipt(c context.Context, id, reason, actor, r string) (Document, error) {
	return s.store.CancelReceipt(c, id, reason, actor, r)
}
