# T11 — Diploma & Digital Certificate Issuance

> Implementation plan. Institution-issued, verifiable diplomas and certificates that land in the learner wallet. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T11 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MINOR |
| **Markets** | HE · K12 |
| **Status (today)** | MISSING — Lextures issues no diplomas or completion certificates. Badges (`375`) and CLR exist, but there is no formal, verifiable diploma/certificate an institution confers on program completion. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Registrar/Credentials squad |
| **Depends on** | T08 (verification/signing), T09 (wallet), [B1 badges](../../completed/) |
| **Unblocks** | Complete credential set; wallet completeness |

---

## 1. Problem Statement

Beyond transcripts, institutions confer **diplomas** (degree/graduation) and **certificates**
(program/CE completion). Parchment issues digital, verifiable versions of these. Lextures has
badges and a CLR but no formal diploma/certificate issuance, so a graduating student can't receive
a verifiable digital diploma. This story adds templated, signed, verifiable diploma/certificate
issuance driven by program completion, delivered into the wallet (T09) and verifiable via T08.

## 2. Goals

- Let registrars define **diploma/certificate templates** (degree, program, honors, seal, signatures).
- **Issue** a diploma/certificate to a learner on program/degree completion (manual or rule-triggered).
- Produce a **signed, verifiable** artifact (VC via T08) with a PDF and a verify link.
- Deliver issued credentials into the **learner wallet** (T09) and optionally via the order/delivery pipeline (T06).
- Support **batch issuance** (e.g. a graduating cohort) and revocation.

## 3. Non-Goals

- Transcript generation (T01) and CLR (14.13).
- Micro-credential badges (already shipped, `375`) — diplomas/certificates are distinct, formal credentials.
- Automated degree-audit/graduation eligibility engine — completion is asserted by the registrar or an existing signal.

## 4. Personas & User Stories

- **As a registrar**, I want to design a diploma template and issue it to graduates so that they get a formal credential.
- **As a graduate**, I want a verifiable digital diploma in my wallet so that I can share proof of my degree.
- **As an employer**, I want to verify a diploma is genuine so that I can trust the candidate.
- **As a program admin**, I want to issue completion certificates for a course/program so that finishers get recognition.
- **As a registrar**, I want to revoke a mistakenly issued diploma so that verification then fails.

## 5. Functional Requirements

- **FR-1.** Registrars MUST be able to create/manage diploma and certificate **templates** (type, title, program, conferral text, seal, signature images, layout).
- **FR-2.** The system MUST **issue** a credential to a learner: manually (single or batch by cohort/program) and optionally rule-triggered on a completion signal.
- **FR-3.** Each issued credential MUST be rendered to a **PDF** and **signed as a VC** (T08) with a verify token and content hash.
- **FR-4.** Issued credentials MUST appear in the learner **wallet** (T09) and MAY be delivered via the order pipeline (T06).
- **FR-5.** The system MUST support **revocation/unrevocation** (RBAC, audited); revoked credentials verify as revoked (T08).
- **FR-6.** Issuance MUST be **idempotent** per (learner, template, program instance) to avoid duplicates in batch runs.
- **FR-7.** Credentials MUST be **immutable** once issued; corrections create a new version + revoke the prior.
- **FR-8.** Diplomas MUST record conferral metadata: degree/credential, program, honors, conferral date, issuing authority.
- **FR-9.** The system SHOULD support standards-based export (Open Badges 3.0 / CLR 2.0 / VC) for portability (via T09).
- **FR-10.** The feature MUST ship behind `ff_diplomas` and be off by default.

## 6. Non-Functional Requirements

- **Performance** — single issuance p95 < 3s; batch via job queue.
- **Security** — signing keys via T08 custody; template assets access-controlled; issuance RBAC-gated.
- **Privacy & Compliance** — diplomas are education records; FERPA-aware; disclosure logged; retention per policy.
- **Accessibility** — PDF/UA diplomas; issuance UI and wallet card WCAG 2.1 AA.
- **Scalability** — cohort batch issuance via queue; idempotent.
- **Reliability** — batch resumable; idempotent; immutable artifacts hash-verified.
- **Observability** — `diploma_issued_total{type}`, `diploma_revoked_total`, batch success/fail.
- **Maintainability** — shares rendering/signing services with T01/T08.
- **Internationalization** — template text/labels localizable; RTL support for layouts.
- **Backward compatibility** — additive; no impact on badges/CLR.

## 7. Acceptance Criteria

- **AC-1.** *Given* a diploma template, *When* a registrar issues it to a graduate, *Then* a signed PDF + VC is created, verifiable via T08, and appears in the learner's wallet.
- **AC-2.** *Given* a cohort batch issuance, *When* run, *Then* each learner gets exactly one credential (idempotent) and failures are reported/resumable.
- **AC-3.** *Given* an issued diploma, *When* revoked, *Then* verification returns "revoked."
- **AC-4.** *Given* a correction, *When* re-issued, *Then* a new version is created and the prior is revoked; both are traceable.
- **AC-5.** *Given* `ff_diplomas` off, *When* issuance endpoints are called, *Then* they return not-enabled.
- **AC-6.** *Given* an issued certificate, *When* the learner opens the wallet, *Then* it shows with issuer, date, and verified status.

## 8. Data Model

Migration `388_diplomas_certificates.sql` (indicative):

