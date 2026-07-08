# IC08 — Admin Governance, Localization & Content Versioning

> Implementation plan. Source: product direction — *"a feature flag that can be disabled by global
> platform feature flags"* + operability/localization of a platform-wide course. Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC08 |
| **Section** | Intro Course |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform + admin/web |
| **Depends on** | IC01 (flag + provisioning), IC03 (content to version/localize) |
| **Unblocks** | GA of the whole epic (governance is a launch gate) |

---

## 1. Problem Statement

A course auto-assigned to every user is a platform-wide artifact that admins must be able to
**govern**, **localize**, and **version safely**. Beyond the global on/off flag, admins need to
re-sync content after edits, run/monitor the enrollment backfill, see completion analytics, and
serve the course in their users' languages. Content changes must ship like code with a clear
version and a non-destructive sync. This plan delivers the admin controls, the localization
pipeline, the content-versioning discipline, and the analytics that make the intro course
operable and a launch gate for GA.

## 2. Goals

- Give platform admins a **console surface** for the intro course: the on/off flag, a **re-sync**
  action, a **backfill** trigger/monitor, content version, and **completion analytics**.
- Localize the course: a **locale-partitioned content pipeline** with English fallback, integrated
  with the existing i18n / translation-memory machinery, honoring RTL.
- Enforce **content versioning**: every change bumps a `content_version`; sync is idempotent,
  update-in-place, and **non-destructive** to learner grades/submissions.
- Provide **analytics**: enrollment count, completion rate, per-module funnel, drop-off, average
  time-to-complete — for judging onboarding effectiveness.
- Record all admin actions to the **audit log** and gate them behind platform-admin scope.

## 3. Non-Goals

- The flag plumbing itself (IC01) — this plan adds the admin *UX and actions* around it.
- Authoring content (IC03) — this plan versions/localizes/syncs it.
- Per-*learner* UI (IC06/IC07).
- Building a general CMS — content stays code-owned fixtures; admins trigger sync, they don't edit
  prose in the DB.

## 4. Personas & User Stories

- **As a platform admin**, I want to turn the intro course off for my deployment, so a district
  with its own onboarding isn't force-enrolled.
- **As a platform admin**, I want to re-sync content and trigger/monitor the backfill from the
  console, so I don't need engineering to operate it.
- **As an admin in a non-English market**, I want the course served in my users' language, so
  onboarding lands.
- **As a product manager**, I want completion analytics, so I can measure and improve activation.
- **As a compliance reviewer**, I want admin actions audited, so changes to a platform-wide,
  auto-enrolling course are accountable.

## 5. Functional Requirements

- **FR-1.** The admin console MUST expose the intro-course panel: current flag state (toggle),
  course status (present, `content_version`, module count), **Re-sync content** action,
  **Run/Resume backfill** action + progress, and **completion analytics**.
- **FR-2.** Toggling the flag, re-syncing, and running the backfill MUST require **platform-admin
  scope** and MUST be written to the **admin audit log** (`admin_audit_log_enabled`).
- **FR-3.** Content MUST carry a monotonically increasing `content_version`; **Re-sync** MUST be
  idempotent and **non-destructive** — update/add content in place, **soft-archive** removed
  items, and **never delete** items that hold student grades/submissions.
- **FR-4.** The course MUST be **localizable**: content fixtures partitioned by locale
  (`content/<locale>/…`), resolved to the learner's language with **English fallback** for missing
  strings/pages; RTL layouts honored (`rtl_enabled`). Translations integrate with the existing
  translation-memory / i18n pipeline.
- **FR-5.** Analytics MUST provide: enrolled, completed, completion rate, per-module completion
  funnel, drop-off point, and average time-to-complete — over the flag-gated population, admin-only.
- **FR-6.** Disabling the flag MUST stop new auto-enroll + hide discovery (IC01/IC02/IC06) but MUST
  **retain** the course, enrollments, grades, submissions, completions, and credentials.
- **FR-7.** The panel MUST show operational health: last provision/sync time and result,
  outstanding backfill remaining, and last content-validation status.
- **FR-8.** (Stretch / OQ) Support a **per-org override** so a multi-tenant deployment can
  disable/enable per organization, layered over the global flag.

## 6. Non-Functional Requirements

- **Performance** — Admin reads (status, analytics) p95 ≤ 300 ms (aggregates precomputed or over
  indexed columns). Re-sync bounded by IC03's < 2 s; backfill per IC02.
- **Security** — All actions platform-admin only; CSRF/authz enforced; audited. Analytics contain
  no per-user PII beyond what admins may already see; no raw student text.
