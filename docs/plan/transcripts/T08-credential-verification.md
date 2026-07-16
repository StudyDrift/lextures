# T08 — Credential Verification & Tamper-Evidence

> Implementation plan. Digitally signed, QR-verifiable documents + a third-party verifier portal. Builds on the shipped CLR/VC signing. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T08 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MAJOR |
| **Markets** | HE · K12 · SL |
| **Status (today)** | PARTIAL — VC signing + public verify + DID exist for the **CLR** (`service/vc_signing`, `/api/v1/verify/{shareToken}`, `/.well-known/did.json`, migration `259`). Official transcripts are **not** signed or verifiable, and there is no unified verifier portal for transcripts/diplomas. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Trust/Credentials squad |
| **Depends on** | T01 (documents to sign), [14.13 CLR](../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md) (VC infra) |
| **Unblocks** | T09 (wallet trust), T11 (diploma verification) |

---

## 1. Problem Statement

A transcript is only useful if a recipient can trust it's genuine and unaltered. Parchment's
"Blue Ribbon" model makes every document tamper-evident and instantly verifiable. Lextures already
signs CLR documents as W3C Verifiable Credentials and exposes a verify endpoint + DID — but that
trust layer doesn't cover official transcripts (T01) or diplomas (T11), and there's no unified,
branded verifier experience for a third party who receives a PDF. This story extends the existing
VC infrastructure to all credential types and ships a proper verifier portal with tamper detection.

## 2. Goals

- **Digitally sign** official transcripts (and later diplomas) reusing `service/vc_signing` + institution DID.
- Embed **tamper-evidence** in delivered PDFs: content hash, PAdES signature, and a **QR/verify link**.
- Ship a **third-party verifier portal** (`/verify/...`) that confirms authenticity, issuer, issue date, and integrity — for transcripts, CLRs, and diplomas.
- Detect and clearly report **tampering / revocation** without exposing the full record unless authorized.
- Maintain a **verification audit log** of who verified what and when.

## 3. Non-Goals

- Re-implementing VC signing (reuse the CLR implementation).
- The wallet UI (T09) or diploma issuance (T11) — this provides their trust primitive.
- Full decentralized-identity ecosystem / blockchain anchoring (may be a future option).

## 4. Personas & User Stories

- **As an employer/registrar receiving a transcript**, I want to scan a QR or open a link to confirm it's genuine and unaltered so that I can trust it.
- **As a student**, I want my official transcript to be verifiable so that recipients accept it without calling my school.
- **As an issuing registrar**, I want to revoke a mistakenly issued document so that verification then fails.
- **As a verifier**, I want to confirm the issuer is really the institution (DID) so that I'm not fooled by a forgery.
- **As a compliance officer**, I want a log of verification lookups so that we can audit access.

## 5. Functional Requirements

- **FR-1.** Official transcript documents (T01) MUST be signable as W3C Verifiable Credentials using the institution DID, reusing `service/vc_signing`.
- **FR-2.** Delivered PDFs MUST embed a content hash and a verification URL/QR that resolves to the verifier portal; MAY carry a PAdES digital signature.
- **FR-3.** The verifier portal MUST validate: issuer DID, signature, content-hash match, issue date, and revocation status, returning a clear genuine/tampered/revoked result.
- **FR-4.** The portal MUST support verification by (a) scanning the QR / opening the link, and (b) **uploading a PDF** to check its hash against issued records.
- **FR-5.** The system MUST support **revocation** (and its reversal) of issued documents; revoked documents MUST verify as revoked.
- **FR-6.** Verification results MUST reveal only minimal fields by default (issuer, type, issue date, validity), with the full record shown only to authorized viewers or when the holder opted into public disclosure.
- **FR-7.** The system MUST log verification attempts (`verifications`) with document, result, and requester context.
- **FR-8.** The unified verifier MUST handle transcripts, CLRs (existing), and diplomas (T11) via one code path.
- **FR-9.** Verification MUST work **offline-of-issuer** to the extent the VC/signature allows (cryptographic verification without needing the private issuer), with revocation requiring an online check.
- **FR-10.** The DID document (`/.well-known/did.json`) MUST expose the keys needed to verify all signed credential types.

## 6. Non-Functional Requirements

- **Performance** — verify lookup p95 < 500ms; PDF-upload hash check < 1s.
- **Security** — signature/DID verification robust; verifier portal rate-limited and abuse-resistant; no enumeration of tokens; minimal disclosure.
- **Privacy & Compliance** — verification reveals least data; lookups logged; holder controls public disclosure; FERPA-aware.
- **Accessibility** — verifier portal + QR fallback (manual code entry) WCAG 2.1 AA.
- **Scalability** — verify endpoint cacheable for genuine/immutable results; revocation check lightweight.
- **Reliability** — verification deterministic; revocation authoritative; fail-closed on signature errors.
- **Observability** — `credential_verify_total{type,result}`, revocation counts, upload-verify usage.
- **Maintainability** — one verification service across credential types; key rotation supported.
- **Internationalization** — verifier UI localized; date/issuer display locale-aware.
- **Backward compatibility** — existing CLR `verify/{shareToken}` continues to work; unified under the same portal.

## 7. Acceptance Criteria

- **AC-1.** *Given* a signed official transcript, *When* a verifier opens its QR link, *Then* the portal shows genuine, issuer DID, type, and issue date.
- **AC-2.** *Given* a tampered PDF (any byte changed), *When* uploaded to the verifier, *Then* the hash mismatch is detected and reported as not authentic.
- **AC-3.** *Given* a revoked document, *When* verified, *Then* the result is "revoked" with revocation date.
- **AC-4.** *Given* a verification lookup, *When* it completes, *Then* a `verifications` row records document, result, and context.
- **AC-5.** *Given* minimal-disclosure default, *When* an unauthorized verifier looks up a document, *Then* only issuer/type/date/validity are shown, not the full record.
- **AC-6.** *Given* the existing CLR verify flow, *When* used after this ships, *Then* it still verifies correctly through the unified portal.

