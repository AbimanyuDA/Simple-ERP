//go:build integration

package integration

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
	inventorypg "github.com/simple-erp/backend/internal/inventory/adapter/postgres"
	inventoryapp "github.com/simple-erp/backend/internal/inventory/application"
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
	config, e := pgxpool.ParseConfig(u)
	if e != nil {
		t.Fatal(e)
	}
	if (config.ConnConfig.Host != "localhost" && config.ConnConfig.Host != "127.0.0.1") || !strings.Contains(strings.ToLower(config.ConnConfig.Database), "test") {
		t.Fatal("refusing database that is not local and explicitly named as a test database")
	}
	p, e := pgxpool.NewWithConfig(context.Background(), config)
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.Exec(context.Background(), `TRUNCATE customer_payment_allocations,customer_payments,sales_invoice_lines,sales_invoices,supplier_payment_allocations,supplier_payments,supplier_invoice_lines,supplier_invoices,stock_movements,stock_balances,delivery_lines,deliveries,sales_order_lines,sales_orders,goods_receipt_lines,goods_receipts,purchase_order_lines,purchase_orders,purchase_request_lines,purchase_requests,inventory_adjustment_lines,inventory_adjustments,organization_settings,products,suppliers,customers,warehouses,units,action_tokens,sessions,audit_entries,idempotency_keys,users CASCADE`)
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
	pr, e := s.CreateRequest(ctx, operatorID, "Partial receipt test", []purchaseapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 100}}, "r1")
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
	poDetail, e := s.GetOrder(ctx, po.ID)
	if e != nil || len(poDetail.Lines) != 1 || poDetail.Lines[0].ID != lineID || poDetail.Lines[0].RemainingQuantity != 10 {
		t.Fatalf("purchase order detail=%#v err=%v", poDetail, e)
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
	poDetail, e = s.GetOrder(ctx, po.ID)
	if e != nil || poDetail.Lines[0].ProcessedQuantity != 4 || poDetail.Lines[0].RemainingQuantity != 6 {
		t.Fatalf("purchase order detail after receipt=%#v err=%v", poDetail, e)
	}
	if _, e = p.Exec(ctx, `UPDATE stock_movements SET quantity=99 WHERE source_id=$1`, gr.ID); e == nil {
		t.Fatal("ledger mutation accepted")
	}
}

func TestInventoryApprovalRejectsChangedSnapshot(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	s := inventoryapp.NewService(inventorypg.Store{Pool: p})
	adjustment, e := s.CreateAdjustment(ctx, warehouseID, operatorID, []inventoryapp.AdjustmentLine{{ProductID: productID, CountedQuantity: 2}}, "inventory-snapshot-1")
	if e != nil {
		t.Fatal(e)
	}
	if adjustment, e = s.Transition(ctx, adjustment.ID, "submit", operatorID, "", adjustment.Version, "inventory-snapshot-2"); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,1)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Transition(ctx, adjustment.ID, "approve", adminID, "", adjustment.Version, "inventory-snapshot-3"); e == nil {
		t.Fatal("inventory adjustment approved after stock snapshot changed")
	}
	assertState(t, p, "inventory_adjustments", adjustment.ID, "SUBMITTED")
}

func TestCopyRejectedDocumentsRequiresActiveMasters(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	purchasing := purchaseapp.NewService(purchasepg.Store{Pool: p})
	sales := salesapp.NewService(salespg.Store{Pool: p})

	request, e := purchasing.CreateRequest(ctx, operatorID, "Copy validation test", []purchaseapp.LineInput{{ProductID: productID, Quantity: 1, UnitPrice: 100}}, "copy-master-pr-1")
	if e != nil {
		t.Fatal(e)
	}
	if request, e = purchasing.TransitionRequest(ctx, request.ID, "submit", operatorID, "", request.Version, "copy-master-pr-2"); e != nil {
		t.Fatal(e)
	}
	if request, e = purchasing.TransitionRequest(ctx, request.ID, "reject", adminID, "rejected", request.Version, "copy-master-pr-3"); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,1)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	order, e := sales.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 1, UnitPrice: 100}}, "copy-master-so-1")
	if e != nil {
		t.Fatal(e)
	}
	if order, e = sales.TransitionOrder(ctx, order.ID, "submit", operatorID, "", order.Version, "copy-master-so-2"); e != nil {
		t.Fatal(e)
	}
	if order, e = sales.TransitionOrder(ctx, order.ID, "reject", adminID, "rejected", order.Version, "copy-master-so-3"); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `UPDATE products SET active=false WHERE id=$1`, productID); e != nil {
		t.Fatal(e)
	}
	if _, e = purchasing.CopyRequest(ctx, request.ID, operatorID, "copy-master-pr-4"); e == nil {
		t.Fatal("purchase request copied with inactive product")
	}
	if _, e = sales.CopyOrder(ctx, order.ID, operatorID, "copy-master-so-4"); e == nil {
		t.Fatal("sales order copied with inactive product")
	}
}

