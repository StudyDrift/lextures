# T02 — Recipient Directory & Multi-Destination Orders

> Implementation plan. Replaces the single-webhook request row with an order + recipient network. Source landscape: [transcripts/README](../../plan/transcripts/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T02 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | BLOCKER |
| **Markets** | HE · K12 · SL |
| **Status (today)** | DONE — recipient directory + multi-item orders; legacy `transcript_requests` backfilled and proxied. |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Registrar/SIS squad (Backend + Web) |
| **Depends on** | T01 (documents to send) |
| **Unblocks** | T03, T05, T06, T07, T10, T12 |

---

## 1. Problem Statement

Parchment's core value is the **network**: a learner picks one or more recipients and Lextures
routes the right document to each. Today a "request" has no recipient at all — it just pings the
institution. Without an order-and-recipient model, we cannot price per destination, deliver
electronically to receiving schools, capture per-recipient consent, or track fulfillment. This
story introduces the `orders`/`order_items` model and a searchable recipient directory, and
migrates the legacy request table onto it.

## 2. Goals

- Model a **transcript order** containing one or more **recipients**, each with its own delivery method.
- Provide a **searchable recipient directory** (institutions, employers, application services, self, custom).
- Let a student build an order (add recipients, choose per-recipient delivery) and submit it once.
- Migrate legacy `transcript_requests` into `orders` + `order_items` without data loss.
- Expose recipient "delivery capabilities" so downstream stories pick the right transport.

## 3. Non-Goals

- Fulfillment workflow, approvals, holds (T03); consent capture (T04); pricing/payment (T05).
- The actual transport/delivery mechanics and standards (T06).
- Editing the academic record (T01).

## 4. Personas & User Stories

- **As a student**, I want to search for my target school/employer and add it as a recipient so that I don't type addresses by hand.
- **As a student**, I want to send to several recipients in one order so that applying to multiple schools is one checkout.
- **As a student**, I want to send a copy to myself so that I keep a personal record.
- **As a registrar/admin**, I want to curate and verify recipients in the directory so that deliveries route correctly.
- **As a self-learner**, I want to send an unofficial record to an employer so that I can share verified progress.

## 5. Functional Requirements

- **FR-1.** The system MUST support a recipient directory with types `institution`, `application_service` (e.g. Common App, AMCAS, NCAA Eligibility Center), `employer`, `self`, `other`.
- **FR-2.** Each recipient MUST record delivery capabilities: `electronic_pesc`, `electronic_pdf`, `secure_link_email`, `postal_mail`, `api_peer` (any subset).
- **FR-3.** Students MUST be able to search the directory (name, location, type) with typeahead, and add an **ad-hoc recipient** when not found.
- **FR-4.** An **order** MUST contain 1..N **order items**, each = one recipient + one document (T01) + one delivery method + per-item status.
- **FR-5.** The system MUST validate that the chosen delivery method is within the recipient's capabilities and the org's enabled methods.
- **FR-6.** The system MUST preserve immutability: each order item references a specific issued `transcript_documents.id` (or triggers issuance at submit).
- **FR-7.** The system MUST migrate every legacy `transcript_requests` row into an order (single-item) preserving delivery type, urgency, status, timestamps.
- **FR-8.** Admins MUST be able to add/verify/deactivate directory recipients scoped to their org; a shared/global directory MAY be seeded.
- **FR-9.** The system SHOULD deduplicate recipients (avoid duplicate institution rows) via a canonical key (CEEB/ACT code, domain, or normalized name).
- **FR-10.** An order MUST be user-scoped; a student MUST NOT see or act on another student's orders.

## 6. Non-Functional Requirements

- **Performance** — directory typeahead p95 < 200ms; order create p95 < 500ms.
- **Security** — order/item access strictly user-scoped; registrar access via RBAC; ad-hoc recipient input sanitized.
- **Privacy & Compliance** — recipient PII (employer contact, address) minimized and access-logged; FERPA release still gated by T04 before anything leaves.
- **Accessibility** — search/typeahead and multi-recipient builder are keyboard- and screen-reader-navigable (combobox ARIA).
- **Scalability** — directory indexed for search; orders partitionable by org/time.
- **Reliability** — order creation atomic across items; partial-failure safe.
- **Observability** — `transcript_order_created_total`, `transcript_order_items{delivery_method,recipient_type}`.
- **Maintainability** — recipient capability enum centralized; delivery-method validation shared with T06.
- **Internationalization** — address formats and directory labels localized.
- **Backward compatibility** — legacy request endpoints proxy to the order model during a deprecation window.

## 7. Acceptance Criteria

- **AC-1.** *Given* a directory with a seeded university, *When* a student searches its name, *Then* it appears with its delivery capabilities and can be added.
- **AC-2.** *Given* two recipients added, *When* the student submits, *Then* one order with two items is created, each linked to a document and a delivery method.
- **AC-3.** *Given* a recipient without `postal_mail` capability, *When* mail is selected, *Then* the API rejects with a clear validation error.
- **AC-4.** *Given* legacy `transcript_requests` rows, *When* the migration runs, *Then* each becomes a one-item order with identical delivery/urgency/status/timestamps and legacy IDs are traceable.
- **AC-5.** *Given* user A's order, *When* user B requests it, *Then* the API returns 404/403.
- **AC-6.** *Given* an ad-hoc recipient with an existing canonical key, *When* added, *Then* it links to the existing directory row instead of duplicating.

## 8. Data Model

Migration `387_transcript_orders_recipients.sql`:

```sql
CREATE TABLE transcripts.recipients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID REFERENCES tenant.organizations(id),   -- NULL = global/seeded
    type          TEXT NOT NULL CHECK (type IN ('institution','application_service','employer','self','other')),
    name          TEXT NOT NULL,
    canonical_key TEXT,                                        -- CEEB/ACT code, domain, or normalized name
    capabilities  TEXT[] NOT NULL DEFAULT '{}',               -- electronic_pesc,electronic_pdf,secure_link_email,postal_mail,api_peer
    email         TEXT,
    address       JSONB,
    peer_config   JSONB,                                       -- endpoint/network id for api_peer (T06)
    verified      BOOLEAN NOT NULL DEFAULT FALSE,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ux_recipients_canonical ON transcripts.recipients (COALESCE(org_id,'00000000-0000-0000-0000-000000000000'), canonical_key) WHERE canonical_key IS NOT NULL;

CREATE TABLE transcripts.orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    org_id        UUID REFERENCES tenant.organizations(id),
    status        TEXT NOT NULL DEFAULT 'draft',   -- lifecycle owned by T03
    consent_id    UUID,                            -- FK added in T04
    total_amount  INT,                             -- minor units; T05
    currency      TEXT,
    legacy_request_id UUID,                         -- provenance from transcript_requests
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at  TIMESTAMPTZ
);
CREATE INDEX idx_orders_user ON transcripts.orders (user_id, created_at DESC);

CREATE TABLE transcripts.order_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES transcripts.orders(id) ON DELETE CASCADE,
    recipient_id  UUID REFERENCES transcripts.recipients(id),
    document_id   UUID REFERENCES transcripts.transcript_documents(id),
    delivery_method TEXT NOT NULL CHECK (delivery_method IN ('electronic_pesc','electronic_pdf','secure_link_email','postal_mail','api_peer')),
    urgency       TEXT NOT NULL DEFAULT 'standard' CHECK (urgency IN ('standard','rush')),
    fee_amount    INT,                              -- T05
    status        TEXT NOT NULL DEFAULT 'pending',  -- per-item; refined in T03/T06
    delivered_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_items_order ON transcripts.order_items (order_id);
```

- **Backfill**: one-time migration maps each `transcript_requests` row → `orders` (+`legacy_request_id`) and a single `order_items` row (self-recipient for email; ad-hoc for mail/pickup). Legacy table kept read-only for one release.

## 9. API Surface

- `GET  /api/v1/transcripts/recipients?q=&type=` — search directory (typeahead).
- `POST /api/v1/transcripts/orders` — create draft `{items:[{recipient|adHocRecipient, deliveryMethod, urgency, terms?}]}`.
- `GET  /api/v1/transcripts/orders` / `GET /api/v1/transcripts/orders/{id}` — list/detail (owner-scoped).
- `POST /api/v1/transcripts/orders/{id}/items` / `DELETE …/items/{itemId}` — edit draft.
- `POST /api/v1/transcripts/orders/{id}/submit` — validate → hand to consent (T04)/payment (T05).
- `GET/POST/PUT /api/v1/admin/transcripts/recipients` — registrar directory management (RBAC).
- OpenAPI updated; legacy `POST /api/v1/transcripts/requests` proxies to a one-item order (deprecated header).

## 10. UI / UX

- **New order flow** in `transcripts-page.tsx` + refactor of `transcript-request-modal.tsx` into a multi-step order builder:
  1. Choose transcript type/terms (T01) → 2. Add recipient(s) via directory search or ad-hoc → 3. Delivery method + urgency per recipient → 4. Review → (consent T04 → payment T05).
- Recipient search: combobox with type filter, capability badges, "Send to myself" shortcut, "Can't find it? Add manually."
- States: empty directory, no-results (offer ad-hoc), invalid method for recipient (inline), draft autosave, submit success.
- Mobile responsive; order builder works on small screens.
- i18n for all copy; recipient-type and capability labels externalized.

## 11. AI / ML Considerations

Optional: fuzzy recipient matching / dedupe suggestions (normalize institution names). Non-blocking; deterministic canonical-key match is primary. No PII sent to any model; if used, redact and cap cost.

## 12. Integration Points

- **Internal:** T01 documents, RBAC, org repos, address/geo formatting, notifications (T10).
- **External:** optional seed from a public institution registry (CEEB/IPEDS) for the directory.
- **Emissions:** `transcript.order.created`, `transcript.order.submitted`.

## 13. Dependencies & Sequencing

- After: T01. Before: T03, T05, T06, T07, T10, T12.
- Shared infra: search index (or trigram index), directory seed job.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate/dirty recipient directory | H | M | Canonical-key dedupe + admin verification flag + seed from authoritative registry |
| Legacy migration data loss | M | H | Reversible migration, `legacy_request_id` provenance, dry-run count reconciliation |
| Wrong delivery method for a recipient | M | M | Capability validation shared with T06; UI hides unsupported methods |

## 15. Rollout Plan

- Flag `ff_transcripts`; new order UI behind `transcripts.orders_ui` sub-flag while legacy modal remains.
- Sequence: schema + migration (backfill) → order APIs → directory search → order builder UI → flip UI flag → deprecate legacy request endpoint next release.
- Pilot: registrar seeds directory; students place multi-recipient orders in a sandbox.
- Rollback: revert UI flag to legacy modal; order tables retained.

## 16. Test Plan

- **Unit** — capability validation; canonical-key dedupe; order/item invariants.
- **Integration** — create multi-item order; legacy backfill reconciliation; owner scoping.
- **E2E** — search → add two recipients → submit → order visible with two items.
- **Security** — cross-user access denied; ad-hoc input sanitization; admin RBAC on directory.
- **Accessibility** — combobox + multi-step builder axe + keyboard/SR scripts.
- **Performance** — typeahead latency; bulk directory seed.

## 17. Documentation & Training

- Student help: "Send to multiple recipients," "Add a recipient manually."
- Registrar docs: managing/verifying the recipient directory, capability meanings.
- API reference for orders/recipients; migration note for legacy request API.

## 18. Open Questions

1. Ship a shared/global seeded directory (IPEDS/CEEB) or per-org only at launch?
2. Do we join an external exchange network for `api_peer` now, or defer peer config to T06?
3. Draft order expiry/GC policy.

## 19. References

- Existing: `server/internal/repos/transcripts/repo.go`, `clients/web/src/components/lms/transcript-request-modal.tsx`, `clients/web/src/lib/transcripts-api.ts`, `clients/web/src/pages/lms/transcripts-page.tsx`.
- Related plans: [T01](T01-official-transcript-generation.md), [T03](T03-order-lifecycle-fulfillment-holds.md), [T04](T04-ferpa-consent-esignature.md), [T05](T05-fees-payments-waivers.md), [T06](T06-electronic-delivery-standards.md).