## 8. Data Model

Migration `385_transcript_verifications.sql` (indicative):

```sql
CREATE TABLE transcripts.verifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID,                        -- transcript_documents.id or credentials.diplomas.id
    document_type TEXT NOT NULL CHECK (document_type IN ('transcript','clr','diploma')),
    result       TEXT NOT NULL CHECK (result IN ('genuine','tampered','revoked','not_found')),
    method       TEXT NOT NULL CHECK (method IN ('link','qr','upload')),
    requester_ip INET,
    requester_ua TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_verifications_doc ON transcripts.verifications (document_id, created_at);

-- Revocation + verify token on issued transcripts (extends T01):
ALTER TABLE transcripts.transcript_documents
    ADD COLUMN verify_token TEXT UNIQUE,
    ADD COLUMN revoked_at   TIMESTAMPTZ,
    ADD COLUMN revoke_reason TEXT;
```

- Reuse `ccr.documents.vc_proof`/DID patterns; store transcript `vc_proof` in the T01 column.

## 9. API Surface

- `GET  /verify` and `GET /verify/{token}` — unified verifier portal (public, rate-limited).
- `POST /api/v1/verify/upload` — upload a PDF; returns authenticity by hash match.
- `POST /api/v1/admin/transcripts/documents/{id}/revoke|unrevoke` — issuer revocation (RBAC, audited).
- `GET  /.well-known/did.json` — extended to cover all signed types (existing route).
- Existing `GET /api/v1/verify/{shareToken}` (CLR) preserved / unified.
- OpenAPI updated.

## 10. UI / UX

- **Verifier portal** (public, institution-branded): scan/enter code or upload PDF → result card (genuine ✓ / tampered ✗ / revoked), issuer, type, issue date, "verified by Lextures" trust mark; minimal fields unless authorized.
- **Delivered PDF**: visible verify QR + short code + "Verify at …" footer (added in T06 render).
- **Student**: "Verification link" per issued document; toggle public disclosure level.
- **Registrar**: revoke/unrevoke action with reason.
- States: verifying, genuine, tampered, revoked, not-found, rate-limited.
- Accessibility: QR has manual-code fallback; result card has text status + icon; WCAG 2.1 AA.
- i18n for portal + result copy.

## 11. AI / ML Considerations

None. Verification is cryptographic/deterministic; no model.

## 12. Integration Points

- **Internal:** `service/vc_signing`, existing CLR verify (`ccr_http.go`), DID route, T01 documents, T06 PDF embedding, T11 diplomas, T09 wallet, audit log.
- **External:** optional interop with open-badge/CLR verifier standards; PAdES for PDF signatures.
- **Emissions:** `credential.verified`, `credential.revoked`.

## 13. Dependencies & Sequencing

- After: T01 (documents), reuse 14.13. Before/with: T09 (wallet trust), T11 (diploma verification).
- Shared infra: signing keys/DID, object storage.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Signing key compromise | L | H | Key rotation + DID key versioning; HSM/KMS custody; revoke + re-issue path |
| Token enumeration / scraping | M | M | High-entropy tokens, rate-limiting, no listing, minimal disclosure |
| Over-disclosure on verify | M | H | Least-data default; holder-controlled disclosure; authz for full record |
| Revocation not reflected offline | M | M | Clearly mark cryptographic vs. revocation-status checks; require online for revocation |

## 15. Rollout Plan

- Flag `ff_transcripts` (transcripts) / `ff_co_curricular_transcript` (CLR reuse).
- Sequence: extend signing to transcripts → verifier portal (link+QR) → PDF embedding (with T06) → upload-verify → revocation → unify CLR path.
- Pilot: employer verifies a signed transcript; registrar tests revoke.
- Rollback: disable QR/verify embedding; documents remain valid, just not self-verifiable.

## 16. Test Plan

- **Unit** — signature verify; hash match; revocation state; minimal-disclosure filter.
- **Integration** — sign → verify genuine; mutate → tampered; revoke → revoked; CLR unified path.
- **E2E** — deliver signed PDF (T06) → scan QR → verifier result; upload tampered PDF → not authentic.
- **Security** — token entropy/rate-limit; enumeration resistance; authz for full record; key rotation.
- **Accessibility** — portal + QR fallback axe + keyboard/SR.
- **Performance** — verify latency; cache behavior.

## 17. Documentation & Training

- Verifier help: "How to verify a Lextures transcript" (QR, link, upload).
- Registrar: revocation policy and effects.
- Trust/security page describing the signing + DID model.

## 18. Open Questions

1. Adopt an external open standard (Open Badges 3.0 / CLR 2.0 / W3C VC-JWT) for cross-wallet interoperability?
2. Anchor issuance to a public ledger for extra tamper-evidence, or keep DID-only?
3. Default disclosure level for transcript verification (validity-only vs. summary).

## 19. References

- Existing: `server/internal/service/vc_signing/`, `server/internal/httpserver/ccr_http.go` (`/api/v1/verify/{shareToken}`, `/.well-known/did.json`), migration `259`, [14.13 CLR](../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md).
- Standards: W3C Verifiable Credentials, DID, PAdES (PDF signatures), Open Badges 3.0 / Comprehensive Learner Record 2.0.
- Related plans: [T01](T01-official-transcript-generation.md), [T06](T06-electronic-delivery-standards.md), [T09](T09-learner-credential-wallet.md), [T11](T11-diploma-certificate-issuance.md).
