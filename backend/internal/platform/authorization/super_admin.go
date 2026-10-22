package authorization

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CanSelfReview checks the persisted role, not a caller-provided bypass flag.
func CanSelfReview(ctx context.Context, tx pgx.Tx, actor string) (bool, error) {
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&allowed)
	return allowed, err
}
