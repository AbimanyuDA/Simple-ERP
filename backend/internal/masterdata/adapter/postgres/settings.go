package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/masterdata/domain"
)

type SettingsStore struct{ Pool *pgxpool.Pool }

func (s SettingsStore) Get(ctx context.Context) (v domain.OrganizationSettings, e error) {
	e = s.Pool.QueryRow(ctx, `SELECT organization_name,currency,timezone,default_warehouse_id FROM organization_settings WHERE singleton`).Scan(&v.OrganizationName, &v.Currency, &v.Timezone, &v.DefaultWarehouseID)
	return
}
func (s SettingsStore) Update(ctx context.Context, v domain.OrganizationSettings, actor, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `UPDATE organization_settings SET organization_name=$1,currency=$2,timezone=$3,default_warehouse_id=$4,updated_by=$5,updated_at=now() WHERE singleton`, v.OrganizationName, v.Currency, v.Timezone, v.DefaultWarehouseID, actor)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return context.Canceled
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'UPDATE_SETTINGS','CAP-MDM-001','ORGANIZATION_SETTINGS','singleton','UPDATED',$2)`, actor, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
