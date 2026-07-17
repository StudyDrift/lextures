# T07 — Inbound Receiving & Transfer-Credit Intake

> Implementation plan. Receive transcripts *from* other institutions, parse PESC, and match to an applicant. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T07 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MAJOR |
| **Markets** | HE |
| **Status (today)** | DONE — inbound API (HMAC peer) + intake queue; PESC `ParseXML` into canonical model; PDF metadata path with manual review; match/accept/reject with audit; course hand-off API; `ff_transcript_inbound`; migration `409`. |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Interop/Integrations squad + Admissions/Registrar |
| **Depends on** | T06 (shared PESC parser + delivery infra), T02 (recipient = us) |
| **Unblocks** | Transfer-credit evaluation workflows; admissions intake |

---

## 1. Problem Statement

A transcript network is bidirectional: institutions must *receive* transcripts (for transfers and
admissions) as well as send them. Lextures can only send. Without an inbound channel, a Lextures
registrar still logs into other systems or opens email PDFs to process incoming records — so the
platform can't be the system of record for transfer credit. This story adds an inbound endpoint,
PESC/PDF parsing, applicant matching, and an intake queue that feeds credit evaluation.

## 2. Goals

- Provide an **inbound channel** (secure API/SFTP + email drop) to receive transcripts addressed to the institution.
- **Parse** received PESC XML into the canonical academic-record model; store PDFs with extracted metadata.
- **Match** an inbound transcript to an existing applicant/student (or hold as unmatched).
- Give registrars an **intake queue** to review, match, accept, or reject inbound documents.
- Expose parsed course data to **transfer-credit evaluation** (hand-off, not full articulation engine).

## 3. Non-Goals

- A full transfer-credit **articulation** engine (course-equivalency rules) — out of scope; this delivers the parsed data and a hand-off point.
- Outbound delivery (T06) and document generation (T01).
- OCR of scanned paper transcripts beyond best-effort text extraction (flag for manual review).

## 4. Personas & User Stories

- **As an admissions officer**, I want incoming transcripts to land in one queue so that I stop hunting through email.
- **As a registrar**, I want a received PESC transcript parsed into structured courses so that credit evaluation starts from data, not a PDF.
- **As a registrar**, I want inbound documents auto-matched to the right applicant so that I don't misfile records.
- **As a transfer student**, I want confirmation my prior school's transcript was received so that I know my application is complete.
- **As a security reviewer**, I want inbound content sandboxed and validated so that malicious payloads can't harm us.

## 5. Functional Requirements

- **FR-1.** The system MUST accept inbound transcripts via authenticated API (peer/network), SFTP drop, and a monitored email address, gated by `ff_transcript_inbound`.
- **FR-2.** The system MUST parse PESC XML into the canonical academic-record model (shared parser with T06) and store the original artifact immutably.
- **FR-3.** For PDF-only inbound, the system MUST store the file and extract available metadata (sender, student name, DOB) with a "needs manual review" flag.
- **FR-4.** The system MUST attempt to **match** inbound documents to an existing applicant/student by name + DOB + sending-institution + provided reference id, producing a confidence score.
- **FR-5.** Unmatched or low-confidence documents MUST go to an **unmatched** queue for manual assignment; matching MUST be auditable and reversible.
- **FR-6.** Registrars MUST be able to accept (attach to applicant record), reject (with reason), or reassign inbound documents (RBAC).
- **FR-7.** Inbound content MUST be validated and sandboxed (schema validation, size/type limits, malware scan) before parsing.
- **FR-8.** Parsed course data MUST be exposed via API for downstream transfer-credit evaluation (structured JSON), without auto-awarding credit.
- **FR-9.** The system MUST notify the applicant/student (T10) when their inbound transcript is received and when accepted.
- **FR-10.** The system MUST deduplicate re-sent inbound documents (same sender + reference id).

## 6. Non-Functional Requirements

- **Performance** — parse + match p95 < 3s per PESC document (async for large batches).
- **Security** — inbound authenticated; content sandboxed, size/type-limited, malware-scanned; parser hardened against XXE/entity expansion.
- **Privacy & Compliance** — inbound records are education records; access-logged; retained per policy; unmatched PII minimized and TTL'd.
- **Accessibility** — intake queue and document viewer WCAG 2.1 AA.
- **Scalability** — batch intake via queue; large SFTP drops chunked.
- **Reliability** — at-least-once intake with dedupe; poison messages dead-lettered; matching idempotent.
- **Observability** — `transcript_inbound_received_total{channel}`, `transcript_inbound_match_rate`, unmatched-queue-age gauge.
- **Maintainability** — one PESC (de)serializer shared with T06; matching strategy pluggable.
- **Internationalization** — international name/date handling in matching; locale-aware parsing.
- **Backward compatibility** — additive; ships dark behind its own flag.

## 7. Acceptance Criteria

- **AC-1.** *Given* a valid PESC XML posted to the inbound API, *When* processed, *Then* it is parsed into the canonical model, stored immutably, and appears in the intake queue.
- **AC-2.** *Given* an inbound document matching an applicant by name+DOB+reference, *When* processed, *Then* it auto-matches with a confidence score and links to that applicant.
- **AC-3.** *Given* a low-confidence match, *When* processed, *Then* it lands in the unmatched queue and a registrar can assign it, with the action audited.
- **AC-4.** *Given* a malformed/oversized/malicious payload, *When* received, *Then* parsing is refused, the item is quarantined, and no unsafe processing occurs.
- **AC-5.** *Given* the same document re-sent, *When* received, *Then* it is deduplicated (single intake row).
- **AC-6.** *Given* an accepted inbound transcript, *When* queried, *Then* structured course data is available for transfer-credit evaluation and the applicant is notified.

