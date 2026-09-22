package domain

import (
	"testing"
	"time"
)

func TestAUTHPWD001ActionTokenLifecycle(t *testing.T) {
	raw, token, err := NewActionToken("t1", "u1", PasswordReset, "admin", []byte("test-only-pepper-with-at-least-32-bytes"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || token.Hash == raw || token.ExpiresAt.Sub(token.CreatedAt) != 24*time.Hour {
		t.Fatal("unsafe token")
	}
	if err := token.Consume(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := token.Consume(time.Now()); err == nil {
		t.Fatal("token reused")
	}
}

func TestExpiredActionTokenCannotBeConsumed(t *testing.T) {
	_, token, _ := NewActionToken("t1", "u1", Invitation, "admin", []byte("test-only-pepper-with-at-least-32-bytes"), time.Now().Add(-25*time.Hour))
	if err := token.Consume(time.Now()); err == nil {
		t.Fatal("expired token consumed")
	}
}
