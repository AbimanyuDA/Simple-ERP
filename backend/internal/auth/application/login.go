package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/simple-erp/backend/internal/auth/domain"
)

var ErrInvalidCredentials = errors.New("email atau kata sandi salah")

type LoginStore interface {
	FindActiveUserByEmail(context.Context, string) (domain.User, error)
	SaveLogin(context.Context, domain.Session, string) error
}

type LoginService struct {
	store  LoginStore
	hasher domain.PasswordHasher
	pepper []byte
}

func NewLoginService(store LoginStore, hasher domain.PasswordHasher, pepper []byte) LoginService {
	return LoginService{store, hasher, pepper}
}
func (s LoginService) Login(ctx context.Context, email, password, requestID string) (domain.PlainTokens, error) {
	u, err := s.store.FindActiveUserByEmail(ctx, email)
	if err != nil {
		return domain.PlainTokens{}, ErrInvalidCredentials
	}
	ok, err := s.hasher.Verify(password, u.PasswordHash)
	if err != nil || !ok {
		return domain.PlainTokens{}, ErrInvalidCredentials
	}
	tokens, session, err := domain.NewSessionTokens(u.ID, uuid.NewString(), s.pepper, time.Now().UTC())
	if err != nil {
		return domain.PlainTokens{}, err
	}
	if err := s.store.SaveLogin(ctx, session, requestID); err != nil {
		return domain.PlainTokens{}, err
	}
	return tokens, nil
}
