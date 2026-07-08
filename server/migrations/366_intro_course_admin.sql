-- IC08: singleton status row for intro course admin operations (sync/provision health).

CREATE TABLE IF NOT EXISTS settings.intro_course_status (
    id               BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    content_version  INTEGER NOT NULL DEFAULT 0,
    last_synced_at   TIMESTAMPTZ,
    last_sync_result TEXT,
    last_validated_at TIMESTAMPTZ,
    last_validation_result TEXT,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE settings.intro_course_status IS
    'Singleton row tracking intro course content sync and validation status (IC08).';

INSERT INTO settings.intro_course_status (id)
VALUES (TRUE)
ON CONFLICT (id) DO NOTHING;