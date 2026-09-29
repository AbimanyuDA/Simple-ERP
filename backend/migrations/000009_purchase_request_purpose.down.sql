ALTER TABLE purchase_requests DROP CONSTRAINT IF EXISTS purchase_requests_purpose_not_blank;
ALTER TABLE purchase_requests DROP COLUMN IF EXISTS purpose;
