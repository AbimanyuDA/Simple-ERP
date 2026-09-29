//go:build integration

package integration

import (
	"context"
	"testing"

	purchasepg "github.com/simple-erp/backend/internal/purchasing/adapter/postgres"
	purchaseapp "github.com/simple-erp/backend/internal/purchasing/application"
	reportingpg "github.com/simple-erp/backend/internal/reporting/adapter/postgres"
	reportingapp "github.com/simple-erp/backend/internal/reporting/application"
)

func TestCAPRPT001DashboardAndCAPAUD001AuditAreReconciledAndRedacted(t *testing.T) {
	pool := operationsPool(t)
	defer pool.Close()
	ctx := context.Background()
	purchasing := purchaseapp.NewService(purchasepg.Store{Pool: pool})
	reporting := reportingapp.NewService(reportingpg.Store{Pool: pool})
	request, err := purchasing.CreateRequest(ctx, operatorID, "Reporting fixture", []purchaseapp.LineInput{{ProductID: productID, Quantity: 2, UnitPrice: 100}}, "report-request")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = purchasing.TransitionRequest(ctx, request.ID, "submit", operatorID, "", request.Version, "report-submit"); err != nil {
		t.Fatal(err)
	}
	summary, err := reporting.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PendingApprovals != 1 || summary.ZeroStockItems != 1 {
		t.Fatalf("summary=%#v", summary)
	}
	audit, err := reporting.Audit(ctx, "PURCHASE_REQUEST", operatorID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 2 || audit[0].TraceID != "PUR-PR-002" || audit[0].RequestID != "report-submit" {
		t.Fatalf("audit=%#v", audit)
	}
	if audit[0].Actor != "operator@example.test" {
		t.Fatalf("actor=%s", audit[0].Actor)
	}
	if _, err = pool.Exec(ctx, `UPDATE audit_entries SET action='TAMPERED' WHERE id=$1`, audit[0].ID); err == nil {
		t.Fatal("audit mutation accepted")
	}
}