- **Privacy & Compliance** — Audit trail for a platform-wide auto-enrolling course (accountability
  for FERPA/GDPR reviewers). Localization introduces no new PII. Analytics aggregate-only.
- **Accessibility** — Admin panel WCAG 2.1 AA (existing admin console standards); analytics charts
  have text/table equivalents.
- **Scalability** — Analytics computed from bounded item set × students; precompute completion-rate
  gauges (IC05) and per-module funnel via a scheduled rollup for large populations.
- **Reliability** — Re-sync/backfill are idempotent and resumable (IC01/IC02); admin actions are
  safe to retry.
- **Observability** — Reuse IC01/IC02/IC05 metrics; add `intro_course_admin_action_total{action}`.
  Surface last-sync/last-backfill status in the panel and metrics.
- **Maintainability** — One admin panel component + one analytics endpoint; localization is a
  fixture-directory convention + existing i18n, not bespoke.
- **Internationalization** — This *is* the i18n plan for the course; English source of truth,
  per-locale overrides, fallback, RTL. Certificate/congrats copy localized.
- **Backward compatibility** — Additive; per-org override (if built) layers over the global flag
  without changing its default.

## 7. Acceptance Criteria

- **AC-1.** *Given* a platform admin, *when* they open the intro-course panel, *then* they see the
  flag state, content version, module count, backfill progress, and completion analytics.
- **AC-2.** *Given* an admin toggles the flag or triggers re-sync/backfill, *then* the action
  requires admin scope and is recorded in the admin audit log.
- **AC-3.** *Given* an edited fixture with a bumped `content_version`, *when* re-sync runs, *then*
  content updates in place, removed items are soft-archived, and no student grade/submission is
  deleted.
- **AC-4.** *Given* a learner with locale `es`, *when* they open the course, *then* Spanish content
  renders where available and falls back to English otherwise; RTL locales render RTL.
- **AC-5.** *Given* the analytics endpoint, *when* an admin queries it, *then* it returns enrolled,
  completed, completion rate, and a per-module funnel; a non-admin is rejected.
- **AC-6.** *Given* the flag is disabled, *when* checked later, *then* enrollments, grades, and
  completions from before remain intact and readable.
- **AC-7.** *(If FR-8 built)* *Given* a per-org override disabling the course for org X, *then*
  org X users are not auto-enrolled while other orgs are unaffected.

## 8. Data Model

Reuses IC01–IC05 tables. Additions:

```sql
-- server/migrations/375_intro_course_admin.sql  (renumber on merge)

-- Content version + last sync/provision status (singleton).
CREATE TABLE settings.intro_course_status (
    id             BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    content_version INTEGER NOT NULL DEFAULT 0,
    last_synced_at  TIMESTAMPTZ,
    last_sync_result TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Optional per-org override (FR-8, stretch). NULL global flag value = follow platform default.
CREATE TABLE settings.intro_course_org_overrides (
    org_id   UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    enabled  BOOLEAN NOT NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Localization uses fixture directories (`content/<locale>/…`) + existing i18n string tables /
translation memory; no per-string DB table beyond what i18n already provides. Analytics derive
from `settings.intro_course_completions` (IC05) + `course.course_grades` + engagement; a scheduled
rollup MAY materialize per-module funnel counts for large populations.

## 9. API Surface

```
GET  /api/v1/admin/intro-course                 Auth: platform-admin
  → { enabled, contentVersion, moduleCount, lastSyncedAt, lastSyncResult,
      backfill: { startedAt, completedAt, remaining }, orgOverride? }

