package application

import (
	"context"
	"github.com/simple-erp/backend/internal/auth/domain"
	"testing"
)

type rotationFake struct{ saved domain.Session }

func (f *rotationFake) ResolveRefresh(context.Context, string) (string, string, error) {
	return "user", "family", nil
}
func (f *rotationFake) Rotate(_ context.Context, _ string, n domain.Session, _ string) error {
	f.saved = n
	return nil
}
func (f *rotationFake) Logout(context.Context, string, string) error { return nil }
func TestCAPAUTH001RefreshRotatesBothTokens(t *testing.T) {
	f := &rotationFake{}
	s := NewSessionService(f, []byte("test-only-pepper-with-at-least-32-bytes"))
	v, e := s.Refresh(context.Background(), "old", "req")
	if e != nil || v.Access == "" || v.Refresh == "" || f.saved.FamilyID != "family" {
		t.Fatalf("tokens=%+v err=%v", v, e)
	}
}
