//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	authpg "github.com/simple-erp/backend/internal/auth/adapter/postgres"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
	masterpg "github.com/simple-erp/backend/internal/masterdata/adapter/postgres"
	masterdomain "github.com/simple-erp/backend/internal/masterdata/domain"
)

func TestRefreshPreservesFamilyAbsoluteDeadline(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	pepper := []byte("integration-test-session-pepper")
	now := time.Now().UTC().Truncate(time.Microsecond)
	plain, old, err := authdomain.NewSessionTokens(adminID, uuid.NewString(), pepper, now)
	if err != nil {
		t.Fatal(err)
	}
	old.RefreshAbsoluteExpiresAt = now.Add(30 * time.Minute)
	if _, err = p.Exec(ctx, `INSERT INTO sessions(user_id,family_id,access_hash,refresh_hash,csrf_hash,access_expires_at,refresh_idle_expires_at,refresh_absolute_expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, old.UserID, old.FamilyID, old.AccessHash, old.RefreshHash, old.CSRFHash, old.AccessExpiresAt, old.RefreshIdleExpiresAt, old.RefreshAbsoluteExpiresAt); err != nil {
		t.Fatal(err)
	}
	service := authapp.NewSessionService(authpg.SessionStore{Pool: p}, pepper)
	rotated, err := service.Refresh(ctx, plain.Refresh, "refresh-deadline-test")
	if err != nil {
		t.Fatal(err)
	}
	var access, idle, absolute time.Time
	if err = p.QueryRow(ctx, `SELECT access_expires_at,refresh_idle_expires_at,refresh_absolute_expires_at FROM sessions WHERE refresh_hash=$1`, authdomain.TokenDigest(rotated.Refresh, pepper)).Scan(&access, &idle, &absolute); err != nil {
		t.Fatal(err)
	}
	if !absolute.Equal(old.RefreshAbsoluteExpiresAt) || idle.After(absolute) || access.After(absolute) {
		t.Fatalf("refresh extended family deadline: access=%s idle=%s absolute=%s want=%s", access, idle, absolute, old.RefreshAbsoluteExpiresAt)
	}
}

func TestCreateProductRequiresActivePCS(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	if _, err := p.Exec(ctx, `UPDATE units SET active=false WHERE code='PCS'`); err != nil {
		t.Fatal(err)
	}
	record, err := masterdomain.NewRecord(masterdomain.Product, "SKU-NO-UNIT", "No unit")
	if err != nil {
		t.Fatal(err)
	}
	record.ID = uuid.NewString()
	if err = (masterpg.Store{Pool: p}).Create(ctx, record, adminID, "missing-pcs-test"); err == nil {
		t.Fatal("product creation succeeded without an active PCS unit")
	}
	var products, audits int
	if err = p.QueryRow(ctx, `SELECT count(*) FROM products WHERE id=$1`, record.ID).Scan(&products); err != nil {
		t.Fatal(err)
	}
	if err = p.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE entity_id=$1`, record.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if products != 0 || audits != 0 {
		t.Fatalf("failed create left product=%d audit=%d", products, audits)
	}
}

func TestInvitationCannotBeAdminActivated(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	id := uuid.NewString()
	if _, err := p.Exec(ctx, `INSERT INTO users(id,email,display_name,role,status) VALUES($1,'invited-readiness@example.test','Invited','STAFF','INVITED')`, id); err != nil {
		t.Fatal(err)
	}
	store := authpg.UserAdminStore{Pool: p}
	if err := store.SetUserStatus(ctx, id, "ACTIVE", 1, adminID, "invite-bypass-test"); err == nil {
		t.Fatal("admin activated invited account without invitation token")
	}
	var status string
	if err := p.QueryRow(ctx, `SELECT status FROM users WHERE id=$1`, id).Scan(&status); err != nil || status != "INVITED" {
		t.Fatalf("invited status changed: status=%s error=%v", status, err)
	}
}

func TestInvitationActivationAndLogin(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	pepper := []byte("integration-test-session-pepper")
	service := authapp.NewLifecycleService(authpg.LifecycleStore{Pool: p}, authdomain.NewPasswordHasher(), pepper)
	raw, _, err := service.Invite(ctx, "invited-activation@example.test", "Invited", "SUPERVISOR", adminID, "invite-activation-test")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Activate(ctx, raw, "Integration@1234", "invite-activation-test"); err != nil {
		t.Fatal(err)
	}
	var status, role string
	if err = p.QueryRow(ctx, `SELECT status,role FROM users WHERE email='invited-activation@example.test'`).Scan(&status, &role); err != nil {
		t.Fatal(err)
	}
	if status != "ACTIVE" || role != "SUPERVISOR" {
		t.Fatalf("activated account has status=%s role=%s", status, role)
	}
	login := authapp.NewLoginService(authpg.LoginStore{Pool: p}, authdomain.NewPasswordHasher(), pepper)
	if _, err = login.Login(ctx, "invited-activation@example.test", "Integration@1234", "invite-login-test"); err != nil {
		t.Fatal(err)
	}
}
