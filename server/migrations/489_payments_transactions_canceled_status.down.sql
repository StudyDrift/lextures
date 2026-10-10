-- Companion to: 489_payments_transactions_canceled_status.sql
UPDATE payments.transactions SET status = 'failed' WHERE status = 'canceled';
ALTER TABLE payments.transactions DROP CONSTRAINT IF EXISTS transactions_status_check;
ALTER TABLE payments.transactions
    ADD CONSTRAINT transactions_status_check
    CHECK (status IN ('pending', 'completed', 'failed', 'refunded'));
