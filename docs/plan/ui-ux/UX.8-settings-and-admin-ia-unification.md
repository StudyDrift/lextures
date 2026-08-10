# UX.8 — Settings and Admin IA Unification

> Implementation plan. Source: [audit.md](audit.md) §4 G-12.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.8 |
| **Section** | UI/UX — Information Architecture |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — two parallel hierarchies for one concept |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Product Design + Web |
| **Depends on** | UX.6, UX.7 |
| **Unblocks** | Admin onboarding time-to-value; reduces support load |

---

## 1. Problem Statement

Configuration lives in two parallel hierarchies. `/settings/*` exposes 33
destinations rendered by 48 panel components; `/admin/*` has its own layout
(`AdminLayout.tsx`), its own 15 KB navigation file and 46 pages. The Settings
navigation itself routes into the admin console for four items
(`/admin/accessibility`, `/admin/bookstore`, `/admin/consortium`,
`/admin/consent-studies`), so the boundary is not even consistent within one menu.
There is no search within settings. An administrator cannot predict where a given
setting lives — the semantic-overlap failure mode in **R-11** — and the cost lands
on the highest-value, lowest-patience persona in the product.

## 2. Goals

- Establish **one predictable configuration hierarchy** with an explicit,
  documented rule for what lives where.
- Add **search across all settings**, indexing individual settings, not just pages.
- Make each settings page scannable via consistent structure and progressive
  disclosure (**R-9**).
- Preserve every existing URL so bookmarks, docs, runbooks and deep links survive.

## 3. Non-Goals

- Adding, removing or changing what any setting *does*.
- Redesigning the permission model (RBAC is out of scope).
- Course-level settings (`/courses/:code/settings/*`) beyond aligning their shell.
- Changing the platform feature-flag system itself.

## 4. Personas & User Stories

- **As an administrator**, I want to type "SAML" and land on the SSO settings,
  without knowing whether it is filed under Settings or Admin.
- **As an administrator**, I want one place that means "configuration", so I stop
  checking two menus.
- **As a new admin at a partner institution**, I want the hierarchy to teach me
  what is configurable.
- **As a support engineer**, I want to give a customer a stable deep link to a
  setting.
- **As an instructor**, I want my personal settings clearly separated from
  organisation settings so I never fear changing something global.

## 5. Functional Requirements

- **FR-1.** A single **configuration scope model** MUST be defined and documented:

  | Scope | Meaning | Route prefix |
  |---|---|---|
  | **Me** | Affects only this user | `/settings/*` |
  | **Organisation** | Affects everyone in the org | `/settings/org/*` |
  | **Platform** | Affects the whole deployment (super-admin) | `/settings/platform/*` |
  | **Course** | Affects one course | `/courses/:code/settings/*` |
  | **Operations** | Non-configuration admin work (audit logs, imports, reports, moderation, jobs) | `/admin/*` |

- **FR-2.** Every one of the 33 settings destinations and 46 admin pages MUST be
  classified into exactly one scope. The classification MUST be committed as a
  reviewed table.
- **FR-3.** `/admin/*` MUST contain **only Operations** — things an admin *does*,
  not things they *configure*. Configuration currently under `/admin/*`
  (accessibility services, bookstore, consortium, consent studies, content filter,
  integrations, SIS) MUST move to the appropriate settings scope.
- **FR-4.** Every existing URL MUST continue to work via permanent redirect to its
  new home.
- **FR-5.** A **settings search** MUST index individual settings — label,
  description, synonyms, scope and section — not only page titles, and MUST deep
  link to the specific control and highlight it.
- **FR-6.** Settings search MUST also be reachable from the command palette, so
  `⌘K → "retention"` works from anywhere.
- **FR-7.** Settings navigation MUST be a scope of the [UX.7](UX.7-navigation-information-architecture.md)
  registry, inheriting its ordering, collision checks and audience filtering.
- **FR-8.** Every settings page MUST use a consistent shell: page title,
  one-sentence description, scope badge ("Affects your organisation"), sections,
  and a single save affordance per section.
- **FR-9.** Settings pages MUST use progressive disclosure — advanced and rarely-
  changed options behind a disclosure, never a wall of 40 controls.
- **FR-10.** Every setting MUST show its **current effective value and where it
  comes from** (default / platform / organisation / course override).
- **FR-11.** Org-wide and platform-wide changes MUST show a **blast-radius
  statement** before saving ("This will apply to 1,240 users in 32 courses").
