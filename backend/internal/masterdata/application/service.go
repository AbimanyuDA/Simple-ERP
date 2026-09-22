package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/simple-erp/backend/internal/masterdata/domain"
)

const TraceID = "CAP-MDM-001"

type Store interface {
	List(context.Context, domain.Kind) ([]domain.Record, error)
	Create(context.Context, domain.Record, string, string) error
	SetActive(context.Context, domain.Kind, string, bool, int, string, string) error
}
type Service struct{ store Store }

func NewService(s Store) Service { return Service{s} }
func (s Service) List(ctx context.Context, k domain.Kind) ([]domain.Record, error) {
	return s.store.List(ctx, k)
}
func (s Service) Create(ctx context.Context, k domain.Kind, code, name, actor, request string) (domain.Record, error) {
	r, err := domain.NewRecord(k, code, name)
	if err != nil {
		return r, err
	}
	r.ID = uuid.NewString()
	err = s.store.Create(ctx, r, actor, request)
	return r, err
}
func (s Service) SetActive(ctx context.Context, k domain.Kind, id string, active bool, version int, actor, request string) error {
	return s.store.SetActive(ctx, k, id, active, version, actor, request)
}
