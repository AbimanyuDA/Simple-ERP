package domain

import (
	"testing"
	"time"
)

func TestCAPAUTH001SessionTokensAreOpaqueAndOnlyHashesPersist(t *testing.T) {
	now := time.Now()
	tokens, stored, err := NewSessionTokens("user-1", "family-1", []byte("test-only-pepper-with-at-least-32-bytes"), now)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.Access == "" || tokens.Refresh == "" || stored.AccessHash == tokens.Access || stored.RefreshHash == tokens.Refresh {
		t.Fatal("tokens must be opaque and stored only as hashes")
	}
	if !VerifyToken(tokens.Refresh, stored.RefreshHash, []byte("test-only-pepper-with-at-least-32-bytes")) {
		t.Fatal("refresh token does not verify")
	}
	if stored.AccessExpiresAt.Sub(now) != 15*time.Minute || stored.RefreshIdleExpiresAt.Sub(now) != 7*24*time.Hour {
		t.Fatal("approved TTLs not applied")
	}
}
