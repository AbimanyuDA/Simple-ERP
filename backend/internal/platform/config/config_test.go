package config

import "testing"

func TestLoadRejectsMissingRequiredValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_PEPPER", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestLoadUsesApprovedDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://erp:erp@localhost:5432/erp?sslmode=disable")
	t.Setenv("SESSION_PEPPER", "test-only-pepper-with-at-least-32-bytes")
	t.Setenv("APP_ENV", "test")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Currency != "IDR" || cfg.Timezone != "Asia/Jakarta" {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}
