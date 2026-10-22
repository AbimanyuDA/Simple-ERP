ALTER TABLE sales_invoices DROP CONSTRAINT sales_invoices_status_check;
ALTER TABLE sales_invoices ADD CONSTRAINT sales_invoices_status_check
  CHECK(status IN('DRAFT','POSTED','PARTIALLY_PAID','PAID','VOID'));