func TestPostingDraftReceiptCannotReviveCancelledOrder(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	s := purchaseapp.NewService(purchasepg.Store{Pool: p})
	ctx := context.Background()

	pr, e := s.CreateRequest(ctx, operatorID, "Cancelled order test", []purchaseapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 100}}, "cancel-po-1")
	if e != nil {
		t.Fatal(e)
	}
	if pr, e = s.TransitionRequest(ctx, pr.ID, "submit", operatorID, "", pr.Version, "cancel-po-2"); e != nil {
		t.Fatal(e)
	}
	if pr, e = s.TransitionRequest(ctx, pr.ID, "approve", adminID, "", pr.Version, "cancel-po-3"); e != nil {
		t.Fatal(e)
	}
	po, e := s.ConvertToOrder(ctx, pr.ID, supplierID, warehouseID, operatorID, "cancel-po-4")
	if e != nil {
		t.Fatal(e)
	}
	if po, e = s.IssueOrder(ctx, po.ID, po.Version, operatorID, "cancel-po-5"); e != nil {
		t.Fatal(e)
	}
	var lineID string
	if e = p.QueryRow(ctx, `SELECT id FROM purchase_order_lines WHERE purchase_order_id=$1`, po.ID).Scan(&lineID); e != nil {
		t.Fatal(e)
	}
	gr, e := s.CreateReceipt(ctx, po.ID, []purchaseapp.ReceiptLineInput{{PurchaseOrderLineID: lineID, ProductID: productID, Quantity: 4}}, operatorID, "cancel-po-6")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CancelOrder(ctx, po.ID, "cancel before posting", operatorID, "cancel-po-7"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PostReceipt(ctx, gr.ID, gr.Version, operatorID, "cancel-po-8"); e == nil {
		t.Fatal("draft receipt posted after purchase order cancellation")
	}

	var poStatus, grStatus string
	var movements int
	if e = p.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1`, po.ID).Scan(&poStatus); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT status FROM goods_receipts WHERE id=$1`, gr.ID).Scan(&grStatus); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE source_id=$1`, gr.ID).Scan(&movements); e != nil {
		t.Fatal(e)
	}
	if poStatus != "CANCELLED" || grStatus != "DRAFT" || movements != 0 {
		t.Fatalf("po=%s receipt=%s movements=%d", poStatus, grStatus, movements)
	}
}
func TestDraftOrderRevisionUsesOptimisticLockingAndAudit(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	s := salesapp.NewService(salespg.Store{Pool: p})
	so, e := s.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "revise-so-1")
	if e != nil {
		t.Fatal(e)
	}
	revised, e := s.UpdateOrder(ctx, so.ID, so.Version, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 4, UnitPrice: 250}}, "revise-so-2")
	if e != nil {
		t.Fatal(e)
	}
	var quantity, price float64
	var traces int
	if e = p.QueryRow(ctx, `SELECT quantity,unit_price FROM sales_order_lines WHERE sales_order_id=$1 AND product_id=$2`, so.ID, productID).Scan(&quantity, &price); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE entity_id=$1 AND trace_id='SAL-SO-001' AND action='UPDATE_SALES_ORDER'`, so.ID).Scan(&traces); e != nil {
		t.Fatal(e)
	}
	if revised.Status != "DRAFT" || revised.Version != so.Version+1 || quantity != 4 || price != 250 || traces != 1 {
		t.Fatalf("order=%#v quantity=%v price=%v traces=%d", revised, quantity, price, traces)
	}
	if _, e = s.UpdateOrder(ctx, so.ID, so.Version, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 8, UnitPrice: 300}}, "revise-so-stale"); e == nil {
		t.Fatal("stale draft revision accepted")
	}
	if e = p.QueryRow(ctx, `SELECT quantity,unit_price FROM sales_order_lines WHERE sales_order_id=$1 AND product_id=$2`, so.ID, productID).Scan(&quantity, &price); e != nil {
		t.Fatal(e)
	}
	if quantity != 4 || price != 250 {
		t.Fatalf("stale revision changed line quantity=%v price=%v", quantity, price)
	}
	if _, e = p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	if revised, e = s.TransitionOrder(ctx, revised.ID, "submit", operatorID, "", revised.Version, "revise-so-submit"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.UpdateOrder(ctx, revised.ID, revised.Version, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 3, UnitPrice: 250}}, "revise-so-submitted"); e == nil {
		t.Fatal("submitted sales order revision accepted")
	}
}
func TestSubmitOrderRequiresAvailableStock(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	s := salesapp.NewService(salespg.Store{Pool: p})
	so, e := s.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "submit-stock-1")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "submit-stock-insufficient"); e == nil {
		t.Fatal("sales order submitted without sufficient stock")
	}
	var status string
	var version int
	var validationTraces int
	if e = p.QueryRow(ctx, `SELECT status,version FROM sales_orders WHERE id=$1`, so.ID).Scan(&status, &version); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE entity_id=$1 AND trace_id='SAL-SO-003'`, so.ID).Scan(&validationTraces); e != nil {
		t.Fatal(e)
	}
	if status != "DRAFT" || version != so.Version || validationTraces != 0 {
		t.Fatalf("status=%s version=%d validation_traces=%d", status, version, validationTraces)
	}
	if _, e = p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "submit-stock-sufficient"); e != nil {
		t.Fatal(e)
	}
	var submitTraces int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE entity_id=$1 AND trace_id='SAL-SO-003'`, so.ID).Scan(&validationTraces); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE entity_id=$1 AND trace_id='SAL-SO-002'`, so.ID).Scan(&submitTraces); e != nil {
		t.Fatal(e)
	}
	if so.Status != "SUBMITTED" || validationTraces != 1 || submitTraces != 1 {
		t.Fatalf("order=%#v validation_traces=%d submit_traces=%d", so, validationTraces, submitTraces)
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
	if so, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "s2"); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "approve", adminID, "", so.Version, "s3"); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "confirm", operatorID, "", so.Version, "s-confirm"); e != nil {
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
	soDetail, e := s.GetOrder(ctx, so.ID)
	if e != nil || !soDetail.Confirmed || len(soDetail.Lines) != 1 || soDetail.Lines[0].ProcessedQuantity != 6 || soDetail.Lines[0].RemainingQuantity != 4 {
		t.Fatalf("sales order detail=%#v err=%v", soDetail, e)
	}
}

func TestPostingDraftDeliveryCannotReviveCancelledOrder(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	if _, e := p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	s := salesapp.NewService(salespg.Store{Pool: p})
	so, e := s.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "cancel-so-1")
	if e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "cancel-so-2"); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "approve", adminID, "", so.Version, "cancel-so-3"); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "confirm", operatorID, "", so.Version, "cancel-so-confirm"); e != nil {
		t.Fatal(e)
	}
	var lineID string
	if e = p.QueryRow(ctx, `SELECT id FROM sales_order_lines WHERE sales_order_id=$1`, so.ID).Scan(&lineID); e != nil {
		t.Fatal(e)
	}
	delivery, e := s.CreateDelivery(ctx, so.ID, []salesapp.DeliveryLineInput{{SalesOrderLineID: lineID, ProductID: productID, Quantity: 4}}, operatorID, "cancel-so-4")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.TransitionOrder(ctx, so.ID, "cancel", operatorID, "cancel before posting", so.Version, "cancel-so-5"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PostDelivery(ctx, delivery.ID, delivery.Version, operatorID, "cancel-so-6"); e == nil {
		t.Fatal("draft delivery posted after sales order cancellation")
	}

	var soStatus, deliveryStatus string
	var balance float64
	var movements int
	if e = p.QueryRow(ctx, `SELECT status FROM sales_orders WHERE id=$1`, so.ID).Scan(&soStatus); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT status FROM deliveries WHERE id=$1`, delivery.ID).Scan(&deliveryStatus); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2`, warehouseID, productID).Scan(&balance); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE source_id=$1`, delivery.ID).Scan(&movements); e != nil {
		t.Fatal(e)
	}
	if soStatus != "CANCELLED" || deliveryStatus != "DRAFT" || balance != 10 || movements != 0 {
		t.Fatalf("so=%s delivery=%s balance=%v movements=%d", soStatus, deliveryStatus, balance, movements)
	}
}

