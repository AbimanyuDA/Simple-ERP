CREATE TABLE organization_settings (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    organization_name text NOT NULL DEFAULT 'Mini ERP UMKM',
    currency char(3) NOT NULL DEFAULT 'IDR',
    timezone text NOT NULL DEFAULT 'Asia/Jakarta',
    default_warehouse_id uuid NOT NULL REFERENCES warehouses(id),
    updated_by uuid REFERENCES users(id),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO organization_settings (default_warehouse_id)
SELECT id FROM warehouses WHERE code = 'MAIN';
