package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/sales/application"
	"time"
)

var ErrConflict = errors.New("invalid state, version, stock, quantity, or maker-checker conflict")

type Store struct{ Pool *pgxpool.Pool }

func number(ctx context.Context, tx pgx.Tx, p string) (string, error) {
	var n int64
	e := tx.QueryRow(ctx, `SELECT nextval('document_number_sequence')`).Scan(&n)
	return fmt.Sprintf("%s-%s-%06d", p, time.Now().Format("20060102"), n), e
}
func audit(ctx context.Context, tx pgx.Tx, actor, action, trace, entity, id, from, to, reason, request string) error {
	_, e := tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,reason,request_id)VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)`, actor, action, trace, entity, id, from, to, reason, request)
	return e
}
func (s Store) CreateOrder(ctx context.Context, customer, warehouse, actor string, lines []application.LineInput, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND active)
		AND EXISTS(SELECT 1 FROM warehouses WHERE id=$2 AND active)`, customer, warehouse).Scan(&active)
	if e != nil || !active {
		return application.Document{}, ErrConflict
	}
	id := uuid.NewString()
	no, e := number(ctx, tx, "SO")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO sales_orders(id,document_no,customer_id,warehouse_id,creator_id,status)VALUES($1,$2,$3,$4,$5,'DRAFT')`, id, no, customer, warehouse, actor)
	if e != nil {
		return application.Document{}, e
	}
	for _, l := range lines {
		if l.Quantity <= 0 || l.UnitPrice < 0 {
			return application.Document{}, ErrConflict
		}
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&active)
		if e != nil || !active {
			return application.Document{}, ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO sales_order_lines(sales_order_id,product_id,quantity,unit_price)VALUES($1,$2,$3,$4)`, id, l.ProductID, l.Quantity, l.UnitPrice)
		if e != nil {
			return application.Document{}, e
		}
	}
	if e = audit(ctx, tx, actor, "CREATE_SALES_ORDER", "SAL-SO-001", "SALES_ORDER", id, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) UpdateOrder(ctx context.Context, id string, version int, actor string, lines []application.LineInput, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version FROM sales_orders WHERE id=$1 FOR UPDATE`, id).
		Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil || d.Status != "DRAFT" || d.Version != version {
		return d, ErrConflict
	}
	if _, e = tx.Exec(ctx, `DELETE FROM sales_order_lines WHERE sales_order_id=$1`, id); e != nil {
		return d, e
	}
	seen := make(map[string]struct{}, len(lines))
	for _, l := range lines {
		if l.Quantity <= 0 || l.UnitPrice < 0 {
			return d, ErrConflict
		}
		if _, duplicate := seen[l.ProductID]; duplicate {
			return d, ErrConflict
		}
		seen[l.ProductID] = struct{}{}
		var active bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&active)
		if e != nil || !active {
			return d, ErrConflict
		}
		if _, e = tx.Exec(ctx, `INSERT INTO sales_order_lines(sales_order_id,product_id,quantity,unit_price)VALUES($1,$2,$3,$4)`, id, l.ProductID, l.Quantity, l.UnitPrice); e != nil {
			return d, e
		}
	}
	tag, e := tx.Exec(ctx, `UPDATE sales_orders SET version=version+1,updated_at=now() WHERE id=$1 AND status='DRAFT' AND version=$2`, id, version)
	if e != nil {
		return d, e
	}
	if tag.RowsAffected() != 1 {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "UPDATE_SALES_ORDER", "SAL-SO-001", "SALES_ORDER", id, "DRAFT", "DRAFT", "", request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Version++
	return d, nil
}
func (s Store) ListOrders(ctx context.Context) ([]application.Document, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM sales_orders ORDER BY created_at DESC`)
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
func (s Store) GetOrder(ctx context.Context, id string) (application.OrderDetail, error) {
	var out application.OrderDetail
	e := s.Pool.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version,confirmed_at IS NOT NULL FROM sales_orders WHERE id=$1`, id).
		Scan(&out.ID, &out.DocumentNo, &out.Status, &out.CreatorID, &out.Version, &out.Confirmed)
	if e != nil {
		return out, e
	}
	rows, e := s.Pool.Query(ctx, `SELECT id,product_id,quantity,unit_price,delivered_quantity,quantity-delivered_quantity FROM sales_order_lines WHERE sales_order_id=$1 ORDER BY id`, id)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	out.Lines = []application.FulfillmentLine{}
	for rows.Next() {
		var line application.FulfillmentLine
		if e = rows.Scan(&line.ID, &line.ProductID, &line.Quantity, &line.UnitPrice, &line.ProcessedQuantity, &line.RemainingQuantity); e != nil {
			return out, e
		}
		out.Lines = append(out.Lines, line)
	}
	return out, rows.Err()
}
func (s Store) ListDeliveries(ctx context.Context) ([]application.Document, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM deliveries ORDER BY created_at DESC`)
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

