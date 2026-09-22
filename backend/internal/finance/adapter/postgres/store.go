package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/finance/application"
	"github.com/simple-erp/backend/internal/finance/domain"
)

var ErrConflict = errors.New("invalid finance state, match, allocation, or version")

type Store struct{ Pool *pgxpool.Pool }

func documentNo(ctx context.Context, tx pgx.Tx, prefix string) (string, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT nextval('document_number_sequence')`).Scan(&n)
	return fmt.Sprintf("%s-%s-%06d", prefix, time.Now().Format("20060102"), n), err
}

func audit(ctx context.Context, tx pgx.Tx, actor, action, trace, entity, id, from, to, reason, request string) error {
	_, err := tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,reason,request_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)`, actor, action, trace, entity, id, from, to, reason, request)
	return err
}

func scanDocuments(rows pgx.Rows) ([]application.Document, error) {
	defer rows.Close()
	documents := []application.Document{}
	for rows.Next() {
		var document application.Document
		if err := rows.Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func (s Store) ListSupplierInvoices(ctx context.Context) ([]application.Document, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,document_no,status,supplier_id,purchase_order_id,currency,total,outstanding,version FROM supplier_invoices ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return scanDocuments(rows)
}

func (s Store) CreateSupplierInvoice(ctx context.Context, input application.SupplierInvoiceInput, actor, request string) (application.Document, error) {
	if len(input.Lines) == 0 || input.SupplierInvoiceNumber == "" || input.InvoiceDate == "" || input.DueDate == "" || input.Currency == "" {
		return application.Document{}, ErrConflict
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var supplierID, status string
	if err = tx.QueryRow(ctx, `SELECT supplier_id,status FROM purchase_orders WHERE id=$1 FOR UPDATE`, input.PurchaseOrderID).Scan(&supplierID, &status); err != nil || status != "RECEIVED" {
		return application.Document{}, ErrConflict
	}
	if input.ReplacesInvoiceID != "" {
		var source, oldStatus string
		if err = tx.QueryRow(ctx, `SELECT purchase_order_id,status FROM supplier_invoices WHERE id=$1`, input.ReplacesInvoiceID).Scan(&source, &oldStatus); err != nil || source != input.PurchaseOrderID || oldStatus != "VOID" {
			return application.Document{}, ErrConflict
		}
	}
	id := uuid.NewString()
	no, err := documentNo(ctx, tx, "SINV")
	if err != nil {
		return application.Document{}, err
	}
	total := float64(0)
	for _, line := range input.Lines {
		if line.Quantity <= 0 || line.UnitPrice < 0 {
			return application.Document{}, ErrConflict
		}
		total += line.Quantity * line.UnitPrice
	}
	_, err = tx.Exec(ctx, `INSERT INTO supplier_invoices(id,document_no,supplier_invoice_number,purchase_order_id,supplier_id,replaces_invoice_id,status,invoice_date,due_date,currency,total,outstanding,created_by)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,'MATCH_PENDING',$7,$8,$9,$10,$10,$11)`, id, no, input.SupplierInvoiceNumber, input.PurchaseOrderID, supplierID, input.ReplacesInvoiceID, input.InvoiceDate, input.DueDate, input.Currency, total, actor)
	if err != nil {
		return application.Document{}, ErrConflict
	}
	for _, line := range input.Lines {
		_, err = tx.Exec(ctx, `INSERT INTO supplier_invoice_lines(supplier_invoice_id,purchase_order_line_id,quantity,unit_price,line_total) VALUES($1,$2,$3,$4,$5)`, id, line.ReferenceID, line.Quantity, line.UnitPrice, line.Quantity*line.UnitPrice)
		if err != nil {
			return application.Document{}, ErrConflict
		}
	}
	if err = audit(ctx, tx, actor, "RECORD_SUPPLIER_INVOICE", "PUR-INV-001", "SUPPLIER_INVOICE", id, "", "MATCH_PENDING", "", request); err != nil {
		return application.Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return application.Document{}, err
	}
	return application.Document{ID: id, DocumentNo: no, Status: "MATCH_PENDING", PartyID: supplierID, SourceID: input.PurchaseOrderID, Currency: input.Currency, Total: total, Outstanding: total, Version: 1}, nil
}

func (s Store) MatchSupplierInvoice(ctx context.Context, id, actor, request string) (application.Document, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,supplier_id,purchase_order_id,currency,total,outstanding,version FROM supplier_invoices WHERE id=$1 FOR UPDATE`, id).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version); err != nil || document.Status != "MATCH_PENDING" {
		return document, ErrConflict
	}
	expectedRows, err := tx.Query(ctx, `SELECT id,quantity,unit_price FROM purchase_order_lines WHERE purchase_order_id=$1 ORDER BY id`, document.SourceID)
	if err != nil {
		return document, err
	}
	expected := []domain.InvoiceLine{}
	for expectedRows.Next() {
		var line domain.InvoiceLine
		if err = expectedRows.Scan(&line.ReferenceID, &line.Quantity, &line.UnitPrice); err != nil {
			expectedRows.Close()
			return document, err
		}
		expected = append(expected, line)
	}
	expectedRows.Close()
	actualRows, err := tx.Query(ctx, `SELECT purchase_order_line_id,quantity,unit_price FROM supplier_invoice_lines WHERE supplier_invoice_id=$1 ORDER BY purchase_order_line_id`, id)
	if err != nil {
		return document, err
	}
	actual := []domain.InvoiceLine{}
	for actualRows.Next() {
		var line domain.InvoiceLine
		if err = actualRows.Scan(&line.ReferenceID, &line.Quantity, &line.UnitPrice); err != nil {
			actualRows.Close()
			return document, err
		}
		actual = append(actual, line)
	}
	actualRows.Close()
	to := "POSTED"
	reason := ""
	if matchErr := domain.ExactMatch(expected, actual); matchErr != nil {
		to = "MATCH_EXCEPTION"
		reason = matchErr.Error()
	}
	_, err = tx.Exec(ctx, `UPDATE supplier_invoices SET status=$1,mismatch_reason=NULLIF($2,''),version=version+1,updated_at=now() WHERE id=$3`, to, reason, id)
	if err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "MATCH_SUPPLIER_INVOICE", "PUR-INV-002", "SUPPLIER_INVOICE", id, "MATCH_PENDING", to, reason, request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status = to
	document.Version++
	return document, nil
}

func (s Store) ResolveSupplierInvoice(ctx context.Context, id string, lines []application.InvoiceLineInput, actor, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,supplier_id,purchase_order_id,currency,total,outstanding,version FROM supplier_invoices WHERE id=$1 FOR UPDATE`, id).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version); err != nil || document.Status != "MATCH_EXCEPTION" {
		return document, ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM supplier_invoice_lines WHERE supplier_invoice_id=$1`, id); err != nil {
		return document, err
	}
	total := float64(0)
	for _, line := range lines {
		if line.Quantity <= 0 || line.UnitPrice < 0 {
			return document, ErrConflict
		}
		total += line.Quantity * line.UnitPrice
		if _, err = tx.Exec(ctx, `INSERT INTO supplier_invoice_lines(supplier_invoice_id,purchase_order_line_id,quantity,unit_price,line_total) VALUES($1,$2,$3,$4,$5)`, id, line.ReferenceID, line.Quantity, line.UnitPrice, line.Quantity*line.UnitPrice); err != nil {
			return document, ErrConflict
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET status='MATCH_PENDING',total=$2,outstanding=$2,mismatch_reason=NULL,version=version+1,updated_at=now() WHERE id=$1`, id, total); err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "RESOLVE_SUPPLIER_MATCH", "PUR-INV-003", "SUPPLIER_INVOICE", id, "MATCH_EXCEPTION", "MATCH_PENDING", "", request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status, document.Total, document.Outstanding = "MATCH_PENDING", total, total
	document.Version++
	return document, nil
}

func (s Store) VoidSupplierInvoice(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	err = tx.QueryRow(ctx, `UPDATE supplier_invoices i SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1,updated_at=now()
		WHERE i.id=$1 AND i.status='POSTED' AND NOT EXISTS(SELECT 1 FROM supplier_payment_allocations a JOIN supplier_payments p ON p.id=a.payment_id WHERE a.invoice_id=i.id AND p.status='POSTED')
		RETURNING id,document_no,status,supplier_id,purchase_order_id,currency,total,outstanding,version`, id, reason, actor).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version)
	if err != nil {
		return document, ErrConflict
	}
	if err = audit(ctx, tx, actor, "VOID_SUPPLIER_INVOICE", "ACT-FIN-SINV-VOID", "SUPPLIER_INVOICE", id, "POSTED", "VOID", reason, request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	return document, nil
}

func allocationTotal(allocations []application.AllocationInput) (float64, error) {
	if len(allocations) == 0 {
		return 0, ErrConflict
	}
	total := float64(0)
	seen := map[string]bool{}
	for _, allocation := range allocations {
		if allocation.InvoiceID == "" || allocation.Amount <= 0 || seen[allocation.InvoiceID] {
			return 0, ErrConflict
		}
		seen[allocation.InvoiceID] = true
		total += allocation.Amount
	}
	return total, nil
}

func (s Store) ListSupplierPayments(ctx context.Context) ([]application.Document, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,document_no,status,supplier_id,''::text,currency,amount,0::numeric,version FROM supplier_payments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return scanDocuments(rows)
}

func (s Store) CreateSupplierPayment(ctx context.Context, input application.PaymentInput, actor, request string) (application.Document, error) {
	total, err := allocationTotal(input.Allocations)
	if err != nil || input.Amount != total || input.Method == "" || input.Currency == "" || input.PaymentDate == "" {
		return application.Document{}, ErrConflict
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.NewString()
	no, err := documentNo(ctx, tx, "SPAY")
	if err != nil {
		return application.Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO supplier_payments(id,document_no,supplier_id,status,payment_date,method,reference,currency,amount,created_by)
		VALUES($1,$2,$3,'DRAFT',$4,$5,NULLIF($6,''),$7,$8,$9)`, id, no, input.PartyID, input.PaymentDate, input.Method, input.Reference, input.Currency, input.Amount, actor); err != nil {
		return application.Document{}, ErrConflict
	}
	for _, allocation := range input.Allocations {
		if _, err = tx.Exec(ctx, `INSERT INTO supplier_payment_allocations(payment_id,invoice_id,amount) VALUES($1,$2,$3)`, id, allocation.InvoiceID, allocation.Amount); err != nil {
			return application.Document{}, ErrConflict
		}
	}
	if err = audit(ctx, tx, actor, "CREATE_SUPPLIER_PAYMENT", "PUR-PAY-001", "SUPPLIER_PAYMENT", id, "", "DRAFT", "", request); err != nil {
		return application.Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return application.Document{}, err
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", PartyID: input.PartyID, Currency: input.Currency, Total: input.Amount, Version: 1}, nil
}

func (s Store) PostSupplierPayment(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	var paymentDate string
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,supplier_id,''::text,currency,amount,0::numeric,version,payment_date::text FROM supplier_payments WHERE id=$1 FOR UPDATE`, id).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version, &paymentDate); err != nil || document.Status != "DRAFT" || document.Version != version {
		return document, ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT a.invoice_id,a.amount,i.status,i.total,i.outstanding,i.currency,i.invoice_date::text,i.purchase_order_id
		FROM supplier_payment_allocations a JOIN supplier_invoices i ON i.id=a.invoice_id WHERE a.payment_id=$1 ORDER BY a.invoice_id FOR UPDATE OF i`, id)
	if err != nil {
		return document, err
	}
	type allocation struct {
		invoice, status, currency, invoiceDate, source string
		amount, total, outstanding                     float64
	}
	allocations := []allocation{}
	for rows.Next() {
		var item allocation
		if err = rows.Scan(&item.invoice, &item.amount, &item.status, &item.total, &item.outstanding, &item.currency, &item.invoiceDate, &item.source); err != nil {
			rows.Close()
			return document, err
		}
		if item.status != "POSTED" && item.status != "PARTIALLY_PAID" {
			rows.Close()
			return document, ErrConflict
		}
		if err = domain.ValidatePayment(item.amount, item.amount, document.Currency, item.currency, paymentDate, item.invoiceDate); err != nil || item.amount > item.outstanding {
			rows.Close()
			return document, ErrConflict
		}
		allocations = append(allocations, item)
	}
	rows.Close()
	if len(allocations) == 0 {
		return document, ErrConflict
	}
	for _, item := range allocations {
		remaining := item.outstanding - item.amount
		status := "PARTIALLY_PAID"
		if remaining == 0 {
			status = "PAID"
		}
		if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, remaining, status); err != nil {
			return document, err
		}
		if remaining == 0 {
			var allPaid bool
			if err = tx.QueryRow(ctx, `SELECT bool_and(status='PAID') FROM supplier_invoices WHERE purchase_order_id=$1 AND status<>'VOID'`, item.source).Scan(&allPaid); err != nil {
				return document, err
			}
			if allPaid {
				if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='CLOSED',version=version+1,updated_at=now() WHERE id=$1 AND status='RECEIVED'`, item.source); err != nil {
					return document, err
				}
				if _, err = tx.Exec(ctx, `UPDATE purchase_requests r SET status='COMPLETED',version=r.version+1,updated_at=now() FROM purchase_orders p WHERE p.id=$1 AND r.id=p.purchase_request_id AND r.status='CONVERTED_TO_PO'`, item.source); err != nil {
					return document, err
				}
				if err = audit(ctx, tx, actor, "CLOSE_PURCHASE", "PUR-CLOSE-001", "PURCHASE_ORDER", item.source, "RECEIVED", "CLOSED", "", request); err != nil {
					return document, err
				}
			}
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE supplier_payments SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id); err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "POST_SUPPLIER_PAYMENT", "PUR-PAY-002", "SUPPLIER_PAYMENT", id, "DRAFT", "POSTED", "", request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status = "POSTED"
	document.Version++
	return document, nil
}

