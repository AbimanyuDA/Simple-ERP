package application

import (
	"context"
	"github.com/simple-erp/backend/internal/masterdata/domain"
)

type SettingsStore interface {
	Get(context.Context) (domain.OrganizationSettings, error)
	Update(context.Context, domain.OrganizationSettings, string, string) error
}
type SettingsService struct{ store SettingsStore }

func NewSettingsService(s SettingsStore) SettingsService { return SettingsService{s} }
func (s SettingsService) Get(ctx context.Context) (domain.OrganizationSettings, error) {
	return s.store.Get(ctx)
}
func (s SettingsService) Update(ctx context.Context, v domain.OrganizationSettings, actor, request string) error {
	if e := v.Validate(); e != nil {
		return e
	}
	return s.store.Update(ctx, v, actor, request)
}
