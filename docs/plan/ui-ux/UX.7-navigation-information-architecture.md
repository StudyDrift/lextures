# UX.7 — Navigation Information Architecture

> Implementation plan. Source: [audit.md](audit.md) §4 G-5, G-5d.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.7 |
| **Section** | UI/UX — Information Architecture |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — flat, alphabetical, feature-flag-driven, up to 40 links |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Product Design + Web |
| **Depends on** | UX.2 (nav components), UX.15 (nav labels are hardcoded English) |
| **Unblocks** | UX.8, UX.9, UX.10 |

---

## 1. Problem Statement

The in-course sidebar renders up to **40 links across 8 sections** for an
instructor with all features enabled; the *Grades & insights* section alone has
**15 links ordered alphabetically**, placing **Gradebook — the most-used
instructor destination in any LMS — sixth**, below "Event log" and "Evaluation
results". The global sidebar renders up to **28 links across 6 sections**. Seven
icons are reused across two-to-four destinations each, and three global items
(*Credential wallet*, *My credentials*, *My achievements*) share one icon with
near-identical names — the semantic-overlap failure mode NN/g identifies as the
primary cause of low findability (**R-11**). Every one of 114 platform features
that owns a nav slot adds a permanent link; there is no overflow, no
prioritisation, no personalisation, and no persistent search field in the top bar.

## 2. Goals

- Restructure both navigations around **user tasks**, not feature ownership.
- Reduce visible top-level choices to a scannable set, with the long tail behind
  deliberate disclosure (**R-9**).
- Order by **frequency and task sequence**, never alphabetically below ~20 items
  (**R-12**).
- Eliminate icon and label collisions so every destination is distinguishable at a
  glance (**R-11**).
- Add a **persistent search affordance** in the primary chrome (**R-30**).
- Give users **autonomy** over their own navigation — pin, hide, reorder (**R-4**).
- Validate the new structure with **tree testing and first-click testing** before
  it ships (**R-15**).

## 3. Non-Goals

- Removing features. Every destination remains reachable; this plan changes
  *arrangement and disclosure*, not availability.
- Changing URLs. Route paths stay stable to preserve deep links, bookmarks, the
  CC deep-link/highlight targeting, and native-client parity.
- Settings/admin restructure — that is [UX.8](UX.8-settings-and-admin-ia-unification.md).
- Redesigning destination pages themselves.
- Native client navigation (though the taxonomy MUST be shared; see §12).

## 4. Personas & User Stories

- **As an instructor**, I want the gradebook one click away and visually first
  among grading tools, because I use it every day.
- **As a student**, I want to see only what applies to me, so that I am not
  scrolling past fifteen instructor reports.
- **As a student**, I want to tell *Credential wallet* from *My credentials* from
  *My achievements* without opening all three.
- **As an administrator**, I want to find a destination by typing its name from
  anywhere.
- **As a power user**, I want to pin the four places I actually go and hide the
  rest.
- **As a new instructor**, I want the sidebar to teach me the shape of the product
  rather than overwhelm me on day one.
- **As a parent**, I want a small, obvious set of destinations relevant to my
  child.

## 5. Functional Requirements

### Structure

- **FR-1.** A single **navigation registry** MUST be the source of truth for every
  destination, declaring: id, stable route, label i18n key, icon, section, task
  category, required permission, required feature flag, audience (student /
  instructor / admin / parent), and a **default priority rank**.
- **FR-2.** Both sidebars MUST render from the registry. Hand-written `<SideNavLink>`
  lists MUST be removed.
- **FR-3.** Sections MUST be ordered by **task sequence**, and links within a
  section by **default priority rank** — never alphabetically while the section
  has fewer than 20 items (**R-12**).
- **FR-4.** Each sidebar MUST show at most **7 primary destinations plus grouped
  sections**; anything beyond a section's visible budget MUST collapse behind a
  **"More"** disclosure that remembers its state.
