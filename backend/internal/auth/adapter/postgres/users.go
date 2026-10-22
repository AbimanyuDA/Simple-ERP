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
	var previous, role string
	var hasPassword bool
	e = tx.QueryRow(ctx, `SELECT status,role,password_hash IS NOT NULL AND btrim(password_hash)<>'' FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&previous, &role, &hasPassword)
	if e != nil {
		return e
	}
	if status == previous {
		return fmt.Errorf("user already has requested status")
	}
	if status == "ACTIVE" && (previous != "DEACTIVATED" || !hasPassword) {
		return fmt.Errorf("only deactivated users with a password can be reactivated")
	}
	if status == "DEACTIVATED" && role == "SUPER_ADMIN" {
		if actor == id {
			return fmt.Errorf("cannot deactivate your own super admin account")
		}
		if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(830002)); e != nil {
			return e
		}
		var active int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE role='SUPER_ADMIN' AND status='ACTIVE'`).Scan(&active); e != nil {
			return e
		}
		if active <= 1 {
			return fmt.Errorf("cannot deactivate the last active super admin")
		}
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
		if _, e = tx.Exec(ctx, `UPDATE action_tokens SET revoked_at=now() WHERE user_id=$1 AND consumed_at IS NULL AND revoked_at IS NULL`, id); e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,request_id)VALUES($1,'SET_USER_STATUS','ACT-AUTH-USER-STATUS','USER',$2,$3,$4,$5)`, actor, id, previous, status, request)
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
