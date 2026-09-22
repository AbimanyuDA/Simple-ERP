DROP TABLE IF EXISTS customer_payment_allocations;
DROP TABLE IF EXISTS customer_payments;
DROP TABLE IF EXISTS sales_invoice_lines;
DROP TABLE IF EXISTS sales_invoices;
DROP TABLE IF EXISTS supplier_payment_allocations;
DROP TABLE IF EXISTS supplier_payments;
DROP TABLE IF EXISTS supplier_invoice_lines;
DROP TABLE IF EXISTS supplier_invoices;

ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
UPDATE sales_orders SET status='FULLY_DELIVERED' WHERE status='DELIVERED';
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED','PARTIALLY_DELIVERED','FULLY_DELIVERED','COMPLETED','CANCELLED'));

ALTER TABLE purchase_orders DROP CONSTRAINT purchase_orders_status_check;
UPDATE purchase_orders SET status='FULLY_RECEIVED' WHERE status='RECEIVED';
UPDATE purchase_orders SET status='COMPLETED' WHERE status='CLOSED';
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check CHECK(status IN('DRAFT','ISSUED','PARTIALLY_RECEIVED','FULLY_RECEIVED','COMPLETED','CANCELLED'));
