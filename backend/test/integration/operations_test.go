//go:build integration

package integration

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
	"github.com/simple-erp/backend/internal/platform/idempotency"
	purchasepg "github.com/simple-erp/backend/internal/purchasing/adapter/postgres"
	purchaseapp "github.com/simple-erp/backend/internal/purchasing/application"
	salespg "github.com/simple-erp/backend/internal/sales/adapter/postgres"
	salesapp "github.com/simple-erp/backend/internal/sales/application"
	"os"
	"strings"
	"sync"
	"testing"
)

const (
	adminID     = "00000000-0000-0000-0000-000000000001"
	operatorID  = "00000000-0000-0000-0000-000000000002"
	unitID      = "00000000-0000-0000-0000-000000000010"
	productID   = "00000000-0000-0000-0000-000000000011"
	supplierID  = "00000000-0000-0000-0000-000000000012"
	customerID  = "00000000-0000-0000-0000-000000000013"
	warehouseID = "00000000-0000-0000-0000-000000000014"
)

func operationsPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	u := os.Getenv("TEST_DATABASE_URL")
	if u == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	if !strings.Contains(u, "localhost") && !strings.Contains(u, "127.0.0.1") {
		t.Fatal("refusing non-local database")
	}
	p, e := pgxpool.New(context.Background(), u)
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.Exec(context.Background(), `TRUNCATE stock_movements,stock_balances,delivery_lines,deliveries,sales_order_lines,sales_orders,goods_receipt_lines,goods_receipts,purchase_order_lines,purchase_orders,purchase_request_lines,purchase_requests,inventory_adjustment_lines,inventory_adjustments,organization_settings,products,suppliers,customers,warehouses,units,action_tokens,sessions,audit_entries,idempotency_keys,users CASCADE`)
	if e != nil {
		t.Fatal(e)
	}
	seeds := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users(id,email,display_name,password_hash,role,status)VALUES($1,'admin@example.test','Admin',$3,'ADMIN','ACTIVE'),($2,'operator@example.test','Operator',$3,'OPERATOR','ACTIVE')`, []any{adminID, operatorID, mustTestHash(t)}},
		{`INSERT INTO units(id,code,name)VALUES($1,'PCS','Pieces')`, []any{unitID}},
		{`INSERT INTO products(id,sku,name,unit_id)VALUES($1,'SKU-1','Product',$2)`, []any{productID, unitID}},
		{`INSERT INTO suppliers(id,code,name)VALUES($1,'SUP-1','Supplier')`, []any{supplierID}},
		{`INSERT INTO customers(id,code,name)VALUES($1,'CUS-1','Customer')`, []any{customerID}},
		{`INSERT INTO warehouses(id,code,name)VALUES($1,'MAIN','Main')`, []any{warehouseID}},
		{`INSERT INTO organization_settings(default_warehouse_id)VALUES($1)`, []any{warehouseID}},
	}
	for _, seed := range seeds {
		if _, e = p.Exec(context.Background(), seed.sql, seed.args...); e != nil {
			t.Fatal(e)
		}
	}
	return p
}
func mustTestHash(t *testing.T) string {
	t.Helper()
	v, e := authdomain.NewPasswordHasher().Hash("Kuat-sekali-2026!")
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestP2PPartialReceiptPostsImmutableSourceLinkedLedger(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	s := purchaseapp.NewService(purchasepg.Store{Pool: p})
	ctx := context.Background()
	pr, e := s.CreateRequest(ctx, operatorID, []purchaseapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 100}}, "r1")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionRequest(ctx, pr.ID, "submit", operatorID, "", pr.Version, "r2"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionRequest(ctx, pr.ID, "approve", operatorID, "", 2, "r3"); e == nil {
		t.Fatal("maker-checker not enforced")
	}
	if _, e = s.TransitionRequest(ctx, pr.ID, "approve", adminID, "", 2, "r4"); e != nil {
		t.Fatal(e)
	}
	po, e := s.ConvertToOrder(ctx, pr.ID, supplierID, warehouseID, operatorID, "r5")
	if e != nil {
		t.Fatal(e)
	}
	po, e = s.IssueOrder(ctx, po.ID, po.Version, operatorID, "r6")
	if e != nil {
		t.Fatal(e)
	}
	var lineID string
	if e = p.QueryRow(ctx, `SELECT id FROM purchase_order_lines WHERE purchase_order_id=$1`, po.ID).Scan(&lineID); e != nil {
		t.Fatal(e)
	}
	gr, e := s.CreateReceipt(ctx, po.ID, []purchaseapp.ReceiptLineInput{{PurchaseOrderLineID: lineID, ProductID: productID, Quantity: 4}}, operatorID, "r7")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PostReceipt(ctx, gr.ID, gr.Version, operatorID, "r8"); e != nil {
		t.Fatal(e)
	}
	var balance, ledger float64
	var source string
	if e = p.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2`, warehouseID, productID).Scan(&balance); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT quantity,source_id FROM stock_movements WHERE source_id=$1`, gr.ID).Scan(&ledger, &source); e != nil {
		t.Fatal(e)
	}
	if balance != 4 || ledger != 4 || source != gr.ID {
		t.Fatalf("balance=%v ledger=%v source=%s", balance, ledger, source)
	}
	if _, e = p.Exec(ctx, `UPDATE stock_movements SET quantity=99 WHERE source_id=$1`, gr.ID); e == nil {
		t.Fatal("ledger mutation accepted")
	}
}
func TestConcurrentDeliveriesCannotOversell(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	_, e := p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID)
	if e != nil {
		t.Fatal(e)
	}
	s := salesapp.NewService(salespg.Store{Pool: p})
	so, e := s.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "s1")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "s2"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionOrder(ctx, so.ID, "approve", adminID, "", 2, "s3"); e != nil {
		t.Fatal(e)
	}
	var lineID string
	if e = p.QueryRow(ctx, `SELECT id FROM sales_order_lines WHERE sales_order_id=$1`, so.ID).Scan(&lineID); e != nil {
		t.Fatal(e)
	}
	d1, e := s.CreateDelivery(ctx, so.ID, []salesapp.DeliveryLineInput{{SalesOrderLineID: lineID, ProductID: productID, Quantity: 6}}, operatorID, "s4")
	if e != nil {
		t.Fatal(e)
	}
	d2, e := s.CreateDelivery(ctx, so.ID, []salesapp.DeliveryLineInput{{SalesOrderLineID: lineID, ProductID: productID, Quantity: 6}}, operatorID, "s5")
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, d := range []salesapp.Document{d1, d2} {
		wg.Add(1)
		go func(v salesapp.Document) {
			defer wg.Done()
			_, e := s.PostDelivery(ctx, v.ID, v.Version, operatorID, "post")
			results <- e
		}(d)
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successes=%d", success)
	}
	var balance float64
	if e = p.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2`, warehouseID, productID).Scan(&balance); e != nil {
		t.Fatal(e)
	}
	if balance != 4 {
		t.Fatalf("balance=%v", balance)
	}
}

func TestOperationalIdempotencyReplaysCompletedResponse(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	store := idempotency.PostgresStore{Pool: p}
	ctx := context.Background()
	record, claimed, err := store.Claim(ctx, operatorID, "operation-one", "request-hash")
	if err != nil || !claimed || record.RequestHash != "request-hash" {
		t.Fatalf("claim record=%#v claimed=%v err=%v", record, claimed, err)
	}
	if err = store.Complete(ctx, operatorID, "operation-one", 201, []byte(`{"id":"document-one"}`)); err != nil {
		t.Fatal(err)
	}
	replayed, claimed, err := store.Claim(ctx, operatorID, "operation-one", "request-hash")
	if err != nil || claimed || replayed.Status != 201 || string(replayed.Body) != `{"id": "document-one"}` {
		t.Fatalf("replay record=%#v claimed=%v err=%v", replayed, claimed, err)
	}
}