```sql
CREATE TABLE credentials.credential_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('diploma','certificate')),
    name        TEXT NOT NULL,
    layout      JSONB NOT NULL,              -- fields, seal/signature asset keys, text
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE credentials.diplomas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    org_id        UUID NOT NULL REFERENCES tenant.organizations(id),
    template_id   UUID REFERENCES credentials.credential_templates(id),
    kind          TEXT NOT NULL CHECK (kind IN ('diploma','certificate')),
    credential_title TEXT NOT NULL,
    program       TEXT,
    honors        TEXT,
    conferred_at  TIMESTAMPTZ NOT NULL,
    version       INT NOT NULL DEFAULT 1,
    canonical     JSONB NOT NULL,
    content_hash  TEXT NOT NULL,
    pdf_key       TEXT,
    vc_proof      JSONB,
    verify_token  TEXT UNIQUE,
    revoked_at    TIMESTAMPTZ,
    revoke_reason TEXT,
    issued_by     UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    issued_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    program_ref   UUID,                        -- idempotency: program/cohort instance
    UNIQUE (user_id, template_id, program_ref)
);
CREATE INDEX idx_diplomas_user ON credentials.diplomas (user_id, issued_at DESC);

ALTER TABLE settings.platform_app_settings ADD COLUMN IF NOT EXISTS ff_diplomas BOOLEAN;
```

## 9. API Surface

- `GET/POST/PUT /api/v1/admin/credentials/templates` — template management (RBAC).
- `POST /api/v1/admin/credentials/issue` — issue to one learner `{userId, templateId, program, honors, conferredAt}`.
- `POST /api/v1/admin/credentials/issue/batch` — cohort/program batch (async).
- `POST /api/v1/admin/credentials/{id}/revoke|unrevoke`.
- `GET  /api/v1/me/credentials` — my diplomas/certificates (also surfaced via wallet T09).
- `GET  /api/v1/me/credentials/{id}/download` — signed PDF.
- Verification via T08 (`/verify/{token}`). OpenAPI updated; add `FFDiplomas` to `platformconfig`.

## 10. UI / UX

- **Registrar template designer**: define layout, upload seal/signatures, preview.
- **Issue flow**: pick learner(s) or cohort/program, set conferral details, preview, issue; batch progress view.
- **Learner**: diploma/certificate card in wallet (T09) with download + verify + share.
- States: template empty, issuing/batch progress, issued, revoked, corrected (new version), feature-off.
- Accessibility: designer + issue flow keyboard/SR; PDF/UA output.
- i18n for template text and UI.

## 11. AI / ML Considerations

Optional: AI-assisted template copy/layout suggestions (advisory, registrar-approved). No PII to models; cost-capped; off by default. Issuance itself is deterministic.

## 12. Integration Points

- **Internal:** T01/T08 rendering + signing, T09 wallet, program/course completion signals, badges (`375`), object storage, job queue, audit log.
- **External:** optional Open Badges 3.0 / CLR issuance endpoints.
- **Emissions:** `credential.diploma.issued/revoked` (T09 wallet + T08 verify consume).

## 13. Dependencies & Sequencing

- After: T08 (signing/verify), T09 (wallet target).
- Shared infra: rendering, signing keys/DID, object storage, queue.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate issuance in batch | M | M | Idempotency on (user, template, program_ref); resumable batch |
| Wrong conferral (before eligibility) | M | H | Registrar review/approve; completion signal + manual confirm; revoke path |
| Template asset (seal/signature) misuse | L | H | Access-controlled assets; RBAC on issuance; audit |
| Immutable artifact vs. corrections | M | M | New version + revoke prior; both traceable |

## 15. Rollout Plan

- Flag `ff_diplomas` (default off) — ships dark.
- Sequence: templates → single issuance + signing/verify → wallet surfacing → batch issuance → revocation → optional delivery via T06.
- Pilot: issue a small cohort in a sandbox org; verify wallet + verification.
- Rollback: disable flag; issued credentials retained/verifiable.

## 16. Test Plan

- **Unit** — issuance idempotency; versioning/revocation; hash/signature.
- **Integration** — issue → wallet + verify; batch idempotency + resume; revoke → verify revoked.
- **E2E** — registrar designs template → issues cohort → learner sees + verifies diploma.
- **Security** — RBAC on issuance/templates; asset access; signing custody.
- **Accessibility** — designer/issue UI + PDF/UA.
- **Performance** — single vs. batch throughput.

## 17. Documentation & Training

- Registrar: template design, issuance (single/batch), revocation.
- Learner help: your diploma/certificate, verifying, sharing.
- API reference for credential/template endpoints.

## 18. Open Questions

1. Completion trigger source: manual only at launch, or wire to a degree-audit/program-completion signal?
2. Adopt Open Badges 3.0 / CLR 2.0 as the issuance format for interoperability?
3. Should diplomas be orderable/deliverable (T06) or wallet-only initially?

## 19. References

- Existing: `service/vc_signing`, competency badges (migration `375_competency_badges`), `ccr` CLR, object storage, `platformconfig` feature flags.
- Standards: W3C Verifiable Credentials, Open Badges 3.0, CLR 2.0, PDF/UA.
- Related plans: [T08](T08-credential-verification.md), [T09](T09-learner-credential-wallet.md), [T01](T01-official-transcript-generation.md).
