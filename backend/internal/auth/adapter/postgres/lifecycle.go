package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/domain"
)

type LifecycleStore struct{ Pool *pgxpool.Pool }

func consumeStatusAllowed(kind domain.ActionTokenKind, status string) bool {
	if kind == domain.Invitation {
		return status == "INVITED"
	}
	return kind == domain.PasswordReset && status == "ACTIVE"
}

func (s LifecycleStore) Invite(ctx context.Context, u domain.User, t domain.ActionToken, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, e = tx.Exec(ctx, `INSERT INTO users(id,email,display_name,role,status,invited_by,created_at,updated_at)VALUES($1,$2,$3,$4,'INVITED',$5,$6,$6)`, u.ID, u.Email, u.DisplayName, u.Role, u.InvitedBy, u.CreatedAt)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO action_tokens(id,user_id,kind,token_hash,expires_at,created_by,created_at)VALUES($1,$2,$3,$4,$5,$6,$7)`, t.ID, t.UserID, t.Kind, t.Hash, t.ExpiresAt, t.CreatedBy, t.CreatedAt)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'INVITE_USER','AUTH-USR-001','USER',$2,'INVITED',$3)`, u.InvitedBy, u.ID, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s LifecycleStore) Consume(ctx context.Context, digest string, kind domain.ActionTokenKind, passwordHash, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tokenID, userID, previousStatus string
	e = tx.QueryRow(ctx, `SELECT t.id,t.user_id,u.status FROM action_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=$1 AND t.kind=$2 AND t.consumed_at IS NULL AND t.revoked_at IS NULL AND t.expires_at>now() FOR UPDATE OF t,u`, digest, kind).Scan(&tokenID, &userID, &previousStatus)
	if e != nil || !consumeStatusAllowed(kind, previousStatus) {
		return domain.ErrTokenUnavailable
	}
	tag, e := tx.Exec(ctx, `UPDATE action_tokens SET consumed_at=now() WHERE id=$1 AND consumed_at IS NULL`, tokenID)
	if e != nil || tag.RowsAffected() != 1 {
		return domain.ErrTokenUnavailable
	}
	tag, e = tx.Exec(ctx, `UPDATE users SET password_hash=$1,status='ACTIVE',version=version+1,updated_at=now() WHERE id=$2 AND status=$3`, passwordHash, userID, previousStatus)
	if e != nil || tag.RowsAffected() != 1 {
		return domain.ErrTokenUnavailable
	}
	if kind == domain.PasswordReset {
		_, e = tx.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
		if e != nil {
			return e
		}
	}
	trace := "AUTH-PWD-005"
	action := "RESET_PASSWORD"
	if kind == domain.Invitation {
		trace = "AUTH-USR-005"
		action = "ACTIVATE_INVITATION"
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,request_id)VALUES($1,$2,$3,'USER',$1,$4,'ACTIVE',$5)`, userID, action, trace, previousStatus, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s LifecycleStore) CreateReset(ctx context.Context, userID string, t domain.ActionToken, actor, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `INSERT INTO action_tokens(id,user_id,kind,token_hash,expires_at,created_by,created_at) SELECT $1,id,$2,$3,$4,$5,$6 FROM users WHERE id=$7 AND status='ACTIVE'`, t.ID, t.Kind, t.Hash, t.ExpiresAt, actor, t.CreatedAt, userID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("active user not found")
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,request_id)VALUES($1,'CREATE_PASSWORD_RESET','AUTH-PWD-001','USER',$2,$3)`, actor, userID, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s LifecycleStore) MarkTokenShared(ctx context.Context, id, actor, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var kind string
	tag, e := tx.Exec(ctx, `UPDATE action_tokens SET shared_at=now() WHERE id=$1 AND shared_at IS NULL AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at>now()`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrTokenUnavailable
	}
	e = tx.QueryRow(ctx, `SELECT kind FROM action_tokens WHERE id=$1`, id).Scan(&kind)
	if e != nil {
		return e
	}
	trace := "AUTH-USR-003"
	if kind == "PASSWORD_RESET" {
		trace = "AUTH-PWD-003"
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'MARK_TOKEN_SHARED',$2,'ACTION_TOKEN',$3,'SHARED',$4)`, actor, trace, id, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s LifecycleStore) RevokeToken(ctx context.Context, id, actor, request string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var kind string
	e = tx.QueryRow(ctx, `UPDATE action_tokens SET revoked_at=now() WHERE id=$1 AND consumed_at IS NULL AND revoked_at IS NULL RETURNING kind`, id).Scan(&kind)
	if e != nil {
		return domain.ErrTokenUnavailable
	}
	trace := "ACT-AUTH-INVITE-REVOKE"
	if kind == "PASSWORD_RESET" {
		trace = "ACT-AUTH-RESET-REVOKE"
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'REVOKE_ACTION_TOKEN',$2,'ACTION_TOKEN',$3,'REVOKED',$4)`, actor, trace, id, request)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
