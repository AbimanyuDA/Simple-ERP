-- Retain cancelled POs for audit while allowing one active replacement per PR.
ALTER TABLE purchase_orders DROP CONSTRAINT purchase_orders_purchase_request_id_key;
CREATE UNIQUE INDEX purchase_orders_one_active_per_request_idx
    ON purchase_orders(purchase_request_id) WHERE status <> 'CANCELLED';
