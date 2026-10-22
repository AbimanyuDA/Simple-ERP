package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/auth/domain"
)

type Authenticator struct {
	Pool   *pgxpool.Pool
	Pepper []byte
}

func (a Authenticator) Authenticate(ctx context.Context, raw string) (application.Principal, error) {
	var p application.Principal
	err := a.Pool.QueryRow(ctx, `SELECT u.id,u.email,u.display_name,u.role,s.csrf_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.access_hash=$1 AND s.revoked_at IS NULL AND s.access_expires_at>$2 AND u.status='ACTIVE'`, domain.TokenDigest(raw, a.Pepper), time.Now().UTC()).Scan(&p.UserID, &p.Email, &p.DisplayName, &p.Role, &p.CSRFHash)
	return p, err
}
func (a Authenticator) VerifyCSRF(raw, hash string) bool {
	return raw != "" && domain.TokenDigest(raw, a.Pepper) == hash
}
