-- T10: Order tracking notification idempotency ledger + email template slots.

CREATE TABLE IF NOT EXISTS transcripts.notification_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    item_id    UUID REFERENCES transcripts.order_items (id) ON DELETE CASCADE,
    -- item_key is COALESCE(item_id, zero-uuid) so UNIQUE works without expression indexes.
    item_key   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    event      TEXT NOT NULL,
    channel    TEXT NOT NULL CHECK (channel IN ('email', 'push', 'in_app')),
    recipient  TEXT NOT NULL,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, item_key, event, channel)
);

CREATE INDEX IF NOT EXISTS notification_log_order_idx
    ON transcripts.notification_log (order_id, sent_at DESC);

COMMENT ON TABLE transcripts.notification_log IS
    'T10: Idempotency ledger for transcript order lifecycle notifications (email/push/in-app).';

-- Lifecycle email slots (localized defaults; orgs may override).
INSERT INTO settings.email_template_slots
    (id, description, merge_fields, default_html, default_text, default_markdown)
VALUES
(
    'transcript_order_update',
    'Transcript order status update (T10)',
    '{"title":"Notification title","message":"Status message","link":"Secure order tracking link","order.id":"Order id"}'::jsonb,
    '<p><strong>{{title}}</strong></p><p>{{message}}</p><p><a href="{{link}}">Track your order</a></p><p>This message does not include your transcript. Use the secure link to view status.</p>',
    '{{title}}

{{message}}

Track your order: {{link}}

This message does not include your transcript. Use the secure link to view status.',
    '**{{title}}**

{{message}}

[Track your order]({{link}})

This message does not include your transcript. Use the secure link to view status.'
),
(
    'transcript_order_exception',
    'Registrar transcript exception alert (T10)',
    '{"title":"Alert title","message":"Exception detail","link":"Fulfillment console link","order.id":"Order id"}'::jsonb,
    '<p><strong>{{title}}</strong></p><p>{{message}}</p><p><a href="{{link}}">Open fulfillment queue</a></p>',
    '{{title}}

{{message}}

Open fulfillment queue: {{link}}',
    '**{{title}}**

{{message}}

[Open fulfillment queue]({{link}})'
)
ON CONFLICT (id) DO NOTHING;