func TestDeliveryRequiresOrderConfirmation(t *testing.T) {
	p := operationsPool(t)
	defer p.Close()
	ctx := context.Background()
	if _, e := p.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID); e != nil {
		t.Fatal(e)
	}
	s := salesapp.NewService(salespg.Store{Pool: p})
	so, e := s.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "confirm-so-1")
	if e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "submit", operatorID, "", so.Version, "confirm-so-2"); e != nil {
		t.Fatal(e)
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "approve", adminID, "", so.Version, "confirm-so-3"); e != nil {
		t.Fatal(e)
	}
	var lineID string
	if e = p.QueryRow(ctx, `SELECT id FROM sales_order_lines WHERE sales_order_id=$1`, so.ID).Scan(&lineID); e != nil {
		t.Fatal(e)
	}
	lines := []salesapp.DeliveryLineInput{{SalesOrderLineID: lineID, ProductID: productID, Quantity: 4}}
	if _, e = s.CreateDelivery(ctx, so.ID, lines, operatorID, "confirm-so-before"); e == nil {
		t.Fatal("delivery created before sales order confirmation")
	}
	if so, e = s.TransitionOrder(ctx, so.ID, "confirm", operatorID, "", so.Version, "confirm-so-4"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateDelivery(ctx, so.ID, lines, operatorID, "confirm-so-after"); e != nil {
		t.Fatal(e)
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
