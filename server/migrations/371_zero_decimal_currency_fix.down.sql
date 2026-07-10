DROP TABLE IF EXISTS billing.zero_decimal_reconciliation_queue;

-- Listing price correction is not safely reversible without a snapshot.