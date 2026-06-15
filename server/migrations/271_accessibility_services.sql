-- Accessibility services intake (plan 14.16): accessibility-office accommodation profiles
-- that propagate to the 2.11 student_accommodations override engine. Disability documentation
-- itself is NOT stored here (it lives in the SIS/disability-office system); only operational
-- accommodation settings (ADA Title II / Section 504 / FERPA need-to-know).

CREATE SCHEMA IF NOT EXISTS accessibility;

CREATE TYPE accessibility.accommodation_type AS ENUM (
    'extended_time_1_5x',
    'extended_time_2x',
    'separate_testing',
    'alternate_format',
    'screen_reader',
    'speech_to_text',
    'reduced_distraction',
    'other'
);

CREATE TABLE accessibility.accommodation_profiles (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    student_id               UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id                   UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    accommodations           accessibility.accommodation_type[] NOT NULL,
    custom_params            JSONB NOT NULL DEFAULT '{}'::jsonb, -- {"timeMultiplier": 1.5, "alternateFormat": "braille"}
    effective_from           DATE NOT NULL DEFAULT CURRENT_DATE,
    effective_until          DATE,
    -- Link to the propagated course.student_accommodations override row (2.11). NULL until applied.
    applied_accommodation_id UUID REFERENCES course.student_accommodations (id) ON DELETE SET NULL,
    notified_at              TIMESTAMPTZ,
    created_by               UUID NOT NULL REFERENCES "user".users (id) ON DELETE RESTRICT,
    is_active                BOOLEAN NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FR-5/AC-3: fast lookup of a student's active profiles.
CREATE INDEX idx_accommodation_profiles_student ON accessibility.accommodation_profiles (student_id)
WHERE
    is_active = TRUE;

CREATE INDEX idx_accommodation_profiles_org ON accessibility.accommodation_profiles (org_id, is_active);

COMMENT ON TABLE accessibility.accommodation_profiles IS
    'Accessibility-office accommodation profiles (plan 14.16). Disability documentation lives in the SIS, not here; only operational settings are stored (ADA/Section 504/FERPA need-to-know).';

-- Platform feature flag (default off; managed in Settings -> Global platform).
ALTER TABLE settings.platform_app_settings
ADD COLUMN IF NOT EXISTS ff_accessibility_intake BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_accessibility_intake IS
    'Enables accessibility services intake: coordinator accommodation profiles propagated to assessments (plan 14.16).';

-- Coordinator capability reuses the existing 2.11 permission (global:user:accommodations:manage),
-- already granted to the "Accessibility Coordinator" and "Global Admin" roles in migration 077.