- **FR-12.** Every settings change MUST be attributable in the existing audit log,
  and the page MUST link to the relevant audit entries.
- **FR-13.** All settings forms MUST be built on [UX.6](../../completed/ui-ux/UX.6-form-and-validation-system.md).
- **FR-14.** The existing PS (Pinned Editor Settings) pattern from `../settings/`
  SHOULD be extended so admins can pin frequently-changed settings.

## 6. Non-Functional Requirements

- **Performance** — Settings search index built at build time, ≤30 KB gzip,
  lazily loaded. Search results ≤50 ms. Settings pages remain route-split.
- **Security** — Redirects MUST NOT bypass authorisation; each destination
  re-checks permission server-side. Search MUST NOT reveal the existence of
  settings the viewer cannot access. Blast-radius counts MUST be computed
  server-side and MUST NOT leak counts across org boundaries.
- **Privacy & Compliance** — Settings changes are already audited; this plan makes
  the audit link discoverable, supporting SOC 2 evidence
  (`../standards/S21-compliance-evidence-continuous-monitoring.md`).
- **Accessibility** — Consistent page shell means consistent heading structure;
  search results are a listbox with correct ARIA; the highlight-on-deep-link must
  not rely on colour alone.
- **Scalability** — Adding a setting means adding a registry/index entry; the
  hierarchy must hold at 200 settings.
- **Reliability** — Redirects MUST be permanent and covered by tests; a broken
  redirect is a support incident.
- **Observability** — Emit `settings_search_query`, `settings_search_no_results`
  (the direct signal for missing synonyms), `settings_page_view`,
  `settings_saved`.
- **Maintainability** — One classification table; no page may exist outside it.
- **Internationalization** — Setting labels, descriptions and synonyms translated
  across all four locales; search matches on the active locale.
- **Backward compatibility** — Every old URL redirects. Documentation, runbooks
  and screenshots MUST be updated in the same release.

## 7. Acceptance Criteria

- **AC-1.** *Given* the classification table, *When* reviewed, *Then* all 33
  settings destinations and 46 admin pages appear exactly once with a scope.
- **AC-2.** *Given* any pre-migration settings or admin URL, *When* requested,
  *Then* it 301/redirects to the new location and the destination renders.
- **AC-3.** *Given* `/admin/*`, *When* audited, *Then* it contains no
  configuration pages — only operational ones.
- **AC-4.** *Given* an admin types "SAML" into settings search, *When* results
  render, *Then* the SSO setting appears and selecting it deep links to and
  highlights that control.
- **AC-5.** *Given* `⌘K` from any route, *When* the admin types "retention",
  *Then* the retention setting is offered.
- **AC-6.** *Given* an admin without a permission, *When* they search, *Then*
  settings they cannot access do not appear in results.
- **AC-7.** *Given* an org-wide setting, *When* the admin saves it, *Then* a
  blast-radius statement is shown before the change is applied.
- **AC-8.** *Given* any setting, *When* viewed, *Then* its effective value and
  source (default/platform/org/course) are displayed.
- **AC-9.** *Given* any settings page, *When* compared to another, *Then* both use
  the identical shell: title, description, scope badge, sections, save.
- **AC-10.** *Given* a settings change, *When* made, *Then* the page links to the
  corresponding audit-log entry.
- **AC-11.** *Given* tree testing with ≥20 admin participants, *When* they are
  asked to locate 10 settings, *Then* task success ≥85%.
- **AC-12.** *Given* the top 20 settings pages, *When* axe runs, *Then* 0
  violations.

## 8. Data Model

```sql
-- server/migrations/NNN_admin_pinned_settings.sql   (FR-14)
CREATE TABLE user_pinned_settings (
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  setting_key text        NOT NULL,
  position    int         NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, setting_key)
);
CREATE INDEX user_pinned_settings_user_pos_idx ON user_pinned_settings (user_id, position);
```

- No schema change is required for the IA work itself; redirects and
  classification are routing concerns.
- **Backfill** — none.
- `setting_key` validated against the settings index on write; unknown keys
  dropped so retired settings self-clean.

## 9. API Surface