func salesOrderStockAvailable(ctx context.Context, tx pgx.Tx, id, warehouse string) (bool, error) {
	var available bool
	e := tx.QueryRow(ctx, `SELECT COALESCE(bool_and(COALESCE(b.quantity,0)>=l.quantity-l.delivered_quantity),false)
		FROM sales_order_lines l
		LEFT JOIN stock_balances b ON b.product_id=l.product_id AND b.warehouse_id=$2
		WHERE l.sales_order_id=$1`, id, warehouse).Scan(&available)
	return available, e
}

func (s Store) ValidateStock(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var warehouse string
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version,warehouse_id FROM sales_orders WHERE id=$1 FOR UPDATE`, id).
		Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &warehouse)
	if e != nil || d.Status != `DRAFT` || d.Version != version {
		return d, ErrConflict
	}
	available, e := salesOrderStockAvailable(ctx, tx, id, warehouse)
	if e != nil || !available {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, `VALIDATE_SALES_ORDER_STOCK`, `SAL-SO-003`, `SALES_ORDER`, id, d.Status, d.Status, ``, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	return d, nil
}
func (s Store) TransitionOrder(ctx context.Context, id, action, actor, reason string, version int, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var warehouse string
	var confirmedAt *time.Time
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version,warehouse_id,confirmed_at FROM sales_orders WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &warehouse, &confirmedAt)
	if e != nil || d.Version != version {
		return d, ErrConflict
	}
	from := d.Status
	trace := "SAL-SO-002"
	switch action {
	case "submit":
		if from != "DRAFT" {
			return d, ErrConflict
		}
		available, stockErr := salesOrderStockAvailable(ctx, tx, id, warehouse)
		if stockErr != nil || !available {
			return d, ErrConflict
		}
		if e = audit(ctx, tx, actor, "VALIDATE_SALES_ORDER_STOCK", "SAL-SO-003", "SALES_ORDER", id, from, from, "", request); e != nil {
			return d, e
		}
		d.Status = "SUBMITTED"
	case "approve":
		trace = "SAL-SO-004"
		if from != "SUBMITTED" || actor == d.CreatorID {
			return d, ErrConflict
		}
		available, stockErr := salesOrderStockAvailable(ctx, tx, id, warehouse)
		if stockErr != nil || !available {
			return d, ErrConflict
		}
		d.Status = "APPROVED"
	case "reject":
		trace = "SAL-SO-004"
		if from != "SUBMITTED" || actor == d.CreatorID || reason == "" {
			return d, ErrConflict
		}
		d.Status = "REJECTED"
	case "confirm":
		trace = "SAL-SO-005"
		if from != "APPROVED" || confirmedAt != nil {
			return d, ErrConflict
		}
		d.Status = "APPROVED"
	case "cancel":
		trace = "ACT-SAL-SO-CANCEL"
		var admin bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='ADMIN')`, actor).Scan(&admin)
		if e != nil || (from != "DRAFT" && from != "APPROVED") || reason == "" || (actor != d.CreatorID && !admin) {
			return d, ErrConflict
		}
		d.Status = "CANCELLED"
	default:
		return d, ErrConflict
	}
	tag, e := tx.Exec(ctx, `UPDATE sales_orders SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject')THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),confirmed_at=CASE WHEN $2='confirm' THEN now() ELSE confirmed_at END,version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version)
	if e != nil || tag.RowsAffected() != 1 {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "SO_"+action, trace, "SALES_ORDER", id, from, d.Status, reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Version++
	return d, nil
}
func (s Store) CreateDelivery(ctx context.Context, so string, lines []application.DeliveryLineInput, actor, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var confirmedAt *time.Time
	e = tx.QueryRow(ctx, `SELECT status,confirmed_at FROM sales_orders WHERE id=$1 FOR UPDATE`, so).Scan(&status, &confirmedAt)
	if e != nil || confirmedAt == nil || (status != "APPROVED" && status != "PARTIALLY_DELIVERED") {
		return application.Document{}, ErrConflict
	}
	id := uuid.NewString()
	no, e := number(ctx, tx, "DLV")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO deliveries(id,document_no,sales_order_id,creator_id,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, so, actor)
	if e != nil {
		return application.Document{}, e
	}
	for _, l := range lines {
		if l.Quantity <= 0 {
			return application.Document{}, ErrConflict
		}
		var valid bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM sales_order_lines
			WHERE id=$1 AND sales_order_id=$2 AND product_id=$3
		)`, l.SalesOrderLineID, so, l.ProductID).Scan(&valid)
		if e != nil || !valid {
			return application.Document{}, ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO delivery_lines(delivery_id,sales_order_line_id,product_id,quantity)VALUES($1,$2,$3,$4)`, id, l.SalesOrderLineID, l.ProductID, l.Quantity)
		if e != nil {
			return application.Document{}, e
		}
	}
	if e = audit(ctx, tx, actor, "CREATE_DELIVERY", "SAL-DLV-001", "DELIVERY", id, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) PostDelivery(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var so, warehouse, parentSOStatus string
	e = tx.QueryRow(ctx, `SELECT d.id,d.document_no,d.status,d.creator_id,d.version,d.sales_order_id,s.warehouse_id,s.status FROM deliveries d JOIN sales_orders s ON s.id=d.sales_order_id WHERE d.id=$1 FOR UPDATE OF d,s`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &so, &warehouse, &parentSOStatus)
	if e != nil || d.Status != "DRAFT" || d.Version != version {
		return d, ErrConflict
	}
	if parentSOStatus != "APPROVED" && parentSOStatus != "PARTIALLY_DELIVERED" {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT dl.id,dl.sales_order_line_id,dl.product_id,dl.quantity,sol.quantity-sol.delivered_quantity FROM delivery_lines dl JOIN sales_order_lines sol ON sol.id=dl.sales_order_line_id WHERE dl.delivery_id=$1 FOR UPDATE OF sol`, id)
	if e != nil {
		return d, e
	}
	type line struct {
		id, sol, product string
		qty, remaining   float64
	}
	ls := []line{}
	for rows.Next() {
		var l line
		if e = rows.Scan(&l.id, &l.sol, &l.product, &l.qty, &l.remaining); e != nil {
			rows.Close()
			return d, e
		}
		if l.qty > l.remaining {
			rows.Close()
			return d, ErrConflict
		}
		ls = append(ls, l)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return d, e
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
		_, e = tx.Exec(ctx, `UPDATE sales_order_lines SET delivered_quantity=delivered_quantity+$1 WHERE id=$2`, l.qty, l.sol)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'SALES_DELIVERY','DELIVERY',$4,$5,$6)`, warehouse, l.product, -l.qty, id, l.id, actor)
		if e != nil {
			return d, e
		}
	}
	var complete bool
	e = tx.QueryRow(ctx, `SELECT bool_and(delivered_quantity=quantity) FROM sales_order_lines WHERE sales_order_id=$1`, so).Scan(&complete)
	if e != nil {
		return d, e
	}
	nextSOStatus := "PARTIALLY_DELIVERED"
	if complete {
		nextSOStatus = "DELIVERED"
	}
	tag, e := tx.Exec(ctx, `UPDATE sales_orders SET status=$1,version=version+1,updated_at=now() WHERE id=$2 AND status IN('APPROVED','PARTIALLY_DELIVERED')`, nextSOStatus, so)
	if e != nil {
		return d, e
	}
	if tag.RowsAffected() != 1 {
		return d, ErrConflict
	}
	_, e = tx.Exec(ctx, `UPDATE deliveries SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id)
	if e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, "POST_DELIVERY", "SAL-DLV-002", "DELIVERY", id, "DRAFT", "POSTED", "", request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "POSTED"
	d.Version++
	return d, nil
}
func (s Store) ReverseDelivery(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var so, warehouse string
	e = tx.QueryRow(ctx, `SELECT d.id,d.document_no,d.status,d.creator_id,d.version,d.sales_order_id,s.warehouse_id FROM deliveries d JOIN sales_orders s ON s.id=d.sales_order_id WHERE d.id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &so, &warehouse)
	if e != nil || d.Status != "POSTED" {
		return d, ErrConflict
	}
	var invoiced bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sales_invoices WHERE sales_order_id=$1 AND status<>'VOID')`, so).Scan(&invoiced); e != nil || invoiced {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT sm.id,sm.product_id,-sm.quantity,sm.source_line_id FROM stock_movements sm WHERE sm.source_id=$1 AND sm.movement_type='SALES_DELIVERY'`, id)
	if e != nil {
		return d, e
	}
	type line struct {
		movement, product, line string
		qty                     float64
	}
	var lines []line
	for rows.Next() {
		var item line
		if e = rows.Scan(&item.movement, &item.product, &item.qty, &item.line); e != nil {
			rows.Close()
			return d, e
		}
		lines = append(lines, item)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return d, e
	}
	rows.Close()
	for _, item := range lines {
		if _, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity+$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, item.qty, warehouse, item.product); e != nil {
			return d, e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'SALES_DELIVERY_REVERSAL','DELIVERY',$4,$5,$6,$7)`, warehouse, item.product, item.qty, id, item.line, item.movement, actor); e != nil {
			return d, e
		}
		if _, e = tx.Exec(ctx, `UPDATE sales_order_lines sol SET delivered_quantity=delivered_quantity-$1 FROM delivery_lines dl WHERE dl.id=$2 AND sol.id=dl.sales_order_line_id`, item.qty, item.line); e != nil {
			return d, e
		}
	}
	if _, e = tx.Exec(ctx, `UPDATE deliveries SET status='REVERSED',reason=$1,version=version+1 WHERE id=$2`, reason, id); e != nil {
		return d, e
	}
	if _, e = tx.Exec(ctx, `UPDATE sales_orders SET status=CASE WHEN EXISTS(SELECT 1 FROM sales_order_lines WHERE sales_order_id=$1 AND delivered_quantity>0) THEN 'PARTIALLY_DELIVERED' ELSE 'APPROVED' END,version=version+1 WHERE id=$1`, so); e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, "REVERSE_DELIVERY", "ACT-SAL-DLV-REVERSE", "DELIVERY", id, "POSTED", "REVERSED", reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "REVERSED"
	d.Version++
	return d, nil
}
func (s Store) CopyOrder(ctx context.Context, id, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var customer, warehouse, status string
	var mastersActive bool
	e = tx.QueryRow(ctx, `SELECT o.customer_id,o.warehouse_id,o.status,c.active AND w.active AND NOT EXISTS(
		SELECT 1 FROM sales_order_lines l JOIN products p ON p.id=l.product_id
		WHERE l.sales_order_id=o.id AND NOT p.active)
		FROM sales_orders o JOIN customers c ON c.id=o.customer_id JOIN warehouses w ON w.id=o.warehouse_id
		WHERE o.id=$1 FOR UPDATE OF o`, id).Scan(&customer, &warehouse, &status, &mastersActive)
	if e != nil || status != "REJECTED" || !mastersActive {
		return application.Document{}, ErrConflict
	}
	newID := uuid.NewString()
	no, e := number(ctx, tx, "SO")
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO sales_orders(id,document_no,customer_id,warehouse_id,creator_id,status,copied_from_id)VALUES($1,$2,$3,$4,$5,'DRAFT',$6)`, newID, no, customer, warehouse, actor, id)
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO sales_order_lines(sales_order_id,product_id,quantity,unit_price)SELECT $1,product_id,quantity,unit_price FROM sales_order_lines WHERE sales_order_id=$2`, newID, id)
	if e != nil {
		return application.Document{}, e
	}
	if e = audit(ctx, tx, actor, "COPY_REJECTED_SO", "ACT-SAL-SO-COPY", "SALES_ORDER", newID, "REJECTED", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: newID, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) CancelDelivery(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `UPDATE deliveries SET status='CANCELLED',reason=$2,version=version+1 WHERE id=$1 AND status='DRAFT' RETURNING id,document_no,status,creator_id,version`, id, reason).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "CANCEL_DELIVERY", "ACT-SAL-DLV-CANCEL", "DELIVERY", id, "DRAFT", "CANCELLED", reason, request); e != nil {
		return d, e
	}
	e = tx.Commit(ctx)
	return d, e
}
