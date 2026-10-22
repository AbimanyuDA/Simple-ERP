//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	financepg "github.com/simple-erp/backend/internal/finance/adapter/postgres"
	financeapp "github.com/simple-erp/backend/internal/finance/application"
	purchasepg "github.com/simple-erp/backend/internal/purchasing/adapter/postgres"
	purchaseapp "github.com/simple-erp/backend/internal/purchasing/application"
	reportingpg "github.com/simple-erp/backend/internal/reporting/adapter/postgres"
	reportingapp "github.com/simple-erp/backend/internal/reporting/application"
	salespg "github.com/simple-erp/backend/internal/sales/adapter/postgres"
	salesapp "github.com/simple-erp/backend/internal/sales/application"
)

func TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid(t *testing.T) {
	pool := operationsPool(t)
	defer pool.Close()
	ctx := context.Background()
	purchasing := purchaseapp.NewService(purchasepg.Store{Pool: pool})
	finance := financeapp.NewService(financepg.Store{Pool: pool})
	reporting := reportingapp.NewService(reportingpg.Store{Pool: pool})

	request, err := purchasing.CreateRequest(ctx, operatorID, "Restock product", []purchaseapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 100}}, "fin-p2p-1")
	if err != nil {
		t.Fatal(err)
	}
	request, err = purchasing.TransitionRequest(ctx, request.ID, "submit", operatorID, "", request.Version, "fin-p2p-2")
	if err != nil {
		t.Fatal(err)
	}
	request, err = purchasing.TransitionRequest(ctx, request.ID, "approve", adminID, "", request.Version, "fin-p2p-3")
	if err != nil {
		t.Fatal(err)
	}
	order, err := purchasing.ConvertToOrder(ctx, request.ID, supplierID, warehouseID, operatorID, "fin-p2p-4")
	if err != nil {
		t.Fatal(err)
	}
	order, err = purchasing.IssueOrder(ctx, order.ID, order.Version, supervisorID, "fin-p2p-5")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.CreateSupplierInvoice(ctx, supplierInvoice(order.ID, "EARLY", 10, 100, ""), adminID, "fin-p2p-early"); err == nil {
		t.Fatal("early invoice accepted")
	}
	var orderLineID string
	if err = pool.QueryRow(ctx, `SELECT id FROM purchase_order_lines WHERE purchase_order_id=$1`, order.ID).Scan(&orderLineID); err != nil {
		t.Fatal(err)
	}
	receipt, err := purchasing.CreateReceipt(ctx, order.ID, []purchaseapp.ReceiptLineInput{{PurchaseOrderLineID: orderLineID, ProductID: productID, Quantity: 10}}, operatorID, "fin-p2p-6")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = purchasing.PostReceipt(ctx, receipt.ID, receipt.Version, supervisorID, "fin-p2p-7"); err != nil {
		t.Fatal(err)
	}

	mismatchInput := supplierInvoice(order.ID, "INV-1", 9, 100, "")
	mismatchInput.Lines[0].ReferenceID = orderLineID
	invoice, err := finance.CreateSupplierInvoice(ctx, mismatchInput, operatorID, "fin-p2p-8")
	if err != nil {
		t.Fatal(err)
	}
	invoice, err = finance.MatchSupplierInvoice(ctx, invoice.ID, supervisorID, "fin-p2p-9")
	if err != nil || invoice.Status != "MATCH_EXCEPTION" {
		t.Fatalf("invoice=%#v err=%v", invoice, err)
	}
	summary, err := reporting.Summary(ctx)
	if err != nil || summary.Payables != 0 {
		t.Fatalf("match-exception invoice leaked into payables: summary=%#v err=%v", summary, err)
	}
	invoice, err = finance.ResolveSupplierInvoice(ctx, invoice.ID, []financeapp.InvoiceLineInput{{ReferenceID: orderLineID, Quantity: 10, UnitPrice: 100}}, operatorID, "fin-p2p-10")
	if err != nil {
		t.Fatal(err)
	}
	invoice, err = finance.MatchSupplierInvoice(ctx, invoice.ID, supervisorID, "fin-p2p-11")
	if err != nil || invoice.Status != "POSTED" {
		t.Fatalf("invoice=%#v err=%v", invoice, err)
	}
	summary, err = reporting.Summary(ctx)
	if err != nil || summary.Payables != 1000 {
		t.Fatalf("posted invoice missing from payables: summary=%#v err=%v", summary, err)
	}
	if _, err = purchasing.ReverseReceipt(ctx, receipt.ID, "should be blocked", adminID, "fin-p2p-guard"); err == nil {
		t.Fatal("receipt reversal accepted with active invoice")
	}

	payment1, err := finance.CreateSupplierPayment(ctx, payment(supplierID, invoice.ID, 400), operatorID, "fin-p2p-12")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.PostSupplierPayment(ctx, payment1.ID, payment1.Version, supervisorID, "fin-p2p-13"); err != nil {
		t.Fatal(err)
	}
	assertInvoice(t, pool, "supplier_invoices", invoice.ID, "PARTIALLY_PAID", 600)
	payment2, err := finance.CreateSupplierPayment(ctx, payment(supplierID, invoice.ID, 600), operatorID, "fin-p2p-14")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.PostSupplierPayment(ctx, payment2.ID, payment2.Version, supervisorID, "fin-p2p-15"); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, "purchase_orders", order.ID, "CLOSED")
	assertState(t, pool, "purchase_requests", request.ID, "COMPLETED")
	if _, err = finance.VoidSupplierPayment(ctx, payment2.ID, "incorrect transfer", adminID, "fin-p2p-16"); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, "purchase_orders", order.ID, "RECEIVED")
	assertState(t, pool, "purchase_requests", request.ID, "CONVERTED_TO_PO")
	assertInvoice(t, pool, "supplier_invoices", invoice.ID, "PARTIALLY_PAID", 600)
	if _, err = finance.VoidSupplierInvoice(ctx, invoice.ID, "blocked by payment", adminID, "fin-p2p-17"); err == nil {
		t.Fatal("supplier invoice void accepted with active allocation")
	}
	if _, err = finance.VoidSupplierPayment(ctx, payment1.ID, "incorrect transfer", adminID, "fin-p2p-18"); err != nil {
		t.Fatal(err)
	}
	if _, err = finance.VoidSupplierInvoice(ctx, invoice.ID, "replace incorrect invoice", adminID, "fin-p2p-19"); err != nil {
		t.Fatal(err)
	}
	missingLink := supplierInvoice(order.ID, "INV-MISSING-LINK", 10, 100, "")
	missingLink.Lines[0].ReferenceID = orderLineID
	if _, err = finance.CreateSupplierInvoice(ctx, missingLink, adminID, "fin-p2p-missing-link"); err == nil {
		t.Fatal("supplier replacement accepted without replacesInvoiceId")
	}
	replacementInput := supplierInvoice(order.ID, "INV-2", 10, 100, invoice.ID)
	replacementInput.Lines[0].ReferenceID = orderLineID
	replacement, err := finance.CreateSupplierInvoice(ctx, replacementInput, operatorID, "fin-p2p-20")
	if err != nil {
		t.Fatal(err)
	}
	if replacement, err = finance.MatchSupplierInvoice(ctx, replacement.ID, supervisorID, "fin-p2p-21"); err != nil || replacement.Status != "POSTED" {
		t.Fatalf("replacement=%#v err=%v", replacement, err)
	}
}

