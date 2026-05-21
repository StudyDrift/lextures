-- 20.2 Trust Center: store sub-processor change notification subscriptions.

CREATE SCHEMA IF NOT EXISTS trust;

CREATE TABLE trust.sub_processor_subscriptions (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email        TEXT        NOT NULL,
  subscribed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS trust_subproc_sub_email_idx
  ON trust.sub_processor_subscriptions (lower(email));
