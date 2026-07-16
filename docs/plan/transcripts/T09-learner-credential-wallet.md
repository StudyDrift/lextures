# T09 — Learner Credential Wallet

> Implementation plan. One learner-owned home for transcripts, diplomas, certificates, badges, and CLRs — portable after graduation. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T09 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MAJOR |
| **Markets** | SL · HE · K12 |
| **Status (today)** | THIN/FRAGMENTED — credentials live in silos: CLR docs (`ccr`), competency badges (migration `375`), transcript requests (`transcripts`), CE/seat-time. There is no single place a learner sees and shares everything, and nothing survives cleanly after they leave the institution. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Learner-experience squad |
| **Depends on** | T01 (transcripts), T08 (verification), [14.13 CLR](../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md), [B1 badges](../../completed/) |
| **Unblocks** | T11 (diplomas land here); learner retention/portability |

---

## 1. Problem Statement

Parchment's "Learner" account is a lasting home for a person's credentials that they control and
can share anywhere, even after leaving school. Lextures scatters credentials across CLR, badges,
transcripts, and CE records, with no unified, portable view. Learners can't see everything they've
earned in one place or share a curated set with an employer, and they lose access when their
enrollment ends. This story unifies all credential types into one learner-owned wallet with
sharing, export, and portability.

## 2. Goals

- Aggregate all credential types (official transcripts, CLR, badges, certificates, diplomas, CE) into one **wallet** view.
- Let learners **share** individual credentials or a curated **collection** via verifiable links (T08).
- Provide **export/portability**: download all credentials (PDFs + VC JSON) as a portable bundle.
- Preserve **post-enrollment access** so alumni keep their wallet after leaving the institution.
- Give the learner **control**: visibility, disclosure level, and revocation of share links.

## 3. Non-Goals

- Issuing new credential types (transcripts T01, diplomas T11) — the wallet indexes what exists.
- The verification cryptography (T08) — the wallet consumes it.
- A cross-vendor identity wallet app — this is the in-product wallet (interop is an open question).

## 4. Personas & User Stories

- **As a learner**, I want all my credentials in one place so that I don't hunt across systems.
- **As a job seeker**, I want to share a curated set of verified credentials with an employer via one link so that they can trust them.
- **As an alum**, I want to keep access to my transcript and diploma after graduation so that I can use them later.
- **As a self-learner**, I want to export everything I've earned so that I own my record.
- **As a privacy-conscious learner**, I want to control what each shared link reveals and revoke it later.

## 5. Functional Requirements

- **FR-1.** The wallet MUST aggregate credential types: `transcript` (T01), `clr` (14.13), `badge` (B1/`375`), `certificate`, `diploma` (T11), `ce_record` (seat-time), each with issuer, issue date, and verification status.
- **FR-2.** The wallet MUST present a unified list/detail with links to download and to verify (T08).
- **FR-3.** Learners MUST be able to create a shareable **collection** (curated subset) with a verifiable public link and a chosen disclosure level.
- **FR-4.** Learners MUST be able to **revoke** any share link and see link access history.
- **FR-5.** The system MUST support **export**: a downloadable bundle (ZIP of PDFs + VC JSON + an index manifest).
- **FR-6.** Access to the wallet MUST persist after enrollment ends (alumni access), subject to retention/legal policy.
- **FR-7.** The wallet MUST reflect **revocation** of underlying credentials (a revoked transcript shows revoked).
- **FR-8.** Sharing MUST default to minimal disclosure and require explicit opt-in for fuller disclosure.
- **FR-9.** The wallet MUST be a **read/index** layer — it MUST NOT mutate source credential records.
- **FR-10.** The wallet SHOULD support standards-based export (Open Badges 3.0 / CLR 2.0 / VC) for interoperability where available.

## 6. Non-Functional Requirements

- **Performance** — wallet list p95 < 400ms; export bundle generated async for large sets.
- **Security** — share links tokenized/expiring/revocable (reuse T08); alumni access re-authentication; least-privilege reads.
- **Privacy & Compliance** — learner-controlled disclosure; FERPA/GDPR export overlaps with DSAR ([S01](../standards/S01-unified-data-subject-rights-orchestration.md)); retention for alumni ([S02](../standards/S02-data-retention-deletion-engine.md)).
- **Accessibility** — wallet + share flows WCAG 2.1 AA; works on mobile.
- **Scalability** — index view over existing tables; caching for read-heavy wallet.
- **Reliability** — wallet reflects source truth; broken source → clear error, no phantom credentials.
- **Observability** — `wallet_view_total`, `wallet_share_created/revoked`, `wallet_export_total`.
- **Maintainability** — credential providers registered via a common interface; adding a type is additive.
- **Internationalization** — labels/dates localized.
- **Backward compatibility** — existing CLR/badge/transcript views link into the wallet; no breaking changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner with a transcript, two badges, and a CLR, *When* they open the wallet, *Then* all four appear with issuer, date, and verification status.
- **AC-2.** *Given* the wallet, *When* the learner creates a shared collection of two credentials, *Then* a verifiable public link shows only those two at the chosen disclosure level.
- **AC-3.** *Given* a share link, *When* the learner revokes it, *Then* the link no longer resolves and access history is retained.
- **AC-4.** *Given* an export request, *When* processed, *Then* a bundle with PDFs + VC JSON + manifest is produced.
- **AC-5.** *Given* a learner whose enrollment ended, *When* they log in, *Then* the wallet remains accessible per policy.
- **AC-6.** *Given* a revoked underlying transcript, *When* viewed in the wallet, *Then* it shows revoked and its share link verifies as revoked.

