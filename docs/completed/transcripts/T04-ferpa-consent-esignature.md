# T04 — FERPA Consent & E-Signature Authorization

> Implementation plan. Signed, scoped, auditable release authorization before education records leave the institution. Source landscape: [transcripts/README](../../plan/transcripts/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T04 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | BLOCKER |
| **Markets** | HE · K12 |
| **Status (today)** | DONE — FERPA e-signature consent per order; hard gate at `pending_consent`; guardian path for minors; revoke + audit export. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Compliance + Registrar/SIS squad |
| **Depends on** | T02 (orders/recipients); complements [S09 FERPA hardening](../../plan/standards/S09-ferpa-hardening.md), [10.1](../10-compliance-privacy-security/10.1-ferpa-workflow.md) |
| **Unblocks** | T03 (consent gate), T06 (release) |

---

## 1. Problem Statement

Under FERPA (34 CFR §99.30), an institution generally needs the eligible student's signed and
dated consent — specifying the records, the purpose, and the recipient — before disclosing
education records to a third party. Lextures captures none of this today. Sending a transcript to
another school or employer without a logged, scoped authorization exposes every institutional
customer to a compliance finding. This story captures a legally sufficient, e-signed, scoped, and
auditable release for each order (and each recipient), and wires it as a hard gate.

## 2. Goals

- Capture a **signed release authorization** per order, scoped to specific recipients and record set.
- Support a compliant **e-signature** (typed/drawn) with identity, timestamp, IP, and consent-text version.
- Handle **minors/K-12**: route authorization to a parent/guardian when the student is not the eligible adult.
- Make consent a **hard gate** in the order state machine (T03) — nothing releases without it.
- Provide an **immutable audit record** and let students **revoke** consent before delivery.

## 3. Non-Goals

- General platform-wide consent ledger (that is [S04](../../plan/standards/S04-unified-consent-preference-ledger.md)); this integrates with it but scopes to transcript release.
- Directory-information opt-out and broader §99 provisions (owned by [S09](../../plan/standards/S09-ferpa-hardening.md)/[10.1](../10-compliance-privacy-security/10.1-ferpa-workflow.md)).
- Payment (T05), transport (T06).

## 4. Personas & User Stories

- **As a student**, I want to review exactly what will be sent and to whom, then sign to authorize it, so that I control my records.
- **As a parent/guardian of a minor**, I want to authorize a K-12 transcript release so that my child's records are protected.
- **As a registrar**, I want a signed authorization on file for every third-party release so that we pass a FERPA audit.
- **As a compliance officer**, I want an immutable, exportable consent record with the exact text version signed so that disclosures are defensible.
- **As a student**, I want to revoke consent before delivery so that I can cancel a mistaken order.

## 5. Functional Requirements

- **FR-1.** Before an order leaves `pending_consent`, the system MUST capture an authorization identifying: signer, records scope, each recipient, purpose, and the consent-text version.
- **FR-2.** The system MUST record an e-signature: signer name, method (`typed`/`drawn`), captured image/text, timestamp, IP, user agent, and authentication context (that the signer was logged in).
- **FR-3.** For self-recipient delivery to the student themselves, the system MAY skip third-party authorization (FERPA does not require consent to disclose to the student) but MUST still log the action.
- **FR-4.** For minors (per org policy / DOB / role), the system MUST route authorization to a linked parent/guardian and record the guardian relationship.
- **FR-5.** The system MUST version the release-authorization text; each signature stores the exact version signed.
- **FR-6.** Consent MUST be a hard gate: T03 MUST NOT advance past `pending_consent` without a valid, unrevoked authorization covering all recipients in the order.
- **FR-7.** Students MUST be able to **revoke** authorization before delivery; revocation MUST block undelivered items and be logged.
- **FR-8.** Authorizations MUST be immutable once signed (append-only; corrections create a new authorization).
- **FR-9.** The system MUST expose an exportable consent record (PDF/JSON) for audit and DSAR ([S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md)).
- **FR-10.** The system SHOULD support an optional expiry on standing authorizations and MUST treat expired authorizations as invalid gates.

## 6. Non-Functional Requirements

- **Performance** — signature capture + persist p95 < 500ms.
- **Security** — signatures tied to authenticated session; tamper-evident (hash of authorization payload); access-controlled.
- **Privacy & Compliance** — FERPA §99.30 fields; retain per retention policy ([S02](../../plan/standards/S02-data-retention-deletion-engine.md)); minimize IP/UA exposure; ESIGN/UETA-consistent e-signature capture.
- **Accessibility** — signature UI operable by keyboard and screen reader; typed-signature alternative always available; drawn canvas has an accessible fallback.
- **Scalability** — append-only table; indexed by order and user.
- **Reliability** — signing is transactional with order-state advance; no half-signed states.
- **Observability** — `transcript_consent_signed_total`, `transcript_consent_revoked_total`, gate-block counts.
- **Maintainability** — consent-text versions in source/config; one signing service.
- **Internationalization** — authorization text localizable; signed version records locale.
- **Backward compatibility** — legacy requests (self email) treated as self-disclosure; no retroactive consent required.

## 7. Acceptance Criteria

- **AC-1.** *Given* an order to a third-party school, *When* the student signs, *Then* an immutable authorization records signer, recipients, scope, text version, timestamp, IP/UA, and the order advances past `pending_consent`.
- **AC-2.** *Given* an unsigned order, *When* T03 tries to advance, *Then* it is blocked at `pending_consent`.
- **AC-3.** *Given* a minor's K-12 order, *When* authorization is required, *Then* it routes to a linked guardian and records the relationship; the student alone cannot authorize.
- **AC-4.** *Given* a signed authorization, *When* the student revokes before delivery, *Then* undelivered items are blocked and the revocation is logged.
- **AC-5.** *Given* a self-recipient order, *When* submitted, *Then* no third-party authorization is required but the disclosure-to-student is logged.
- **AC-6.** *Given* a signed authorization, *When* exported, *Then* the export shows the exact consent-text version signed.

## 8. Data Model

Migration `395_transcript_consents.sql`:

```sql
CREATE TABLE transcripts.consents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES transcripts.orders(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    signer_id     UUID NOT NULL REFERENCES "user".users(id),   -- student or guardian
    signer_role   TEXT NOT NULL CHECK (signer_role IN ('student','guardian')),
    guardian_relationship TEXT,
    recipients    JSONB NOT NULL,                               -- snapshot of authorized recipients
    scope         TEXT NOT NULL DEFAULT 'full_academic_record',
    purpose       TEXT,
    text_version  TEXT NOT NULL,                                -- authorization text version signed
    signature_method TEXT NOT NULL CHECK (signature_method IN ('typed','drawn')),
    signature_data TEXT,                                        -- typed name or drawn PNG (data URI / key)
    signed_ip     INET,
    signed_ua     TEXT,
    payload_hash  TEXT NOT NULL,                                -- sha256 of the authorization payload
    signed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at    TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ
);
CREATE INDEX idx_consents_order ON transcripts.consents (order_id);
CREATE INDEX idx_consents_user ON transcripts.consents (user_id, signed_at DESC);
```

- Add FK `transcripts.orders.consent_id → transcripts.consents(id)` (deferred FK from T02).
- Records are append-only; revocation sets `revoked_at` only.
- Consent-text versions stored in config/source and referenced by `text_version`.

## 9. API Surface

- `GET  /api/v1/transcripts/orders/{id}/consent/preview` — the exact authorization text + recipient/scope summary to be signed.
- `POST /api/v1/transcripts/orders/{id}/consent` — submit signature `{method, signatureData, agree:true}` → creates authorization, advances state.
- `POST /api/v1/transcripts/orders/{id}/consent/revoke` — revoke before delivery.
- `GET  /api/v1/transcripts/orders/{id}/consent/export` — PDF/JSON audit copy.
- `POST /api/v1/guardian/transcripts/orders/{id}/consent` — guardian-signed path (guardian-linked auth).
- OpenAPI + links to S04 consent ledger emission.

## 10. UI / UX

- **Consent step** in the order flow (after review, before payment): shows recipients, records scope, purpose, full FERPA authorization text; typed or drawn signature; explicit "I authorize" checkbox; date auto-stamped.
- **Guardian flow** for minors: order enters a "waiting for guardian" state; guardian receives a link (T10) and signs.
- **Order detail**: shows "Authorized by … on …," with a "Revoke authorization" action while undelivered.
- States: text loading, signature validation (non-empty), submit success, revoke confirm, guardian-pending banner.
- Accessibility: typed signature is the default accessible path; drawn canvas labeled with fallback; focus order defined.
- Mobile: signature capture works on touch; i18n for all legal copy.

## 11. AI / ML Considerations

None. Legal text is fixed, versioned, and human-reviewed. No AI in the authorization path.

## 12. Integration Points

- **Internal:** T02 orders, T03 state gate, guardian/parent linkage, [S04 consent ledger](../../plan/standards/S04-unified-consent-preference-ledger.md) (emit a ledger entry), audit log, retention ([S02](../../plan/standards/S02-data-retention-deletion-engine.md)), DSAR export ([S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md)).
- **External:** none required (self-hosted e-signature); ESIGN/UETA compliance is procedural.
- **Emissions:** `transcript.consent.signed/revoked` → S04 ledger + T10.

## 13. Dependencies & Sequencing

- After: T02. Concurrent with T03 (provides the consent gate). Before: T06 release.
- Shared infra: guardian linkage, consent ledger, retention engine.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| E-signature not legally sufficient | M | H | Capture ESIGN/UETA elements (intent, attribution, association, retention); legal review of text; store text version |
| Minor authorized by the wrong party | M | H | Guardian routing driven by DOB/role/org policy; block student self-auth for minors |
| Consent gate bypassed by another delivery path | L | H | Single delivery guard checks consent; deny-by-default; tests for every release path |
| Retention conflict (delete vs. audit) | M | M | Retain authorization per legal hold rules; coordinate with S02 |

## 15. Rollout Plan

- Flag `ff_transcripts`; consent step behind `transcripts.consent_required` (default **on** for third-party recipients once shipped).
- Sequence: schema → authorization text + versioning → signing service + gate → guardian path → export → enable gate.
- Pilot: registrar + counsel review the captured authorization for a sample order.
- Rollback: cannot disable the gate for third-party releases once GA (compliance); self-delivery path unaffected.

## 16. Test Plan

- **Unit** — payload hashing; text-version pinning; gate logic; minor detection.
- **Integration** — sign → advance; unsigned → blocked; revoke → block; guardian sign path; self-recipient skip + log.
- **E2E** — full order with consent step; guardian email→sign→deliver.
- **Security** — signature bound to authenticated session; cannot sign others' orders; export authz.
- **Accessibility** — signature UI axe + keyboard/SR; typed fallback.
- **Compliance** — audit export contains all §99.30 elements; retention behavior.

## 17. Documentation & Training

- Compliance doc: how Lextures satisfies §99.30 for transcript releases; e-signature evidence model.
- Registrar/help: guardian authorization for minors; revocation.
- API reference for consent endpoints.

## 18. Open Questions

1. Standing (blanket) authorizations for application services vs. per-order signing — allowed and for how long?
2. Do we require step-up authentication (re-auth/MFA) at signing for official third-party releases?
3. Exact authorization text per market (US HE vs. K-12) — legal sign-off owner.

## 19. References

- Existing: [10.1 FERPA workflow](../10-compliance-privacy-security/10.1-ferpa-workflow.md) (`server/internal/service/ferpa`), guardian/parent linkage, `service/vc_signing`.
- Standards: FERPA 34 CFR §99.30–§99.31; ESIGN Act; UETA.
- Related plans: [T02](T02-recipient-directory-and-orders.md), [T03](T03-order-lifecycle-fulfillment-holds.md), [S04](../../plan/standards/S04-unified-consent-preference-ledger.md), [S09](../../plan/standards/S09-ferpa-hardening.md).
- Shipped: migration `395`, `service/transcriptconsent`, `repos/transcripts/consents.go`, `transcripts_consent_http.go`, consent step in order builder, `consent_required` config flag.