## 8. Data Model

Migration `409_transcript_inbound.sql`:

```sql
CREATE TABLE transcripts.inbound_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES tenant.organizations(id),
    channel        TEXT NOT NULL CHECK (channel IN ('api_peer','sftp','email')),
    source_name    TEXT,                       -- sending institution (parsed/declared)
    external_ref   TEXT,                       -- sender's reference id (dedupe)
    format         TEXT NOT NULL CHECK (format IN ('pesc_xml','pdf','edi','other')),
    raw_key        TEXT NOT NULL,              -- object storage key of original
    parsed         JSONB,                      -- canonical academic-record model (if parsed)
    matched_user_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    match_confidence NUMERIC(4,3),
    status         TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN ('received','quarantined','parsed','matched','accepted','rejected','unmatched')),
    reviewer_id    UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    reject_reason  TEXT,
    received_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at   TIMESTAMPTZ,
    UNIQUE (org_id, source_name, external_ref)
);
CREATE INDEX idx_inbound_queue ON transcripts.inbound_documents (org_id, status, received_at);
```

- Add `ff_transcript_inbound` to `settings.platform_app_settings` + `platformconfig`.

## 9. API Surface

- `POST /api/v1/integrations/transcripts/inbound` — authenticated peer/network intake (PESC/PDF).
- SFTP + email-drop ingestion workers → same pipeline.
- `GET  /api/v1/admin/transcripts/inbound?status=` — intake queue (RBAC).
- `GET  /api/v1/admin/transcripts/inbound/{id}` — detail + parsed data + original.
- `POST /api/v1/admin/transcripts/inbound/{id}/match` — assign to applicant `{userId}`.
- `POST /api/v1/admin/transcripts/inbound/{id}/accept|reject` — decision `{reason?}`.
- `GET  /api/v1/admin/transcripts/inbound/{id}/courses` — structured course data for credit eval.
- OpenAPI updated; inbound payload/auth documented.

## 10. UI / UX

- **Admissions/registrar intake inbox**: filterable list (status, sender, date, matched/unmatched), row → viewer with parsed courses side-by-side with the original document.
- **Match panel**: applicant search, confidence indicator, one-click accept/reject/reassign.
- **Applicant/student view**: "Transcript from {school} received / accepted" status.
- States: empty inbox, quarantined (unsafe), parse-failed (manual), unmatched, accepted.
- Accessibility: document viewer + match panel keyboard/SR navigable.
- i18n for statuses and notifications.

## 11. AI / ML Considerations

- **Optional** for PDF-only inbound: LLM/OCR-assisted extraction of student/course fields to pre-fill the review form. Non-authoritative — always human-confirmed. PII redaction before any model call; cost-capped; deterministic PESC path preferred whenever XML is present. Fallback = manual entry.

## 12. Integration Points

- **Internal:** shared PESC parser (T06), applicant/student records, object storage, malware scanner, job queue, notifications (T10), audit log.
- **External:** exchange network inbound (PESC EdExchange / Clearinghouse), SFTP, inbound email (SES inbound / mailbox).
- **Emissions:** `transcript.inbound.received/matched/accepted/rejected`.

## 13. Dependencies & Sequencing

- After: T06 (parser + infra). Independent of the outbound order flow otherwise.
- Before: transfer-credit articulation (future, out of scope).
- Shared infra: object storage, malware scan, queue, inbound email.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Malicious inbound payload (XXE, zip bomb, malware) | M | H | Hardened parser (no external entities), size/type limits, sandbox, malware scan, quarantine-first |
| Wrong applicant match (privacy breach) | M | H | Confidence threshold + human confirmation for low confidence; reversible + audited matches |
| PDF-only extraction errors | H | M | Flag for manual review; never auto-accept unparsed; optional AI pre-fill is advisory |
| Duplicate/late re-sends | M | L | Dedupe on (sender, external_ref) |

## 15. Rollout Plan

- Flag `ff_transcript_inbound` (default off) — ships dark.
- Sequence: schema + intake API → sandbox/validation → PESC parse + match → intake UI → SFTP/email channels → optional AI PDF pre-fill.
- Pilot: one partner school sends real PESC into a sandbox org; validate parse + match + accept.
- Rollback: disable flag; retain received documents read-only.

## 16. Test Plan

- **Unit** — parser (valid/invalid/malicious PESC); matching + confidence; dedupe.
- **Integration** — intake → parse → match → accept; quarantine on unsafe; unmatched → manual assign.
- **E2E** — partner sends PESC → appears in inbox → registrar accepts → applicant notified → courses available.
- **Security** — XXE/entity-expansion/zip-bomb defenses; malware scan; auth on inbound; RBAC on queue.
- **Accessibility** — inbox + viewer axe + keyboard/SR.
- **Performance** — batch SFTP intake throughput; parse latency.

## 17. Documentation & Training

- Interop guide: inbound endpoints/auth, supported formats, sender onboarding.
- Admissions/registrar runbook: working the intake queue, matching, credit-eval hand-off.
- Student help: how received transcripts are confirmed.

## 18. Open Questions

1. Which inbound channels at launch (network peer only, or SFTP + email too)?
2. Where does transfer-credit articulation live (separate plan) and what's the exact hand-off contract?
3. Retention/TTL for unmatched inbound PII.

## 19. References

- Existing: shared PESC serializer (T06), object storage, `server/internal/workers`, notifications, applicant/student repos.
- Standards: PESC XML, secure XML parsing (OWASP XXE prevention).
- Related plans: [T06](T06-electronic-delivery-standards.md), [T02](T02-recipient-directory-and-orders.md), [T10](T10-order-tracking-notifications.md).
