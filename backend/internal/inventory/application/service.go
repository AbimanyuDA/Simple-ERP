package application

import "context"

type AdjustmentLine struct {
	ProductID       string  `json:"productId"`
	CountedQuantity float64 `json:"countedQuantity"`
}
type Document struct {
	ID         string `json:"id"`
	DocumentNo string `json:"documentNo"`
	Status     string `json:"status"`
	CreatorID  string `json:"creatorId"`
	Version    int    `json:"version"`
}
type Balance struct {
	WarehouseID string  `json:"warehouseId"`
	ProductID   string  `json:"productId"`
	Quantity    float64 `json:"quantity"`
}
type Movement struct {
	ID          string  `json:"id"`
	WarehouseID string  `json:"warehouseId"`
	ProductID   string  `json:"productId"`
	Quantity    float64 `json:"quantity"`
	Type        string  `json:"type"`
	SourceType  string  `json:"sourceType"`
	SourceID    string  `json:"sourceId"`
}
type Store interface {
	Balances(context.Context) ([]Balance, error)
	Movements(context.Context) ([]Movement, error)
	CreateAdjustment(context.Context, string, string, []AdjustmentLine, string) (Document, error)
	TransitionAdjustment(context.Context, string, string, string, string, int, string) (Document, error)
	PostAdjustment(context.Context, string, int, string, string) (Document, error)
	ReverseAdjustment(context.Context, string, string, string, string) (Document, error)
}
type Service struct{ store Store }

func NewService(s Store) Service                                  { return Service{s} }
func (s Service) Balances(c context.Context) ([]Balance, error)   { return s.store.Balances(c) }
func (s Service) Movements(c context.Context) ([]Movement, error) { return s.store.Movements(c) }
func (s Service) CreateAdjustment(c context.Context, w, a string, l []AdjustmentLine, r string) (Document, error) {
	return s.store.CreateAdjustment(c, w, a, l, r)
}
func (s Service) Transition(c context.Context, id, action, actor, reason string, v int, r string) (Document, error) {
	return s.store.TransitionAdjustment(c, id, action, actor, reason, v, r)
}
func (s Service) Post(c context.Context, id string, v int, a, r string) (Document, error) {
	return s.store.PostAdjustment(c, id, v, a, r)
}
func (s Service) Reverse(c context.Context, id, reason, a, r string) (Document, error) {
	return s.store.ReverseAdjustment(c, id, reason, a, r)
}
