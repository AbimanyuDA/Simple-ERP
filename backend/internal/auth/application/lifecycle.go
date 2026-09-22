package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/simple-erp/backend/internal/auth/domain"
	"time"
)

type LifecycleStore interface {
	Invite(context.Context, domain.User, domain.ActionToken, string) error
	Consume(context.Context, string, domain.ActionTokenKind, string, string) error
	CreateReset(context.Context, string, domain.ActionToken, string, string) error
	MarkTokenShared(context.Context, string, string, string) error
	RevokeToken(context.Context, string, string, string) error
}
type LifecycleService struct {
	store  LifecycleStore
	hasher domain.PasswordHasher
	pepper []byte
}

func NewLifecycleService(s LifecycleStore, h domain.PasswordHasher, p []byte) LifecycleService {
	return LifecycleService{s, h, p}
}
func (s LifecycleService) Invite(ctx context.Context, email, name, role, actor, request string) (string, string, error) {
	now := time.Now().UTC()
	u, e := domain.InviteUser(uuid.NewString(), email, role, actor, now)
	if e != nil {
		return "", "", e
	}
	u.DisplayName = name
	raw, t, e := domain.NewActionToken(uuid.NewString(), u.ID, domain.Invitation, actor, s.pepper, now)
	if e != nil {
		return "", "", e
	}
	e = s.store.Invite(ctx, *u, *t, request)
	return raw, t.ID, e
}
func (s LifecycleService) Activate(ctx context.Context, raw, password, request string) error {
	h, e := s.hasher.Hash(password)
	if e != nil {
		return e
	}
	return s.store.Consume(ctx, domain.TokenDigest(raw, s.pepper), domain.Invitation, h, request)
}
func (s LifecycleService) RequestReset(ctx context.Context, userID, actor, request string) (string, string, error) {
	now := time.Now().UTC()
	raw, t, e := domain.NewActionToken(uuid.NewString(), userID, domain.PasswordReset, actor, s.pepper, now)
	if e != nil {
		return "", "", e
	}
	e = s.store.CreateReset(ctx, userID, *t, actor, request)
	return raw, t.ID, e
}
func (s LifecycleService) MarkShared(ctx context.Context, id, actor, request string) error {
	return s.store.MarkTokenShared(ctx, id, actor, request)
}
func (s LifecycleService) Revoke(ctx context.Context, id, actor, request string) error {
	return s.store.RevokeToken(ctx, id, actor, request)
}
func (s LifecycleService) Reset(ctx context.Context, raw, password, request string) error {
	h, e := s.hasher.Hash(password)
	if e != nil {
		return e
	}
	return s.store.Consume(ctx, domain.TokenDigest(raw, s.pepper), domain.PasswordReset, h, request)
}
