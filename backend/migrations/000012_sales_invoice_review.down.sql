-- Rollback requires all DRAFT sales invoices to be resolved first.
ALTER TABLE sales_invoices DROP CONSTRAINT sales_invoices_status_check;
ALTER TABLE sales_invoices ADD CONSTRAINT sales_invoices_status_check
  CHECK(status IN('POSTED','PARTIALLY_PAID','PAID','VOID'));
