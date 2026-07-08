# LP08 — Privacy, Consent & Transparency Controls

> Implementation plan. Governance for the autonomous learner profile: view/export/pause/reset,
> DSAR + retention integration. Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP08 |
| **Section** | Learner Profile |
| **Severity** | BLOCKER (launch gate for LP07 GA) |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform + Privacy team |
| **Depends on** | LP01 (pause/erase hooks), LP07 (hosts controls); DSAR/Privacy Center (10.x / S-series) |
| **Unblocks** | LP07 GA, LP10 controls |

---

## 1. Problem Statement

The learner profile is built **autonomously** from behaviour — which is exactly what makes it a GDPR
Art. 4(4) *profiling* artifact and a FERPA education record. "No setup" cannot mean "no control." A
learner (or a guardian for a minor) must be able to see it, understand it, export it, pause its
computation, and reset it — and it must be covered by the platform's existing DSAR, retention, and
consent machinery. Without these controls, the feature is a legal and trust liability and cannot GA.

## 2. Goals

- Give learners first-class **controls**: download (export), **pause** (stop deriving), **resume**,
  and **reset/delete** the profile — all self-service, no ticket.
- Wire the profile into existing **DSAR** (`gdpr_http.go`), **data export** (`report_export.go`),
  **state-privacy** (`stateprivacy_http.go`), and **retention/deletion** flows so it's included, not
  forgotten, in access/portability/erasure requests.
- Provide **guardian** access for minors consistent with the existing parent-portal/COPPA model.
- Publish clear **disclosure** (privacy policy + in-product "How this works") that the profile exists,
  what it's derived from, and that it's not used for punitive/consequential automated decisions.

## 3. Non-Goals

- The profile UI itself (LP07) — this plan supplies the control endpoints it hosts.
- Re-implementing DSAR/retention engines — integrate with the shipped ones (10.x, S01, S02).
- Consent to *tracking* the underlying signals — that's governed where those signals are collected
  (9.7 engagement disclosure, etc.); this plan governs the *derived profile*.

## 4. Personas & User Stories

- **As a student**, I want to pause my learner profile if it makes me uncomfortable, and have that
  respected immediately.
- **As a privacy-conscious learner**, I want to download everything the profile holds about me, with
  its provenance, in a portable format.
- **As a parent of a minor**, I want to view and reset my child's profile through the parent portal.
- **As a DPO/registrar**, I want a learner's profile automatically included in DSAR exports and
  erasure runs, with an audit trail.

## 5. Functional Requirements

- **FR-1.** `POST /api/v1/me/learner-profile/pause` MUST set profile `status='paused'` (LP01), and
  while paused **no deriver runs** and the profile is frozen (LP07 shows paused state). `.../resume`
  reverts and schedules a recompute.
- **FR-2.** `POST /api/v1/me/learner-profile/reset` MUST delete all `learner.*` rows for the user
  (facets, insights, evidence); the profile rebuilds from future activity unless also paused.
- **FR-3.** `GET /api/v1/me/learner-profile/export` MUST return a portable (JSON, human-readable)
  export of the full profile **including provenance/evidence**, suitable for GDPR portability.
- **FR-4.** The profile MUST be **included in the platform DSAR/export pipeline**: register a data
  source with `report_export.go` / `gdpr_http.go` so access & portability requests contain it
  automatically.
- **FR-5.** User **erasure** MUST cascade to all `learner.*` rows (LP01 `ON DELETE CASCADE`) and be
  verified in the erasure job; a re-created account starts with no profile.
- **FR-6.** The profile MUST honour **retention** policy (S02): if a learner is inactive beyond the
  configured window, evidence/insights age out per policy.
- **FR-7.** **Guardian access** for minors MUST allow a linked guardian to view/pause/reset via the
  existing parent-portal authorization (357 parent-portal-v2 / COPPA model), scoped to their child.
- **FR-8.** All control actions MUST be written to the **audit log** (who, what, when) and be
  rate-limited.
- **FR-9.** Disclosure copy MUST state the profile is **not** used for consequential automated
  decisions without human oversight (GDPR Art. 22 posture); LP09 consumers are advisory/assistive.

## 6. Non-Functional Requirements

- **Performance** — Pause/reset ≤ 300 ms; export may be async for large profiles (reuse export job
  pattern). 
- **Security** — Control endpoints self-only (or authorized guardian/DSAR-admin); CSRF-safe; audited;
  rate-limited. Reset/erase are irreversible — require explicit confirmation (UI, LP07).
- **Privacy & Compliance** — FERPA (education record access/amendment posture), GDPR Arts. 15/17/20/22,
  COPPA guardian rights, US state privacy (S11). Reuse the shipped compliance surfaces, do not fork.
- **Accessibility** — Confirmation dialogs and controls WCAG 2.1 AA (owned with LP07).
- **Reliability** — Pause takes effect before the next deriver run (check status at job start);
  erasure idempotent.
- **Observability** — Metrics `learner_profile_control_total{action}`; audit entries for every action.
- **Internationalization** — All control/disclosure copy localised.
- **Backward compatibility** — Additive endpoints; integrates with existing DSAR/export/retention.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner pauses their profile, *when* the next recompute is scheduled, *then* no
  deriver runs and LP07 shows paused until resume.
