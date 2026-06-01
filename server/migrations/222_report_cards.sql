-- Plan 13.4: Report cards (district-formatted, comment banks, narrative, AI-assisted comments).

CREATE SCHEMA IF NOT EXISTS report_card;

CREATE TABLE IF NOT EXISTS report_card.comment_bank (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    category   TEXT        NOT NULL,
    text       TEXT        NOT NULL,
    active     BOOLEAN     NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_comment_bank_org
    ON report_card.comment_bank (org_id, active);

COMMENT ON TABLE report_card.comment_bank IS
    'Plan 13.4: Admin-curated comment bank phrases per org, grouped by category. Used for report card comment authoring.';

CREATE TABLE IF NOT EXISTS report_card.report_cards (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id        UUID        NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    grading_period   TEXT        NOT NULL,
    final_grade_pct  NUMERIC(5,2),
    letter_grade     TEXT,
    comment          TEXT,
    status           TEXT        NOT NULL DEFAULT 'draft'
                                 CHECK (status IN ('draft', 'submitted', 'approved', 'released')),
    pdf_url          TEXT,
    generated_at     TIMESTAMPTZ,
    released_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, course_id, grading_period)
);

CREATE INDEX IF NOT EXISTS idx_report_cards_course_period
    ON report_card.report_cards (course_id, grading_period);
CREATE INDEX IF NOT EXISTS idx_report_cards_student
    ON report_card.report_cards (student_id);

COMMENT ON TABLE report_card.report_cards IS
    'Plan 13.4: Per-student per-course per-period report cards. FERPA-protected education records.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_report_cards BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_report_cards IS
    'Plan 13.4: Enables district-formatted report cards with comment banks and PDF generation for K-12.';
