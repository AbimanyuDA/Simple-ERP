package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/simple-erp/backend/internal/reporting/application"
)

type Store struct{ Pool *pgxpool.Pool }

func (s Store) Summary(ctx context.Context) (application.Summary, error) {
	var summary application.Summary
	err := s.Pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM purchase_requests WHERE status='SUBMITTED')+(SELECT count(*) FROM sales_orders WHERE status='SUBMITTED')+(SELECT count(*) FROM inventory_adjustments WHERE status='SUBMITTED'),
		(SELECT count(*) FROM purchase_orders WHERE status NOT IN('CLOSED','CANCELLED')),
		(SELECT count(*) FROM sales_orders WHERE status NOT IN('COMPLETED','CANCELLED','REJECTED')),
		(SELECT count(*) FROM products p CROSS JOIN organization_settings o LEFT JOIN stock_balances b ON b.product_id=p.id AND b.warehouse_id=o.default_warehouse_id WHERE p.active AND COALESCE(b.quantity,0)=0),
		(SELECT COALESCE(sum(outstanding),0) FROM supplier_invoices WHERE status IN('POSTED','PARTIALLY_PAID')),
		(SELECT COALESCE(sum(outstanding),0) FROM sales_invoices WHERE status IN('POSTED','PARTIALLY_PAID')),
		(SELECT COALESCE(sum(outstanding),0) FROM supplier_invoices WHERE status IN('POSTED','PARTIALLY_PAID') AND due_date<CURRENT_DATE),
		(SELECT COALESCE(sum(outstanding),0) FROM sales_invoices WHERE status IN('POSTED','PARTIALLY_PAID') AND due_date<CURRENT_DATE)`).Scan(
		&summary.PendingApprovals, &summary.ActivePurchases, &summary.ActiveSales, &summary.ZeroStockItems,
		&summary.Payables, &summary.Receivables, &summary.OverduePayables, &summary.OverdueReceivables)
	return summary, err
}

func reportRows(rows pgx.Rows) ([]application.ReportRow, error) {
	defer rows.Close()
	result := []application.ReportRow{}
	for rows.Next() {
		var row application.ReportRow
		if err := rows.Scan(&row.DocumentNo, &row.Party, &row.Status, &row.Amount, &row.Outstanding, &row.DueDate, &row.DaysOverdue); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s Store) Purchases(ctx context.Context) ([]application.ReportRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT p.document_no,s.name,p.status,sum(l.quantity*l.unit_price),0::numeric,''::text,0 FROM purchase_orders p JOIN suppliers s ON s.id=p.supplier_id JOIN purchase_order_lines l ON l.purchase_order_id=p.id GROUP BY p.id,s.name ORDER BY p.created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	return reportRows(rows)
}
func (s Store) Sales(ctx context.Context) ([]application.ReportRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT o.document_no,c.name,o.status,sum(l.quantity*l.unit_price),0::numeric,''::text,0 FROM sales_orders o JOIN customers c ON c.id=o.customer_id JOIN sales_order_lines l ON l.sales_order_id=o.id GROUP BY o.id,c.name ORDER BY o.created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	return reportRows(rows)
}
func (s Store) Payables(ctx context.Context) ([]application.ReportRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT i.document_no,s.name,i.status,i.total,i.outstanding,i.due_date::text,GREATEST(CURRENT_DATE-i.due_date,0) FROM supplier_invoices i JOIN suppliers s ON s.id=i.supplier_id WHERE i.status IN('POSTED','PARTIALLY_PAID') ORDER BY i.due_date LIMIT 100`)
	if err != nil {
		return nil, err
	}
	return reportRows(rows)
}
func (s Store) Receivables(ctx context.Context) ([]application.ReportRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT i.document_no,c.name,i.status,i.total,i.outstanding,i.due_date::text,GREATEST(CURRENT_DATE-i.due_date,0) FROM sales_invoices i JOIN customers c ON c.id=i.customer_id WHERE i.status IN('POSTED','PARTIALLY_PAID') ORDER BY i.due_date LIMIT 100`)
	if err != nil {
		return nil, err
	}
	return reportRows(rows)
}
func (s Store) Inventory(ctx context.Context) ([]application.InventoryRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT w.name,p.sku,p.name,COALESCE(b.quantity,0) FROM products p CROSS JOIN warehouses w LEFT JOIN stock_balances b ON b.product_id=p.id AND b.warehouse_id=w.id WHERE p.active AND w.active ORDER BY w.name,p.name LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []application.InventoryRow{}
	for rows.Next() {
		var row application.InventoryRow
		if err = rows.Scan(&row.Warehouse, &row.SKU, &row.Product, &row.Quantity); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
func (s Store) Audit(ctx context.Context, entity, actor, action string) ([]application.AuditRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT a.id,a.occurred_at::text,COALESCE(u.email,'SYSTEM'),a.action,a.trace_id,a.entity_type,a.entity_id,COALESCE(a.previous_state,''),COALESCE(a.new_state,''),COALESCE(a.reason,''),a.request_id
		FROM audit_entries a LEFT JOIN users u ON u.id=a.actor_id WHERE ($1='' OR a.entity_type=$1) AND ($2='' OR a.actor_id::text=$2) AND ($3='' OR a.action=$3) ORDER BY a.occurred_at DESC,a.id DESC LIMIT 200`, entity, actor, action)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []application.AuditRow{}
	for rows.Next() {
		var row application.AuditRow
		if err = rows.Scan(&row.ID, &row.OccurredAt, &row.Actor, &row.Action, &row.TraceID, &row.EntityType, &row.EntityID, &row.PreviousState, &row.NewState, &row.Reason, &row.RequestID); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