func (s Store) VoidSupplierPayment(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,supplier_id,''::text,currency,amount,0::numeric,version FROM supplier_payments WHERE id=$1 FOR UPDATE`, id).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version); err != nil || document.Status != "POSTED" {
		return document, ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT a.invoice_id,a.amount,i.total,i.outstanding,i.purchase_order_id FROM supplier_payment_allocations a JOIN supplier_invoices i ON i.id=a.invoice_id WHERE a.payment_id=$1 FOR UPDATE OF i`, id)
	if err != nil {
		return document, err
	}
	type allocation struct {
		invoice, source            string
		amount, total, outstanding float64
	}
	items := []allocation{}
	for rows.Next() {
		var item allocation
		if err = rows.Scan(&item.invoice, &item.amount, &item.total, &item.outstanding, &item.source); err != nil {
			rows.Close()
			return document, err
		}
		items = append(items, item)
	}
	rows.Close()
	for _, item := range items {
		outstanding := item.outstanding + item.amount
		status := "PARTIALLY_PAID"
		if outstanding == item.total {
			status = "POSTED"
		}
		if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, outstanding, status); err != nil {
			return document, err
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='RECEIVED',version=version+1,updated_at=now() WHERE id=$1 AND status='CLOSED'`, item.source); err != nil {
			return document, err
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_requests r SET status='CONVERTED_TO_PO',version=r.version+1,updated_at=now() FROM purchase_orders p WHERE p.id=$1 AND r.id=p.purchase_request_id AND r.status='COMPLETED'`, item.source); err != nil {
			return document, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE supplier_payments SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1 WHERE id=$1`, id, reason, actor); err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "VOID_SUPPLIER_PAYMENT", "ACT-FIN-SPAY-VOID", "SUPPLIER_PAYMENT", id, "POSTED", "VOID", reason, request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status = "VOID"
	document.Version++
	return document, nil
}

func (s Store) ListSalesInvoices(ctx context.Context) ([]application.Document, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,document_no,status,customer_id,sales_order_id,currency,total,outstanding,version FROM sales_invoices ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return scanDocuments(rows)
}

func (s Store) CreateSalesInvoice(ctx context.Context, input application.SalesInvoiceInput, actor, request string) (application.Document, error) {
	if input.SalesOrderID == "" || input.InvoiceDate == "" || input.DueDate == "" || input.Currency == "" {
		return application.Document{}, ErrConflict
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var customerID, status string
	if err = tx.QueryRow(ctx, `SELECT customer_id,status FROM sales_orders WHERE id=$1 FOR UPDATE`, input.SalesOrderID).Scan(&customerID, &status); err != nil || status != "DELIVERED" {
		return application.Document{}, ErrConflict
	}
	if input.ReplacesInvoiceID != "" {
		var source, oldStatus string
		if err = tx.QueryRow(ctx, `SELECT sales_order_id,status FROM sales_invoices WHERE id=$1`, input.ReplacesInvoiceID).Scan(&source, &oldStatus); err != nil || source != input.SalesOrderID || oldStatus != "VOID" {
			return application.Document{}, ErrConflict
		}
	}
	var total float64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(delivered_quantity*unit_price),0) FROM sales_order_lines WHERE sales_order_id=$1`, input.SalesOrderID).Scan(&total); err != nil || total <= 0 {
		return application.Document{}, ErrConflict
	}
	id := uuid.NewString()
	no, err := documentNo(ctx, tx, "CINV")
	if err != nil {
		return application.Document{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO sales_invoices(id,document_no,sales_order_id,customer_id,replaces_invoice_id,status,invoice_date,due_date,currency,total,outstanding,created_by)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,'POSTED',$6,$7,$8,$9,$9,$10)`, id, no, input.SalesOrderID, customerID, input.ReplacesInvoiceID, input.InvoiceDate, input.DueDate, input.Currency, total, actor)
	if err != nil {
		return application.Document{}, ErrConflict
	}
	if _, err = tx.Exec(ctx, `INSERT INTO sales_invoice_lines(sales_invoice_id,sales_order_line_id,quantity,unit_price,line_total)
		SELECT $1,id,delivered_quantity,unit_price,delivered_quantity*unit_price FROM sales_order_lines WHERE sales_order_id=$2`, id, input.SalesOrderID); err != nil {
		return application.Document{}, err
	}
	if err = audit(ctx, tx, actor, "CREATE_SALES_INVOICE", "SAL-INV-001", "SALES_INVOICE", id, "", "POSTED", "", request); err != nil {
		return application.Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return application.Document{}, err
	}
	return application.Document{ID: id, DocumentNo: no, Status: "POSTED", PartyID: customerID, SourceID: input.SalesOrderID, Currency: input.Currency, Total: total, Outstanding: total, Version: 1}, nil
}

func (s Store) VoidSalesInvoice(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	err = tx.QueryRow(ctx, `UPDATE sales_invoices i SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1,updated_at=now()
		WHERE i.id=$1 AND i.status='POSTED' AND NOT EXISTS(SELECT 1 FROM customer_payment_allocations a JOIN customer_payments p ON p.id=a.payment_id WHERE a.invoice_id=i.id AND p.status='POSTED')
		RETURNING id,document_no,status,customer_id,sales_order_id,currency,total,outstanding,version`, id, reason, actor).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version)
	if err != nil {
		return document, ErrConflict
	}
	if err = audit(ctx, tx, actor, "VOID_SALES_INVOICE", "ACT-FIN-CINV-VOID", "SALES_INVOICE", id, "POSTED", "VOID", reason, request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	return document, nil
}

func (s Store) ListCustomerPayments(ctx context.Context) ([]application.Document, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,document_no,status,customer_id,''::text,currency,amount,0::numeric,version FROM customer_payments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return scanDocuments(rows)
}

func (s Store) CreateCustomerPayment(ctx context.Context, input application.PaymentInput, actor, request string) (application.Document, error) {
	total, err := allocationTotal(input.Allocations)
	if err != nil || input.Amount != total || input.Method == "" || input.Currency == "" || input.PaymentDate == "" {
		return application.Document{}, ErrConflict
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.NewString()
	no, err := documentNo(ctx, tx, "CPAY")
	if err != nil {
		return application.Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO customer_payments(id,document_no,customer_id,status,payment_date,method,reference,currency,amount,created_by) VALUES($1,$2,$3,'DRAFT',$4,$5,NULLIF($6,''),$7,$8,$9)`, id, no, input.PartyID, input.PaymentDate, input.Method, input.Reference, input.Currency, input.Amount, actor); err != nil {
		return application.Document{}, ErrConflict
	}
	for _, allocation := range input.Allocations {
		if _, err = tx.Exec(ctx, `INSERT INTO customer_payment_allocations(payment_id,invoice_id,amount)VALUES($1,$2,$3)`, id, allocation.InvoiceID, allocation.Amount); err != nil {
			return application.Document{}, ErrConflict
		}
	}
	if err = audit(ctx, tx, actor, "CREATE_CUSTOMER_PAYMENT", "SAL-PAY-001", "CUSTOMER_PAYMENT", id, "", "DRAFT", "", request); err != nil {
		return application.Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return application.Document{}, err
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", PartyID: input.PartyID, Currency: input.Currency, Total: input.Amount, Version: 1}, nil
}

