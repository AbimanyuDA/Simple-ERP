package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/domain"
)

type LoginStore struct{ Pool *pgxpool.Pool }

func (s LoginStore) FindActiveUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx, `SELECT id,email,display_name,password_hash,role,status FROM users WHERE email=$1 AND status='ACTIVE'`, strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Status)
	return u, err
}
func (s LoginStore) SaveLogin(ctx context.Context, session domain.Session, requestID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO sessions(user_id,family_id,access_hash,refresh_hash,csrf_hash,access_expires_at,refresh_idle_expires_at,refresh_absolute_expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, session.UserID, session.FamilyID, session.AccessHash, session.RefreshHash, session.CSRFHash, session.AccessExpiresAt, session.RefreshIdleExpiresAt, session.RefreshAbsoluteExpiresAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,new_state,request_id) VALUES($1,'LOGIN','CAP-AUTH-001','SESSION',$2,'ACTIVE',$3)`, session.UserID, session.FamilyID, requestID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