- **FR-5.** The registry MUST enforce **icon uniqueness** within a navigation
  scope. A CI check MUST fail on a duplicate icon or a duplicate/near-duplicate
  label (Levenshtein threshold) within the same scope.
- **FR-6.** Destinations whose names are currently ambiguous MUST be renamed or
  merged. Specifically: consolidate *Credential wallet* / *My credentials* /
  *My achievements* into one **Achievements** destination with internal tabs;
  disambiguate *Standards coverage* vs *Standards gradebook*; disambiguate
  global *Reports* vs course *Reports*.
- **FR-7.** Sections MUST be collapsible, and collapse state MUST persist per user
  per scope.

### Audience and permission

- **FR-8.** Navigation MUST be filtered by **audience first**, then permission,
  then feature flag. A student MUST NOT see instructor analytics sections at all.
- **FR-9.** The existing "View as: Student/Teacher" control MUST switch the
  navigation audience, so a teacher previewing as a student sees the student
  navigation.
- **FR-10.** When a feature flag is off, its destinations MUST be absent — not
  disabled — and MUST NOT leave an empty section heading.

### Personalisation (autonomy — R-4)

- **FR-11.** Users MUST be able to **pin** any destination to a "Pinned" group at
  the top of the sidebar, reorder pins, and unpin.
- **FR-12.** Users MUST be able to **hide** any non-essential destination; hidden
  items remain reachable via search and a "Show hidden" toggle.
- **FR-13.** Personalisation MUST persist server-side per user per scope
  (global / per-course) and sync across devices.
- **FR-14.** The system MUST provide "Reset to default" for both pins and hidden
  items.
- **FR-15.** The system SHOULD surface a **"Recent"** group of the last 3–5
  visited destinations in the current scope, computed client-side.

### Findability

- **FR-16.** A **persistent search affordance MUST be present in the top bar on
  every authenticated route**, at every viewport — not sidebar-only on desktop and
  top-bar-only on mobile as today.
- **FR-17.** The command palette MUST index every registry destination, including
  hidden and overflow items, with fuzzy matching over label, section and synonyms.
- **FR-18.** The palette MUST support **synonyms** declared in the registry (e.g.
  "marks", "grades", "scores" → Gradebook) so users find things by their own
  vocabulary.
- **FR-19.** Breadcrumbs MUST reflect the new section taxonomy and remain accurate
  for every route.

### Feature-flag governance

- **FR-20.** Adding a platform feature MUST NOT automatically add a top-level nav
  link. New destinations MUST declare a section and priority rank in the registry,
  and a CI check MUST fail if a feature adds a destination without them.

## 6. Non-Functional Requirements

- **Performance** — Registry evaluation (permissions × flags × audience ×
  personalisation) MUST be memoised and MUST NOT re-run on every route change.
  Sidebar render ≤5 ms. Personalisation MUST load with the initial session payload
  to avoid a nav flash.
- **Security** — Navigation filtering is **presentation only**; server-side
  authorisation remains authoritative. The registry MUST NOT leak the existence of
  destinations the user cannot access (no disabled placeholders).
- **Privacy & Compliance** — "Recent destinations" is behavioural data; it MUST be
  client-local unless the user opts into sync, and MUST be covered by the RoPA
  entry (`../standards/S05-ropa-data-inventory-mapping.md`).
- **Accessibility** — Sidebar is a `nav` landmark with an accessible name;
  sections use a disclosure pattern with correct `aria-expanded`; the current page
  is `aria-current="page"`; collapsed sidebar tooltips are accessible components,
  not `title` (see [UX.4](UX.4-aria-widget-and-focus-management-remediation.md)).
- **Scalability** — Adding a destination is a registry entry. The structure must
  hold at 150 destinations.
- **Reliability** — If personalisation fails to load, the default navigation
  renders; personalisation is never a blocking dependency.
