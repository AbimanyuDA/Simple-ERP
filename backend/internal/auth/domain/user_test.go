package domain

import (
	"testing"
	"time"
)

func TestAUTHUSR004UserLifecycle(t *testing.T) {
	u, err := InviteUser("u-1", "operator@example.test", "OPERATOR", "admin-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if u.Status != UserInvited {
		t.Fatalf("status=%s", u.Status)
	}
	if err := u.Activate("hash", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := u.Deactivate(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := u.Activate("hash", time.Now()); err == nil {
		t.Fatal("deactivated user must not reactivate with an old invitation")
	}
}

func TestRoleValidation(t *testing.T) {
	if _, err := InviteUser("u", "u@example.test", "OWNER", "admin", time.Now()); err == nil {
		t.Fatal("unsupported role accepted")
	}
}
