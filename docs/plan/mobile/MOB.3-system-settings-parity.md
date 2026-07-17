# MOB.3 — System Settings Parity (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: [`clients/web/src/components/layout/side-nav-admin-links.tsx`](../../../clients/web/src/components/layout/side-nav-admin-links.tsx),
> `clients/web/src/pages/admin/*`, `clients/web/src/components/settings/*`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.3 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Mobile team (phased) |
| **Depends on** | — |
| **Unblocks** | Admin-on-mobile workflows |

## 1. Problem Statement

The web app exposes a full System Settings / Admin console: a left-nav grouped
into **Platform**, **School operations**, **Student records**, **Integrations**,
and **Compliance & security**, plus dozens of pages under `pages/admin/*`. The
mobile apps ship ~20 admin/settings screens (`Features/Settings/Admin/*`) —
Platform settings, Integrations, People, Roles & permissions, Org structure, Org
branding, Terms, Archived courses, Advising, Transcripts, and the AI admin hub —
but the menu and many pages are absent. Admins therefore cannot run the platform
from a phone/tablet, which undercuts the "admin in the hallway" and district-IT
use cases and creates an uneven, confusing settings surface.

## 2. Goals

- Establish a single **mobile Settings/Admin menu** that mirrors the web nav
  grouping and visibility rules (permission + feature-flag gated).
- Reach page-level parity for the highest-value admin pages, phased.
- Reuse existing mobile admin screens; fill the gaps consistently.
- Make every settings screen permission-gated and flag-gated exactly as web.

## 3. Non-Goals

- Re-implementing settings that another MOB plan owns:
  boards governance → [MOB.8](MOB.8-collaboration-boards-completion.md);
  live-quizzes governance → [MOB.5](MOB.5-interactive-quizzes.md).
- Net-new settings not present on web.
- Changing server settings schemas or RBAC.
- Super-admin/global-operator tooling that is deliberately web/desktop-only
  (see Open Questions).

## 4. Personas & User Stories

- **As an org admin**, I want the same settings menu on mobile so I can change a
  platform toggle or invite a user from my phone.
- **As a registrar (HE)**, I want student-records pages (incompletes, final
  grades status, demographics) on a tablet.
- **As a district IT admin (K12)**, I want integrations (SIS, webhooks, content
  filter) and compliance pages reachable on mobile.
- **As a compliance officer**, I want audit log, quarantine, and security
  reports visible on mobile for spot checks.

## 5. Functional Requirements

- **FR-1.** The app MUST render a Settings/Admin menu grouped identically to web
  (Platform, School operations, Student records, Integrations, Compliance &
  security), showing only groups/items the viewer's permissions and platform
  feature flags allow.
- **FR-2.** Menu visibility MUST derive from the same signals web uses
  (`usePermissions`, `usePlatformFeatures` equivalents already present in mobile
  via `LMSAPIFeatures` / platform-features context).
- **FR-3.** Each in-scope page MUST reach functional parity with its web
  counterpart (read + the write actions web exposes).
- **FR-4.** Already-shipped mobile admin screens MUST be linked from the new
  menu rather than duplicated.
- **FR-5.** Missing pages MUST be delivered per the phased inventory in §8/§13.
- **FR-6.** Every settings write MUST be permission-checked server-side; the UI
  MUST hide/disable actions the viewer cannot perform.
- **FR-7.** Deep links (e.g. from a push notification to "Scheduled jobs") MUST
  route into the correct settings page.

## 6. Non-Functional Requirements

- **Performance** — menu renders from cached feature/permission state; each page
  lazy-loads its data; list pages paginate.
- **Security** — parity with web authz; no client-only gating; audit-log and
  security pages read-only unless permitted.
- **Privacy & Compliance** — demographics/Title I, consent studies, and audit
  data are sensitive: enforce role checks, avoid caching PII to disk, respect
  FERPA/PPRA handling already on the server.
- **Accessibility** — WCAG 2.1 AA across all new screens; large data tables use
  responsive card layouts.
- **Scalability** — list endpoints paginated; no full-table loads.
- **Reliability** — settings writes idempotent; optimistic UI with rollback on
  error.
- **Observability** — `admin_settings_view/edit` events with page id.
- **Maintainability** — one `SettingsMenu` registry mapping id → gate → screen,
  shared shape across iOS/Android to keep parity auditable.
- **Internationalization** — `mobile.settings.*` keys; reuse existing admin
  strings.
- **Backward compatibility** — no API change.

## 7. Acceptance Criteria

- **AC-1.** *Given* an org admin, *when* they open Settings, *then* they see the
  same menu groups/items (minus flags they lack) as web, in the same grouping.
