-- Rollback requires no PR to have multiple historical POs.
DROP INDEX purchase_orders_one_active_per_request_idx;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_purchase_request_id_key UNIQUE (purchase_request_id);
