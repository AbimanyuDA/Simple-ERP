package application

import (
	"context"
	"errors"
	"testing"

	"github.com/simple-erp/backend/internal/auth/domain"
)

type bootstrapStoreFake struct{ called bool }

func (s *bootstrapStoreFake) CreateFirstAdmin(_ context.Context, u domain.User, traceID string) error {
	s.called = true
	if traceID != "ACT-AUTH-BOOTSTRAP" || u.Role != "ADMIN" || u.Status != domain.UserActive {
		return errors.New("invalid bootstrap payload")
	}
	return nil
}

func TestACTAUTHBOOTSTRAPCreatesActiveAdmin(t *testing.T) {
	store := &bootstrapStoreFake{}
	svc := NewBootstrapService(store, domain.NewPasswordHasher())
	if err := svc.Execute(context.Background(), "admin@example.test", "Admin", "Kuat-sekali-2026!"); err != nil {
		t.Fatal(err)
	}
	if !store.called {
		t.Fatal("store was not called")
	}
}
