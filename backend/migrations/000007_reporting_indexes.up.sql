CREATE INDEX purchase_orders_status_created_idx ON purchase_orders(status,created_at DESC);
CREATE INDEX sales_orders_status_created_idx ON sales_orders(status,created_at DESC);
CREATE INDEX purchase_requests_status_created_idx ON purchase_requests(status,created_at DESC);
CREATE INDEX inventory_adjustments_status_created_idx ON inventory_adjustments(status,created_at DESC);
CREATE INDEX audit_trace_occurred_idx ON audit_entries(trace_id,occurred_at DESC);
CREATE INDEX audit_action_occurred_idx ON audit_entries(action,occurred_at DESC);
CREATE FUNCTION reject_audit_entry_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit entries are immutable'; END $$;
CREATE TRIGGER audit_entries_no_update BEFORE UPDATE OR DELETE ON audit_entries FOR EACH ROW EXECUTE FUNCTION reject_audit_entry_mutation();
