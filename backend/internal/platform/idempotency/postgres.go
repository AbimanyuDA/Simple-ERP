package idempotency

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ Pool *pgxpool.Pool }

func (s PostgresStore) Claim(ctx context.Context, actor, key, hash string) (Record, bool, error) {
	tag, err := s.Pool.Exec(ctx, `INSERT INTO idempotency_keys(actor_id,key,request_hash,expires_at)
		VALUES($1,$2,$3,$4) ON CONFLICT(actor_id,key) DO NOTHING`, actor, key, hash, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		return Record{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return Record{RequestHash: hash}, true, nil
	}
	var record Record
	err = s.Pool.QueryRow(ctx, `SELECT request_hash,COALESCE(response_status,0),COALESCE(response_body,'null'::jsonb)::text::bytea
		FROM idempotency_keys WHERE actor_id=$1 AND key=$2`, actor, key).Scan(&record.RequestHash, &record.Status, &record.Body)
	return record, false, err
}

func (s PostgresStore) Complete(ctx context.Context, actor, key string, status int, body []byte) error {
	_, err := s.Pool.Exec(ctx, `UPDATE idempotency_keys SET response_status=$3,response_body=$4::jsonb
		WHERE actor_id=$1 AND key=$2`, actor, key, status, string(body))
	return err
}

func (s PostgresStore) Release(ctx context.Context, actor, key string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE actor_id=$1 AND key=$2 AND response_status IS NULL`, actor, key)
	return err
}
