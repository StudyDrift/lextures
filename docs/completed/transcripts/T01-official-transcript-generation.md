# T01 — Official Academic Transcript Generation

> Implementation plan. Foundation for the transcript platform. Extends the thin request feature (migrations 263–265). Source landscape: [transcripts/README](../../plan/transcripts/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T01 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | BLOCKER |
| **Markets** | HE · K12 |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Registrar/SIS squad (Backend) + PDF/rendering |
| **Depends on** | Gradebook, enrollments, sections (5.4), grading scales, `vc_signing` |
| **Unblocks** | T02, T03, T06, T08, T11 |

---

## 1. Problem Statement

Today Lextures can *ask* an institution for a transcript but cannot *produce* one — there is no
code that turns a student's enrollments and grades into an official academic record. Registrars
cannot adopt Lextures as a system of record for transcripts without this, and every downstream
feature (ordering, delivery, verification) is blocked. Building a canonical academic-record
model plus deterministic PDF and PESC-XML renderers makes Lextures the authoritative issuer.

## 2. Goals

- Assemble a canonical, machine-readable **academic record** from enrollment + gradebook data.
- Render that record to an **official PDF** (registrar letterhead, signature, security marks).
- Emit the same record as **PESC XML College Transcript** for electronic exchange (T06).
- Persist each issued document as an **immutable, hash-sealed artifact** with a version chain.
- Support **official vs. unofficial** and **partial/in-progress vs. final** transcript variants.

## 3. Non-Goals

- Recipients, ordering, and payment (T02/T05) — this story issues documents, it does not sell them.
- Delivery / transport (T06) and third-party verification portal (T08).
- Diplomas/certificates (T11) and CLR/co-curricular content (14.13) — separate artifact types.
- Editing grades. This reads the gradebook; it never mutates academic data.

## 4. Personas & User Stories

- **As a student**, I want to preview an unofficial transcript so that I can check it before ordering an official one.
- **As a registrar**, I want the official transcript to reflect our GPA scale, credit rules, honors, and standing so that it is defensible.
- **As a receiving institution**, I want a standards-conformant PESC XML file so that it imports into my SIS.
- **As an admin**, I want issued transcripts to be immutable and hashed so that a document cannot be silently altered after issuance.
- **As a self-learner**, I want a clean unofficial record of completed courses so that I can share progress.

## 5. Functional Requirements

- **FR-1.** The system MUST assemble a canonical academic-record document per user from enrollments, sections, gradebook final grades, credit hours, and term structure.
- **FR-2.** The record MUST include: student identity block, program/plan, per-term course lines (code, title, credits attempted/earned, grade, quality points), term & cumulative GPA and credit totals, degrees/honors conferred, and academic standing.
- **FR-3.** The system MUST render the record to a PDF using the institution's configured letterhead, registrar signature image, seal, and a legend explaining grade symbols.
- **FR-4.** The system MUST render the same record to **PESC XML (AcademicRecord / College Transcript, v1.x)** validated against the schema.
- **FR-5.** Each issued document MUST be stored immutably with a SHA-256 content hash and a monotonic `version` per user; re-issuance creates a new version, never overwrites.
- **FR-6.** The system MUST distinguish `official` (sealed, signed, non-editable) from `unofficial` (watermarked "UNOFFICIAL") and MUST watermark previews.
- **FR-7.** GPA/credit computation MUST be pluggable per grading scale (4.0/percentage/SBG mastery) and MUST document rounding and inclusion rules (repeats, withdrawals, transfer credit, in-progress).
- **FR-8.** The system SHOULD support a **partial transcript** (single term or date range) and an **in-progress** variant flagging non-final grades.
- **FR-9.** Generation MUST be reproducible: the same inputs and template version MUST yield the same canonical JSON (byte-stable hash).
- **FR-10.** The system MAY sign the PDF and canonical JSON as a W3C Verifiable Credential reusing `service/vc_signing` for later verification (T08).

## 6. Non-Functional Requirements

- **Performance** — p95 < 3s to render a typical (≤4-year) transcript PDF; async job for bulk/large records.
- **Security** — official rendering only via server-side job; letterhead/signature assets access-controlled; no client-side assembly of official records.
- **Privacy & Compliance** — FERPA education record; generation is logged; release governed by consent (T04). PII minimization in logs.
- **Accessibility** — PDFs tagged (PDF/UA) with a logical reading order and text layer; the on-screen preview meets WCAG 2.1 AA.
- **Scalability** — batch issuance (e.g. end-of-term) via the job queue; idempotent per (user, version, template).
- **Reliability** — hash verified on read; corrupt artifacts fail closed.
- **Observability** — metrics `transcript_generate_total{variant,format,result}`, `transcript_generate_latency`; log record version + template version (see [observability 17.7](../../plan/17-platform-performance-operability/)).
- **Maintainability** — canonical model in one package; renderers are pure functions of the model.
- **Internationalization** — grade legends and labels externalized; locale-aware date/number formatting.
- **Backward compatibility** — additive; does not change gradebook or enrollment schemas.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student with graded enrollments across three terms, *When* an official transcript is generated, *Then* per-term and cumulative GPA/credits match a fixture computed by the grading-scale rules and the PDF renders those totals.
- **AC-2.** *Given* an issued official document, *When* any byte of stored content is altered, *Then* the stored SHA-256 no longer matches and reads fail closed.
- **AC-3.** *Given* the same inputs and template version, *When* generation runs twice, *Then* the canonical JSON hashes are identical.
- **AC-4.** *Given* a PESC XML render, *When* validated against the PESC College Transcript schema, *Then* validation passes with zero errors.
- **AC-5.** *Given* an unofficial preview, *When* rendered, *Then* it carries an "UNOFFICIAL" watermark and is not stored as an official artifact.
- **AC-6.** *Given* a re-issuance, *When* generated, *Then* `version` increments and the prior version remains retrievable.

## 8. Data Model

New table (owning story; migration `386_transcript_documents.sql`):

```sql
CREATE TABLE transcripts.transcript_documents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    org_id        UUID REFERENCES tenant.organizations(id),
    variant       TEXT NOT NULL CHECK (variant IN ('official','unofficial','partial','in_progress')),
    version       INT  NOT NULL,                       -- monotonic per (user_id, variant='official')
    canonical     JSONB NOT NULL,                      -- academic-record model (schema-versioned)
    schema_version TEXT NOT NULL,                       -- e.g. 'acadrec/1.0'
    template_version TEXT NOT NULL,
    content_hash  TEXT NOT NULL,                        -- sha256 of canonical (byte-stable)
    pdf_key       TEXT,                                 -- object storage key
    pesc_xml_key  TEXT,
    vc_proof      JSONB,                                -- optional signed VC (T08)
    gpa_cumulative NUMERIC(4,3),
    credits_earned NUMERIC(7,2),
    generated_by  UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ux_transcript_documents_official_version
    ON transcripts.transcript_documents (user_id, version) WHERE variant = 'official';
CREATE INDEX idx_transcript_documents_user ON transcripts.transcript_documents (user_id, generated_at DESC);
```

- Canonical JSON is the source of truth; PDF/XML are derived and cached in object storage.
- **Backfill**: none required; documents are generated on demand going forward.

## 9. API Surface

- `POST /api/v1/transcripts/documents` — generate (body: `{variant, terms?, format[]}`); async → returns `documentId` + job.
- `GET  /api/v1/transcripts/documents` — list my issued documents (id, variant, version, generatedAt).
- `GET  /api/v1/transcripts/documents/{id}` — metadata.
- `GET  /api/v1/transcripts/documents/{id}/download?format=pdf|xml` — signed URL (self or authorized fulfillment path).
- `GET  /api/v1/transcripts/preview` — unofficial watermarked preview (no persistence).
- `GET  /api/v1/admin/transcripts/students/{uid}/documents` — registrar view (RBAC).
- Rate limit generation per user; OpenAPI updated.

Canonical record (pseudo-TypeScript):

```ts
type AcademicRecord = {
  schemaVersion: string;
  student: { name: string; studentId?: string; birthDateMasked?: string };
  institution: { name: string; ceebActId?: string };
  program?: { degree?: string; major?: string[]; minor?: string[] };
  terms: Array<{ label: string; startedOn: string;
    courses: Array<{ code: string; title: string; creditsAttempted: number; creditsEarned: number; grade: string; qualityPoints?: number; inProgress?: boolean }>;
    termGpa?: number; termCredits?: number }>;
  cumulative: { gpa?: number; creditsAttempted: number; creditsEarned: number };
  honors?: string[]; degreesConferred?: Array<{ degree: string; conferredOn: string }>;
  standing?: string; legend: Record<string,string>;
};
```

## 10. UI / UX

- **Student → Transcripts page** (`pages/lms/transcripts-page.tsx`): add "Preview unofficial transcript" and "My issued documents" list with download.
- Preview renders the canonical record in-app (accessible HTML) plus a PDF download.
- **Registrar** (T12 console) can generate/reissue for a student.
- States: empty (no graded enrollments → explain what's needed), loading (async job spinner), error, in-progress banner when non-final grades present.
- Mobile: read-only preview + download.
- i18n keys for all labels and the grade legend.

## 11. AI / ML Considerations

None. Generation is deterministic; no model involved. (Explicitly no AI in the official-record path.)

## 12. Integration Points

- **Internal:** gradebook + final-grade computation, enrollments/sections repos, grading-scale config, `service/vc_signing`, object storage, job queue, `platformconfig` (letterhead/seal/signature assets).
- **External:** PESC XML schemas (College Transcript / AcademicRecord).
- **Emissions:** `transcript.document.generated` event (consumed by T10 notifications, T12 analytics).

## 13. Dependencies & Sequencing

- Must ship **after**: gradebook final grades + grading scales (already present).
- Must ship **before**: T02, T03, T06, T08, T11.
- Shared infra: object storage, job queue, institution branding assets.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| GPA/credit rules vary per institution and get computed wrong | H | H | Pluggable scale engine + golden fixtures per scale; registrar sign-off before "official" flips |
| PESC schema drift / version mismatch with receivers | M | H | Pin schema version; validate on generate; conformance suite in CI |
| Immutable artifacts + later grade corrections | M | M | Re-issuance = new version + amendment note; never mutate prior versions |
| PDF/UA tagging complexity | M | M | Use a tagging-capable renderer; axe/pac checks in CI |

## 15. Rollout Plan

- Flag: reuse `ff_transcripts`; add sub-config `transcripts.official_enabled` (default off) so unofficial preview ships first.
- Sequence: schema → canonical model + unofficial preview → PDF renderer → PESC XML → official sealing → enable per pilot org.
- Pilot: 1 registrar cohort validates official output against their SIS golden set.
- GA: after fixture parity + PESC conformance + a11y pass.
- Rollback: disable `official_enabled`; unofficial preview unaffected.

## 16. Test Plan

- **Unit** — GPA/credit computation per scale; canonical hash stability; variant/watermark logic.
- **Integration** — generate from seeded enrollments; artifact persistence + hash verification on read.
- **E2E** — student preview → generate official → download PDF + XML.
- **Security** — authz (student can only generate own; registrar via RBAC); signed-URL scoping; tamper detection.
- **Accessibility** — axe on preview; PDF/UA validation (PAC/veraPDF).
- **Performance** — p95 render; bulk term-close batch throughput.
- **Standards** — PESC schema validation fixtures.

## 17. Documentation & Training

- Student help: "Preview vs. official transcript," what's included, how GPA is computed.
- Registrar runbook: grade-scale config, letterhead/seal upload, reissue/amendment process.
- API reference for `documents` endpoints; PESC mapping doc.

## 18. Open Questions

1. Which PESC version(s) must we certify against first (College Transcript vs. High School Transcript for K12)?
2. Do we sign every official PDF as a VC by default, or only when a verification link is requested (T08)?
3. How are transfer-credit and repeat/replace policies represented in the canonical model (link to 3.9 drop/replace rules)?
4. Masking policy for DOB/SSN-like identifiers on the rendered PDF.

## 19. References

- Existing: `server/internal/httpserver/transcripts_http.go`, `server/internal/repos/transcripts/repo.go`, `server/internal/service/masterytranscriptpdf/`, `server/internal/service/vc_signing/`, `clients/web/src/pages/lms/transcripts-page.tsx`.
- Standards: PESC (Postsecondary Electronic Standards Council) AcademicRecord/College Transcript XML; PDF/UA (ISO 14289).
- Related plans: [T02](T02-recipient-directory-and-orders.md), [T06](T06-electronic-delivery-standards.md), [T08](T08-credential-verification.md), [14.13 CLR](../14-higher-ed-specific/14.13-co-curricular-transcript.md).