```ts
// GET /api/v1/settings/index                          (auth: any authenticated)
// Returns only entries the caller may access.
type SettingsIndexEntry = {
  key: string
  scope: 'me' | 'org' | 'platform' | 'course' | 'operations'
  route: string
  anchor: string | null       // deep-link target within the page
  labelKey: string
  descriptionKey: string
  synonyms: string[]
  requiredPermission: string | null
}

// GET /api/v1/settings/{key}/blast-radius             (auth: scope admin)
type BlastRadius = { users: number; courses: number; orgs: number }

// GET /api/v1/settings/{key}/effective                (auth: scope reader)
type EffectiveValue = {
  value: unknown
  source: 'default' | 'platform' | 'org' | 'course'
  overriddenBy: { scope: string; id: string } | null
}
```

- No WebSocket events. Standard settings rate limits.
- **OpenAPI** — all routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — a settings landing page per scope with search-first layout.
- **Modified pages** — all 48 settings panels adopt the shared shell; the ~15
  admin pages that are actually configuration move scope; `AdminLayout.tsx` is
  reframed as an Operations console.
- **Key user flows**
  1. Admin opens Settings → types in search → jumps straight to the control,
     highlighted.
  2. Admin changes an org setting → sees blast radius → confirms → sees the audit
     link.
  3. Admin follows a stale bookmark → is redirected transparently.
- **States** — search: idle (recent + pinned), typing, results, **no results**
  (offer the closest matches and a "request this setting" link — no-result queries
  are the synonym backlog), error, offline (index is cached).
- **Mobile/responsive** — settings nav collapses to a drawer; search is primary on
  small viewports.
- **Accessibility** — search is a combobox with `aria-activedescendant`; deep-link
  highlight uses the existing focus-anchor pattern documented in
  `docs/accessibility/focus-anchor-highlight.md`, moving focus and announcing,
  not merely tinting; scope badge is text, not colour alone.
- **Copy & i18n** — every setting label, description and synonym is an i18n key at
  parity across four locales. Scope badges use the shared status vocabulary
  (`components/ui/status-vocabulary.tsx`).

## 11. AI / ML Considerations

Not AI-touching in v1. *(Deferred: natural-language settings search — "stop
students seeing each other's grades" → the relevant controls. Attractive, but it
must not act; it may only navigate. If adopted it needs grounding in the settings
index only, no free generation, and the standard PII-redaction and cost-budget
treatment.)*

## 12. Integration Points

- **External** — none.
- **Internal**
  - `clients/web/src/pages/lms/settings.tsx`, `components/settings/**` (48 files)
  - `clients/web/src/pages/admin/**` (46 pages), `pages/admin/AdminLayout.tsx`
  - `clients/web/src/components/layout/side-nav-settings-links.tsx`,
    `side-nav-admin-links.tsx`, `side-nav-course-settings-links.tsx`
  - `clients/web/src/app.tsx` — redirect routes
  - `clients/web/src/components/command-palette/**` — settings index
  - `clients/web/src/components/checklist/**` — CC deep-link/highlight pattern
    is reused for settings anchors
  - `server/internal/httpserver` — index, blast-radius, effective-value routes
  - `docs/runbooks/**` — every runbook referencing a settings URL
- **Events** — settings telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.7](UX.7-navigation-information-architecture.md)
  (settings nav is a registry scope), [UX.6](../../completed/ui-ux/UX.6-form-and-validation-system.md)
  (restructured pages should land on the new form system).
- **Must ship before** — nothing blocking, but it materially improves admin
  onboarding and should precede any large admin-facing feature push.
- **Shared infra** — admin participant recruitment for AC-11.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Moving admin URLs breaks customer bookmarks, runbooks and support macros | **H** | **H** | FR-4 permanent redirects, tested (AC-2); update all runbooks in the same release; announce to customers with a mapping table |
| The scope model has genuinely ambiguous cases (is "content filter" config or ops?) | **H** | M | Classification table is **reviewed and committed** before any move; ambiguity resolved once, in writing, with a documented tiebreak rule (does it change behaviour for others? → config) |
| Search returns nothing useful because synonyms are missing | M | M | `settings_search_no_results` telemetry is the synonym backlog; seed synonyms from support-ticket language |
| 48 panels adopting a shared shell is a large mechanical change | M | M | Shell is additive; migrate panel-by-panel; visual regression per panel |
| Blast-radius counts are expensive to compute | M | M | Computed on demand, cached briefly; approximate counts acceptable and labelled as such |
| Admins dislike the change more than the status quo | M | M | AC-11 tree test with real admins **before** shipping; flag-gated rollout |

## 15. Rollout Plan

