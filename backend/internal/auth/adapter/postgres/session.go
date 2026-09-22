package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/domain"
)

var ErrRefreshReuse = errors.New("refresh token reuse detected")

type SessionStore struct{ Pool *pgxpool.Pool }

func (s SessionStore) ResolveRefresh(ctx context.Context, hash string) (user, family string, e error) {
	var rotated, revoked bool
	e = s.Pool.QueryRow(ctx, `SELECT user_id,family_id,rotated_at IS NOT NULL,revoked_at IS NOT NULL FROM sessions WHERE refresh_hash=$1`, hash).Scan(&user, &family, &rotated, &revoked)
	if e != nil {
		return
	}
	if rotated || revoked {
		_, _ = s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE family_id=$1`, family)
		e = ErrRefreshReuse
	}
	return
}
func (s SessionStore) Rotate(ctx context.Context, old string, n domain.Session, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `UPDATE sessions SET rotated_at=now(),revoked_at=now() WHERE refresh_hash=$1 AND rotated_at IS NULL AND revoked_at IS NULL AND refresh_idle_expires_at>now() AND refresh_absolute_expires_at>now()`, old)
	if e != nil || tag.RowsAffected() != 1 {
		return ErrRefreshReuse
	}
	_, e = tx.Exec(ctx, `INSERT INTO sessions(user_id,family_id,access_hash,refresh_hash,csrf_hash,access_expires_at,refresh_idle_expires_at,refresh_absolute_expires_at)VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, n.UserID, n.FamilyID, n.AccessHash, n.RefreshHash, n.CSRFHash, n.AccessExpiresAt, n.RefreshIdleExpiresAt, n.RefreshAbsoluteExpiresAt)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'REFRESH_SESSION','CAP-AUTH-001','SESSION',$2,'ROTATED',$3)`, n.UserID, n.FamilyID, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s SessionStore) Logout(ctx context.Context, accessHash, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var user, id string
	e = tx.QueryRow(ctx, `UPDATE sessions SET revoked_at=now() WHERE access_hash=$1 AND revoked_at IS NULL RETURNING user_id,id`, accessHash).Scan(&user, &id)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'LOGOUT','CAP-AUTH-001','SESSION',$2,'REVOKED',$3)`, user, id, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
