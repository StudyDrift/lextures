-- BUG-MKT2-01: Correct JPY (zero-decimal) course prices stored under the ×100 assumption.
-- price_cents stores Stripe smallest-unit amounts; JPY has no subunit.

UPDATE course.courses
SET price_cents = price_cents / 100,
    updated_at = NOW()
WHERE LOWER(price_currency) = 'jpy'
  AND price_cents > 0
  AND price_cents % 100 = 0;

-- Flag completed JPY purchases that were likely overcharged for ops reconciliation.
CREATE TABLE IF NOT EXISTS billing.zero_decimal_reconciliation_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entitlement_id UUID NOT NULL REFERENCES billing.user_entitlements(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    recorded_amount_cents INT NOT NULL,
    expected_amount_cents INT NOT NULL,
    excess_cents INT NOT NULL,
    flagged_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO billing.zero_decimal_reconciliation_queue (
    entitlement_id, currency, recorded_amount_cents, expected_amount_cents, excess_cents
)
SELECT
    e.id,
    e.currency,
    e.amount_paid_cents,
    e.amount_paid_cents / 100,
    e.amount_paid_cents - (e.amount_paid_cents / 100)
FROM billing.user_entitlements e
WHERE LOWER(e.currency) = 'jpy'
  AND e.amount_paid_cents > 0
  AND e.amount_paid_cents % 100 = 0
  AND e.entitlement_type = 'course_purchase'
  AND e.status = 'active'
  AND NOT EXISTS (
      SELECT 1 FROM billing.zero_decimal_reconciliation_queue q
      WHERE q.entitlement_id = e.id
  );

COMMENT ON TABLE billing.zero_decimal_reconciliation_queue IS
    'JPY overcharge reconciliation queue (BUG-MKT2-01). Review and refund excess_cents.';