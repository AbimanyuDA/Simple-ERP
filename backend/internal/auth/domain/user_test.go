package domain

import (
	"testing"
	"time"
)

func TestAUTHUSR004UserLifecycle(t *testing.T) {
	u, err := InviteUser("u-1", "operator@example.test", RoleStaff, "admin-1", time.Now())
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
	for _, role := range []string{RoleStaff, RoleSupervisor, RoleSuperAdmin} {
		if _, err := InviteUser("u", "u@example.test", role, "admin", time.Now()); err != nil {
			t.Fatalf("role %s rejected: %v", role, err)
		}
	}
	for _, role := range []string{"ADMIN", "OPERATOR"} {
		if _, err := InviteUser("u", "u@example.test", role, "admin", time.Now()); err == nil {
			t.Fatalf("legacy role %s accepted", role)
		}
	}
	if _, err := InviteUser("u", "u@example.test", "OWNER", "admin", time.Now()); err == nil {
		t.Fatal("unsupported role accepted")
	}
}

func TestRoleCapabilities(t *testing.T) {
	if !CanCreate(RoleStaff) || CanApprove(RoleStaff) || CanManageUsers(RoleStaff) {
		t.Fatal("staff permissions incorrect")
	}
	if CanCreate(RoleSupervisor) || !CanApprove(RoleSupervisor) || CanManageUsers(RoleSupervisor) {
		t.Fatal("supervisor permissions incorrect")
	}
	if !CanCreate(RoleSuperAdmin) || !CanApprove(RoleSuperAdmin) || !CanManageUsers(RoleSuperAdmin) {
		t.Fatal("super admin permissions incorrect")
	}
}