- **AC-2.** *Given* a learner resets their profile, *then* all `learner.*` rows for them are gone and a
  fresh read returns the empty state.
- **AC-3.** *Given* a learner requests an export, *then* they receive a portable file containing every
  facet, insight, and its evidence/provenance.
- **AC-4.** *Given* a DSAR access request for a user, *when* it runs, *then* the export includes the
  learner profile automatically (no manual step).
- **AC-5.** *Given* a user erasure, *then* all `learner.*` rows are deleted and verified by the job.
- **AC-6.** *Given* a guardian linked to a minor, *then* they can view/pause/reset the minor's profile
  and only that minor's.
- **AC-7.** *Given* any control action, *then* an audit-log entry records actor, action, and time.

## 8. Data Model

- No new profile tables. Adds: a `paused` usage of `learner.profiles.status` (LP01), an audit entry
  type for profile controls (reuse the admin audit-log schema), and registration rows/config in the
  existing DSAR data-source registry (10.x/S01) pointing at the `learner` schema.
- Retention config for the `learner` schema added to the retention engine (S02) tables.

## 9. API Surface

```
POST /api/v1/me/learner-profile/pause      -> 200 { status:"paused" }
POST /api/v1/me/learner-profile/resume     -> 200 { status:"active" }
POST /api/v1/me/learner-profile/reset      -> 200 { status:"reset" }   (irreversible; confirmed in UI)
GET  /api/v1/me/learner-profile/export     -> 200 application/json (portable, with provenance)
# Guardian variants scoped via parent-portal authz; DSAR inclusion via report_export.go registry.
```

## 10. UI / UX

- LP07 hosts a "Manage your profile" area: **Download**, **Pause/Resume**, **Reset** with a
  confirming dialog for destructive actions and a link to the Privacy Center.
- Guardian controls surface in the existing parent portal for linked minors.
- Disclosure: privacy-policy section + the in-product "How this works" (LP07 FR-7) + Art. 22 statement.
- States: pause confirmation, reset confirmation (irreversible warning), export-preparing, paused
  banner.

## 11. AI / ML Considerations

Establishes the **Art. 22 posture**: the profile and its LP09 consumers are *assistive/advisory*, not
consequential automated decisions; any future consequential use requires human oversight + a DPIA
(S06). Export/disclosure must state this.

## 12. Integration Points

- `server/internal/httpserver/gdpr_http.go`, `stateprivacy_http.go`, `report_export.go`,
  `research_consent_http.go`; retention engine (S02); parent portal (`357_ff_parent_portal_v2`).
- `server/internal/httpserver/admin_audit_log_http.go` (audit); LP01 `Pause/Resume/Erase` hooks.
- Privacy Center UI (`gdprModuleEnabled`, `/privacy-centre`).

## 13. Dependencies & Sequencing

- After LP01 (hooks) + LP07 (host UI). MUST ship before LP07 GA. Integrates with shipped 10.x/S01/S02.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Profile forgotten in DSAR/erasure | M | H | Register in the DSAR data-source registry + erasure verification test (AC-4/AC-5) |
| Pause races an in-flight recompute | M | M | Check `status` at job start; skip if paused |
| Guardian scope leak (sees other children) | L | H | Reuse parent-portal authz; scope tests |
| "Reset" misunderstood as reversible | M | M | Explicit irreversible confirmation; distinct from Pause |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Ship controls + DSAR registration + retention config → verify
erasure/export in staging → enable with LP07 pilot → GA (LP07 GA gated on this). Rollback: flag; data
retained; controls hidden.

## 16. Test Plan

- **Unit** — pause/resume/reset semantics; export serialization incl. provenance; retention aging.
- **Integration** — DSAR access includes profile; erasure removes all `learner.*` + verification;
  guardian scope.
- **E2E** — learner pauses→LP07 paused; reset→empty; export downloads; guardian path.
- **Security** — self/guardian/DSAR-admin authz matrix; rate-limit; audit entries present.
- **Compliance** — erasure completeness; export portability; Art. 22 disclosure present.

## 17. Documentation & Training

- Student/guardian help: "Managing your Learner Profile — download, pause, reset."
- Privacy policy update: profile disclosure + Art. 22 statement.
- DPO runbook: profile in DSAR/erasure; retention config for `learner` schema.

## 18. Open Questions

1. Does "pause" also suppress LP09 consumption, or only stop *deriving*? (Proposed: suppress both.)
2. Retention window for an inactive learner's profile — align with engagement-event retention (9.7 OQ)?
3. FERPA "amendment" right: since the profile is derived, is Pause/Reset the amendment mechanism, or do
   we need per-insight dismissal? (Leaning: Reset + per-insight "not accurate" dismissal in a follow-on.)

## 19. References

- [LP01](LP01-foundation-derivation-engine.md), [LP07](LP07-settings-page-transparency-ui.md).
- Existing: `gdpr_http.go`, `stateprivacy_http.go`, `report_export.go`, `admin_audit_log_http.go`,
  `357_ff_parent_portal_v2.sql`; standards [S01](../standards/S01-unified-data-subject-rights-orchestration.md),
  [S02](../standards/S02-data-retention-deletion-engine.md), [S08](../standards/S08-childrens-privacy-age-assurance-design-codes.md).
- External: GDPR Arts. 15/17/20/22; FERPA 34 CFR §99.20 (amendment); COPPA parental rights.
