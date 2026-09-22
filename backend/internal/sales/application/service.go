package application

import "context"

type LineInput struct {
	ProductID string  `json:"productId"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}
type DeliveryLineInput struct {
	SalesOrderLineID string  `json:"salesOrderLineId"`
	ProductID        string  `json:"productId"`
	Quantity         float64 `json:"quantity"`
}
type Document struct {
	ID         string `json:"id"`
	DocumentNo string `json:"documentNo"`
	Status     string `json:"status"`
	CreatorID  string `json:"creatorId"`
	Version    int    `json:"version"`
}
type Store interface {
	CreateOrder(context.Context, string, string, string, []LineInput, string) (Document, error)
	ListOrders(context.Context) ([]Document, error)
	TransitionOrder(context.Context, string, string, string, string, int, string) (Document, error)
	CreateDelivery(context.Context, string, []DeliveryLineInput, string, string) (Document, error)
	PostDelivery(context.Context, string, int, string, string) (Document, error)
	ReverseDelivery(context.Context, string, string, string, string) (Document, error)
	CopyOrder(context.Context, string, string, string) (Document, error)
	CancelDelivery(context.Context, string, string, string, string) (Document, error)
}
type Service struct{ store Store }

func NewService(s Store) Service { return Service{s} }
func (s Service) CreateOrder(c context.Context, customer, warehouse, actor string, l []LineInput, r string) (Document, error) {
	return s.store.CreateOrder(c, customer, warehouse, actor, l, r)
}
func (s Service) ListOrders(c context.Context) ([]Document, error) { return s.store.ListOrders(c) }
func (s Service) TransitionOrder(c context.Context, id, action, actor, reason string, v int, r string) (Document, error) {
	return s.store.TransitionOrder(c, id, action, actor, reason, v, r)
}
func (s Service) CreateDelivery(c context.Context, id string, l []DeliveryLineInput, actor, r string) (Document, error) {
	return s.store.CreateDelivery(c, id, l, actor, r)
}
func (s Service) PostDelivery(c context.Context, id string, v int, actor, r string) (Document, error) {
	return s.store.PostDelivery(c, id, v, actor, r)
}
func (s Service) ReverseDelivery(c context.Context, id, reason, actor, r string) (Document, error) {
	return s.store.ReverseDelivery(c, id, reason, actor, r)
}
func (s Service) CopyOrder(c context.Context, id, actor, r string) (Document, error) {
	return s.store.CopyOrder(c, id, actor, r)
}
func (s Service) CancelDelivery(c context.Context, id, reason, actor, r string) (Document, error) {
	return s.store.CancelDelivery(c, id, reason, actor, r)
}
