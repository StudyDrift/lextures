-- 20.1 Privacy Policy & Terms: track user acknowledgements when legal documents change.

CREATE TABLE IF NOT EXISTS settings.user_legal_acknowledgements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  document        TEXT NOT NULL CHECK (document IN ('privacy_policy', 'terms_of_service')),
  version         TEXT NOT NULL,
  acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ula_user_doc_version
  ON settings.user_legal_acknowledgements (user_id, document, version);

CREATE INDEX IF NOT EXISTS idx_ula_user_document
  ON settings.user_legal_acknowledgements (user_id, document);
