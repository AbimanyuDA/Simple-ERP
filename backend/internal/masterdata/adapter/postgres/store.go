package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/masterdata/domain"
)

type Store struct{ Pool *pgxpool.Pool }

func table(k domain.Kind) (string, error) {
	switch k {
	case domain.Product:
		return "products", nil
	case domain.Supplier:
		return "suppliers", nil
	case domain.Customer:
		return "customers", nil
	case domain.Unit:
		return "units", nil
	case domain.Warehouse:
		return "warehouses", nil
	}
	return "", fmt.Errorf("invalid kind")
}
func (s Store) List(ctx context.Context, k domain.Kind) ([]domain.Record, error) {
	t, e := table(k)
	if e != nil {
		return nil, e
	}
	code := "code"
	if k == domain.Product {
		code = "sku"
	}
	rows, e := s.Pool.Query(ctx, "SELECT id,"+code+",name,active,version FROM "+t+" ORDER BY "+code)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Record{}
	for rows.Next() {
		r := domain.Record{Kind: k}
		if e = rows.Scan(&r.ID, &r.Code, &r.Name, &r.Active, &r.Version); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s Store) Create(ctx context.Context, r domain.Record, actor, request string) error {
	t, e := table(r.Kind)
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if r.Kind == domain.Product {
		_, e = tx.Exec(ctx, `INSERT INTO products(id,sku,name,unit_id)SELECT $1,$2,$3,id FROM units WHERE code='PCS'`, r.ID, r.Code, r.Name)
	} else {
		_, e = tx.Exec(ctx, "INSERT INTO "+t+"(id,code,name) VALUES($1,$2,$3)", r.ID, r.Code, r.Name)
	}
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'CREATE_MASTER_DATA','CAP-MDM-001',$2,$3,'ACTIVE',$4)`, actor, string(r.Kind), r.ID, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Store) SetActive(ctx context.Context, k domain.Kind, id string, active bool, version int, actor, request string) error {
	t, e := table(k)
	if e != nil {
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, "UPDATE "+t+" SET active=$1,version=version+1,updated_at=now() WHERE id=$2 AND version=$3", active, id, version)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("record conflict")
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'SET_MASTER_DATA_STATUS','CAP-MDM-001',$2,$3,$4,$5)`, actor, string(k), id, map[bool]string{true: "ACTIVE", false: "INACTIVE"}[active], request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
