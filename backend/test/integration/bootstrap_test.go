//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	authpostgres "github.com/simple-erp/backend/internal/auth/adapter/postgres"
	"github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/auth/domain"
)

func TestBootstrapConcurrentAttemptsCreateExactlyOneAdmin(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	if !strings.Contains(url, "localhost") && !strings.Contains(url, "127.0.0.1") {
		t.Fatal("integration test refuses non-local database")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(context.Background(), "TRUNCATE users CASCADE"); err != nil {
		t.Fatal(err)
	}
	svc := application.NewBootstrapService(authpostgres.BootstrapStore{Pool: pool}, domain.NewPasswordHasher())
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- svc.Execute(context.Background(), "admin@example.test", "Admin", "Kuat-sekali-2026!")
		}()
	}
	wg.Wait()
	close(results)
	var successes int
	for err := range results {
		if err == nil {
			successes++
		} else {
			t.Logf("bootstrap result: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successes=%d, want 1", successes)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("users=%d err=%v", count, err)
	}
}
