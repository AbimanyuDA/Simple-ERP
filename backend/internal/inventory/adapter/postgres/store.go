package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/inventory/application"
	"github.com/simple-erp/backend/internal/platform/authorization"
	"time"
)

var ErrConflict = errors.New("invalid state, version, stock snapshot, or maker-checker conflict")

type Store struct{ Pool *pgxpool.Pool }

func number(ctx context.Context, tx pgx.Tx) (string, error) {
	var n int64
	e := tx.QueryRow(ctx, `SELECT nextval('document_number_sequence')`).Scan(&n)
	return fmt.Sprintf("ADJ-%s-%06d", time.Now().Format("20060102"), n), e
}
func audit(ctx context.Context, tx pgx.Tx, actor, action, trace, id, from, to, reason, request string) error {
	_, e := tx.Exec(ctx, `INSERT INTO audit_entries(actor_id,action,trace_id,entity_type,entity_id,previous_state,new_state,reason,request_id)VALUES($1,$2,$3,'INVENTORY_ADJUSTMENT',$4,$5,$6,NULLIF($7,''),$8)`, actor, action, trace, id, from, to, reason, request)
	return e
}
func (s Store) Balances(ctx context.Context) ([]application.Balance, error) {
	rows, e := s.Pool.Query(ctx, `SELECT warehouse_id,product_id,quantity FROM stock_balances ORDER BY warehouse_id,product_id`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []application.Balance{}
	for rows.Next() {
		var v application.Balance
		if e = rows.Scan(&v.WarehouseID, &v.ProductID, &v.Quantity); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Store) Movements(ctx context.Context) ([]application.Movement, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,warehouse_id,product_id,quantity,movement_type,source_type,source_id FROM stock_movements ORDER BY created_at DESC,id DESC LIMIT 500`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []application.Movement{}
	for rows.Next() {
		var v application.Movement
		if e = rows.Scan(&v.ID, &v.WarehouseID, &v.ProductID, &v.Quantity, &v.Type, &v.SourceType, &v.SourceID); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Store) ListAdjustments(ctx context.Context) ([]application.Document, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM inventory_adjustments ORDER BY created_at DESC`)
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
func (s Store) CreateAdjustment(ctx context.Context, warehouse, actor string, lines []application.AdjustmentLine, request string) (application.Document, error) {
	if len(lines) == 0 {
		return application.Document{}, ErrConflict
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id=$1 AND active)`, warehouse).Scan(&active)
	if e != nil || !active {
		return application.Document{}, ErrConflict
	}
	id := uuid.NewString()
	no, e := number(ctx, tx)
	if e != nil {
		return application.Document{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO inventory_adjustments(id,document_no,warehouse_id,creator_id,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, warehouse, actor)
	if e != nil {
		return application.Document{}, e
	}
	for _, l := range lines {
		if l.CountedQuantity < 0 {
			return application.Document{}, ErrConflict
		}
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&active)
		if e != nil || !active {
			return application.Document{}, ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO inventory_adjustment_lines(inventory_adjustment_id,product_id,snapshot_quantity,counted_quantity)VALUES($1,$2,COALESCE((SELECT quantity FROM stock_balances WHERE warehouse_id=$3 AND product_id=$2),0),$4)`, id, l.ProductID, warehouse, l.CountedQuantity)
		if e != nil {
			return application.Document{}, e
		}
	}
	if e = audit(ctx, tx, actor, "CREATE_ADJUSTMENT", "INV-ADJ-001", id, "", "DRAFT", "", request); e != nil {
		return application.Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return application.Document{}, e
	}
	return application.Document{ID: id, DocumentNo: no, Status: "DRAFT", CreatorID: actor, Version: 1}, nil
}
func (s Store) TransitionAdjustment(ctx context.Context, id, action, actor, reason string, version int, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version FROM inventory_adjustments WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version)
	if e != nil || d.Version != version {
		return d, ErrConflict
	}
	from := d.Status
	trace := "INV-ADJ-002"
	switch action {
	case "submit":
		if from != "DRAFT" {
			return d, ErrConflict
		}
		d.Status = "SUBMITTED"
	case "approve":
		trace = "INV-ADJ-003"
		if from != "SUBMITTED" {
			return d, ErrConflict
		}
		if actor == d.CreatorID {
			allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)
			if roleErr != nil || !allowed {
				return d, ErrConflict
			}
		}
		var snapshotCurrent bool
		e = tx.QueryRow(ctx, `SELECT COALESCE(bool_and(l.snapshot_quantity=COALESCE(b.quantity,0)),false)
			FROM inventory_adjustment_lines l
			JOIN inventory_adjustments a ON a.id=l.inventory_adjustment_id
			LEFT JOIN stock_balances b ON b.warehouse_id=a.warehouse_id AND b.product_id=l.product_id
			WHERE l.inventory_adjustment_id=$1`, id).Scan(&snapshotCurrent)
		if e != nil || !snapshotCurrent {
			return d, ErrConflict
		}
		d.Status = "APPROVED"
	case "reject":
		trace = "INV-ADJ-003"
		if from != "SUBMITTED" || reason == "" {
			return d, ErrConflict
		}
		if actor == d.CreatorID {
			allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)
			if roleErr != nil || !allowed {
				return d, ErrConflict
			}
		}
		d.Status = "REJECTED"
	case "cancel":
		trace = "ACT-INV-ADJ-CANCEL"
		var admin bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&admin)
		if e != nil || (from != "DRAFT" && from != "APPROVED") || reason == "" || (actor != d.CreatorID && !admin) {
			return d, ErrConflict
		}
		d.Status = "CANCELLED"
	default:
		return d, ErrConflict
	}
	tag, e := tx.Exec(ctx, `UPDATE inventory_adjustments SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject')THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version)
	if e != nil || tag.RowsAffected() != 1 {
		return d, ErrConflict
	}
	if e = audit(ctx, tx, actor, "ADJUSTMENT_"+action, trace, id, from, d.Status, reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Version++
	return d, nil
}
func (s Store) PostAdjustment(ctx context.Context, id string, version int, actor, request string) (application.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var warehouse string
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version,warehouse_id FROM inventory_adjustments WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &warehouse)
	if e != nil || d.Status != "APPROVED" || d.Version != version {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT id,product_id,snapshot_quantity,counted_quantity FROM inventory_adjustment_lines WHERE inventory_adjustment_id=$1`, id)
	if e != nil {
		return d, e
	}
	type line struct {
		id, product       string
		snapshot, counted float64
	}
	ls := []line{}
	for rows.Next() {
		var l line
		if e = rows.Scan(&l.id, &l.product, &l.snapshot, &l.counted); e != nil {
			rows.Close()
			return d, e
		}
		ls = append(ls, l)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return d, e
	}
	rows.Close()
	for _, l := range ls {
		var current float64
		if _, e = tx.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity) VALUES($1,$2,0) ON CONFLICT DO NOTHING`, warehouse, l.product); e != nil {
			return d, e
		}
		e = tx.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2 FOR UPDATE`, warehouse, l.product).Scan(&current)
		if e != nil || current != l.snapshot {
			return d, ErrConflict
		}
		delta := l.counted - l.snapshot
		if delta == 0 {
			continue
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,$3)ON CONFLICT(warehouse_id,product_id)DO UPDATE SET quantity=$3,updated_at=now()`, warehouse, l.product, l.counted)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'STOCK_ADJUSTMENT','INVENTORY_ADJUSTMENT',$4,$5,$6)`, warehouse, l.product, delta, id, l.id, actor)
		if e != nil {
			return d, e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE inventory_adjustments SET status='COMPLETED',version=version+1,updated_at=now() WHERE id=$1`, id)
	if e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, `POST_ADJUSTMENT_MOVEMENT`, `INV-ADJ-004`, id, `APPROVED`, `APPROVED`, ``, request); e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, `COMPLETE_ADJUSTMENT`, `INV-ADJ-005`, id, `APPROVED`, `COMPLETED`, ``, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "COMPLETED"
	d.Version++
	return d, nil
}
func (s Store) ReverseAdjustment(ctx context.Context, id, reason, actor, request string) (application.Document, error) {
	if reason == "" {
		return application.Document{}, errors.New("reason required")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return application.Document{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var d application.Document
	var warehouse string
	e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version,warehouse_id FROM inventory_adjustments WHERE id=$1 FOR UPDATE`, id).Scan(&d.ID, &d.DocumentNo, &d.Status, &d.CreatorID, &d.Version, &warehouse)
	if e != nil || d.Status != "COMPLETED" {
		return d, ErrConflict
	}
	rows, e := tx.Query(ctx, `SELECT id,product_id,quantity,source_line_id FROM stock_movements WHERE source_id=$1 AND movement_type='STOCK_ADJUSTMENT'`, id)
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
	if e = rows.Err(); e != nil {
		rows.Close()
		return d, e
	}
	rows.Close()
	for _, l := range ls {
		var balance float64
		e = tx.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE warehouse_id=$1 AND product_id=$2 FOR UPDATE`, warehouse, l.product).Scan(&balance)
		if e != nil || balance-l.qty < 0 {
			return d, ErrConflict
		}
		_, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, l.qty, warehouse, l.product)
		if e != nil {
			return d, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'STOCK_ADJUSTMENT_REVERSAL','INVENTORY_ADJUSTMENT',$4,$5,$6,$7)`, warehouse, l.product, -l.qty, id, l.line, l.movement, actor)
		if e != nil {
			return d, e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE inventory_adjustments SET status='REVERSED',reason=$1,version=version+1,updated_at=now() WHERE id=$2`, reason, id)
	if e != nil {
		return d, e
	}
	if e = audit(ctx, tx, actor, "REVERSE_ADJUSTMENT", "ACT-INV-ADJ-REVERSE", id, "COMPLETED", "REVERSED", reason, request); e != nil {
		return d, e
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.Status = "REVERSED"
	d.Version++
	return d, nil
}
