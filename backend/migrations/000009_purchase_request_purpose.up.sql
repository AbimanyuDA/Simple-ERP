ALTER TABLE purchase_requests ADD COLUMN purpose text;
UPDATE purchase_requests SET purpose='Migrated request ' || document_no WHERE purpose IS NULL;
ALTER TABLE purchase_requests ALTER COLUMN purpose SET NOT NULL;
ALTER TABLE purchase_requests ADD CONSTRAINT purchase_requests_purpose_not_blank CHECK (btrim(purpose)<>'');
