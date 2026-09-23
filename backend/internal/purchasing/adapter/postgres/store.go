package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/purchasing/application"
	"time"
)

var ErrConflict = errors.New("invalid state, version, quantity, or maker-checker conflict")

type Store struct{ Pool *pgxpool.Pool }

func docNo(ctx context.Context, tx pgx.Tx, prefix string) (string, error) {
	var n int64
	e := tx.QueryRow(ctx, `SELECT nextval('document_number_sequence')`).Scan(&n)
	return fmt.Sprintf("%s-%s-%06d", prefix, time.Now().Format("20060102"), n), e
}
func audit(ctx context.Context, tx pgx.Tx, actor, action, trace, entity, id, from, to, reason, request string) error {
	_, e := tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,reason,request_id)VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)`, actor, action, trace, entity, id, from, to, reason, request)
	return e
}
func (s Store) CreateRequest(ctx context.Context, actor string, lines []application.LineInput, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, errors.New("at least one line required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.NewString()
	no, e := docNo(ctx, tx, "PR")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO purchase_requests(id,document_no,creator_id,status)VALUES($1,$2,$3,'DRAFT')`, id, no, actor)
	if e != nil {
		return application.Document{}, e
	}
	for _, l := range lines {
		if l.Quantity <= 0 || l.UnitPrice < 0 {
			return application.Document{}, errors.New("invalid quantity or price")
		}
		var active bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&active)
		if e != nil || !active {
			return application.Document{}, ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO purchase_request_lines(purchase_request_id,product_id,quantity,estimated_price)VALUES($1,$2,$3,$4)`, id, l.ProductID, l.Quantity, l.UnitPrice)
		if e != nil {
			return application.Document{}, e
		}
	}
	if e = audit(ctx, tx, actor, "CREATE_PURCHASE_REQUEST", "PUR-PR-001", "PURCHASE_REQUEST", id, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) ListRequests(ctx context.Context) ([]application.Document, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM purchase_requests ORDER BY created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []application.Document{}
	for rows.Next() {
		var d application.Document
		if e = rows.Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version); e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s Store) ListOrders(ctx context.Context) ([]application.Document, error) {
	return s.listDocuments(ctx, `SELECT id,document_no,status,creator_id,version FROM purchase_orders ORDER BY created_at DESC`)
}
func (s Store) ListReceipts(ctx context.Context) ([]application.Document, error) {
	return s.listDocuments(ctx, `SELECT id,document_no,status,creator_id,version FROM goods_receipts ORDER BY created_at DESC`)
}
func (s Store) listDocuments(ctx context.Context, query string) ([]application.Document, error) {
	rows, e := s.Pool.Query(ctx, query)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []application.Document{}
	for rows.Next() {
		var d application.Document
		if e = rows.Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version); e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s Store) TransitionRequest(ctx context.Context, id, action, actor, reason string, version int, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version FROM purchase_requests WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil {
		return d, e
	}
	if d.Version != version {
		return d, ErrConflict
	}
	from := d.Status
	trace := "PUR-PR-002"
	switch action {
	case "submit":
		if from != "DRAFT" {
			return d, ErrConflict
		}
		d.Status = "SUBMITTED"
	case "approve":
		trace = "PUR-PR-003"
		if from != "SUBMITTED" || actor == d.CreatorID {
			return d, ErrConflict
		}
		d.Status = "APPROVED"
	case "reject":
		trace = "PUR-PR-003"
		if from != "SUBMITTED" || actor == d.CreatorID || reason == "" {
			return d, ErrConflict
		}
		d.Status = "REJECTED"
	case "cancel":
		trace = "ACT-PUR-PR-CANCEL"
		var admin bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='ADMIN')`, actor).Scan(&admin)
		if e != nil || from != "DRAFT" || reason == "" || (actor != d.CreatorID && !admin) {
			return d, ErrConflict
		}
		d.Status = "CANCELLED"
	default:
		return d, ErrConflict
	}
	tag, e := tx.Exec(ctx, `UPDATE purchase_requests SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject') THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version)
	if e != nil || tag.RowsAffected() != 1 {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "PR_"+action, trace, "PURCHASE_REQUEST", id, from, d.Status, reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Version++
	return d, nil
}
func (s Store) ConvertToOrder(ctx context.Context, id, supplier, warehouse, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var creator, status string
	e = tx.QueryRow(ctx, `SELECT creator_id,status FROM purchase_requests WHERE id=$1 FOR UPDATE`, id).Scan(&creator, &status)
	if e != nil || status != "APPROVED" {
		return application.Document{}, ErrConflict
	}
	var active bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE id=$1 AND active)
		AND EXISTS(SELECT 1 FROM warehouses WHERE id=$2 AND active)`, supplier, warehouse).Scan(&active)
	if e != nil || !active {
		return application.Document{}, ErrConflict
	}
	po := uuid.NewString()
	no, e := docNo(ctx, tx, "PO")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO purchase_orders(id,document_no,purchase_request_id,supplier_id,warehouse_id,creator_id,status)VALUES($1,$2,$3,$4,$5,$6,'DRAFT')`, po, no, id, supplier, warehouse, actor)
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO purchase_order_lines(purchase_order_id,product_id,quantity,unit_price)SELECT $1,product_id,quantity,estimated_price FROM purchase_request_lines WHERE purchase_request_id=$2`, po, id)
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `UPDATE purchase_requests SET status='CONVERTED_TO_PO',version=version+1,updated_at=now() WHERE id=$1`, id)
	if e != nil {
		return application.Document{}, e
	}
	if e = audit(ctx, tx, actor, "CONVERT_TO_PO", "PUR-PO-001", "PURCHASE_ORDER", po, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: po, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) IssueOrder(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `UPDATE purchase_orders SET status='ISSUED',version=version+1,updated_at=now() WHERE id=$1 AND status='DRAFT' AND version=$2 RETURNING id,document_no,status,creator_id,version`, id, version).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "ISSUE_PO", "PUR-PO-002", "PURCHASE_ORDER", id, "DRAFT", "ISSUED", "", request); e != nil {
		return d, e
	}
	e = tx.Commit(ctx)
	return d, e
}
func (s Store) CreateReceipt(ctx context.Context, po string, lines []application.ReceiptLineInput, actor, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	e = tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1`, po).Scan(&status)
	if e != nil || (status != "ISSUED" && status != "PARTIALLY_RECEIVED") {
		return application.Document{}, ErrConflict
	}
	id := uuid.NewString()
	no, e := docNo(ctx, tx, "GR")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO goods_receipts(id,document_no,purchase_order_id,creator_id,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, po, actor)
	if e != nil {
		return application.Document{}, e
	}
	for _, l := range lines {
		if l.Quantity <= 0 {
			return application.Document{}, ErrConflict
		}
		var valid bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM purchase_order_lines
			WHERE id=$1 AND purchase_order_id=$2 AND product_id=$3
		)`, l.PurchaseOrderLineID, po, l.ProductID).Scan(&valid)
		if e != nil || !valid {
			return application.Document{}, ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO goods_receipt_lines(goods_receipt_id,purchase_order_line_id,product_id,quantity)VALUES($1,$2,$3,$4)`, id, l.PurchaseOrderLineID, l.ProductID, l.Quantity)
		if e != nil {
			return application.Document{}, e
		}
	}
	if e = audit(ctx, tx, actor, "CREATE_GOODS_RECEIPT", "PUR-GR-001", "GOODS_RECEIPT", id, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) PostReceipt(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var po, warehouse string
	e = tx.QueryRow(ctx, `SELECT g.id,g.document_no,g.status,g.creator_id,g.version,g.purchase_order_id,p.warehouse_id FROM goods_receipts g JOIN purchase_orders p ON p.id=g.purchase_order_id WHERE g.id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &po, &warehouse)
	if e != nil || d.Status != "DRAFT" || d.Version != version {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT gl.id,gl.purchase_order_line_id,gl.product_id,gl.quantity,pol.quantity-pol.received_quantity FROM goods_receipt_lines gl JOIN purchase_order_lines pol ON pol.id=gl.purchase_order_line_id WHERE gl.goods_receipt_id=$1 FOR UPDATE OF pol`, id)
	if e != nil {
		return d, e
	}
	type line struct {
		id, pol, product string
		qty, remaining   float64
	}
	ls := []line{}
	for rows.Next() {
		var l line
		if e = rows.Scan(&l.id, &l.pol, &l.product, &l.qty, &l.remaining); e != nil {
			rows.Close()
			return d, e
		}
		if l.qty > l.remaining {
			return d, ErrConflict
		}
		ls = append(ls, l)
	}
	rows.Close()
	for _, l := range ls {
		_, e = tx.Exec(ctx, `UPDATE purchase_order_lines SET received_quantity=received_quantity+$1 WHERE id=$2`, l.qty, l.pol)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,$3)ON CONFLICT(warehouse_id,product_id)DO UPDATE SET quantity=stock_balances.quantity+EXCLUDED.quantity,updated_at=now()`, warehouse, l.product, l.qty)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'GOODS_RECEIPT','GOODS_RECEIPT',$4,$5,$6)`, warehouse, l.product, l.qty, id, l.id, actor)
		if e != nil {
			return d, e
		}
	}
	var complete bool
	e = tx.QueryRow(ctx, `SELECT bool_and(received_quantity=quantity) FROM purchase_order_lines WHERE purchase_order_id=$1`, po).Scan(&complete)
	if e != nil {
		return d, e
	}
	poStatus := "PARTIALLY_RECEIVED"
	if complete {
		poStatus = "RECEIVED"
	}
	_, e = tx.Exec(ctx, `UPDATE purchase_orders SET status=$1,version=version+1,updated_at=now() WHERE id=$2`, poStatus, po)
	if e != nil {
		return d, e
	}
	_, e = tx.Exec(ctx, `UPDATE goods_receipts SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id)
	if e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, "POST_GOODS_RECEIPT", "PUR-GR-002", "GOODS_RECEIPT", id, "DRAFT", "POSTED", "", request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "POSTED"
	d.Version++
	return d, nil
}
func (s Store) ReverseReceipt(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var po, warehouse string
	e = tx.QueryRow(ctx, `SELECT g.id,g.document_no,g.status,g.creator_id,g.version,g.purchase_order_id,p.warehouse_id FROM goods_receipts g JOIN purchase_orders p ON p.id=g.purchase_order_id WHERE g.id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &po, &warehouse)
	if e != nil || d.Status != "POSTED" {
		return d, ErrConflict
	}
	var invoiced bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_invoices WHERE purchase_order_id=$1 AND status<>'VOID')`, po).Scan(&invoiced); e != nil || invoiced {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT sm.id,sm.product_id,sm.quantity,sm.source_line_id FROM stock_movements sm WHERE sm.source_id=$1 AND sm.movement_type='GOODS_RECEIPT'`, id)
	if e != nil {
		return d, e
	}
	type line struct {
		movement, product, line string
		qty                     float64
	}
	ls := []line{}
	for rows.Next() {
		var l line
		if e = rows.Scan(&l.movement, &l.product, &l.qty, &l.line); e != nil {
			rows.Close()
			return d, e
		}
		ls = append(ls, l)
	}
	rows.Close()
	for _, l := range ls {
		var balance float64
		e = tx.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2 FOR UPDATE`, warehouse, l.product).Scan(&balance)
		if e != nil || balance < l.qty {
			return d, ErrConflict
		}
		_, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, l.qty, warehouse, l.product)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'GOODS_RECEIPT_REVERSAL','GOODS_RECEIPT',$4,$5,$6,$7)`, warehouse, l.product, -l.qty, id, l.line, l.movement, actor)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `UPDATE purchase_order_lines pol SET received_quantity=received_quantity-$1 FROM goods_receipt_lines gl WHERE gl.id=$2 AND pol.id=gl.purchase_order_line_id`, l.qty, l.line)
		if e != nil {
			return d, e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE goods_receipts SET status='REVERSED',reason=$1,version=version+1 WHERE id=$2`, reason, id)
	if e != nil {
		return d, e
	}
	_, e = tx.Exec(ctx, `UPDATE purchase_orders SET status=CASE WHEN EXISTS(SELECT 1 FROM purchase_order_lines WHERE purchase_order_id=$1 AND received_quantity>0)THEN'PARTIALLY_RECEIVED'ELSE'ISSUED'END,version=version+1 WHERE id=$1`, po)
	if e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, "REVERSE_GOODS_RECEIPT", "ACT-PUR-GR-REVERSE", "GOODS_RECEIPT", id, "POSTED", "REVERSED", reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "REVERSED"
	d.Version++
	return d, nil
}
func (s Store) CopyRequest(ctx context.Context, id, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	e = tx.QueryRow(ctx, `SELECT status FROM purchase_requests WHERE id=$1`, id).Scan(&status)
	if e != nil || status != "REJECTED" {
		return application.Document{}, ErrConflict
	}
	newID := uuid.NewString()
	no, e := docNo(ctx, tx, "PR")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO purchase_requests(id,document_no,creator_id,status,copied_from_id)VALUES($1,$2,$3,'DRAFT',$4)`, newID, no, actor, id)
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO purchase_request_lines(purchase_request_id,product_id,quantity,estimated_price)SELECT $1,product_id,quantity,estimated_price FROM purchase_request_lines WHERE purchase_request_id=$2`, newID, id)
	if e != nil {
		return application.Document{}, e
	}
	if e = audit(ctx, tx, actor, "COPY_REJECTED_PR", "ACT-PUR-PR-COPY", "PURCHASE_REQUEST", newID, "REJECTED", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: newID, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) CancelOrder(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `UPDATE purchase_orders p SET status='CANCELLED',reason=$2,version=version+1,updated_at=now() WHERE p.id=$1 AND p.status IN('DRAFT','ISSUED') AND NOT EXISTS(SELECT 1 FROM goods_receipts g WHERE g.purchase_order_id=p.id AND g.status='POSTED') RETURNING id,document_no,status,creator_id,version`, id, reason).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "CANCEL_PO", "ACT-PUR-PO-CANCEL", "PURCHASE_ORDER", id, "", "CANCELLED", reason, request); e != nil {
		return d, e
	}
	e = tx.Commit(ctx)
	return d, e
}
func (s Store) CancelReceipt(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `UPDATE goods_receipts SET status='CANCELLED',reason=$2,version=version+1 WHERE id=$1 AND status='DRAFT' RETURNING id,document_no,status,creator_id,version`, id, reason).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "CANCEL_GR", "ACT-PUR-GR-CANCEL", "GOODS_RECEIPT", id, "DRAFT", "CANCELLED", reason, request); e != nil {
		return d, e
	}
	e = tx.Commit(ctx)
	return d, e
}
