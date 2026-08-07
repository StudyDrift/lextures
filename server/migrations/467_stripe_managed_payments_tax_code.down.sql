ALTER TABLE billing.org_tax_settings
    ALTER COLUMN default_tax_category SET DEFAULT 'txcd_99999999';

-- Do not rewrite rows back to the ineligible code; leave corrected values in place.
