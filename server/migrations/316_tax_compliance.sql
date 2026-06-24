-- Plan 15.13 — Tax compliance (Sales Tax / VAT / GST via Stripe Tax).

ALTER TABLE billing.user_entitlements
    ADD COLUMN IF NOT EXISTS subtotal_cents INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_amount_cents INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_rate NUMERIC(8, 6),
    ADD COLUMN IF NOT EXISTS tax_jurisdiction TEXT,
    ADD COLUMN IF NOT EXISTS tax_type TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS tax_inclusive BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS customer_country TEXT,
    ADD COLUMN IF NOT EXISTS customer_region TEXT,
    ADD COLUMN IF NOT EXISTS customer_tax_id_enc TEXT,
    ADD COLUMN IF NOT EXISTS reverse_charge BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS stripe_tax_calculation_id TEXT,
    ADD COLUMN IF NOT EXISTS invoice_id UUID;

COMMENT ON COLUMN billing.user_entitlements.subtotal_cents IS
    'Pre-tax line amount in smallest currency unit (plan 15.13).';
COMMENT ON COLUMN billing.user_entitlements.tax_amount_cents IS
    'Tax collected in smallest currency unit (plan 15.13).';
COMMENT ON COLUMN billing.user_entitlements.tax_jurisdiction IS
    'ISO country or state/province code for tax jurisdiction (plan 15.13).';
COMMENT ON COLUMN billing.user_entitlements.tax_type IS
    'Tax type: vat, gst, sales_tax, or none (plan 15.13).';
COMMENT ON COLUMN billing.user_entitlements.customer_tax_id_enc IS
    'Encrypted customer VAT/GST ID at rest (plan 15.13).';

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_tax_type_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_tax_type_check
    CHECK (tax_type IN ('vat', 'gst', 'sales_tax', 'none'));

CREATE TABLE IF NOT EXISTS billing.org_tax_settings (
    org_id                      UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    enabled                     BOOLEAN NOT NULL DEFAULT FALSE,
    registered_jurisdictions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    default_tax_category        TEXT NOT NULL DEFAULT 'txcd_99999999',
    price_display               TEXT NOT NULL DEFAULT 'exclusive',
    filing_mode                 TEXT NOT NULL DEFAULT 'manual',
    record_retention_years      INT NOT NULL DEFAULT 7,
    seller_name                 TEXT,
    seller_address              TEXT,
    seller_tax_id               TEXT,
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (price_display IN ('inclusive', 'exclusive')),
    CHECK (record_retention_years >= 1)
);

COMMENT ON TABLE billing.org_tax_settings IS
    'Per-org Stripe Tax configuration and seller details for invoices (plan 15.13).';

CREATE TABLE IF NOT EXISTS billing.tax_invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entitlement_id  UUID NOT NULL REFERENCES billing.user_entitlements (id) ON DELETE CASCADE,
    invoice_number  TEXT NOT NULL UNIQUE,
    pdf_storage_key TEXT,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    credited_by     UUID REFERENCES billing.tax_invoices (id)
);

COMMENT ON TABLE billing.tax_invoices IS
    'Tax-compliant invoice records linked to entitlements (plan 15.13).';

CREATE INDEX IF NOT EXISTS idx_billing_entitlements_tax_report
    ON billing.user_entitlements (tax_jurisdiction, created_at)
    WHERE tax_type <> 'none';

CREATE INDEX IF NOT EXISTS idx_billing_tax_invoices_entitlement
    ON billing.tax_invoices (entitlement_id);

-- Historical transactions: no retroactive tax.
UPDATE billing.user_entitlements
SET tax_type = 'none'
WHERE tax_type IS NULL OR tax_type = '';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_tax_collection BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_tax_collection IS
    'Enables Stripe Tax calculation, collection, and reporting (plan 15.13).';