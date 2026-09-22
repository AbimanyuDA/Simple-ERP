package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/domain"
)

var ErrAlreadyBootstrapped = errors.New("bootstrap refused: a user already exists")

type BootstrapStore struct{ Pool *pgxpool.Pool }

func (s BootstrapStore) CreateFirstAdmin(ctx context.Context, u domain.User, traceID string) error {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(830001)); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users)").Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrAlreadyBootstrapped
	}
	_, err = tx.Exec(ctx, `INSERT INTO users (id,email,display_name,password_hash,role,status,created_at,updated_at)
		VALUES ($1,$2,$3,$4,'ADMIN','ACTIVE',$5,$5)`, u.ID, u.Email, u.DisplayName, u.PasswordHash, u.CreatedAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)
		VALUES ($1::uuid,'BOOTSTRAP_ADMIN',$2,'USER',$1::text,'ACTIVE','bootstrap-cli')`, u.ID, traceID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
