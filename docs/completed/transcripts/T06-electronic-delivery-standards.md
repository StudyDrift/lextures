# T06 — Electronic Delivery & Interoperability Standards

> Implementation plan. The transport layer: PESC/EDI/SPEEDE + signed-PDF adapters, secure links, and delivery receipts. Replaces the fire-and-forget webhook. Source landscape: [transcripts/README](../../plan/transcripts/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T06 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | BLOCKER |
| **Markets** | HE · K12 |
| **Status (today)** | DONE — adapter framework (`electronic_pesc`, `edi_speede`, `electronic_pdf`/`secure_link_email`, `postal_mail`, `api_peer`); release guard; job-queue delivery worker; share links + open/download receipts; delivery_attempts; postal jobs; `delivery_v2` flag. |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Interop/Integrations squad (Backend) |
| **Depends on** | T01 (documents), T02 (order items/recipients), T03 (release trigger + hold guard), T04 (consent gate), T05 (payment gate) |
| **Unblocks** | T07 (shared parser/formats), T10 (receipts) |

---

## 1. Problem Statement

Receiving institutions expect transcripts in industry-standard formats — **PESC XML** and the
older **EDI TS130 / SPEEDE** — or as a **digitally signed, tamper-evident PDF** delivered through a
secure link. Lextures ships none of these; it just POSTs student identity JSON to a webhook. To
join the transcript exchange ecosystem, we need a pluggable delivery layer with real adapters,
secure recipient links, retries, and receipts, all gated on holds/consent/payment.

## 2. Goals

- Deliver each order item via the adapter matching the recipient's capability: PESC XML, EDI/SPEEDE, signed PDF (secure link/email), postal mail (print job), or network peer/API.
- Generate **secure, expiring download links** for PDF recipients with download tracking.
- Produce **delivery attempts** with retries/backoff and **delivery + open receipts**.
- Enforce a single **release guard** re-checking holds (T03), consent (T04), and payment (T05) immediately before send.
- Keep the legacy institution webhook as one adapter option (backward compatible) — but carrying the actual document.

## 3. Non-Goals

- Generating the document (T01) or pricing it (T05).
- Inbound receiving/parsing (T07) — though the PESC (de)serializer is shared.
- Third-party verification portal (T08) — delivery embeds a verify link but the portal is T08.

## 4. Personas & User Stories

- **As a receiving registrar**, I want a PESC XML file that imports into my SIS so that I don't re-key grades.
- **As an employer**, I want a secure link to a signed PDF so that I can confirm it's genuine.
- **As a student**, I want to know when my recipient actually received/opened the transcript so that I'm not left guessing.
- **As a sending registrar**, I want failed deliveries retried and surfaced so that nothing silently drops.
- **As a security reviewer**, I want download links to expire and be access-limited so that records don't leak.

## 5. Functional Requirements

- **FR-1.** The system MUST support delivery adapters: `electronic_pesc` (PESC XML), `edi_speede` (EDI TS130), `electronic_pdf`/`secure_link_email` (signed PDF via expiring link), `postal_mail` (print/mail job), `api_peer` (network/webhook carrying the document).
- **FR-2.** Adapter selection MUST match the order item's delivery method and the recipient's capabilities (validated in T02).
- **FR-3.** Secure links MUST be tokenized, expiring, and download-limited, and MUST record opened/downloaded timestamps and requester IP.
- **FR-4.** The system MUST record a `delivery_attempts` row per attempt with adapter, status, response, and timestamps; failures MUST retry with capped exponential backoff.
- **FR-5.** A single **release guard** MUST re-verify no active hold (T03), valid consent (T04), and satisfied payment (T05) atomically before any send; failure aborts and reflects order state.
- **FR-6.** Delivered documents MUST be the immutable T01 artifact (by `document_id`); the PESC XML MUST validate against the schema before send.
- **FR-7.** The system MUST emit **delivery receipts** (queued/sent/delivered/opened/failed) per item, feeding tracking (T10).
- **FR-8.** The legacy institution webhook MUST be supported as the `api_peer` adapter, now POSTing the actual document (or a fetch link) plus the existing HMAC signature — preserving backward compatibility.
- **FR-9.** Postal mail MUST create a fulfillment job (print vendor or manual queue) with address validation.
- **FR-10.** All adapters MUST be idempotent per (order_item, attempt) to avoid duplicate sends on retry.

## 6. Non-Functional Requirements

- **Performance** — enqueue-to-first-attempt < 5s; PESC validation < 500ms.
- **Security** — signed PDFs; secure links scoped, expiring, rate-limited; SFTP/API creds encrypted; HMAC on peer webhook; no document in logs.
- **Privacy & Compliance** — transcripts are education records; delivery logged for FERPA disclosure log; TLS enforced; link tokens single-purpose.
- **Accessibility** — recipient download page (for PDF links) meets WCAG 2.1 AA; delivered PDFs are PDF/UA (from T01).
- **Scalability** — delivery via job queue; per-adapter concurrency limits; backoff on receiver throttling.
- **Reliability** — at-least-once with idempotency; dead-letter after max retries; guard denies on any gate regression.
- **Observability** — `transcript_delivery_attempt_total{adapter,result}`, delivery latency, retry counts, open-rate; alerts on dead-letter growth.
- **Maintainability** — adapter interface; PESC (de)serializer shared with T07; config per org.
- **Internationalization** — recipient-facing link page and emails localized; address formats per country.
- **Backward compatibility** — existing webhook config continues to work as `api_peer` adapter.

## 7. Acceptance Criteria

- **AC-1.** *Given* a recipient with `electronic_pesc`, *When* an item is released, *Then* a schema-valid PESC XML is delivered and a `delivered` receipt recorded.
- **AC-2.** *Given* a `secure_link_email` item, *When* released, *Then* the recipient gets an expiring link; opening it records an `opened` receipt and downloading decrements the remaining-download count.
- **AC-3.** *Given* an active hold appears after submit, *When* the release guard runs, *Then* delivery is aborted and the order returns to `on_hold`.
- **AC-4.** *Given* a transient adapter failure, *When* delivery is retried, *Then* it backs off and does not double-send (idempotent), and dead-letters after the cap.
- **AC-5.** *Given* the legacy webhook config, *When* an item uses `api_peer`, *Then* it POSTs the document/link with the HMAC signature and succeeds against the existing receiver.
- **AC-6.** *Given* an expired secure link, *When* opened, *Then* access is denied and no document is served.

## 8. Data Model

Migration `408_transcript_delivery.sql`:

```sql
CREATE TABLE transcripts.delivery_attempts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID NOT NULL REFERENCES transcripts.order_items(id) ON DELETE CASCADE,
    adapter       TEXT NOT NULL CHECK (adapter IN ('electronic_pesc','edi_speede','electronic_pdf','secure_link_email','postal_mail','api_peer')),
    attempt_no    INT  NOT NULL,
    status        TEXT NOT NULL CHECK (status IN ('queued','sent','delivered','opened','failed')),
    response_code INT,
    detail        TEXT,
    idempotency_key TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_item_id, idempotency_key)
);
CREATE INDEX idx_delivery_attempts_item ON transcripts.delivery_attempts (order_item_id, created_at);

CREATE TABLE transcripts.share_links (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID NOT NULL REFERENCES transcripts.order_items(id) ON DELETE CASCADE,
    token         TEXT NOT NULL UNIQUE,
    expires_at    TIMESTAMPTZ NOT NULL,
    max_downloads INT NOT NULL DEFAULT 5,
    download_count INT NOT NULL DEFAULT 0,
    opened_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- Per-org adapter config (SFTP creds, peer endpoints, print-vendor keys) stored encrypted in `settings.transcripts_config` extensions or a dedicated `transcripts.delivery_config`.
- Extends the legacy `webhook_url`/`webhook_secret` as the default `api_peer` config.

## 9. API Surface

- Internal: delivery worker consumes `transcript.item.ready` (from T03) → runs adapter → writes attempts/receipts.
- `GET  /r/t/{token}` — public recipient download page for secure links (rate-limited, records open).
- `GET  /r/t/{token}/download` — serves the signed PDF (decrements count).
- `POST /api/v1/admin/transcripts/orders/{id}/items/{itemId}/resend` — registrar/student resend (new attempt).
- `GET  /api/v1/transcripts/orders/{id}/items/{itemId}/receipts` — delivery receipt timeline.
- `GET/PUT /api/v1/admin/transcripts/delivery-config` — adapter/endpoint config (RBAC).
- OpenAPI updated; peer webhook payload documented (now includes document reference).

## 10. UI / UX

- **Student order detail**: per-recipient delivery timeline (queued → sent → delivered → opened), resend action, link expiry indicator.
- **Recipient download page** (`/r/t/{token}`): institution-branded, shows document metadata + verify link (T08), download button, expiry notice; no login required but token-scoped.
- **Registrar delivery config** (T12): adapter setup per recipient type, SFTP/peer endpoints, test-send.
- States: delivering spinner, failed with retry/resend, dead-lettered (contact support), link expired.
- Accessibility: download page WCAG 2.1 AA; receipts timeline has text status.
- i18n for recipient emails and download page.

## 11. AI / ML Considerations

None. (Format mapping is deterministic; no model.)

## 12. Integration Points

- **Internal:** T01 artifacts + PESC serializer (shared with T07), T02 items, T03 guard/trigger, T04/T05 gates, object storage, job queue, email (SES/`372`), audit/FERPA disclosure log, T08 verify links, T10 receipts.
- **External:** PESC network / receiver SFTP/API endpoints; EDI/SPEEDE clearinghouse; print/mail vendor (e.g. Lob) for postal.
- **Emissions:** `transcript.delivery.attempted/succeeded/failed`, `transcript.item.opened`.

## 13. Dependencies & Sequencing

- After: T01, T02, T03, T04, T05. Before: full T10 tracking; shares parser with T07.
- Shared infra: job queue, object storage, email, encrypted config store.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate sends on retry | M | H | Idempotency key per attempt; adapter-level dedupe |
| Gate regression (hold/refund) after submit but before send | M | H | Release guard re-checks all gates atomically at send time; deny-by-default |
| PESC/EDI conformance rejected by receiver | M | H | Validate before send; conformance fixtures; per-receiver profile overrides |
| Secure link leakage | M | H | Short expiry, download caps, token entropy, rate-limit, no directory listing |
| Receiver throttling/outage | M | M | Backoff + dead-letter + registrar visibility + resend |

## 15. Rollout Plan

- Flag `ff_transcripts`; new adapters behind `transcripts.delivery_v2` while `api_peer` (legacy webhook) stays default.
- Sequence: adapter framework + guard → secure-link + PDF adapter → PESC adapter → EDI/SPEEDE → postal → migrate default from legacy webhook.
- Pilot: one receiver pair validates PESC import; one employer validates secure link.
- Rollback: revert to `api_peer` legacy adapter; attempts/receipts retained.

## 16. Test Plan

- **Unit** — adapter selection; idempotency key; link expiry/download-cap; guard logic.
- **Integration** — PESC round-trip (with T07 parser); secure link open/download receipts; retry/backoff/dead-letter; legacy webhook parity.
- **E2E** — order → deliver via each adapter → receipts visible; recipient downloads via link.
- **Security** — link expiry/limits/rate-limit; SFTP/API cred handling; HMAC on peer; no doc in logs; release-guard denial.
- **Accessibility** — recipient download page axe + keyboard/SR.
- **Standards** — PESC/EDI conformance fixtures against reference validators.

## 17. Documentation & Training

- Interop guide: supported formats, per-adapter setup, receiver onboarding, peer webhook payload (now with document).
- Registrar: configuring delivery, resending, reading receipts.
- Recipient help: using the secure link + verify.

## 18. Open Questions

1. Which exchange network(s) do we peer with first (PESC EdExchange, National Student Clearinghouse, direct SFTP)?
2. Do we build EDI/SPEEDE in v1 or defer to a fast-follow after PESC XML?
3. Postal mail: in-house print queue vs. third-party vendor (Lob) at launch?

## 19. References

- Existing: `deliverTranscriptWebhook` in `server/internal/httpserver/transcripts_http.go` (becomes the `api_peer` adapter), email provider (`372_email_provider_ses`), job queue/workers (`server/internal/workers`).
- Standards: PESC XML (AcademicRecord/College Transcript), ANSI X12 EDI TS130 / SPEEDE, PDF signatures (PAdES).
- Related plans: [T01](T01-official-transcript-generation.md), [T03](T03-order-lifecycle-fulfillment-holds.md), [T07](T07-inbound-receiving-transfer-credit.md), [T08](T08-credential-verification.md), [T10](T10-order-tracking-notifications.md).
