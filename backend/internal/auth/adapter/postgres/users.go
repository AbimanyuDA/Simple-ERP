package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/application"
)

type UserAdminStore struct{ Pool *pgxpool.Pool }

func (s UserAdminStore) ListUsers(ctx context.Context) ([]application.UserView, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,email,display_name,role,status,version FROM users ORDER BY created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []application.UserView{}
	for rows.Next() {
		var v application.UserView
		if e = rows.Scan(&v.ID, &v.Email, &v.DisplayName, &v.Role, &v.Status, &v.Version); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s UserAdminStore) SetUserStatus(ctx context.Context, id, status string, version int, actor, request string) error {
	if status != "ACTIVE" && status != "DEACTIVATED" {
		return fmt.Errorf("invalid status")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previous string
	e = tx.QueryRow(ctx, `SELECT status FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&previous)
	if e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, `UPDATE users SET status=$1,version=version+1,updated_at=now() WHERE id=$2 AND version=$3`, status, id, version)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("record conflict")
	}
	if status == "DEACTIVATED" {
		if _, e = tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id); e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,request_id)VALUES($1,'SET_USER_STATUS','AUTH-USR-004','USER',$2,$3,$4,$5)`, actor, id, previous, status, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s UserAdminStore) RevokeSessions(ctx context.Context, id, actor, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, e = tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'REVOKE_SESSIONS','ACT-AUTH-SESSION-REVOKE','USER',$2,'SESSIONS_REVOKED',$3)`, actor, id, request); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
