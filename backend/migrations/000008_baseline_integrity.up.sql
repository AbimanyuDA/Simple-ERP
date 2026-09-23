ALTER TABLE purchase_requests
    ADD COLUMN copied_from_id uuid REFERENCES purchase_requests(id);

ALTER TABLE sales_orders
    ADD COLUMN copied_from_id uuid REFERENCES sales_orders(id),
    ADD COLUMN confirmed_at timestamptz;

CREATE INDEX purchase_requests_copied_from_idx ON purchase_requests(copied_from_id);
CREATE INDEX sales_orders_copied_from_idx ON sales_orders(copied_from_id);
