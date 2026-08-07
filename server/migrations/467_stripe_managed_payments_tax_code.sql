-- Managed Payments rejects the legacy catch-all tax code txcd_99999999.
-- Default digital LMS sales to SaaS personal (txcd_10103000), which is eligible.

ALTER TABLE billing.org_tax_settings
    ALTER COLUMN default_tax_category SET DEFAULT 'txcd_10103000';

UPDATE billing.org_tax_settings
SET default_tax_category = 'txcd_10103000'
WHERE default_tax_category IS NULL
   OR btrim(default_tax_category) = ''
   OR default_tax_category = 'txcd_99999999';
