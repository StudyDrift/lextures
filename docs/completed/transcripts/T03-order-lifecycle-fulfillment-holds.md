# T03 — Order Lifecycle, Registrar Fulfillment & Holds

> Implementation plan. The workflow engine that moves an order from submit to delivered, with holds that can block issuance. Source landscape: [transcripts/README](../../plan/transcripts/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T03 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | BLOCKER |
| **Markets** | HE · K12 |
| **Status (today)** | DONE — order/item state machine; holds (financial/disciplinary/registrar/library/other); registrar fulfillment queue; auto-approval; SIS hold webhook; audit `order_events`. |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Registrar/SIS squad (Backend + Web) |
| **Depends on** | T02 (orders/items) |
| **Unblocks** | T05, T06, T10, T12 |

---

## 1. Problem Statement

Real transcript operations require gatekeeping: a registrar reviews orders, a **financial or
disciplinary hold** can block release, and each recipient's delivery has its own state. Lextures'
current two-hop status cannot express any of this, so no institution can trust it to withhold a
transcript from a student who owes tuition. This story adds a proper order/item state machine, a
registrar fulfillment console, and a holds system that stops issuance until cleared.

## 2. Goals

- Define an explicit **order and per-item state machine** with legal transitions.
- Give registrars a **fulfillment queue** to review, approve, reject, place on hold, and complete orders.
- Implement **holds** (financial, disciplinary, registrar, library) that block issuance and are surfaced to the student.
- Enforce holds automatically at submit and re-check before any delivery.
- Record a full **audit trail** of every transition and who performed it.

## 3. Non-Goals

- Payment collection (T05) — this story reads payment state as a gate but does not process charges.
- Transport mechanics (T06) — it marks items ready-to-deliver and consumes delivery results.
- Consent capture (T04) — it treats missing consent as a blocking gate.

## 4. Personas & User Stories

- **As a registrar**, I want a queue of pending orders so that I can review and release them.
- **As a registrar**, I want to place a financial hold so that a student who owes fees cannot pull an official transcript.
- **As a bursar/admin**, I want holds I set in the SIS to block transcripts so that policy is enforced.
- **As a student**, I want to see why my order is stuck (e.g. "financial hold — contact the bursar") so that I can resolve it.
- **As an auditor**, I want a complete transition log so that releases are defensible.

## 5. Functional Requirements

- **FR-1.** Orders MUST follow: `draft → pending_consent → pending_payment → in_review → (on_hold ↔ in_review) → processing → completed`, with terminal `canceled` and `rejected`.
- **FR-2.** Each order item MUST track its own state: `pending → ready → delivering → delivered | failed | canceled`.
- **FR-3.** The system MUST support **holds** with types `financial`, `disciplinary`, `registrar`, `library`, `other`, each with reason, placer, timestamps, and release info.
- **FR-4.** At submit and again immediately before delivery, the system MUST evaluate active holds for the student and org; any blocking hold MUST move the order to `on_hold` and prevent issuance/delivery.
- **FR-5.** Registrars MUST be able to approve (`in_review → processing`), reject (with reason), place/release holds, and cancel — all RBAC-gated and audit-logged.
- **FR-6.** Configurable **auto-approval**: an org MAY auto-approve orders with no holds and satisfied consent/payment, skipping manual review.
- **FR-7.** The system MUST expose hold status and human-readable resolution guidance to the student without leaking sensitive detail.
- **FR-8.** Holds MAY be created via API/webhook from an external SIS/bursar system (idempotent upsert keyed by external hold id).
- **FR-9.** Every state transition MUST be recorded with actor, from/to state, reason, and timestamp (immutable log).
- **FR-10.** The system MUST re-verify consent (T04) and payment (T05) gates on each forward transition; a regression (e.g. refund) MUST block further delivery.

## 6. Non-Functional Requirements

- **Performance** — registrar queue list p95 < 400ms; hold evaluation < 100ms per order.
- **Security** — transitions authorized by RBAC role (registrar/admin); students read-only on their orders; external hold API authenticated (HMAC/API key).
- **Privacy & Compliance** — FERPA: withholding and releases logged; hold reasons shown to student are sanitized.
- **Accessibility** — registrar console meets WCAG 2.1 AA; status chips have text equivalents.
- **Scalability** — queue paginated/filterable; hold evaluation indexed by (user, org, active).
- **Reliability** — transitions are transactional; illegal transitions rejected; idempotent external hold upserts.
- **Observability** — `transcript_order_state_transition_total{from,to}`, `transcript_hold_blocked_total{type}`, queue-age gauge.
- **Maintainability** — state machine defined declaratively in one place; guards are pure functions.
- **Internationalization** — status labels and hold-resolution copy localized.
- **Backward compatibility** — legacy `submitted/failed` map onto `completed/failed`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student with an active financial hold, *When* they submit an order, *Then* the order becomes `on_hold` and no document is delivered.
- **AC-2.** *Given* an on-hold order, *When* the hold is released, *Then* the order returns to `in_review` (or auto-processes if configured) and can complete.
- **AC-3.** *Given* an order `in_review`, *When* a registrar rejects it with a reason, *Then* status is `rejected`, the reason is visible to the student, and no delivery occurs.
- **AC-4.** *Given* an illegal transition request (e.g. `draft → completed`), *When* attempted, *Then* it is rejected and logged.
- **AC-5.** *Given* an external SIS hold webhook, *When* posted twice with the same external id, *Then* only one hold exists (idempotent).
- **AC-6.** *Given* auto-approval enabled and no holds/consent/payment gaps, *When* an order is submitted, *Then* it advances to `processing` without manual action.

## 8. Data Model

Migration `393_transcript_order_lifecycle_holds.sql`:

```sql
CREATE TABLE transcripts.holds (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    org_id        UUID REFERENCES tenant.organizations(id),
    type          TEXT NOT NULL CHECK (type IN ('financial','disciplinary','registrar','library','other')),
    reason        TEXT,
    student_message TEXT,                       -- sanitized guidance shown to student
    external_id   TEXT,                          -- SIS/bursar idempotency key
    placed_by     UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    placed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_by   UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    released_at   TIMESTAMPTZ
);
CREATE INDEX idx_holds_active ON transcripts.holds (user_id, org_id) WHERE released_at IS NULL;
CREATE UNIQUE INDEX ux_holds_external ON transcripts.holds (org_id, external_id) WHERE external_id IS NOT NULL;

CREATE TABLE transcripts.order_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES transcripts.orders(id) ON DELETE CASCADE,
    item_id    UUID REFERENCES transcripts.order_items(id) ON DELETE CASCADE,
    from_state TEXT,
    to_state   TEXT NOT NULL,
    actor_id   UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_events_order ON transcripts.order_events (order_id, created_at);
```

- `transcripts.orders.status` / `order_items.status` CHECK constraints extended to the full state set.
- Org auto-approval flag stored in `settings.transcripts_config` (or per-org override).

## 9. API Surface

- `GET  /api/v1/admin/transcripts/orders?status=&hold=&q=` — registrar fulfillment queue (RBAC).
- `POST /api/v1/admin/transcripts/orders/{id}/transition` — `{action: approve|reject|cancel, reason?}`.
- `GET/POST /api/v1/admin/transcripts/holds` , `POST /api/v1/admin/transcripts/holds/{id}/release`.
- `POST /api/v1/integrations/transcripts/holds` — external SIS hold upsert (HMAC-authenticated, idempotent).
- `GET  /api/v1/transcripts/orders/{id}` (T02) extended: exposes state, hold status + `student_message`, event history summary.
- OpenAPI updated; state machine documented.

## 10. UI / UX

- **Registrar console** (new, feeds T12): filterable order queue (status, hold, urgency, age), row → detail drawer with actions (approve/reject/hold/release), audit timeline.
- **Holds admin**: place/release holds on a student; list active holds.
- **Student order detail**: prominent status banner; on-hold shows `student_message` + resolution CTA; rejected shows reason.
- States: empty queue, bulk actions, optimistic transition with rollback, permission-denied.
- Mobile: registrar console usable read-first with key actions; student status fully responsive.
- i18n for statuses, actions, hold copy.

## 11. AI / ML Considerations

None required. (Optional future: anomaly flag for unusual order spikes — out of scope.)

## 12. Integration Points

- **Internal:** T02 orders/items, T04 consent gate, T05 payment gate, T06 delivery trigger, RBAC, audit log ([10.11]), notifications (T10).
- **External:** SIS/bursar hold webhooks (inbound).
- **Emissions:** `transcript.order.transitioned`, `transcript.hold.placed/released` (T10, T12 consume).

## 13. Dependencies & Sequencing

- After: T02. Before: T05 gating UX, T06 delivery trigger, T10, T12.
- Shared infra: RBAC, audit log, job queue (for re-evaluation before delivery).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hold not enforced before delivery (compliance failure) | M | H | Re-check holds in the delivery guard (T06), not only at submit; deny-by-default |
| Illegal transitions corrupt state | M | H | Declarative state machine + DB CHECK + transactional guards + tests for every edge |
| External SIS holds out of sync | M | M | Idempotent upsert + periodic reconcile; TTL/last-seen on external holds |
| Registrar bottleneck | M | M | Auto-approval config for no-hold orders; bulk actions |

## 15. Rollout Plan

- Flag `ff_transcripts`; registrar console behind `transcripts.registrar_console` sub-flag.
- Sequence: schema + state machine → hold evaluation guard → registrar APIs/console → external hold webhook → enable auto-approval per org.
- Pilot: registrar processes a batch with a seeded financial hold; verify blocking + release.
- Rollback: disable console flag; state machine defaults to auto-approve to avoid stuck orders.

## 16. Test Plan

- **Unit** — transition guards (legal/illegal matrix); hold evaluation; sanitized student message.
- **Integration** — submit-with-hold blocks; release resumes; external hold idempotency; consent/payment regression blocks delivery.
- **E2E** — registrar approves an order end-to-end; rejects another; student sees correct banners.
- **Security** — RBAC on all admin routes; external webhook auth; students cannot transition.
- **Accessibility** — console axe + keyboard/SR; status chips text equivalents.
- **Performance** — queue pagination; hold-eval latency under load.

## 17. Documentation & Training

- Registrar runbook: reviewing orders, placing/releasing holds, auto-approval policy.
- Bursar/SIS integration guide for the holds webhook.
- Student help: "Why is my transcript on hold?"

## 18. Open Questions

1. Do holds apply to *unofficial* transcripts/previews, or only official issuance? (Default: official only.)
2. Should partial fulfillment be allowed (deliver un-held items) or all-or-nothing per order?
3. SLA/queue-age alerting thresholds per org.

## 19. References

- Existing: `server/internal/httpserver/transcripts_http.go` (status handling), RBAC repos, audit log ([10.11](../../completed/10-compliance-privacy-security/)).
- Related plans: [T02](T02-recipient-directory-and-orders.md), [T04](T04-ferpa-consent-esignature.md), [T05](T05-fees-payments-waivers.md), [T06](T06-electronic-delivery-standards.md), [T12](T12-registrar-console-analytics.md).
- Shipped: migration `393`, `models/transcriptorder`, `repos/transcripts/{holds,lifecycle}.go`, `transcripts_lifecycle_http.go`, registrar UI at `/admin/transcripts`.