- **Observability** — Emit `nav_item_click` (destination id, scope, source:
  sidebar/palette/pinned/recent), `nav_search_open`, `nav_search_select`,
  `nav_pin_add/remove`, `nav_hide`. These directly drive the priority ranks.
- **Maintainability** — One registry file per scope; no navigation logic in JSX.
- **Internationalization** — Every label from an i18n key (today they are
  hardcoded English — see [UX.15](UX.15-i18n-coverage-and-rtl-completion.md)).
  Sidebar mirrors in RTL. Synonyms are per-locale.
- **Backward compatibility** — All routes unchanged. Deep links, CC checklist
  highlight targeting, and native-client parity preserved.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor with all features enabled, *When* they open a
  course, *Then* the sidebar shows at most 7 primary destinations plus collapsed
  sections, and **Gradebook is the first item in its section**.
- **AC-2.** *Given* the registry, *When* the collision check runs, *Then* there
  are **0** duplicate icons and **0** near-duplicate labels within any scope.
- **AC-3.** *Given* a student, *When* they open a course, *Then* no instructor
  analytics destination is present in the DOM.
- **AC-4.** *Given* a teacher using "View as: Student", *When* the sidebar
  renders, *Then* it matches what a student sees.
- **AC-5.** *Given* any authenticated route at any viewport, *When* rendered,
  *Then* a search affordance is visible in the top bar.
- **AC-6.** *Given* a user types "marks", *When* the palette searches, *Then*
  Gradebook appears via the synonym index.
- **AC-7.** *Given* a user pins three destinations and hides two, *When* they sign
  in on another device, *Then* the same pins and hidden items apply.
- **AC-8.** *Given* a hidden destination, *When* searched in the palette, *Then*
  it is still findable and navigable.
- **AC-9.** *Given* a feature flag is disabled, *When* the sidebar renders, *Then*
  neither its destinations nor an empty section heading appear.
- **AC-10.** *Given* the proposed taxonomy, *When* tree-tested with ≥30
  participants per persona, *Then* task success ≥80% and directness ≥70% on the 12
  core findability tasks — **and this must pass before implementation begins**.
- **AC-11.** *Given* the shipped navigation, *When* first-click tested against the
  same 12 tasks, *Then* first-click success ≥75%.
- **AC-12.** *Given* a new platform feature is added without a registry section
  and rank, *When* CI runs, *Then* it fails.
- **AC-13.** *Given* the sidebar, *When* operated by keyboard and screen reader,
  *Then* landmarks, disclosure state, and `aria-current` are all correct, and axe
  reports 0 violations.

## 8. Data Model

```sql
-- server/migrations/NNN_user_nav_preferences.sql
CREATE TABLE user_nav_preferences (
  user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  scope        text        NOT NULL,          -- 'global' | 'course:<course_id>'
  pinned       jsonb       NOT NULL DEFAULT '[]'::jsonb,   -- ordered destination ids
  hidden       jsonb       NOT NULL DEFAULT '[]'::jsonb,   -- destination ids
  collapsed    jsonb       NOT NULL DEFAULT '[]'::jsonb,   -- section ids
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, scope)
);

CREATE INDEX user_nav_preferences_user_idx ON user_nav_preferences (user_id);
```

- **Constraints** — `scope` validated server-side against the known scope grammar.
  Destination ids validated against the registry on write; unknown ids are dropped
  (registry is the authority, so a retired destination self-cleans).
- **Backfill** — none. Absent row means defaults.
- **Retention** — deleted with the user (FK cascade), satisfying
  `../standards/S02-data-retention-deletion-engine.md`.

## 9. API Surface

```ts
// GET /api/v1/nav/preferences?scope=global            (auth: self)
// PUT /api/v1/nav/preferences                         (auth: self)
type NavPreferences = {
  scope: string
  pinned: string[]     // ordered destination ids
  hidden: string[]
  collapsed: string[]
}

// DELETE /api/v1/nav/preferences?scope=global         (auth: self) — reset to default
```

- Unknown destination ids are silently dropped on write (not an error) so a client
  running an older registry cannot corrupt state.
