package domain

import (
	"errors"
	"time"
)

type ActionTokenKind string

const (
	Invitation    ActionTokenKind = "INVITATION"
	PasswordReset ActionTokenKind = "PASSWORD_RESET"
)

var ErrTokenUnavailable = errors.New("token expired, consumed, or revoked")

type ActionToken struct {
	ID, UserID, Hash, CreatedBy string
	Kind                        ActionTokenKind
	CreatedAt, ExpiresAt        time.Time
	ConsumedAt, RevokedAt       *time.Time
}

func NewActionToken(id, userID string, kind ActionTokenKind, createdBy string, pepper []byte, now time.Time) (string, *ActionToken, error) {
	raw, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	return raw, &ActionToken{ID: id, UserID: userID, Hash: hashToken(raw, pepper), Kind: kind, CreatedBy: createdBy, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}, nil
}
func (t *ActionToken) Consume(now time.Time) error {
	if t.ConsumedAt != nil || t.RevokedAt != nil || !now.Before(t.ExpiresAt) {
		return ErrTokenUnavailable
	}
	t.ConsumedAt = &now
	return nil
}
func TokenDigest(raw string, pepper []byte) string { return hashToken(raw, pepper) }
