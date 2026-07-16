# T10 — Order Tracking & Notifications

> Implementation plan. Real-time status, delivery/open receipts, resend/cancel, and email + push at each step. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T10 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MINOR |
| **Markets** | HE · K12 · SL |
| **Status (today)** | THIN — a student can list their requests and see queued/submitted/failed; there are no lifecycle notifications, no delivery/open receipts surfaced, and no resend/cancel. |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Learner-experience squad |
| **Depends on** | T02 (orders), T03 (states), T06 (delivery receipts) |
| **Unblocks** | Support-load reduction; T12 insights |

---

## 1. Problem Statement

Ordering a transcript is anxiety-inducing when you can't see where it is. Parchment sends status
updates at every step and shows when a recipient received/opened the document. Lextures shows only
a coarse status and never notifies. This story turns the order lifecycle (T03) and delivery
receipts (T06) into a clear tracking timeline plus proactive email/push notifications, and adds
resend/cancel self-service — cutting "where's my transcript?" support load.

## 2. Goals

- Show a **tracking timeline** per order and per recipient (submitted → in review → processing → sent → delivered → opened).
- **Notify** the learner (and guardian where relevant) by email + push at each meaningful transition.
- Surface **delivery + open receipts** (from T06) to the learner.
- Provide **self-service** resend and cancel where the state allows.
- Notify **registrars** of exceptions (failures, dead-letters, holds) needing action.

## 3. Non-Goals

- The lifecycle/state machine (T03) and delivery mechanics/receipts (T06) — this presents them.
- Building a new notification transport (reuse the existing email + push infrastructure).

## 4. Personas & User Stories

- **As a student**, I want a live timeline of my order so that I know exactly where it is.
- **As a student**, I want an email/push when my transcript is delivered and opened so that I have proof.
- **As a student**, I want to resend or cancel an order myself so that I don't file a support ticket.
- **As a guardian**, I want updates on a minor's order I authorized so that I stay informed.
- **As a registrar**, I want alerts on failed deliveries so that I can intervene.

## 5. Functional Requirements

- **FR-1.** The system MUST present a per-order and per-item tracking timeline derived from T03 events and T06 delivery attempts/receipts.
- **FR-2.** The system MUST send learner notifications (email + push) on: submitted, on-hold, consent needed, payment needed, approved/rejected, sent, delivered, opened, failed.
- **FR-3.** Notifications MUST respect user notification preferences and per-channel opt-outs; transactional/compliance messages MAY be non-optional.
- **FR-4.** For guardian-authorized minor orders, the guardian MUST receive the relevant updates.
- **FR-5.** Students MUST be able to **resend** a delivered/failed item (creates a new delivery attempt via T06) and **cancel** an order while its state permits (pre-delivery), triggering refund logic (T05) if paid.
- **FR-6.** Registrars MUST receive exception notifications (delivery failure/dead-letter, new holds) via their configured channel.
- **FR-7.** Notification content MUST be localized and MUST NOT include the transcript itself (link to secure download instead).
- **FR-8.** All notifications MUST be idempotent per (order/item, event) to avoid duplicates on retries.
- **FR-9.** The timeline MUST update in near-real-time (poll or push) without a full page reload.
- **FR-10.** Notification sends MUST be logged for support/audit.

## 6. Non-Functional Requirements

- **Performance** — timeline load p95 < 300ms; notification enqueue < 1s after event.
- **Security** — notifications never embed the document; links are secure/expiring (T06/T08); no PII beyond necessary.
- **Privacy & Compliance** — respect opt-outs; CAN-SPAM/CASL footer where applicable; disclosure logging for compliance messages.
- **Accessibility** — timeline WCAG 2.1 AA; emails accessible (semantic HTML + text alt).
- **Scalability** — reuse existing notification queue; batch registrar digests where noisy.
- **Reliability** — at-least-once with idempotency dedupe; failed sends retried.
- **Observability** — `transcript_notification_sent_total{event,channel}`, delivery/open surfaced rates.
- **Maintainability** — event→notification mapping table-driven; templates in the existing email-template system.
- **Internationalization** — templates localized (reuse markdown email templates, migration `373`).
- **Backward compatibility** — existing request-list view augmented, not replaced.

## 7. Acceptance Criteria

- **AC-1.** *Given* an order progressing through states, *When* each transition occurs, *Then* the timeline updates and the matching notification is sent once.
- **AC-2.** *Given* a recipient opens a secure link (T06), *When* the open receipt lands, *Then* the student sees "opened" and gets a notification.
- **AC-3.** *Given* a user who opted out of push, *When* an event fires, *Then* only email (or nothing, for optional messages) is sent.
- **AC-4.** *Given* a delivered/failed item, *When* the student resends, *Then* a new delivery attempt is created and tracked.
- **AC-5.** *Given* a paid, pre-delivery order, *When* the student cancels, *Then* the order cancels and refund logic (T05) runs.
- **AC-6.** *Given* a delivery failure, *When* it dead-letters, *Then* the registrar is notified.

