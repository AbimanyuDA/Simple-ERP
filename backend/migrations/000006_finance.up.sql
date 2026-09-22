ALTER TABLE purchase_orders DROP CONSTRAINT purchase_orders_status_check;
UPDATE purchase_orders SET status='RECEIVED' WHERE status='FULLY_RECEIVED';
UPDATE purchase_orders SET status='CLOSED' WHERE status='COMPLETED';
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check CHECK(status IN('DRAFT','ISSUED','PARTIALLY_RECEIVED','RECEIVED','CLOSED','CANCELLED'));

ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
UPDATE sales_orders SET status='DELIVERED' WHERE status='FULLY_DELIVERED';
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED','PARTIALLY_DELIVERED','DELIVERED','COMPLETED','CANCELLED'));

CREATE TABLE supplier_invoices (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE,
 supplier_invoice_number text NOT NULL, purchase_order_id uuid NOT NULL REFERENCES purchase_orders(id), supplier_id uuid NOT NULL REFERENCES suppliers(id),
 replaces_invoice_id uuid REFERENCES supplier_invoices(id), status text NOT NULL CHECK(status IN('MATCH_PENDING','MATCH_EXCEPTION','POSTED','PARTIALLY_PAID','PAID','VOID')),
 invoice_date date NOT NULL, due_date date NOT NULL CHECK(due_date>=invoice_date), currency char(3) NOT NULL DEFAULT 'IDR',
 total numeric(19,2) NOT NULL CHECK(total>0), outstanding numeric(19,2) NOT NULL CHECK(outstanding>=0 AND outstanding<=total),
 mismatch_reason text, version integer NOT NULL DEFAULT 1, void_reason text, voided_by uuid REFERENCES users(id), voided_at timestamptz,
 created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(supplier_id,supplier_invoice_number)
);
CREATE UNIQUE INDEX supplier_invoice_one_active_po ON supplier_invoices(purchase_order_id) WHERE status<>'VOID';
CREATE INDEX supplier_invoice_outstanding_idx ON supplier_invoices(status,due_date) WHERE outstanding>0;
CREATE TABLE supplier_invoice_lines (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), supplier_invoice_id uuid NOT NULL REFERENCES supplier_invoices(id) ON DELETE CASCADE,
 purchase_order_line_id uuid NOT NULL REFERENCES purchase_order_lines(id), quantity numeric(19,4) NOT NULL CHECK(quantity>0),
 unit_price numeric(19,2) NOT NULL CHECK(unit_price>=0), line_total numeric(19,2) NOT NULL CHECK(line_total>=0), UNIQUE(supplier_invoice_id,purchase_order_line_id)
);
CREATE TABLE supplier_payments (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, supplier_id uuid NOT NULL REFERENCES suppliers(id),
 status text NOT NULL CHECK(status IN('DRAFT','POSTED','VOID')), payment_date date NOT NULL, method text NOT NULL, reference text,
 currency char(3) NOT NULL DEFAULT 'IDR', amount numeric(19,2) NOT NULL CHECK(amount>0), version integer NOT NULL DEFAULT 1,
 void_reason text, voided_by uuid REFERENCES users(id), voided_at timestamptz, created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(), posted_at timestamptz
);
CREATE TABLE supplier_payment_allocations (
 payment_id uuid NOT NULL REFERENCES supplier_payments(id), invoice_id uuid NOT NULL REFERENCES supplier_invoices(id),
 amount numeric(19,2) NOT NULL CHECK(amount>0), PRIMARY KEY(payment_id,invoice_id)
);

CREATE TABLE sales_invoices (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, sales_order_id uuid NOT NULL REFERENCES sales_orders(id), customer_id uuid NOT NULL REFERENCES customers(id),
 replaces_invoice_id uuid REFERENCES sales_invoices(id), status text NOT NULL CHECK(status IN('POSTED','PARTIALLY_PAID','PAID','VOID')),
 invoice_date date NOT NULL, due_date date NOT NULL CHECK(due_date>=invoice_date), currency char(3) NOT NULL DEFAULT 'IDR',
 total numeric(19,2) NOT NULL CHECK(total>0), outstanding numeric(19,2) NOT NULL CHECK(outstanding>=0 AND outstanding<=total),
 version integer NOT NULL DEFAULT 1, void_reason text, voided_by uuid REFERENCES users(id), voided_at timestamptz,
 created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX sales_invoice_one_active_so ON sales_invoices(sales_order_id) WHERE status<>'VOID';
CREATE INDEX sales_invoice_outstanding_idx ON sales_invoices(status,due_date) WHERE outstanding>0;
CREATE TABLE sales_invoice_lines (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), sales_invoice_id uuid NOT NULL REFERENCES sales_invoices(id) ON DELETE CASCADE,
 sales_order_line_id uuid NOT NULL REFERENCES sales_order_lines(id), quantity numeric(19,4) NOT NULL CHECK(quantity>0),
 unit_price numeric(19,2) NOT NULL CHECK(unit_price>=0), line_total numeric(19,2) NOT NULL CHECK(line_total>=0), UNIQUE(sales_invoice_id,sales_order_line_id)
);
CREATE TABLE customer_payments (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, customer_id uuid NOT NULL REFERENCES customers(id),
 status text NOT NULL CHECK(status IN('DRAFT','POSTED','VOID')), payment_date date NOT NULL, method text NOT NULL, reference text,
 currency char(3) NOT NULL DEFAULT 'IDR', amount numeric(19,2) NOT NULL CHECK(amount>0), version integer NOT NULL DEFAULT 1,
 void_reason text, voided_by uuid REFERENCES users(id), voided_at timestamptz, created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(), posted_at timestamptz
);
CREATE TABLE customer_payment_allocations (
 payment_id uuid NOT NULL REFERENCES customer_payments(id), invoice_id uuid NOT NULL REFERENCES sales_invoices(id),
 amount numeric(19,2) NOT NULL CHECK(amount>0), PRIMARY KEY(payment_id,invoice_id)
);