func (s Store) PostCustomerPayment(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	var paymentDate string
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,customer_id,''::text,currency,amount,0::numeric,version,payment_date::text FROM customer_payments WHERE id=$1 FOR UPDATE`, id).
		Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version, &paymentDate); err != nil || document.Status != "DRAFT" || document.Version != version {
		return document, ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT a.invoice_id,a.amount,i.status,i.total,i.outstanding,i.currency,i.invoice_date::text,i.sales_order_id FROM customer_payment_allocations a JOIN sales_invoices i ON i.id=a.invoice_id WHERE a.payment_id=$1 ORDER BY a.invoice_id FOR UPDATE OF i`, id)
	if err != nil {
		return document, err
	}
	type allocation struct {
		invoice, status, currency, invoiceDate, source string
		amount, total, outstanding                     float64
	}
	items := []allocation{}
	for rows.Next() {
		var item allocation
		if err = rows.Scan(&item.invoice, &item.amount, &item.status, &item.total, &item.outstanding, &item.currency, &item.invoiceDate, &item.source); err != nil {
			rows.Close()
			return document, err
		}
		if item.status != "POSTED" && item.status != "PARTIALLY_PAID" {
			rows.Close()
			return document, ErrConflict
		}
		if err = domain.ValidatePayment(item.amount, item.amount, document.Currency, item.currency, paymentDate, item.invoiceDate); err != nil || item.amount > item.outstanding {
			rows.Close()
			return document, ErrConflict
		}
		items = append(items, item)
	}
	rows.Close()
	if len(items) == 0 {
		return document, ErrConflict
	}
	for _, item := range items {
		remaining := item.outstanding - item.amount
		status := "PARTIALLY_PAID"
		if remaining == 0 {
			status = "PAID"
		}
		if _, err = tx.Exec(ctx, `UPDATE sales_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, remaining, status); err != nil {
			return document, err
		}
		if remaining == 0 {
			var allPaid bool
			if err = tx.QueryRow(ctx, `SELECT bool_and(status='PAID') FROM sales_invoices WHERE sales_order_id=$1 AND status<>'VOID'`, item.source).Scan(&allPaid); err != nil {
				return document, err
			}
			if allPaid {
				if _, err = tx.Exec(ctx, `UPDATE sales_orders SET status='COMPLETED',version=version+1,updated_at=now() WHERE id=$1 AND status='DELIVERED'`, item.source); err != nil {
					return document, err
				}
				if err = audit(ctx, tx, actor, "COMPLETE_SALES_ORDER", "SAL-SO-006", "SALES_ORDER", item.source, "DELIVERED", "COMPLETED", "", request); err != nil {
					return document, err
				}
			}
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE customer_payments SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id); err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "POST_CUSTOMER_PAYMENT", "SAL-PAY-002", "CUSTOMER_PAYMENT", id, "DRAFT", "POSTED", "", request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status = "POSTED"
	document.Version++
	return document, nil
}

func (s Store) VoidCustomerPayment(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return application.Document{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var document application.Document
	if err = tx.QueryRow(ctx, `SELECT id,document_no,status,customer_id,''::text,currency,amount,0::numeric,version FROM customer_payments WHERE id=$1 FOR UPDATE`, id).Scan(&document.ID, &document.DocumentNo, &document.Status, &document.PartyID, &document.SourceID, &document.Currency, &document.Total, &document.Outstanding, &document.Version); err != nil || document.Status != "POSTED" {
		return document, ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT a.invoice_id,a.amount,i.total,i.outstanding,i.sales_order_id FROM customer_payment_allocations a JOIN sales_invoices i ON i.id=a.invoice_id WHERE a.payment_id=$1 FOR UPDATE OF i`, id)
	if err != nil {
		return document, err
	}
	type allocation struct {
		invoice, source            string
		amount, total, outstanding float64
	}
	items := []allocation{}
	for rows.Next() {
		var item allocation
		if err = rows.Scan(&item.invoice, &item.amount, &item.total, &item.outstanding, &item.source); err != nil {
			rows.Close()
			return document, err
		}
		items = append(items, item)
	}
	rows.Close()
	for _, item := range items {
		outstanding := item.outstanding + item.amount
		status := "PARTIALLY_PAID"
		if outstanding == item.total {
			status = "POSTED"
		}
		if _, err = tx.Exec(ctx, `UPDATE sales_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, outstanding, status); err != nil {
			return document, err
		}
		if _, err = tx.Exec(ctx, `UPDATE sales_orders SET status='DELIVERED',version=version+1,updated_at=now() WHERE id=$1 AND status='COMPLETED'`, item.source); err != nil {
			return document, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE customer_payments SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1 WHERE id=$1`, id, reason, actor); err != nil {
		return document, err
	}
	if err = audit(ctx, tx, actor, "VOID_CUSTOMER_PAYMENT", "ACT-FIN-CPAY-VOID", "CUSTOMER_PAYMENT", id, "POSTED", "VOID", reason, request); err != nil {
		return document, err
	}
	if err = tx.Commit(ctx); err != nil {
		return document, err
	}
	document.Status = "VOID"
	document.Version++
	return document, nil
}
