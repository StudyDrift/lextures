-- Issue #740 — abandoned/expired Stripe Checkout Sessions stayed "pending" forever.
-- Allow a terminal 'canceled' status for payments.transactions.

ALTER TABLE payments.transactions DROP CONSTRAINT IF EXISTS transactions_status_check;
ALTER TABLE payments.transactions
    ADD CONSTRAINT transactions_status_check
    CHECK (status IN ('pending', 'completed', 'failed', 'refunded', 'canceled'));
