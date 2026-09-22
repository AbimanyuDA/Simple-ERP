//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationOrderingContract(t *testing.T) {
	root := filepath.Join("..", "..", "migrations")
	identity := read(t, filepath.Join(root, "000001_identity.up.sql"))
	master := read(t, filepath.Join(root, "000002_masterdata.up.sql"))
	settings := read(t, filepath.Join(root, "000003_settings.up.sql"))
	for _, required := range []string{"users", "sessions", "action_tokens", "audit_entries", "idempotency_keys"} {
		if !strings.Contains(identity, required) {
			t.Errorf("identity migration missing %s", required)
		}
	}
	if !strings.Contains(master, "warehouses") {
		t.Error("master-data migration must create warehouses")
	}
	if !strings.Contains(settings, "organization_settings") || !strings.Contains(settings, "warehouses") {
		t.Error("settings must be created after and reference warehouses")
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
