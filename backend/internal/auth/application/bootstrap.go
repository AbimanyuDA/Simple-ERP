package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/simple-erp/backend/internal/auth/domain"
)

const BootstrapTraceID = "ACT-AUTH-BOOTSTRAP"

type FirstAdminStore interface {
	CreateFirstAdmin(context.Context, domain.User, string) error
}

type BootstrapService struct {
	store  FirstAdminStore
	hasher domain.PasswordHasher
}

func NewBootstrapService(store FirstAdminStore, hasher domain.PasswordHasher) BootstrapService {
	return BootstrapService{store: store, hasher: hasher}
}

func (s BootstrapService) Execute(ctx context.Context, email, displayName, password string) error {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	u := domain.User{
		ID: uuid.NewString(), Email: strings.ToLower(strings.TrimSpace(email)), DisplayName: strings.TrimSpace(displayName), Role: domain.RoleSuperAdmin,
		Status: domain.UserActive, PasswordHash: hash, CreatedAt: now, UpdatedAt: now,
	}
	return s.store.CreateFirstAdmin(ctx, u, BootstrapTraceID)
}