## 8. Data Model

Mostly reuses existing notification + email-template infra and the T03 `order_events` / T06
`delivery_attempts` tables. Minimal additions (migration `387_transcript_order_events.sql` may be
folded into T03):

```sql
-- Idempotency ledger for order notifications (avoid dupes on retry).
CREATE TABLE transcripts.notification_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES transcripts.orders(id) ON DELETE CASCADE,
    item_id    UUID REFERENCES transcripts.order_items(id) ON DELETE CASCADE,
    event      TEXT NOT NULL,
    channel    TEXT NOT NULL,
    recipient  TEXT NOT NULL,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, COALESCE(item_id,'00000000-0000-0000-0000-000000000000'), event, channel)
);
```

## 9. API Surface

- `GET  /api/v1/transcripts/orders/{id}/timeline` — merged state + delivery events for the timeline.
- `POST /api/v1/transcripts/orders/{id}/items/{itemId}/resend` — (shared with T06) learner resend.
- `POST /api/v1/transcripts/orders/{id}/cancel` — learner cancel (state-permitting) → T05 refund.
- Notification templates registered in the existing email-template system; push via existing service.
- WebSocket/poll channel for live timeline updates.
- OpenAPI updated.

## 10. UI / UX

- **Order detail timeline**: vertical stepper per order with per-recipient sub-tracks (sent/delivered/opened), timestamps, and status chips; resend/cancel buttons where allowed.
- **Notifications**: in-app notification entries link to the order; email + push per event.
- **Registrar**: exception alerts link to the fulfillment console (T03/T12).
- States: live-updating, delivered/opened highlights, failed with resend, canceled, refund-pending.
- Accessibility: stepper has text status + ARIA; live region announces updates.
- i18n via markdown email templates (`373`).

## 11. AI / ML Considerations

None.

## 12. Integration Points

- **Internal:** existing notification + push services, email templates (`373_email_templates_markdown`), SES (`372`), T03 events, T06 receipts, T05 refund on cancel, WebSocket infra.
- **External:** email/push providers (already integrated).
- **Emissions:** consumes T03/T06 events; emits `transcript.notification.sent`.

## 13. Dependencies & Sequencing

- After: T02, T03, T06 (for receipts). Enhances all of them.
- Shared infra: notification queue, email templates, push, WebSocket.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate/noisy notifications | M | M | Idempotency ledger; digest for registrar exceptions; respect preferences |
| Notification leaks document/PII | L | H | Never embed document; secure links only; minimal content |
| Timeline out of sync with truth | M | M | Derive from authoritative events; live refresh; reconcile on load |

## 15. Rollout Plan

- Flag `ff_transcripts`; notifications default on for transactional events.
- Sequence: timeline read → notification mapping + templates → resend/cancel → registrar exception alerts → live updates.
- Pilot: verify a full order emits the correct notifications and timeline.
- Rollback: disable non-essential notifications; timeline remains.

## 16. Test Plan

- **Unit** — event→notification mapping; idempotency dedupe; preference gating.
- **Integration** — full lifecycle emits correct, single notifications; resend/cancel + refund; guardian updates.
- **E2E** — student places order → receives step notifications → sees delivered/opened → resends.
- **Security** — no document in messages; link scoping; opt-out honored.
- **Accessibility** — timeline + emails axe; live-region announcements.
- **Reliability** — duplicate-event idempotency; retry behavior.

## 17. Documentation & Training

- Student help: tracking your order, notifications, resend/cancel.
- Registrar: exception alerts and how to act.
- Admin: notification preferences and non-optional compliance messages.

## 18. Open Questions

1. Which events are non-optional (compliance) vs. preference-gated?
2. Registrar exception delivery: per-event vs. digest, and channel (email/Slack/in-app)?
3. Live updates via existing WebSocket vs. polling for the timeline.

## 19. References

- Existing: notification/push services, email templates (`373_email_templates_markdown_system_scope`), SES (`372_email_provider_ses`), `clients/web/src/pages/lms/transcripts-page.tsx`.
- Related plans: [T03](../../completed/transcripts/T03-order-lifecycle-fulfillment-holds.md), [T06](T06-electronic-delivery-standards.md), [T05](T05-fees-payments-waivers.md), [T12](T12-registrar-console-analytics.md).
