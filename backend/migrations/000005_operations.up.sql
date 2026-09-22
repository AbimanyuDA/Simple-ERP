CREATE TABLE purchase_requests (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE,
 creator_id uuid NOT NULL REFERENCES users(id), reviewer_id uuid REFERENCES users(id),
 status text NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED','CONVERTED_TO_PO','COMPLETED','CANCELLED')),
 reason text, version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE purchase_request_lines (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), purchase_request_id uuid NOT NULL REFERENCES purchase_requests(id) ON DELETE CASCADE,
 product_id uuid NOT NULL REFERENCES products(id), quantity numeric(19,4) NOT NULL CHECK(quantity>0), estimated_price numeric(19,2) NOT NULL CHECK(estimated_price>=0), UNIQUE(purchase_request_id,product_id)
);
CREATE TABLE purchase_orders (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, purchase_request_id uuid NOT NULL UNIQUE REFERENCES purchase_requests(id),
 supplier_id uuid NOT NULL REFERENCES suppliers(id), warehouse_id uuid NOT NULL REFERENCES warehouses(id), creator_id uuid NOT NULL REFERENCES users(id),
 status text NOT NULL CHECK(status IN('DRAFT','ISSUED','PARTIALLY_RECEIVED','FULLY_RECEIVED','COMPLETED','CANCELLED')),
 reason text, version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE purchase_order_lines (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), purchase_order_id uuid NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
 product_id uuid NOT NULL REFERENCES products(id), quantity numeric(19,4) NOT NULL CHECK(quantity>0), unit_price numeric(19,2) NOT NULL CHECK(unit_price>=0), received_quantity numeric(19,4) NOT NULL DEFAULT 0 CHECK(received_quantity>=0 AND received_quantity<=quantity), UNIQUE(purchase_order_id,product_id)
);
CREATE TABLE goods_receipts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, purchase_order_id uuid NOT NULL REFERENCES purchase_orders(id), creator_id uuid NOT NULL REFERENCES users(id),
 status text NOT NULL CHECK(status IN('DRAFT','POSTED','CANCELLED','REVERSED')), reason text, version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), posted_at timestamptz
);
CREATE TABLE goods_receipt_lines (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),goods_receipt_id uuid NOT NULL REFERENCES goods_receipts(id) ON DELETE CASCADE,purchase_order_line_id uuid NOT NULL REFERENCES purchase_order_lines(id),product_id uuid NOT NULL REFERENCES products(id),quantity numeric(19,4) NOT NULL CHECK(quantity>0));

CREATE TABLE sales_orders (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_no text NOT NULL UNIQUE, customer_id uuid NOT NULL REFERENCES customers(id), warehouse_id uuid NOT NULL REFERENCES warehouses(id),
 creator_id uuid NOT NULL REFERENCES users(id), reviewer_id uuid REFERENCES users(id), status text NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED','PARTIALLY_DELIVERED','FULLY_DELIVERED','COMPLETED','CANCELLED')),
 reason text,version integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sales_order_lines (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),sales_order_id uuid NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,product_id uuid NOT NULL REFERENCES products(id),quantity numeric(19,4) NOT NULL CHECK(quantity>0),unit_price numeric(19,2) NOT NULL CHECK(unit_price>=0),delivered_quantity numeric(19,4) NOT NULL DEFAULT 0 CHECK(delivered_quantity>=0 AND delivered_quantity<=quantity),UNIQUE(sales_order_id,product_id));
CREATE TABLE deliveries (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),document_no text NOT NULL UNIQUE,sales_order_id uuid NOT NULL REFERENCES sales_orders(id),creator_id uuid NOT NULL REFERENCES users(id),status text NOT NULL CHECK(status IN('DRAFT','POSTED','CANCELLED','REVERSED')),reason text,version integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),posted_at timestamptz);
CREATE TABLE delivery_lines (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),delivery_id uuid NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,sales_order_line_id uuid NOT NULL REFERENCES sales_order_lines(id),product_id uuid NOT NULL REFERENCES products(id),quantity numeric(19,4) NOT NULL CHECK(quantity>0));

CREATE TABLE inventory_adjustments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),document_no text NOT NULL UNIQUE,warehouse_id uuid NOT NULL REFERENCES warehouses(id),creator_id uuid NOT NULL REFERENCES users(id),reviewer_id uuid REFERENCES users(id),status text NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED','COMPLETED','CANCELLED','REVERSED')),reason text,version integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE inventory_adjustment_lines (id uuid PRIMARY KEY DEFAULT gen_random_uuid(),inventory_adjustment_id uuid NOT NULL REFERENCES inventory_adjustments(id) ON DELETE CASCADE,product_id uuid NOT NULL REFERENCES products(id),snapshot_quantity numeric(19,4) NOT NULL,counted_quantity numeric(19,4) NOT NULL,UNIQUE(inventory_adjustment_id,product_id));

CREATE TABLE stock_movements (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),warehouse_id uuid NOT NULL REFERENCES warehouses(id),product_id uuid NOT NULL REFERENCES products(id),
 quantity numeric(19,4) NOT NULL CHECK(quantity<>0),movement_type text NOT NULL CHECK(movement_type IN('GOODS_RECEIPT','SALES_DELIVERY','STOCK_ADJUSTMENT','GOODS_RECEIPT_REVERSAL','SALES_DELIVERY_REVERSAL','STOCK_ADJUSTMENT_REVERSAL')),
 source_type text NOT NULL,source_id uuid NOT NULL,source_line_id uuid NOT NULL,reversal_of uuid REFERENCES stock_movements(id),created_by uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(source_type,source_id,source_line_id,movement_type)
);
CREATE INDEX stock_movements_ledger_idx ON stock_movements(warehouse_id,product_id,created_at,id);
CREATE TABLE stock_balances (warehouse_id uuid NOT NULL REFERENCES warehouses(id),product_id uuid NOT NULL REFERENCES products(id),quantity numeric(19,4) NOT NULL DEFAULT 0 CHECK(quantity>=0),updated_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(warehouse_id,product_id));
CREATE FUNCTION reject_stock_movement_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'stock movements are immutable'; END $$;
CREATE TRIGGER stock_movements_no_update BEFORE UPDATE OR DELETE ON stock_movements FOR EACH ROW EXECUTE FUNCTION reject_stock_movement_mutation();
