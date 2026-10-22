package application

import (
	"context"
	"github.com/simple-erp/backend/internal/auth/domain"
	"testing"
)

type lifecycleFake struct {
	user  domain.User
	token domain.ActionToken
}

func (f *lifecycleFake) Invite(_ context.Context, u domain.User, t domain.ActionToken, _ string) error {
	f.user = u
	f.token = t
	return nil
}
func (f *lifecycleFake) Consume(_ context.Context, _ string, _ domain.ActionTokenKind, _ string, _ string) error {
	return nil
}
func (f *lifecycleFake) CreateReset(context.Context, string, domain.ActionToken, string, string) error {
	return nil
}
func (f *lifecycleFake) MarkTokenShared(context.Context, string, string, string) error { return nil }
func (f *lifecycleFake) RevokeToken(context.Context, string, string, string) error     { return nil }
func TestAUTHUSR001InviteReturnsRawTokenOnce(t *testing.T) {
	f := &lifecycleFake{}
	s := NewLifecycleService(f, domain.NewPasswordHasher(), []byte("test-only-pepper-with-at-least-32-bytes"))
	raw, _, err := s.Invite(context.Background(), "operator@example.test", "Operator", domain.RoleStaff, "admin", "req")
	if err != nil || raw == "" || f.token.Hash == raw || f.user.Status != domain.UserInvited {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
}
