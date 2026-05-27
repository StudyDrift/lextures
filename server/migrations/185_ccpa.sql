-- 10.4 CCPA/CPRA: Do Not Sell opt-out and California privacy rights requests.
-- Depends on: compliance schema (179_ferpa), "user".users (011).

ALTER TABLE "user".users
  ADD COLUMN IF NOT EXISTS ccpa_do_not_sell        BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS ccpa_limit_sensitive_pi  BOOLEAN NOT NULL DEFAULT FALSE;

-- California privacy rights requests (CPRA § 1798.100 et seq.).
-- due_at defaults to 45 days per CPRA § 1798.130(a)(2).
CREATE TABLE IF NOT EXISTS compliance.ccpa_requests (
  id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                 UUID        REFERENCES "user".users(id) ON DELETE SET NULL,
  requester_email         TEXT        NOT NULL,
  request_type            TEXT        NOT NULL CHECK (request_type IN (
                              'know_categories','know_specific','delete','correct','limit_sensitive')),
  status                  TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN (
                              'pending','verified','in_progress','completed','denied')),
  verification_token_hash TEXT,
  response_payload        TEXT,
  requested_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  due_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '45 days',
  completed_at            TIMESTAMPTZ,
  extended                BOOLEAN     NOT NULL DEFAULT FALSE,
  actioned_by             UUID        REFERENCES "user".users(id)
);

CREATE INDEX IF NOT EXISTS idx_ccpa_requests_due
  ON compliance.ccpa_requests(due_at) WHERE status != 'completed';

CREATE INDEX IF NOT EXISTS idx_ccpa_requests_user
  ON compliance.ccpa_requests(user_id, status, requested_at DESC)
  WHERE user_id IS NOT NULL;

COMMENT ON TABLE compliance.ccpa_requests IS
  'California Consumer Privacy Act rights requests (CCPA/CPRA § 1798.100 et seq.); plan 10.4.';