- **Feature flag** — `ffSettingsIaV2`, default off, for the hierarchy change.
  Redirects ship unflagged (they must work regardless). Search ships unflagged as
  a pure addition.
- **Sequencing**
  1. Classification table authored and reviewed. **Gate.**
  2. Settings index + search (pure addition, unflagged) — immediate value even
     before restructure.
  3. Shared page shell adopted across 48 panels.
  4. Redirect infrastructure + tests.
  5. Scope moves behind `ffSettingsIaV2`.
  6. Admin console reframed as Operations.
  7. Effective-value display and blast-radius statements.
  8. Pinned settings (FR-14).
- **Dogfood** — internal org admins, 2 weeks.
- **GA criteria** — AC-1…AC-12 green; tree-test success ≥85%; redirect suite 100%
  green; runbooks updated.
- **Rollback** — `ffSettingsIaV2` off restores the old hierarchy; redirects are
  bidirectional-safe during the transition window.

## 16. Test Plan

- **Unit** — classification completeness (every route classified exactly once);
  search index build; synonym matching; permission filtering of index entries;
  effective-value resolution across the four scopes.
- **Integration** — index/blast-radius/effective-value authz matrix (self / org
  admin / platform admin / cross-org); audit-log linkage; blast-radius correctness
  against seeded data.
- **End-to-end** — Playwright: a **generated redirect suite** asserting every
  pre-migration URL resolves; search → deep link → highlight → focus; org setting
  save showing blast radius; stale-bookmark journey.
- **Security** — cross-org blast-radius leakage; index enumeration by an
  under-privileged user; redirect open-redirect check; authz re-verified at each
  destination.
- **Accessibility** — axe on the top 20 settings pages (AC-12); screen-reader
  script for search → result → deep-linked control (must announce arrival);
  keyboard-only settings traversal.
- **Performance / load** — search latency ≤50 ms; index bundle ≤30 KB gzip;
  blast-radius query p95 ≤300 ms.
- **User research** — tree test with ≥20 admins (AC-11); 5 moderated sessions on
  the 10 hardest-to-find settings.
- **Manual exploratory** — QA checklist per scope × permission level; verify no
  configuration remains under `/admin/*`.

## 17. Documentation & Training

- **End-user** — none.
- **Admin** — help-centre: "Where settings live" with the scope model and a
  full mapping table of old → new URLs; "Finding a setting".
- **Engineer** — `docs/guides/settings-architecture.md`: the scope model, the
  tiebreak rule, how to add a setting to the index, how deep-link anchors work.
- **API reference** — OpenAPI for index/blast-radius/effective-value.
- **Runbook** — update **every** runbook in `docs/runbooks/` that cites a settings
  or admin URL; add "A customer's settings bookmark 404s".
- **Customer comms** — release note with the mapping table before GA.

## 18. Open Questions

1. Is "Operations vs Configuration" the right primary split, or should the split
   be by **scope** alone (Me / Org / Platform) with operations folded in as a
   section? The admin card sort decides.
2. Do org admins and platform super-admins need visibly distinct consoles, or one
   console with scope badges? *Recommendation: one console with badges — two
   consoles is how we got here.*
3. Should course settings (`/courses/:code/settings/*`) join the same index and
   search? *Recommendation: yes for search, no for navigation — an instructor
   should not browse org settings.*
4. Who owns the classification table long-term, and what stops a new page being
   added outside it? (Proposed: a CI check that every route under
   `/settings` or `/admin` appears in the table.)
5. Do we need a per-setting "changed recently" indicator to help admins diagnose
   configuration drift?

## 19. References

- Existing files: `clients/web/src/pages/lms/settings.tsx`,
  `clients/web/src/components/settings/` (48 files),
  `clients/web/src/pages/admin/` (46 pages), `pages/admin/AdminLayout.tsx`,
  `clients/web/src/components/layout/side-nav-settings-links.tsx`,
  `side-nav-admin-links.tsx`, `clients/web/src/app.tsx`,
  `docs/accessibility/focus-anchor-highlight.md`
- Research: [research.md](research.md) R-9, R-10, R-11, R-15, R-29
- Audit: [audit.md](audit.md) G-12, G-5
- Related plans: [UX.6](../../completed/ui-ux/UX.6-form-and-validation-system.md),
  [UX.7](UX.7-navigation-information-architecture.md),
  `../settings/` (PS pinned-settings precedent), `../checklist/` (deep-link
  highlight precedent), `../../completed/18-admin-experience/`
