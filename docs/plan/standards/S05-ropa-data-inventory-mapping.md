# S05 — Records of Processing Activities & Live Data Inventory

> Implementation plan. Hardens: [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md) (Art 30 RoPA), [10.10 ISO 27701](../../completed/10-compliance-privacy-security/10.10-iso-27001-27701.md), [10.14 PII redaction](../../completed/10-compliance-privacy-security/10.14-pii-redaction-logs.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S05 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | EU/UK · Global |
| **Status (today)** | THIN — 10.3 promised an in-system RoPA register but it is a static list; there is no machine-readable inventory mapping *which tables/stores hold which personal-data categories*, which every other standards plan needs |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Data Platform + DPO |
| **Depends on** | 10.3, 10.14 |
| **Unblocks** | S01 (provider registry), S02 (categories), S03 (affected data), S04 (purposes), S06 (DPIA inputs) |

---

## 1. Problem Statement

Nearly every obligation in this folder needs one thing first: an accurate answer to "what personal data do we hold, where, for what purpose, under what basis, shared with whom, kept how long?" GDPR Art 30 mandates it as the RoPA; ISO 27701 and every DPA require the underlying data map; S01 needs it to know which stores to export/erase, S02 to schedule retention, S03 to scope a breach, S04 to define purposes. Today this exists as prose. A prose RoPA drifts from reality the moment an engineer adds a column, and a data map that is wrong makes every downstream compliance guarantee false. We need a **living, machine-readable data inventory** that is partly generated from the schema and partly curated, feeding all the engines.

## 2. Goals

- A **data inventory**: every store (table, object bucket, search index, analytics dataset, embedding index, external subprocessor) tagged with the personal-data **categories** it holds, sensitivity, subjects, and the store's system location.
- A structured, exportable **RoPA** (Art 30) generated from the inventory + purposes (S04) + transfers (S07).
- **Schema drift detection**: flag new columns/tables that are untagged so the inventory never silently goes stale.
- A **category taxonomy** (e.g. identity, contact, academic-record, behavioural, biometric, special-category, financial) reused everywhere.
- Machine-readable outputs consumed by S01/S02/S03 (not just a human PDF).

## 3. Non-Goals

- Automatic PII classification via ML (v1 is schema-annotation + curation; ML-assist is a later enhancement).
- The transfer register itself (S07) — S05 references it.
- Consent purposes' UX (S04) — S05 provides the processing records they link to.

## 4. Personas & User Stories

- **As a DPO**, I want an Art 30 RoPA that is generated from reality so that it's accurate at audit time, not aspirational.
- **As a privacy engineer**, I want new database columns holding PII to be flagged until classified so that our data map never silently drifts.
- **As the S01 orchestrator**, I want to enumerate every store holding a subject's data so that access/erasure is complete.
- **As a security responder (S03)**, I want to know exactly which categories a breached store held so that I scope notifications correctly.
- **As an auditor**, I want to export the RoPA and the data map so that I can verify accountability.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain a `data_stores` inventory (name, type, system, jurisdiction, subjects) and a `data_elements` mapping each store's fields to a **category** in the taxonomy with a sensitivity level.
- **FR-2.** The system MUST maintain `records_of_processing` linking purpose (S04) → categories → subjects → lawful basis → retention (S02) → recipients/transfers (S07).
- **FR-3.** A **drift check** MUST compare the live DB schema against `data_elements` and flag untagged columns/tables (fail CI or raise a review task).
- **FR-4.** The inventory MUST expose a **machine-readable API** enumerating, for a given subject or category, the stores/fields involved (consumed by S01/S02/S03).
- **FR-5.** The system MUST generate an **Art 30 RoPA** export (both controller and processor records) in CSV/JSON/PDF.
- **FR-6.** Special-category data (Art 9: health/504-IEP, biometric proctoring, etc.) MUST be explicitly flagged and gated for extra controls.
- **FR-7.** The taxonomy and inventory MUST be tenant-aware where stores differ by deployment/residency (ties to 10.12 / S07).
- **FR-8.** Inventory edits MUST be versioned and auditable.

## 6. Non-Functional Requirements

- **Performance** — Subject/category lookups < 100 ms (indexed); drift check runs in CI and nightly.
- **Security** — The inventory itself is sensitive (a map to all PII); read access gated by `privacy:inventory_read`; no raw PII stored in the inventory, only metadata.
- **Privacy & Compliance** — GDPR Art 30; ISO 27701 6.x; supports every DPA's Annex I/II (categories/purposes).
- **Accessibility** — Inventory/RoPA admin UI WCAG 2.1 AA.
- **Scalability** — Hundreds of stores, thousands of elements; export streams.
- **Reliability** — Drift check is deterministic; API reflects committed inventory only.
- **Observability** — `data_inventory_untagged_elements`, `ropa_last_generated_at`; alert if untagged count > 0 for > N days.
- **Maintainability** — Inventory in `server/internal/service/datainventory/`; taxonomy is versioned data; annotations can live as struct tags/migration comments harvested by a generator.
- **Internationalization** — RoPA export localised for regulator language where required.
- **Backward compatibility** — Seeded from the existing 10.3 RoPA prose; additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a new migration adds a `phone_number` column, *when* the drift check runs in CI, *then* it fails until the column is classified in `data_elements`.
- **AC-2.** *Given* the S01 orchestrator asks "which stores hold subject X's data," *when* it queries the inventory API, *then* it receives every store/field, including object storage and embeddings.
- **AC-3.** *Given* a DPO exports the RoPA, *when* generation runs, *then* controller and processor records are produced with purpose, categories, basis, retention, and recipients populated from S04/S02/S07.
- **AC-4.** *Given* a store holding IEP/504 health data, *when* it is tagged, *then* it is flagged special-category and appears in the extra-controls report.
- **AC-5.** *Given* a breach in store Y (S03), *when* responders open the case, *then* the affected categories auto-populate from the inventory.
- **AC-6.** *Given* an untagged element exists for > 14 days, *when* the alert job runs, *then* `data_inventory_untagged_elements` alerts and a review task is created.

## 8. Data Model

New migration `361_data_inventory_ropa.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.data_categories (
  key         TEXT PRIMARY KEY,               -- 'identity','contact','academic_record','behavioural',
  label       TEXT NOT NULL,                  --  'special_category','biometric','financial','device'
  sensitivity TEXT NOT NULL CHECK (sensitivity IN ('normal','sensitive','special')),
  is_art9     BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS compliance.data_stores (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name         TEXT NOT NULL UNIQUE,          -- 'pg.submissions','r2.media','opensearch.courses','vec.embeddings'
  store_type   TEXT NOT NULL,                 -- 'postgres','object','search','analytics','embedding','external'
  system       TEXT NOT NULL,                 -- owning service/module
  jurisdiction TEXT,                          -- residency of the store (10.12)
  subprocessor_id UUID                         -- FK to S07 when external
);

CREATE TABLE IF NOT EXISTS compliance.data_elements (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id     UUID NOT NULL REFERENCES compliance.data_stores(id) ON DELETE CASCADE,
  path         TEXT NOT NULL,                 -- 'submissions.content','users.dob'
  category_key TEXT NOT NULL REFERENCES compliance.data_categories(key),
  subjects     TEXT[] NOT NULL,               -- {'student','parent','staff'}
  notes        TEXT,
  UNIQUE (store_id, path)
);

CREATE TABLE IF NOT EXISTS compliance.records_of_processing (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  role         TEXT NOT NULL CHECK (role IN ('controller','processor')),
  purpose_key  TEXT NOT NULL,                 -- FK-by-name to S04 processing_purposes
  category_keys TEXT[] NOT NULL,
  lawful_basis TEXT,
  retention_ref TEXT,                          -- S02 schedule key
  recipients   TEXT[],                         -- S07 subprocessors / third parties
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET` | `/api/v1/compliance/inventory/stores` | `privacy:inventory_read` | List stores + tags |
| `GET` | `/api/v1/compliance/inventory/resolve` | internal (S01/S02/S03) | Stores/fields for subject or category |
| `GET/PUT` | `/api/v1/compliance/inventory/elements` | `privacy:inventory_admin` | Classify elements |
| `GET` | `/api/v1/compliance/ropa` | `privacy:dpo` | Generate Art 30 RoPA (CSV/JSON/PDF) |
| `GET` | `/api/v1/compliance/inventory/drift` | `privacy:inventory_admin` | Untagged schema report |

## 10. UI / UX

- **Data-map console:** store list, element classification grid, special-category flags, per-store jurisdiction and subprocessor link.
- **RoPA generator:** preview + export; diff vs. last version.
- **Drift dashboard:** untagged columns/tables with "classify" quick-action.
- States: empty (seed from schema), loading (export), error (drift gate blocking a merge), warning banner for special-category stores.
- Accessibility: large-grid keyboard navigation, sortable/filterable with ARIA; i18n keys `inventory.*`.

## 11. AI / ML Considerations

Embedding/vector stores and AI-log stores are first-class inventory entries so that S02 erasure and S06 DPIAs cover them. Free-text stores (submissions, messages) are flagged as potentially containing any category (unstructured PII risk) and are prioritised for the redaction proxy (10.14).

## 12. Integration Points

- `server/internal/service/datainventory/` (new); schema-drift generator hooks into `server/internal/migrate` / CI.
- Consumed by S01, S02, S03; links to S04 purposes and S07 subprocessors; feeds S06 DPIAs.
- `server/internal/service/logredaction` / 10.14 for unstructured-store flags.

## 13. Dependencies & Sequencing

- Must ship **first** among the cross-cutting engines (S01/S02/S03/S06 depend on it).
- Must ship before: S01, S02, S03, S06.
- Shared infra: CI pipeline (drift gate), export/object storage.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Inventory drifts from schema → downstream guarantees false | H | H | Automated drift gate in CI; nightly reconciliation; alert on untagged |
| Manual classification burden discourages upkeep | M | M | Struct-tag/comment harvesting to pre-fill; quick-classify UI; block merges on untagged |
| Inventory becomes a treasure map for attackers | L | H | Strict authz; metadata only (no PII); audit all reads |
| Unstructured stores under-classified | M | H | Default free-text to "any category"; redaction-proxy priority |

## 15. Rollout Plan

- Flag `data_inventory_enabled`. Phase 1: taxonomy + stores + elements seeded from schema + 10.3 RoPA. Phase 2: resolve API for S01/S02/S03. Phase 3: drift gate in CI (warn, then enforce). GA when untagged = 0 and RoPA export validated by DPO. Rollback: drift gate to warn-only; inventory is read infrastructure (non-destructive).

## 16. Test Plan

- **Unit** — taxonomy validation; RoPA generation from fixtures; drift diff logic.
- **Integration** — resolve API returns complete store set for a seeded subject; special-category gating.
- **E2E** — add a PII column → CI drift gate fails → classify → passes; export RoPA.
- **Security** — authz on inventory reads; confirm no PII values stored.
- **Accessibility** — axe on data-map grid; keyboard classification.
- **Performance** — resolve < 100 ms across hundreds of stores.
- **Manual** — DPO validates generated RoPA against a known processing activity.

## 17. Documentation & Training

- Engineering guide: "Classifying new personal-data columns" (part of the migration checklist).
- DPO guide: reading/exporting the RoPA.
- Data-map maintained as living doc, linked from ISO 27701 evidence.

## 18. Open Questions

1. Struct-tag vs. migration-comment vs. separate annotation file as the source of element tags?
2. How granular for unstructured stores (per-column "any category" vs. sampled classification)?
3. Do we track *field-level* transfers or is store-level sufficient for the RoPA's Annex?
4. Ownership: who signs off classification for a new store before it can hold prod data?

## 19. References

- `server/internal/migrate`, `server/internal/service/logredaction`, `server/migrations/`
- GDPR Art 30, Art 9; ISO/IEC 27701 clause 6–8; DPA Annex I/II conventions
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S02](S02-data-retention-deletion-engine.md), [S03](S03-global-breach-notification-incident-response.md), [S07](S07-cross-border-transfer-subprocessor-governance.md)