POST /api/v1/admin/intro-course/resync          Auth: platform-admin  (from IC01)
POST /api/v1/admin/intro-course/backfill        Auth: platform-admin  (from IC02)
GET  /api/v1/admin/intro-course/analytics       Auth: platform-admin  (from IC05, extended funnel)
PUT  /api/v1/admin/intro-course/org-override    Auth: platform-admin  (FR-8, stretch)
```

Flag toggle continues to flow through the existing platform-features patch endpoint (audited). All
new routes documented in OpenAPI.

## 10. UI / UX

- **Admin console → Intro course panel** (in the platform settings/admin area): flag toggle
  (reuses platform-features), status block (version, module count, last sync), **Re-sync** and
  **Run backfill** buttons with progress, and an **analytics** section (enrolled, completion rate,
  per-module funnel chart + table equivalent, avg time-to-complete).
- Confirmation dialogs for re-sync/backfill (they affect all users); success/error toasts; audit
  note ("this action is logged").
- Empty/loading/error states; charts have accessible table fallbacks. Copy via i18n.

## 11. AI / ML Considerations

- Localization MAY use the existing **translation-memory** machinery (and optionally machine
  translation) to seed non-English content, but human review is required before a locale is marked
  authoritative; English remains the source of truth. No model calls at learner runtime.

## 12. Integration Points

- **Admin console:** existing admin/platform-settings surfaces (`settings_platform.go`, admin
  console handlers), audit log (`admin_audit_log_enabled`).
- **Flag stack:** platform-features patch (IC01).
- **Sync/backfill:** IC01 `EnsureProvisioned`/`SyncContent`, IC02 `RunBackfill`.
- **i18n / translation memory:** existing localization pipeline (`translation_memory_enabled`,
  `rtl_enabled`), mobile + web i18n.
- **Analytics:** IC05 completion data + `course.course_grades` + engagement; scheduler for rollups.

## 13. Dependencies & Sequencing

- **After:** IC01 (flag/provision/resync), IC03 (content to version/localize); consumes IC02/IC05
  actions/data.
- **Before:** **GA of the epic** (an admin must be able to disable, re-sync, and localize before
  broad rollout).
- **Shared infra:** admin console, audit log, i18n/translation memory, scheduler — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Admin re-sync destroys learner grades | L | H | Non-destructive sync; soft-archive; never delete graded items (FR-3, AC-3) |
| Untranslated locale shows broken/empty pages | M | M | English fallback per-string/page (FR-4); validation flags missing keys |
| Backfill/re-sync run without accountability | M | M | Admin scope + audit log (FR-2) + confirmation dialogs |
| Per-org override complexity / precedence bugs | M | M | Clear precedence (org override > global default); default follow-global; heavily tested; ship as stretch |
| Analytics heavy at scale | M | M | Precompute gauges + scheduled funnel rollup |

## 15. Rollout Plan

- **Flag:** `intro_course_enabled` (global) + optional per-org override.
- **Sequencing:** admin status/analytics endpoints → admin panel (toggle/resync/backfill/analytics)
  → localization pipeline + first non-English locale → (stretch) per-org override → GA.
- **Dogfood:** admins on staging disable/enable, re-sync after a content edit, run backfill, read
  analytics; validate one non-English locale end-to-end.
- **GA criteria (epic-level gate):** admin can disable + re-sync + localize; actions audited;
  non-destructive sync proven; analytics accurate.
- **Rollback:** disable flag (non-destructive); revert content version + re-sync; per-org override
  removable.

## 16. Test Plan

- **Unit** — content-version bump + no-op sync; non-destructive removal (soft-archive); locale
  resolution + English fallback; override precedence.
- **Integration (DB)** — re-sync preserves grades; backfill trigger/monitor; analytics aggregation
  correctness; org override gates enrollment (if built).
- **End-to-end** — admin toggles flag (audited), re-syncs after an edit, runs backfill, reads
  analytics; learner in `es` sees localized content with fallback; RTL renders.
- **Security** — all admin routes reject non-admins; audit entries written; no PII in analytics.
- **Accessibility** — admin panel + charts (with table equivalents) pass axe.
- **Performance** — analytics p95; rollup within window at scale.
- **Manual exploratory** — disable then re-enable (data intact); partially-translated locale;
  per-org override interplay with global default.

## 17. Documentation & Training

- Admin guide: operating the intro course (enable/disable, re-sync, backfill, analytics, per-org
  override).
- Localization guide: adding a locale to the intro course; translation-memory workflow; fallback
  rules.
- Runbook additions: interpreting analytics, safe re-sync, audit review.
- OpenAPI for all admin endpoints.

## 18. Open Questions

1. Is per-org enable/disable (FR-8) required for v1, or is the global flag sufficient? (Leaning:
   global for v1; per-org as fast-follow for multi-tenant deployments — ties to IC01/IC02 OQs.)
2. Which locales ship first, and machine-translate-then-review vs. human-only? (Product/i18n.)
3. Should analytics be per-org scoped for tenant admins vs. platform-wide for platform admins?
4. Where does the panel live — platform feature settings, a dedicated "Onboarding" admin section,
   or the admin console? (Leaning platform settings alongside the flag.)

## 19. References

- Existing: `server/internal/httpserver/settings_platform.go`, admin console handlers, audit log,
  translation-memory + i18n pipeline (`translation_memory_enabled`, `rtl_enabled`),
  `server/internal/openapi/openapi.go`.
- Related plans: [IC01](IC01-foundation-provisioning-flag.md), [IC02](IC02-automatic-enrollment.md),
  [IC03](IC03-curriculum-content.md), [IC05](IC05-progress-completion-credential.md).