func TestO2CConsolidatedInvoicePartialPaymentCompletionAndVoid(t *testing.T) {
	pool := operationsPool(t)
	defer pool.Close()
	ctx := context.Background()
	sales := salesapp.NewService(salespg.Store{Pool: pool})
	finance := financeapp.NewService(financepg.Store{Pool: pool})
	if _, err := pool.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,10)`, warehouseID, productID); err != nil {
		t.Fatal(err)
	}
	order, err := sales.CreateOrder(ctx, customerID, warehouseID, operatorID, []salesapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 200}}, "fin-o2c-1")
	if err != nil {
		t.Fatal(err)
	}
	order, err = sales.TransitionOrder(ctx, order.ID, "submit", operatorID, "", order.Version, "fin-o2c-2")
	if err != nil {
		t.Fatal(err)
	}
	order, err = sales.TransitionOrder(ctx, order.ID, "approve", adminID, "", order.Version, "fin-o2c-3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.CreateSalesInvoice(ctx, salesInvoice(order.ID, ""), adminID, "fin-o2c-early"); err == nil {
		t.Fatal("early sales invoice accepted")
	}
	order, err = sales.TransitionOrder(ctx, order.ID, "confirm", operatorID, "", order.Version, "fin-o2c-confirm")
	if err != nil {
		t.Fatal(err)
	}
	var orderLineID string
	if err = pool.QueryRow(ctx, `SELECT id FROM sales_order_lines WHERE sales_order_id=$1`, order.ID).Scan(&orderLineID); err != nil {
		t.Fatal(err)
	}
	delivery, err := sales.CreateDelivery(ctx, order.ID, []salesapp.DeliveryLineInput{{SalesOrderLineID: orderLineID, ProductID: productID, Quantity: 10}}, operatorID, "fin-o2c-4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sales.PostDelivery(ctx, delivery.ID, delivery.Version, adminID, "fin-o2c-5"); err != nil {
		t.Fatal(err)
	}
	invoice, err := finance.CreateSalesInvoice(ctx, salesInvoice(order.ID, ""), operatorID, "fin-o2c-6")
	if err != nil {
		t.Fatal(err)
	}
	if invoice.Status != "DRAFT" {
		t.Fatalf("new sales invoice must be draft, got %s", invoice.Status)
	}
	if _, err = finance.PostSalesInvoice(ctx, invoice.ID, invoice.Version, operatorID, "fin-o2c-self-post"); err == nil {
		t.Fatal("staff self-posted sales invoice")
	}
	if _, err = finance.CreateCustomerPayment(ctx, payment(customerID, invoice.ID, 500), operatorID, "fin-o2c-before-post"); err == nil {
		t.Fatal("payment draft accepted against unposted invoice")
	}
	invoice, err = finance.PostSalesInvoice(ctx, invoice.ID, invoice.Version, adminID, "fin-o2c-post-invoice")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sales.ReverseDelivery(ctx, delivery.ID, "should be blocked", adminID, "fin-o2c-guard"); err == nil {
		t.Fatal("delivery reversal accepted with active invoice")
	}
	payment1, err := finance.CreateCustomerPayment(ctx, payment(customerID, invoice.ID, 500), operatorID, "fin-o2c-7")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.PostCustomerPayment(ctx, payment1.ID, payment1.Version, operatorID, "fin-o2c-self-payment"); err == nil {
		t.Fatal("staff self-posted customer payment")
	}
	if _, err = finance.PostCustomerPayment(ctx, payment1.ID, payment1.Version, adminID, "fin-o2c-8"); err != nil {
		t.Fatal(err)
	}
	assertInvoice(t, pool, "sales_invoices", invoice.ID, "PARTIALLY_PAID", 1500)
	payment2, err := finance.CreateCustomerPayment(ctx, payment(customerID, invoice.ID, 1500), operatorID, "fin-o2c-9")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = finance.PostCustomerPayment(ctx, payment2.ID, payment2.Version, adminID, "fin-o2c-10"); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, "sales_orders", order.ID, "COMPLETED")
	if _, err = finance.VoidCustomerPayment(ctx, payment2.ID, "duplicate receipt", adminID, "fin-o2c-11"); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, "sales_orders", order.ID, "DELIVERED")
	assertInvoice(t, pool, "sales_invoices", invoice.ID, "PARTIALLY_PAID", 1500)
	if _, err = finance.VoidSalesInvoice(ctx, invoice.ID, "blocked by payment", adminID, "fin-o2c-12"); err == nil {
		t.Fatal("sales invoice void accepted with active allocation")
	}
	if _, err = finance.VoidCustomerPayment(ctx, payment1.ID, "duplicate receipt", adminID, "fin-o2c-13"); err != nil {
		t.Fatal(err)
	}
	if _, err = finance.VoidSalesInvoice(ctx, invoice.ID, "replace invoice", adminID, "fin-o2c-14"); err != nil {
		t.Fatal(err)
	}
	if _, err = finance.CreateSalesInvoice(ctx, salesInvoice(order.ID, ""), operatorID, "fin-o2c-missing-link"); err == nil {
		t.Fatal("sales replacement accepted without replacesInvoiceId")
	}
	if _, err = finance.CreateSalesInvoice(ctx, salesInvoice(order.ID, invoice.ID), operatorID, "fin-o2c-15"); err != nil {
		t.Fatal(err)
	}
}

func supplierInvoice(orderID, number string, quantity, price float64, replaces string) financeapp.SupplierInvoiceInput {
	return financeapp.SupplierInvoiceInput{PurchaseOrderID: orderID, SupplierInvoiceNumber: number, InvoiceDate: "2026-09-22", DueDate: "2026-10-22", Currency: "IDR", ReplacesInvoiceID: replaces, Lines: []financeapp.InvoiceLineInput{{Quantity: quantity, UnitPrice: price}}}
}
func salesInvoice(orderID, replaces string) financeapp.SalesInvoiceInput {
	return financeapp.SalesInvoiceInput{SalesOrderID: orderID, InvoiceDate: "2026-09-22", DueDate: "2026-10-22", Currency: "IDR", ReplacesInvoiceID: replaces}
}
func payment(partyID, invoiceID string, amount float64) financeapp.PaymentInput {
	return financeapp.PaymentInput{PartyID: partyID, PaymentDate: "2026-09-22", Method: "BANK_TRANSFER", Currency: "IDR", Amount: amount, Allocations: []financeapp.AllocationInput{{InvoiceID: invoiceID, Amount: amount}}}
}

func assertState(t *testing.T, pool *pgxpool.Pool, table, id, expected string) {
	t.Helper()
	query := map[string]string{
		"purchase_orders":       `SELECT status FROM purchase_orders WHERE id=$1`,
		"purchase_requests":     `SELECT status FROM purchase_requests WHERE id=$1`,
		"supplier_invoices":     `SELECT status FROM supplier_invoices WHERE id=$1`,
		"sales_orders":          `SELECT status FROM sales_orders WHERE id=$1`,
		"inventory_adjustments": `SELECT status FROM inventory_adjustments WHERE id=$1`,
	}[table]
	var actual string
	if err := pool.QueryRow(context.Background(), query, id).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("%s state=%s expected=%s", table, actual, expected)
	}
}

func assertInvoice(t *testing.T, pool *pgxpool.Pool, table, id, expectedStatus string, expectedOutstanding float64) {
	t.Helper()
	query := map[string]string{
		"supplier_invoices": `SELECT status,outstanding FROM supplier_invoices WHERE id=$1`,
		"sales_invoices":    `SELECT status,outstanding FROM sales_invoices WHERE id=$1`,
	}[table]
	var status string
	var outstanding float64
	if err := pool.QueryRow(context.Background(), query, id).Scan(&status, &outstanding); err != nil {
		t.Fatal(err)
	}
	if status != expectedStatus || outstanding != expectedOutstanding {
		t.Fatalf("%s status=%s outstanding=%v", table, status, outstanding)
	}
}
