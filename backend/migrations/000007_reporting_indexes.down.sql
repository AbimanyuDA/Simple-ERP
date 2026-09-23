DROP TRIGGER IF EXISTS audit_entries_no_update ON audit_entries;
DROP FUNCTION IF EXISTS reject_audit_entry_mutation();
DROP INDEX IF EXISTS audit_action_occurred_idx;
DROP INDEX IF EXISTS audit_trace_occurred_idx;
DROP INDEX IF EXISTS inventory_adjustments_status_created_idx;
DROP INDEX IF EXISTS purchase_requests_status_created_idx;
DROP INDEX IF EXISTS sales_orders_status_created_idx;
DROP INDEX IF EXISTS purchase_orders_status_created_idx;
