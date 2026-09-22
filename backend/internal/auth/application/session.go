package application

import (
	"context"
	"github.com/simple-erp/backend/internal/auth/domain"
	"time"
)

type SessionStore interface {
	ResolveRefresh(context.Context, string) (string, string, error)
	Rotate(context.Context, string, domain.Session, string) error
	Logout(context.Context, string, string) error
}
type SessionService struct {
	store  SessionStore
	pepper []byte
}

func NewSessionService(s SessionStore, p []byte) SessionService { return SessionService{s, p} }
func (s SessionService) Refresh(ctx context.Context, raw, request string) (domain.PlainTokens, error) {
	old := domain.TokenDigest(raw, s.pepper)
	user, family, e := s.store.ResolveRefresh(ctx, old)
	if e != nil {
		return domain.PlainTokens{}, e
	}
	tokens, next, e := domain.NewSessionTokens(user, family, s.pepper, time.Now().UTC())
	if e != nil {
		return tokens, e
	}
	if e = s.store.Rotate(ctx, old, next, request); e != nil {
		return domain.PlainTokens{}, e
	}
	return tokens, nil
}
func (s SessionService) Logout(ctx context.Context, raw, request string) error {
	return s.store.Logout(ctx, domain.TokenDigest(raw, s.pepper), request)
}