## 8. Data Model

Migration `386_credential_wallet.sql` (indicative). The wallet is primarily an **index/view** over
existing tables, plus tables for collections and share settings:

```sql
CREATE SCHEMA IF NOT EXISTS credentials;

-- Optional materialized index for fast wallet reads (rebuilt from sources).
CREATE TABLE credentials.wallet_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN ('transcript','clr','badge','certificate','diploma','ce_record')),
    source_id     UUID NOT NULL,               -- id in the owning table
    title         TEXT NOT NULL,
    issuer        TEXT,
    issued_at     TIMESTAMPTZ,
    verify_token  TEXT,
    revoked       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kind, source_id)
);

CREATE TABLE credentials.collections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    share_token TEXT UNIQUE,
    disclosure  TEXT NOT NULL DEFAULT 'summary' CHECK (disclosure IN ('validity','summary','full')),
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE credentials.collection_items (
    collection_id UUID NOT NULL REFERENCES credentials.collections(id) ON DELETE CASCADE,
    wallet_item_id UUID NOT NULL REFERENCES credentials.wallet_items(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, wallet_item_id)
);
```

- `wallet_items` is a cache refreshed on credential issuance/revocation events (or a live UNION view if perf allows).

## 9. API Surface

- `GET  /api/v1/me/wallet` — unified credential list (kind, issuer, date, verify status).
- `GET  /api/v1/me/wallet/{itemId}` — detail + download + verify link.
- `POST /api/v1/me/wallet/collections` / `PUT` / `DELETE` — manage curated collections + share links.
- `POST /api/v1/me/wallet/collections/{id}/revoke` — revoke a share link.
- `GET  /api/v1/me/wallet/export` — async bundle (ZIP: PDFs + VC JSON + manifest).
- `GET  /wallet/s/{token}` — public shared-collection view (via T08 verifier for each item).
- OpenAPI updated.

## 10. UI / UX

- **Wallet page** (new learner surface, links from CLR `MyCCR.tsx`, badges, transcripts): grouped by kind, each card shows issuer, date, verified ✓, download, share.
- **Collections**: build/curate, set disclosure level, copy share link, view access history, revoke.
- **Export**: "Download all" → async bundle with progress.
- **Public shared view**: branded, minimal by default, per-item verify.
- States: empty wallet, source error, export generating, revoked item, expired/revoked share link.
- Accessibility: card grid + share dialog keyboard/SR; mobile-first.
- i18n for labels; alumni "your enrollment ended, wallet still yours" messaging.

## 11. AI / ML Considerations

Optional: suggest a credential collection tailored to a job description (advisory only). Non-blocking; no PII to models beyond titles the user opts to share; cost-capped. Default off.

## 12. Integration Points

- **Internal:** T01 transcripts, `ccr` CLR, badges (`375`), CE/seat-time, T08 verification, T11 diplomas, object storage, DSAR export ([S01](../standards/S01-unified-data-subject-rights-orchestration.md)), notifications.
- **External:** optional Open Badges 3.0 / CLR 2.0 export targets; external wallet import (future).
- **Emissions:** `wallet.collection.shared/revoked`, `wallet.exported`.

## 13. Dependencies & Sequencing

- After: T01, T08, 14.13, badges. With/before: T11 (diplomas index into wallet).
- Shared infra: object storage, verifier (T08), export/DSAR.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Wallet drifts from source truth | M | M | Event-driven refresh + periodic reconcile; verify status computed at read |
| Alumni access vs. retention/deletion conflict | M | H | Policy-driven retention (S02); honor deletion; distinguish institutional vs. learner-owned copies |
| Over-sharing via links | M | M | Minimal-disclosure default, expiry, revocation, access history (T08) |
| Adding a new credential type is invasive | M | L | Provider interface; additive registration |

## 15. Rollout Plan

- Flag: reuse `ff_co_curricular_transcript` (overlaps CLR) + `ff_transcripts`.
- Sequence: wallet index/read → collections + share (T08) → export → alumni access policy → public shared view.
- Pilot: learners curate and share a collection with an external reviewer.
- Rollback: hide wallet page; source credentials unaffected.

## 16. Test Plan

- **Unit** — provider aggregation; disclosure filtering; collection/share token logic.
- **Integration** — wallet reflects issuance/revocation; export bundle contents; share revoke.
- **E2E** — learner views wallet → shares collection → external viewer verifies → learner revokes.
- **Security** — share token scoping/expiry; alumni re-auth; least-disclosure; cross-user isolation.
- **Accessibility** — wallet + share dialog + public view axe + keyboard/SR.
- **Performance** — wallet list latency; async export.

## 17. Documentation & Training

- Learner help: "Your credential wallet," sharing collections, exporting, alumni access.
- Admin: retention/alumni-access policy configuration.
- API reference for wallet endpoints.

## 18. Open Questions

1. Live UNION view vs. materialized `wallet_items` cache — which meets the perf/consistency bar?
2. Alumni access model: free-forever learner account vs. institution-scoped with export-on-exit?
3. External wallet interop (import/export to third-party VC wallets) at launch or later?

## 19. References

- Existing: `ccr` schema + `clients/web/src/pages/lms/MyCCR.tsx`, competency badges (migration `375`), CE/seat-time (`CeTranscript.tsx`), `service/vc_signing`.
- Standards: Open Badges 3.0, Comprehensive Learner Record 2.0, W3C Verifiable Credentials.
- Related plans: [T01](T01-official-transcript-generation.md), [T08](T08-credential-verification.md), [T11](T11-diploma-certificate-issuance.md), [14.13](../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md).
