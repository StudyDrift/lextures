-- 10.6 State-specific student data privacy laws: CA SOPIPA, NY Ed Law 2-d, IL SOPPA.
-- Depends on: compliance schema (179_ferpa), tenant.organizations (127), "user".users (011).

ALTER TABLE tenant.organizations
  ADD COLUMN IF NOT EXISTS state_privacy_jurisdiction TEXT
    CHECK (state_privacy_jurisdiction IN ('CA','NY','IL'));

-- Sub-processor and school-official access log required by CA SOPIPA § 49073.1(b)(7).
-- Also used for NY and IL disclosure obligations.
CREATE TABLE IF NOT EXISTS compliance.state_disclosure_events (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID        NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  student_id    UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  accessor      TEXT        NOT NULL,
  purpose       TEXT        NOT NULL,
  data_elements TEXT[]      NOT NULL DEFAULT '{}',
  occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_state_disclosure_student
  ON compliance.state_disclosure_events(student_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_state_disclosure_org
  ON compliance.state_disclosure_events(org_id, occurred_at DESC);

-- IL SOPPA parent data-deletion requests (105 ILCS 85/25); due_at defaults to 30 calendar days.
CREATE TABLE IF NOT EXISTS compliance.state_deletion_requests (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID        NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  student_id      UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  requester_id    UUID        REFERENCES "user".users(id) ON DELETE SET NULL,
  requester_email TEXT        NOT NULL,
  status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','completed','denied')),
  response_notes  TEXT,
  submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  due_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
  completed_at    TIMESTAMPTZ,
  actioned_by     UUID        REFERENCES "user".users(id)
);

CREATE INDEX IF NOT EXISTS idx_state_deletion_student
  ON compliance.state_deletion_requests(student_id, status);

CREATE INDEX IF NOT EXISTS idx_state_deletion_due
  ON compliance.state_deletion_requests(due_at) WHERE status != 'completed';

-- NY Ed Law 2-d annual parent notice batch job tracker (§ 2-d(5)(a)).
CREATE TABLE IF NOT EXISTS compliance.annual_notice_jobs (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID        NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  jurisdiction TEXT        NOT NULL,
  year         INTEGER     NOT NULL,
  sent_at      TIMESTAMPTZ,
  UNIQUE (org_id, jurisdiction, year)
);

COMMENT ON COLUMN tenant.organizations.state_privacy_jurisdiction IS
  'State-specific student data privacy jurisdiction: CA (SOPIPA), NY (Ed Law 2-d), IL (SOPPA); plan 10.6.';
COMMENT ON TABLE compliance.state_disclosure_events IS
  'Parent-accessible sub-processor and school-official data access log; CA SOPIPA § 49073.1(b)(7); plan 10.6.';
COMMENT ON TABLE compliance.state_deletion_requests IS
  'IL SOPPA parent data-deletion requests (105 ILCS 85/25); 30-day statutory deadline; plan 10.6.';
COMMENT ON TABLE compliance.annual_notice_jobs IS
  'NY Ed Law 2-d annual parent notice batch job tracker (§ 2-d(5)(a)); plan 10.6.';