- **AC-2.** *Given* a user lacking `rbac:manage`, *when* they open Settings,
  *then* admin-only groups are hidden (parity with web's early return).
- **AC-3.** *Given* a shipped page (e.g. People), *when* opened from the menu,
  *then* the existing mobile screen loads (no duplicate).
- **AC-4.** *Given* a Phase-1 page (e.g. Audit log), *when* opened, *then* it
  shows the same data and filters as web read-only.
- **AC-5.** *Given* a feature flag off (e.g. SIS integration), *then* that menu
  item is absent on mobile as on web.

## 8. Data Model

- **No new tables.** Every page consumes existing admin endpoints/settings
  tables. Client adds a settings-menu registry (static config), no persistence.

### Settings inventory & parity (mobile status)

| Group | Item | Web source | Mobile today | Phase |
|---|---|---|---|---|
| Platform | Platform settings | `components/settings/platform-settings-panel.tsx` | ✅ `PlatformSettingsView` | — |
| Platform | Org structure | `AdminSettings`/org | ✅ `OrgStructure(Admin)View` | — |
| Platform | Org branding | branding panel | ✅ `OrgBranding(Admin)View` | — |
| Platform | Terms | terms admin | ✅ `TermsAdminView` | — |
| Platform | Roles & permissions | `rbac-api.ts` | ✅ `RolesPermissionsAdminView` | — |
| Platform | People / users | `people-api.ts`, `AdminUsers` | ✅ `PeopleAdminView`/`UserDetailAdminView` | — |
| Platform | Custom fields | `AdminCustomFields` | ❌ | 2 |
| Platform | Banners / maintenance | `AdminBanners`, `banner-api.ts` | ❌ | 2 |
| Platform | Email templates | `AdminEmailTemplates` | ❌ | 2 |
| Platform | Courses admin | `AdminCourses` | ❌ | 3 |
| Platform | Archived courses | `archived-courses-api.ts` | ✅ `ArchivedCoursesAdminView` | — |
| AI | AI models/providers/governance/reports/prompts | AI admin pages | ✅ AI admin hub views | — |
| School ops | Academic calendar | `academic-calendar.tsx` | ❌ | 2 |
| School ops | Broadcasts | `BroadcastComposer` | ❌ | 3 |
| School ops | Conference scheduling | `conference-schedule-grid.tsx` | ⚠️ partial (`Conference` feature) | 3 |
| School ops | Course evaluations | `EvaluationTemplates`/`EvaluationReport` | ⚠️ `Evaluations` feature exists | 2 |
| School ops | Library / Learning paths | library / paths admin | ⚠️ features exist | 3 |
| Student records | Incompletes | `incompletes.tsx` | ❌ | 2 |
| Student records | Final grades status | `grade-submission-status.tsx` | ❌ | 2 |
| Student records | Demographics / Title I | `student-demographics.tsx`, `title1-report.tsx` | ❌ | 3 |
| Student records | CCR achievements | `ccr-api.ts` | ⚠️ transcripts/advising partial | 3 |
| Student records | Attendance dashboard/export | `AttendanceDashboard`/`Export` | ⚠️ course attendance exists | 3 |
| Integrations | Integrations hub | `AdminIntegrations`/`integrations.tsx` | ✅ `IntegrationsAdminView` | — |
| Integrations | SIS integration | `sis-integration.tsx` | ❌ | 3 |
| Integrations | Webhooks | `webhooks.tsx` | ❌ | 3 |
| Integrations | Content filter | `content-filter-settings.tsx` | ❌ | 3 |
| Integrations | Bookstore | `BookstoreIntegration` | ❌ | 3 |
| Compliance | Audit log | `AdminAuditLog` | ❌ | 1 |
| Compliance | AV quarantine | `av-scan-api.ts` | ❌ | 2 |
| Compliance | Scheduled jobs | `scheduled-jobs.tsx` | ❌ | 2 |
| Compliance | Backup ops | `backup-ops-admin-page.tsx` | ❌ | 3 |
| Compliance | ISO / security reports | `iso-compliance-admin-page.tsx` | ❌ | 3 |
| Compliance | Caption compliance | `caption-compliance-report.tsx` | ❌ | 3 |
| Compliance | Consent studies | `consent-studies.tsx` | ❌ | 3 |
| Compliance | Accessibility services | `accessibility-services.tsx` | ❌ | 2 |
| Compliance | Consortium agreements | `consortium-agreements.tsx` | ❌ | 3 |
| Governance | Course reviews moderation | `course-reviews-moderation.tsx` | ❌ | 2 |
| Governance | Boards governance | `boards-governance.tsx` | → MOB.8 | — |
| Governance | Live-quizzes governance | `live-quizzes-governance.tsx` | → MOB.5 | — |

✅ shipped · ⚠️ partial · ❌ missing.

## 9. API Surface

No new server routes. Each page consumes its existing admin API:
`people-api`, `rbac-api`, `banner-api`, `custom-fields-api`,
`email-templates-api`, `scheduler-api`, `av-scan-api`, `sis-api`,
`webhooks-api`, `content-filter-api`, `demographics-api`,
`incomplete-grades-api`, `course-evaluations-api`, `consortium-api`,
`research-consent-api`, `accessibility-api`, `admin-console-api`, etc.
Mobile adds matching `LMSAPI*`/`*Api.kt` wrappers where absent.

## 10. UI / UX

- **New:** a Settings/Admin hub screen with grouped, searchable menu; per-page
  screens for the phased items.
- **Flows:** Settings → group → page → edit → save. Search jumps to a page.
- **States:** loading, empty, error, permission-denied (item hidden),
  offline (read cached where safe, block writes).
- **Mobile/responsive:** convert web tables to card lists; heavy editors
  (email templates, custom fields) may use focused single-purpose screens.
- **Accessibility:** menu is a proper list; each page labelled; destructive
  actions confirmed.
- **Copy & i18n:** `mobile.settings.*`, reuse admin strings.

## 11. AI / ML Considerations

AI admin pages already shipped on mobile; no new model usage here.

## 12. Integration Points

- iOS `Features/Settings/*` + `Core/LMS/LMSAPIAdmin.swift` (already large; extend
  per page). Android `features/settings/*` + `core/lms/*Api.kt`.
- Feature/permission gating via existing mobile platform-features + RBAC state.
- Cross-refs: MOB.5 (quizzes governance), MOB.8 (boards governance).

## 13. Dependencies & Sequencing

- **Phase 1 (menu + top compliance read):** Settings menu shell + Audit log.
- **Phase 2 (high-value ops):** Custom fields, Banners, Email templates,
  Academic calendar, Incompletes, Final grades status, Scheduled jobs, AV
  quarantine, Accessibility services, Course reviews moderation, Evaluations.
- **Phase 3 (long tail):** SIS, Webhooks, Content filter, Bookstore, Broadcasts,
  Demographics/Title I, Backup ops, ISO/security, Caption compliance, Consent
  studies, Consortium, Courses admin, Conference scheduling.
- Shared infra: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Menu/visibility drifts from web gating | H | M | Single registry with gate rules; parity snapshot test vs `side-nav-admin-links` |
| Complex web editors don't fit mobile | M | M | Purpose-built mobile screens; defer truly desktop-class editors (Open Qs) |
| Sensitive data cached to disk | M | H | No-persist policy for PII pages; security review |
| Scope creep (36+ pages) | H | M | Strict phasing; each phase independently shippable |

## 15. Rollout Plan

- Flag: `ff_mobile_admin_console` gates the whole menu; per-page sub-flags
  optional.
- Sequence: Phase 1 behind flag → dogfood with admins → enable phases
  incrementally.
- GA criteria per phase: parity ACs pass; authz tests green.
- Rollback: flag off hides the menu, leaving today's individual screens.

## 16. Test Plan

- **Unit** — menu registry gating (permission×flag matrix) mirrors web.
- **Integration** — each page's read/write against a seeded admin org.
- **End-to-end** — per-phase smoke: open menu → open page → perform one write.
- **Security** — authz matrix per page; PII-not-persisted checks on sensitive
  pages; audit-log read-only.
- **Accessibility** — screen-reader pass on menu + representative pages.
- **Performance** — list pagination; menu render from cache.
- **Manual** — exploratory per page vs web parity checklist.

## 17. Documentation & Training

- Admin help center: "Manage settings on mobile" with per-page notes and which
  pages remain web-only.
- Internal parity checklist doc kept in sync with §8 table.

## 18. Open Questions

1. Which pages stay intentionally web/desktop-only (e.g. bulk imports, backup
   ops, heavy email-template editor)? Mark them "web-only" in the menu.
2. Do we need super-admin/global-operator pages on mobile at all?
3. Should Phase boundaries be per-market (HE registrar pages first vs K12 IT
   pages first)?

## 19. References

- Web: `clients/web/src/components/layout/side-nav-admin-links.tsx`,
  `clients/web/src/pages/admin/*`, `clients/web/src/components/settings/*`.
- iOS: `clients/ios/Lextures/Features/Settings/*`,
  `Core/LMS/LMSAPIAdmin.swift`.
- Android: `.../features/settings/*`, `.../core/lms/*Api.kt`.
- Related: [MOB.5](MOB.5-interactive-quizzes.md),
  [MOB.8](MOB.8-collaboration-boards-completion.md).