- Session bootstrap payload MUST include the global scope preferences to avoid a
  nav flash.
- No WebSocket events. Standard per-user settings rate limit.
- **OpenAPI** — all three routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none. New UI: a "Customise navigation" sheet (pin, hide,
  reorder, reset).
- **Modified pages** — `side-nav.tsx` and all six `side-nav-*-links.tsx` files
  (~62 KB of hand-written nav) collapse into registry-driven rendering;
  `top-bar.tsx` gains a persistent search field; `top-bar-breadcrumbs.tsx` follows
  the new taxonomy.
- **Proposed taxonomy** (to be validated by AC-10 tree testing, not assumed):

  *In-course, instructor:*
  | Section | Contents |
  |---|---|
  | **(primary)** | Dashboard · Modules · Gradebook · People · Checklist |
  | **Teach** | Syllabus · Files · Question bank · Live quizzes · Content tools · Whiteboard |
  | **Engage** | Discussions · Feed · Groups · Boards · Live sessions · Office hours · Collab docs |
  | **Assess & analyse** | At-risk · Outcomes · Standards · Mastery · Report cards · Reports · What's working · Misconceptions · Event log · Evaluations |
  | **Manage** | Settings · Enrollments · Final grades · Attendance |

  *In-course, student:*
  | Section | Contents |
  |---|---|
  | **(primary)** | Dashboard · Modules · My grades · Syllabus |
  | **Participate** | Discussions · Feed · Groups · Live sessions · Office hours · Boards |
  | **My work** | Notebook · Calendar · Attendance |

- **Key user flows**
  1. Instructor opens a course → Gradebook is visible without scrolling.
  2. User types `⌘K` or clicks top-bar search → types "marks" → lands on Gradebook.
  3. User opens "Customise navigation" → pins Gradebook and Modules → hides
     "Event log" → order persists everywhere.
  4. Student opens the same course → sees a short, student-shaped list.
- **States** — sidebar: loading (skeleton preserving item count to avoid shift),
  empty (no course access → explanatory state), error (preferences failed → silent
  fallback to defaults), offline (last-known nav from cache).
- **Mobile/responsive** — sidebar becomes a drawer; pinned + recent surface first;
  search is always in the top bar (FR-16).
- **Accessibility annotations** — `nav` landmarks named "Main" and "Course";
  sections are disclosures with `aria-expanded`; `aria-current="page"`; skip link
  to `main`; collapsed-rail tooltips are accessible components.
- **Copy & i18n** — every label becomes an i18n key. Section names and synonyms
  translated across all four locales. **Renames (FR-6) require terminology review
  against `scripts/check-homeschool-terminology.sh` and `docs/brand/`.**

## 11. AI / ML Considerations

Not AI-touching in v1. Priority ranks are **manually curated from
`nav_item_click` telemetry**, not model-driven — deliberately, because an
adaptive nav that reorders itself violates the spatial-memory expectation users
rely on. *(Deferred: a "suggested pins" prompt after N visits, mirroring the
approach already shipped in `../settings/` PS.4. If adopted, it needs the same
opt-in and explainability treatment.)*

## 12. Integration Points

- **External** — none.
- **Internal**
  - `clients/web/src/components/layout/side-nav*.tsx` (10 files, ~62 KB) — replaced
  - `clients/web/src/components/layout/top-bar.tsx` — persistent search
  - `clients/web/src/components/layout/top-bar-breadcrumbs.tsx` — taxonomy
  - `clients/web/src/components/command-palette/command-palette-dialog.tsx` —
    registry + synonym index
  - `clients/web/src/context/platform-features-context.tsx`,
    `components/settings/platform-feature-definitions.ts` — flag → registry link
  - `clients/web/src/context/use-permissions.ts`, `lib/rbac-api.ts`
  - `clients/web/src/lib/course-view-as.ts` — audience switching
  - `clients/web/src/components/checklist/**` — CC deep-link targeting must keep
    working
  - `server/internal/httpserver` — nav preference routes
  - `clients/ios`, `clients/android` — **MUST consume the same taxonomy**; the
    registry SHOULD be emitted as a shared JSON artefact
