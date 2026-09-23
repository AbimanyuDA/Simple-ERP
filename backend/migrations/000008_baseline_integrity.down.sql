DROP INDEX IF EXISTS sales_orders_copied_from_idx;
DROP INDEX IF EXISTS purchase_requests_copied_from_idx;

ALTER TABLE sales_orders
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS copied_from_id;

ALTER TABLE purchase_requests
    DROP COLUMN IF EXISTS copied_from_id;
