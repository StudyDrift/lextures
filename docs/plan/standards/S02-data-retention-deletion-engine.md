# S02 — Data Retention & Deletion Schedule Engine

> Implementation plan. Hardens: [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md) (Art 17 erasure), [10.1 FERPA](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md), [10.15 backup/RPO-RTO](../../completed/10-compliance-privacy-security/10.15-backup-restore-rpo-rto.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S02 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL (Global) |
| **Status (today)** | THIN — erasure exists ad hoc per domain; no jurisdiction-aware retention schedules, no legal hold, no proof-of-deletion, backups excluded from erasure |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Backend/Data Platform + Compliance |
| **Depends on** | 10.3, 10.13 (encryption), 10.15 (backups), S05 (data inventory) |
| **Unblocks** | S01, S08, S11–S19 (every erasure/retention obligation resolves here) |

---

## 1. Problem Statement

"Storage limitation" (GDPR Art 5(1)(e)), FERPA/state retention *minimums*, COPPA's "only as long as necessary," and Quebec Law 25 / LGPD / DPDP deletion duties all pull in different directions — some laws require us to **keep** records for years, others require us to **delete** promptly, and litigation can require us to **freeze** everything. Right now deletion is scattered, backups are never purged, and we cannot prove a record was actually destroyed. Without a single retention engine that encodes each data category's minimum-keep and maximum-keep windows per jurisdiction, honours legal holds, cascades to backups and derived stores, and emits a deletion certificate, we are simultaneously over-retaining (a GDPR fine) and under-retaining (a records-law violation), with no evidence either way.

## 2. Goals

- A **retention schedule registry**: per data category × jurisdiction × tenant-policy, defining minimum-keep, maximum-keep, and disposition (delete / anonymise / archive).
- A **legal-hold** mechanism that overrides deletion for named subjects/matters and is auditable.
- A **deletion executor** that cascades across primary DB, derived stores (search, analytics, embeddings), object storage, and **backups** (crypto-shredding where hard-delete is infeasible).
- **Proof of deletion**: a signed certificate per executed disposition, linkable from an S01 erasure case.
- Tenant-configurable overrides within legal bounds (a district can keep longer, never shorter than the statutory minimum).

## 3. Non-Goals

- Deciding *which* data is PII — that inventory is S05 (this engine consumes it).
- The rights-request workflow that *triggers* an erasure — that is S01.
- Backup infrastructure itself (10.15); this plan adds purge/crypto-shred hooks to it.

## 4. Personas & User Stories

- **As a compliance officer**, I want each data category to have an explicit retention rule per jurisdiction so that we neither over- nor under-retain.
- **As legal counsel**, I want to place a hold on a student's records during a dispute so that nothing is deleted until I release it.
- **As a data subject**, I want confirmation my data was actually deleted (including from backups) so that erasure is real, not cosmetic.
- **As a district admin**, I want to extend retention to match our records policy so long as it exceeds the legal minimum.
- **As an auditor**, I want a log of every scheduled disposition with proof so that I can verify storage-limitation compliance.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain a `retention_schedule` registry keyed by `(data_category, jurisdiction)` with `min_keep`, `max_keep`, and `disposition`.
- **FR-2.** The system MUST compute an effective rule per record as the **intersection** of applicable jurisdictions (never delete before the longest minimum; never keep past the shortest maximum unless a longer minimum forces it — surfacing conflicts for human resolution).
- **FR-3.** A scheduled **disposition worker** MUST scan for records past `max_keep` and execute the disposition, skipping any under legal hold.
- **FR-4.** The system MUST support **legal holds** (subject- or matter-scoped) that block all disposition and erasure for matched records until explicitly released, with both actions logged.
- **FR-5.** Deletion MUST cascade to derived stores: full-text search index, analytics warehouse, recommendation/embedding vectors, caches, and object storage.
- **FR-6.** For backups where selective delete is infeasible, the system MUST use **crypto-shredding** (destroy the per-subject/tenant data key) and record it as the disposition method.
- **FR-7.** The system MUST emit a signed **deletion certificate** (subject, categories, method, timestamp, operator/job id) retrievable by S01.
- **FR-8.** "Anonymise" disposition MUST irreversibly de-identify to the GDPR Recital 26 standard (no re-identification via retained keys) rather than hard-delete, where analytics value must be preserved.
- **FR-9.** All schedule changes, holds, and dispositions MUST be written to `admin_audit_log` (10.11).

## 6. Non-Functional Requirements

- **Performance** — Disposition worker batches; must process a 1M-row category within a maintenance window without blocking OLTP (throttled, off-peak).
- **Security** — Only `data:retention_admin` may edit schedules; only `legal:hold_admin` may place/release holds; crypto-shred key destruction requires two-person approval for tenant-wide scope.
- **Privacy & Compliance** — GDPR Art 5(1)(e), 17; FERPA/state minimums; COPPA §312.10; Quebec Law 25; LGPD Art 16; DPDP §8(7). Anonymisation meets Recital 26.
- **Accessibility** — Admin schedule/hold UIs meet WCAG 2.1 AA.
- **Scalability** — Registry evaluated in bulk; incremental scans via `deleted_watermark` per category.
- **Reliability** — Dispositions idempotent and resumable; a crash mid-cascade re-runs safely; certificate written only after all cascade steps confirm.
- **Observability** — `retention_dispositions_total{category,method}`, `legal_holds_active`, `retention_conflicts_total`; alert on worker lag or conflict backlog.
- **Maintainability** — Engine in `server/internal/service/retention/`; each derived-store cascade is a pluggable `Purger` interface.
- **Internationalization** — Certificates and admin copy localised.
- **Backward compatibility** — Ships with schedules that match current implicit behaviour (effectively "keep") so nothing is deleted until a tenant/admin opts a category into active disposition.

## 7. Acceptance Criteria

- **AC-1.** *Given* a K12 grade record with a 5-year state minimum and a tenant policy of 7 years, *when* the worker runs at year 6, *then* the record is retained (max is not yet reached) and no disposition occurs.
- **AC-2.** *Given* a legal hold on student X, *when* an S01 erasure and the disposition worker both run, *then* X's records are not deleted, and both the erasure case and the worker log the hold deferral.
- **AC-3.** *Given* an executed erasure, *when* the certificate is generated, *then* it lists every category and method (including `crypto_shred` for backups) and is retrievable from the S01 case.
- **AC-4.** *Given* a record past `max_keep` with disposition `anonymise`, *when* the worker runs, *then* direct identifiers are irreversibly removed, the row survives for analytics, and re-identification via retained keys is impossible.
- **AC-5.** *Given* conflicting rules (a 10-year minimum vs a 3-year maximum), *when* the engine evaluates, *then* it retains, flags a `retention_conflict`, and surfaces it to a compliance admin rather than silently choosing.
- **AC-6.** *Given* a released legal hold, *when* the next worker run executes, *then* previously frozen records become eligible and are disposed per schedule.

## 8. Data Model

New migration `358_retention_engine.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.retention_schedules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID REFERENCES org.organizations(id),   -- null = platform default
  data_category TEXT NOT NULL,                            -- FK-by-name to S05 inventory categories
  jurisdiction  TEXT NOT NULL,                            -- 'us_ferpa','eu_gdpr','ca_quebec',...
  min_keep      INTERVAL,                                 -- statutory minimum (null = none)
  max_keep      INTERVAL,                                 -- storage-limitation cap (null = indefinite w/ basis)
  disposition   TEXT NOT NULL DEFAULT 'delete'
                  CHECK (disposition IN ('delete','anonymise','archive')),
  legal_ref     TEXT NOT NULL,
  updated_by    UUID REFERENCES "user".users(id),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (org_id, data_category, jurisdiction)
);

CREATE TABLE IF NOT EXISTS compliance.legal_holds (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID REFERENCES org.organizations(id),
  matter       TEXT NOT NULL,
  subject_id   UUID REFERENCES "user".users(id),          -- null = matter-scoped by query
  scope_query  JSONB,                                     -- optional structured selector
  placed_by    UUID NOT NULL REFERENCES "user".users(id),
  placed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  released_by  UUID REFERENCES "user".users(id),
  released_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS compliance.disposition_certificates (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID REFERENCES org.organizations(id),
  subject_id   UUID,
  rights_request_id UUID,                                 -- links to S01 when triggered by erasure
  categories   TEXT[] NOT NULL,
  method       TEXT NOT NULL,                             -- 'hard_delete','crypto_shred','anonymise'
  executed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  signature    TEXT NOT NULL                              -- HMAC/asymmetric signature over the record
);

CREATE INDEX idx_legal_holds_active ON compliance.legal_holds(subject_id) WHERE released_at IS NULL;
```

Backfill: seed `retention_schedules` platform defaults from statutory tables (documented in the runbook); default `disposition` inert until a category is activated.

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/retention/schedules` | `data:retention_admin` | Read/edit the registry |
| `POST` | `/api/v1/compliance/retention/holds` | `legal:hold_admin` | Place a legal hold |
| `DELETE` | `/api/v1/compliance/retention/holds/{id}` | `legal:hold_admin` | Release (two-person for tenant-wide) |
| `GET` | `/api/v1/compliance/retention/certificates/{id}` | admin or linked subject | Retrieve deletion proof |
| `POST` | `/api/v1/compliance/retention/dry-run` | `data:retention_admin` | Preview what a run would dispose |
| `GET` | `/api/v1/compliance/retention/conflicts` | `data:retention_admin` | Unresolved rule conflicts |

## 10. UI / UX

- **Retention admin console:** schedule grid (category × jurisdiction), conflict panel, dry-run preview with counts before any destructive run.
- **Legal-hold manager:** place/release holds, see impacted record counts, hold history.
- **Certificate viewer:** surfaced inside the S01 case and downloadable.
- States: empty (no schedules yet → guided seed), loading (dry-run counting), error (cascade partial failure with retry), confirm-modal with typed confirmation before tenant-wide disposition.
- Accessibility: destructive actions need explicit focus-managed confirmation; i18n keys `retention.*`.

## 11. AI / ML Considerations

Deletion MUST cascade to **AI-derived stores**: embedding/vector indexes, fine-tune/training corpora, and cached model outputs tied to the subject (see S06). Anonymisation must account for the re-identification risk of free-text stored in AI logs.

## 12. Integration Points

- `server/internal/service/retention/` (new); `Purger` implementations for search, analytics, `filestorage`, embeddings, cache.
- `server/internal/service/backup` + 10.15 — crypto-shred hooks; per-tenant/subject data keys via `server/internal/crypto`.
- S01 (erasure trigger + certificate link), S05 (category inventory), `adminaudit`, job scheduler (`server/internal/scheduler`).

## 13. Dependencies & Sequencing

- Must ship after: S05 (needs the category inventory) and 10.13 (per-subject/tenant keys for crypto-shred).
- Must ship before: S01 GA (erasure needs the executor + hold), and S08/S11–S19 retention duties.
- Shared infra: scheduler, object storage, backup subsystem, KMS.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accidental over-deletion destroys records under a keep-minimum | M | H | Intersection rule keeps the longer minimum; dry-run + typed confirm; certificates before/after counts |
| Backups retain "deleted" data | H | H | Crypto-shred keys so restored backups yield unreadable ciphertext |
| Cascade misses a derived store → residual PII | M | H | S05-driven registry of stores; cascade coverage test asserts every PII store has a Purger |
| Legal hold bypassed by a direct erasure path | L | H | All deletes route through the executor; hold check is centralized, not per-caller |

## 15. Rollout Plan

- Flag `retention_engine_enabled` (default off). Phase 1: registry + holds + certificates (no auto-disposition). Phase 2: dry-run + manual disposition. Phase 3: scheduled worker per category, opt-in per tenant. Pilot on a low-risk category (e.g. expired transient logs) before student records. GA after crypto-shred verified against a real backup restore. Rollback: disable worker; registry/holds are non-destructive.

## 16. Test Plan

- **Unit** — intersection math (min/max/conflict); anonymisation irreversibility; certificate signing/verification.
- **Integration** — cascade across all Purgers with a seeded subject; hold blocks disposition; crypto-shred renders a restored backup row unreadable.
- **E2E** — erasure from S01 → certificate visible; hold placed → deletion deferred → hold released → deletion completes.
- **Security** — two-person control on tenant-wide shred; authz on schedule/hold; tamper-evidence of certificates.
- **Accessibility** — axe on consoles; confirm-modal keyboard trap correctness.
- **Performance** — 1M-row category disposition within window; OLTP latency unaffected (throttled).
- **Manual** — auditor walkthrough: pick a subject, show every store purged + certificate.

## 17. Documentation & Training

- Statutory retention reference table (the seed source) in the compliance runbook.
- Runbook: placing/releasing legal holds; responding to a restore-then-shred scenario.
- Admin docs: configuring tenant retention within legal bounds.

## 18. Open Questions

1. Backup crypto-shred granularity — per subject, per tenant, or per data-category key? (Cost vs precision.)
2. For "archive" disposition, where do archived records live and who can read them?
3. Do we hard-block a tenant from setting a maximum below a statutory minimum, or warn-and-allow with attestation?
4. How long do disposition certificates themselves retain (they contain subject identifiers)?

## 19. References

- `server/internal/service/backup`, `server/internal/crypto`, `server/internal/scheduler`, `server/internal/service/filestorage`
- GDPR Art 5(1)(e), 17, Recital 26; COPPA 16 CFR §312.10; Quebec Law 25; LGPD Art 16; India DPDP §8(7)
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S05](S05-ropa-data-inventory-mapping.md), [10.15](../../completed/10-compliance-privacy-security/10.15-backup-restore-rpo-rto.md)