- **Events** — nav telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md)
  (disclosure/tooltip components), [UX.15](UX.15-i18n-coverage-and-rtl-completion.md)
  (labels must be i18n keys before they are centralised — or the two land
  together).
- **Must ship before** — [UX.8](UX.8-settings-and-admin-ia-unification.md)
  (settings nav is a registry scope), [UX.9](UX.9-role-aware-dashboard.md).
- **Gated by research** — AC-10 tree testing MUST pass **before** implementation.
- **Shared infra** — user-research participant recruitment; telemetry pipeline.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Users' spatial memory is broken by re-arrangement | **H** | **H** | One-time "what moved" overlay; keep all URLs; palette finds everything; ship once, not iteratively |
| The taxonomy we design is wrong | M | **H** | AC-10 tree testing is a **gate before implementation**, with ≥30 participants per persona (**R-15**) |
| "More" disclosure hides things users need daily | M | M | Priority ranks derived from real `nav_item_click` telemetry, collected for 4 weeks before the taxonomy is finalised |
| Personalisation creates unsupportable per-user states ("it's not there for me") | M | M | "Reset to default" always available; support tooling shows a user's nav prefs; hidden items always findable via search |
| Renames (FR-6) confuse existing users and break docs/screenshots | M | M | Terminology review; redirect old labels via palette synonyms; update help centre and the CC checklist copy in the same release |
| Registry becomes a god-file | M | L | One file per scope; conforms to `docs/ARCHITECTURE_CONVENTIONS.md` budgets |
| Native clients drift from web taxonomy | **H** | M | Shared JSON artefact (§12); parity check in CI |

## 15. Rollout Plan

- **Feature flag** — `ffNavigationV2`, defaulting off. This *is* flagged, unlike
  the refactor plans, because it is a genuine experience change with spatial-memory
  risk.
- **Sequencing**
  1. Instrument current navigation; collect `nav_item_click` for **4 weeks**.
  2. Card sort + tree test the proposed taxonomy (AC-10). **Gate.**
  3. Build the registry; render current nav from it with **no visible change**
     (pure refactor, unflagged). Removes ~62 KB of hand-written nav.
  4. Collision check + renames (FR-6) behind `ffNavigationV2`.
  5. New taxonomy, sections, "More", audience filtering behind the flag.
  6. Persistent top-bar search + palette synonyms (unflagged — pure addition).
  7. Personalisation (pin/hide/reorder).
  8. Internal dogfood → 10% → 50% → GA, watching `nav_search_open` rate (a spike
     means people cannot find things) and task telemetry.
- **Dogfood** — internal org, 3 weeks, with a "what moved" overlay.
- **GA criteria** — AC-1…AC-13 green; first-click success ≥75% (AC-11); no
  sustained increase in nav-search rate; support-ticket signal flat or improved.
- **Rollback** — `ffNavigationV2` off restores the previous arrangement. The
  registry refactor (step 3) stays, since it is behaviour-identical.

## 16. Test Plan

- **Unit** — registry filtering (audience × permission × flag × personalisation);
  priority ordering; overflow budget; collision detection; synonym matching;
  unknown-id dropping.
- **Integration** — preferences GET/PUT/DELETE authz (self only); scope validation;
  session-bootstrap inclusion; graceful degradation when preferences 500.
- **End-to-end** — Playwright per persona (student, instructor, admin, parent) ×
  flag matrix: correct destinations present/absent; "View as" switching; pin/hide
  persistence across sessions; palette finds hidden items; deep links and CC
  checklist highlight targeting still work.
- **Security** — assert that a student's DOM contains no instructor destination;
  assert nav filtering cannot be used to enumerate features the org has disabled;
  authz on preference writes.
- **Accessibility** — axe on both sidebars in all states; screen-reader script:
  traverse landmarks, expand/collapse a section, identify the current page,
  operate the collapsed rail; keyboard-only navigation of the whole sidebar.
- **Performance / load** — sidebar render ≤5 ms; no nav flash on first paint
  (measured as zero CLS attributable to nav); memoisation verified by re-render
  counts across route changes.
- **User research** — card sort (open + closed), tree test (AC-10), first-click
  test (AC-11), and 8 moderated sessions covering the 12 core findability tasks.
- **Manual exploratory** — QA matrix of persona × 114 feature flags sampled at the
  10 most common org configurations.

## 17. Documentation & Training

- **End-user** — help-centre: "Finding your way around", "Customising your
  sidebar"; a one-time in-product "what moved" overlay for existing users.
- **Admin / instructor** — a note in admin docs that disabling a feature removes
  its destinations entirely.
- **Engineer** — `docs/guides/navigation-registry.md`: how to add a destination,
  how ranks are set, why alphabetical ordering is banned below 20 items, how the
  collision check works, how native clients consume the artefact.
- **API reference** — OpenAPI for nav preference routes.
- **Runbook** — "A destination is missing for a user": how to inspect their nav
  preferences and reset them.
- **Update** `AGENTS.md` — new features must register a destination, not append a
  link.

## 18. Open Questions

1. Should the taxonomy be **task-based** ("Teach / Engage / Assess") or
   **object-based** ("Content / People / Grades")? The card sort decides this;
   do not pre-commit.
2. Should *Credential wallet / My credentials / My achievements* merge into one
   destination (FR-6) or be renamed in place? Merging is better IA but touches
   three shipped feature areas — needs product owner sign-off.
3. Does "Recent destinations" (FR-15) sync across devices, and does that make it a
   privacy artefact requiring a RoPA entry?
4. Do native clients adopt the registry in this cycle or the next? Web-only first
   risks a parity gap; simultaneous costs more.
5. How do we handle the 4-week telemetry window — does it block the whole plan, or
   can the registry refactor (step 3) proceed in parallel? *Recommendation:
   parallel; the refactor is behaviour-neutral.*
6. Should the collapsed icon-rail mode survive at all, given it forces reliance on
   tooltips? Test in moderated sessions.

## 19. References

- Existing files: `clients/web/src/components/layout/side-nav.tsx`,
  `side-nav-main-links.tsx`, `side-nav-course-links.tsx`,
  `side-nav-admin-links.tsx`, `side-nav-settings-links.tsx`,
  `side-nav-course-settings-links.tsx`, `side-nav-pinned-courses.tsx`,
  `side-nav-command-palette.tsx`, `top-bar.tsx`, `top-bar-breadcrumbs.tsx`,
  `clients/web/src/components/command-palette/command-palette-dialog.tsx`,
  `clients/web/src/components/settings/platform-feature-definitions.ts`,
  `clients/web/src/app.tsx` (200 routes)
- Research: [research.md](research.md) R-4, R-7, R-8, R-9, R-10, R-11, R-12,
  R-13, R-14, R-15, R-29, R-30, R-33, R-34
- Audit: [audit.md](audit.md) G-5, G-5d, G-12
- External: [NN/g — Top 3 IA Questions about Navigation Menus](https://www.nngroup.com/articles/ia-questions-navigation-menus/),
  [NN/g — Low Findability and Discoverability](https://www.nngroup.com/articles/navigation-ia-tests/),
  [NN/g — Intranet IA Trends](https://www.nngroup.com/articles/intranet-information-architecture-ia/)
- Related plans: [UX.8](UX.8-settings-and-admin-ia-unification.md),
  [UX.9](UX.9-role-aware-dashboard.md),
  [UX.15](UX.15-i18n-coverage-and-rtl-completion.md),
  `../checklist/` (CC deep-link targeting), `../settings/` (PS pinning precedent)
