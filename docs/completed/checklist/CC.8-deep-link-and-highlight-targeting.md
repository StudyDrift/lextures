# CC.8 — Deep-Link & Highlight Targeting ("take me to the exact spot")

> Implementation plan. Source: Course Checklist product request — "navigate directly to the item
> highlighting the spot". Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.8 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web client team (+ mobile for the native target table) |
| **Depends on** | CC.1 (target shape), CC.7 (consumer) |
| **Unblocks** | CC.9 (mobile targets), CC.10 |

---

## 1. Problem Statement

"Set your course dates" is useless if it drops the instructor on a settings page with forty controls. The
checklist is only as good as its ability to land someone **on the exact control that is wrong** and make it
obvious. Lextures has one precedent for addressable controls — the PS.1 settings registry gives every
assignment/quiz editor control a stable string ID — but nothing that turns an ID into a navigation, and no
convention for pages outside those two panels. CC.8 builds the missing layer: a target grammar, a registry
of focus anchors across the course surfaces, a `?focus=` URL contract, and a focus-and-highlight behaviour
that is accessible, reduced-motion-safe and reusable by anything (not just the checklist).

## 2. Goals

- One **target grammar** shared by server rules (CC.1 `NavTarget`), web (CC.7) and mobile (CC.9).
- A **focus-anchor registry** covering every control the checklist rules point at, with an integrity test
  that fails the build when a rule targets an anchor that does not exist.
- A **`?focus=` URL contract** so a checklist target is a plain shareable link, not an in-app-only action.
- A **highlight behaviour** that scrolls, opens whatever container the control is inside (accordion, tab,
  collapsed section), draws attention without relying on colour or motion alone, and announces itself to
  assistive tech.
- Reuse of the existing PS.1 setting IDs rather than a second, parallel ID space.

## 3. Non-Goals

- No new authoring pages or controls; CC.8 only makes existing ones addressable.
- No change to the PS.1 registry's ID contract — CC.8 consumes it.
- No general-purpose product tour / coach-mark framework.
- No native mobile implementation here — CC.8 defines the target table and the web behaviour; CC.9 implements
  the native side.
- No highlight persistence (a highlight is transient by design).

## 4. Personas & User Stories

- **As a teacher**, I want clicking "Set your course dates" to land me on the dates field with it visibly
  called out, so that I fix it in one action.
- **As a teacher**, I want clicking an unmapped assignment to open that assignment's editor with the
  Outcomes section already expanded, so that I do not hunt through collapsed accordions.
- **As a screen-reader user**, I want the focused control announced when I arrive, so that the "highlight"
  is not purely visual.
- **As a support agent**, I want to paste a link that takes an instructor to a specific control, so that
  help articles can be precise.
- **As an engineer**, I want the build to fail when I delete a control a checklist rule points at, so that
  targets never rot silently.
- **As a user with vestibular sensitivity**, I want the highlight to work without a pulsing animation, so
  that the product does not make me ill.

## 5. Functional Requirements

### Target grammar

- **FR-1.** A target MUST be expressible as `{ route, anchor?, params? }` where `route` is a web path
  template (`/courses/{courseCode}/settings/general`), `anchor` is a **focus-anchor ID**, and `params` are
  substitution values supplied per evidence row.
- **FR-2.** Focus-anchor IDs MUST match `^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$` — the same shape as CC.1 item
  IDs and PS.1 setting IDs — and MUST be globally unique across the anchor registry.
- **FR-3.** Anchors that address an assignment or quiz editor control MUST **be** the PS.1 setting ID
  (e.g. `assignment.outcomes-mapping`, `quiz.scores-review`), resolved through PS.1's
  `resolveSettingId` including its alias map. CC.8 MUST NOT mint a parallel ID for a control PS.1 already
  names.
- **FR-4.** A new client registry `clients/web/src/lib/focus-anchors.ts` MUST declare every non-PS.1 anchor
  with: `id`, `route` (the page it lives on), `label` (for the announcement), `container` (optional —
  accordion/tab/section that must be opened first), and `kind` (`control | region | entity`).
- **FR-5.** `entity`-kind anchors MUST accept an ID parameter (e.g. `modules.item:{itemId}`) so evidence
  rows can address a specific row on a list page.

### URL contract

- **FR-6.** Navigating with `?focus={anchorId}` (plus `&focusEntity={id}` for entity anchors) MUST trigger
  the focus behaviour on arrival.
- **FR-7.** The `focus` params MUST be **stripped from the URL** after the behaviour runs (history replace),
  so a refresh or a back-navigation does not re-fire the highlight.
- **FR-8.** An unknown or retired anchor MUST navigate to the route and do nothing else — never error, never
  blank the page — and MUST log a dev-only warning.
- **FR-9.** The contract MUST work for a cold load (pasted link) and for an in-app navigation.

### Focus behaviour

- **FR-10.** A `useFocusAnchor()` hook MUST, on arrival: (1) wait for the target element via a bounded
  observer (max 5 s, then give up silently), (2) open its declared `container` (expand accordion, select
  tab, expand collapsed section), (3) scroll it into view with `block: 'center'`, (4) move DOM focus to the
  control (or to a `tabindex="-1"` wrapper for `region` anchors), and (5) apply the highlight treatment.
- **FR-11.** The highlight treatment MUST be a persistent outline plus an offset ring that remains for 4 s,
  then fades. It MUST NOT rely on colour alone: the ring is paired with a small "Here" marker chip that is
  also readable text.
- **FR-12.** Under `prefers-reduced-motion: reduce`, the scroll MUST be instant (`behavior: 'auto'`) and the
  ring MUST appear and disappear without animation. There MUST be no pulsing, flashing or looping animation
  under any setting (WCAG 2.3.1 — nothing flashes more than three times per second; here: nothing flashes at
  all).
- **FR-13.** Arrival MUST announce via a polite live region: "{label} — this is the setting from your
  checklist." The announcement MUST fire once.
- **FR-14.** Moving focus MUST NOT trap it; the next Tab continues the page's natural order.
- **FR-15.** If the element is inside a virtualised or lazily-rendered list, the hook MUST first request the
  list scroll to the entity (via a registered `revealEntity(id)` callback) before applying focus.
- **FR-16.** The highlight MUST clear early on any user interaction (keypress, click, scroll of > 200 px).

### Coverage & integrity

- **FR-17.** CC.8 MUST provide anchors for at least every target referenced by CC.3–CC.6, including:
  `course.general.title`, `course.general.description`, `course.general.dates`, `course.general.timezone`,
  `course.general.published`, `course.general.visibility`, `course.general.home-landing`,
  `course.general.hero-image`, `course.features.grid`, `course.grading.scheme`, `course.grading.groups`,
  `course.grading.posting-policy`, `course.outcomes.list`, `course.outcomes.item:{id}`,
  `course.sections.list`, `course.accessibility.settings`, `course.import-export.export`,
  `syllabus.section:{id}`, `syllabus.editor`, `modules.list`, `modules.module:{id}`, `modules.item:{id}`,
  `feed.channel:announcements`, `enrollments.list`, `enrollments.invitations`, `discussions.list`,
  `office-hours.slots`, `groups.sets`, `standards-coverage.grid`, `files.item:{id}` — plus the PS.1 IDs
  `assignment.outcomes-mapping`, `assignment.rubric`, `assignment.scheduling`, `assignment.grading`,
  `quiz.outcomes`, `quiz.scores-review`, `quiz.attempts-grading`, `quiz.scheduling`.
- **FR-18.** A build-time test MUST assert that **every** `NavTarget.anchor` emitted by the server catalog
  exists in either the CC.8 anchor registry or the PS.1 settings registry. The server catalog is exported to
  a JSON fixture (`server/internal/service/coursechecklist/testdata/catalog_targets.json`, regenerated by
  `go test -update`) which the web test consumes — so a server-side rule change that breaks a target fails
  the web build.
- **FR-19.** A parity test MUST assert every anchor in the registry is actually rendered by its declared
  route in at least one component test (no dead anchors).
- **FR-20.** Anchors MUST be attachable with a single `useAnchorRef(anchorId)` hook or an `<Anchor id=…>`
  wrapper that adds `data-focus-anchor={id}` — no bespoke IDs per page.
- **FR-21.** CC.8 MUST publish the **native target table** (anchor ID → iOS destination + Android
  destination, or "web-only") as a shared JSON asset consumed by CC.9, so mobile does not re-derive routing.

## 6. Non-Functional Requirements

- **Performance** — The observer MUST use `MutationObserver` scoped to the page root, disconnect on
  resolution or timeout, and add no measurable cost when no `focus` param is present (zero work on the
  common path). Highlight uses CSS classes only — no layout thrash, no JS animation loop.
- **Security** — `focus` and `focusEntity` are opaque IDs validated against the registry before use; they
  MUST NOT be interpolated into selectors without escaping (`CSS.escape`) and MUST NOT be rendered as HTML.
  An unknown value is discarded (FR-8). No new data exposure — the route's own authz still applies.
- **Privacy & Compliance** — `focusEntity` may be a course-object ID; it is stripped from the URL after use
  (FR-7), so it does not linger in browser history or referrers.
- **Accessibility** — WCAG 2.1 AA: 2.4.3 (focus order preserved, FR-14), 2.4.7 (focus visible — the ring
  meets 3:1 against adjacent colours), 1.4.1 (not colour alone — the "Here" chip, FR-11), 2.3.1 (no flashing,
  FR-12), 4.1.3 (status message via live region, FR-13), 2.2.2 (nothing auto-updates or moves after arrival).
  The programmatic focus move is user-initiated (they clicked a checklist item), which is the case where
  moving focus is correct rather than disorienting.
- **Scalability** — Registry of ~60 anchors; O(1) lookup by ID. Adding a surface is one registry entry plus
  one `<Anchor>`.
- **Reliability** — Every failure mode is silent-and-safe: element never appears → give up after 5 s;
  container fails to open → still scroll to the nearest ancestor; entity not in list → land on the list.
- **Observability** — Client events (CC.10 dictionary): `checklist_target_navigated` with `anchorId` and
  `resolved: true|false`. A rising `resolved: false` rate is the signal that an anchor has rotted in a way
  the build test missed.
- **Maintainability** — One registry file, one hook, one CSS class set. The `<Anchor>` wrapper must be
  render-transparent (no extra DOM node where a `ref` will do).
- **Internationalization** — Announcement copy and anchor `label`s are i18n keys; RTL verified (ring offset
  and "Here" chip must mirror).
- **Backward compatibility** — Unknown anchors are ignored forever (FR-8), so an older client following a
  newer server target degrades to plain navigation. Retired anchors go in an alias/retired map like PS.1.

## 7. Acceptance Criteria

- **AC-1.** *Given* a checklist item targeting `course.general.dates`, *When* it is activated, *Then* the app
  navigates to course settings → General, the dates field receives DOM focus, the ring is applied, and the
  URL no longer contains `?focus=`.
- **AC-2.** *Given* an anchor whose control lives inside a collapsed accordion, *Then* the accordion is
  expanded before focus is applied.
- **AC-3.** *Given* an anchor on a tabbed page, *Then* the correct tab is selected first.
- **AC-4.** *Given* an evidence row targeting `assignment.outcomes-mapping` for assignment X, *Then* the app
  opens assignment X's editor with the Outcomes section expanded and focused, resolved through the PS.1
  registry (including via a PS.1 alias).
- **AC-5.** *Given* `?focus=does.not.exist`, *Then* the page renders normally, nothing is focused, no error
  is shown, and a dev-only warning is logged.
- **AC-6.** *Given* `prefers-reduced-motion: reduce`, *Then* scrolling is instant and the ring has no
  transition; no animation runs at any point.
- **AC-7.** *Given* a screen reader, *When* arrival completes, *Then* a single polite announcement is made
  naming the control.
- **AC-8.** *Given* the highlight is showing, *When* the user presses any key or scrolls 200 px, *Then* the
  highlight clears immediately.
- **AC-9.** *Given* an entity anchor for an item far down a virtualised module list, *Then* the list reveals
  the row before focus is applied.
- **AC-10.** *Given* the server catalog fixture, *When* the integrity test runs, *Then* it fails if any rule
  target names an anchor absent from both registries.
- **AC-11.** *Given* the anchor registry, *When* the parity test runs, *Then* it fails for any anchor no
  component renders.
- **AC-12.** *Given* a pasted link `…/settings/general?focus=course.general.timezone` on a cold load, *Then*
  the behaviour runs after hydration exactly as for an in-app navigation.
- **AC-13.** *Given* a malicious `?focus=` value containing selector metacharacters, *Then* it is rejected by
  registry validation and never reaches a DOM query.
- **AC-14.** *Given* an axe scan while a highlight is active, *Then* there are zero serious or critical
  violations and the ring meets 3:1 contrast against its background.

## 8. Data Model

No database changes. Two artefacts:

1. `clients/web/src/lib/focus-anchors.ts` — the anchor registry (static array + derived `Map`), mirroring
   the PS.1 registry conventions (aliases, retired IDs, integrity test).
2. `server/internal/service/coursechecklist/testdata/catalog_targets.json` — generated fixture listing every
   `(itemId, route, anchor)` the server catalog emits; regenerated with `go test -update` and consumed by the
   web integrity test (FR-18). Checked in, so a server change that breaks a target shows up as a diff.
3. `clients/packages/shared/checklist-targets.json` (or the repo's existing shared-package location) — the
   anchor → native destination table (FR-21) consumed by CC.9.

## 9. API Surface

No HTTP changes. Web surface:

```ts
// clients/web/src/lib/focus-anchors.ts
export type FocusAnchorKind = 'control' | 'region' | 'entity'
export type FocusAnchor = {
  id: string
  route: string                    // path template
  labelKey: string; label: string
  kind: FocusAnchorKind
  container?: { type: 'accordion' | 'tab' | 'section'; id: string }
}
export const FOCUS_ANCHORS: FocusAnchor[]
export const FOCUS_ANCHOR_ALIASES: Record<string, string>
export function resolveFocusAnchor(id: string): FocusAnchor | null

// clients/web/src/lib/use-focus-anchor.ts
export function useFocusAnchorRuntime(): void        // mounted once in the course layout
export function useAnchorRef<T extends HTMLElement>(anchorId: string): React.RefObject<T>
export function hrefForTarget(t: NavTarget, params?: Record<string, string>): string
export function registerEntityRevealer(routeKey: string, reveal: (id: string) => void): () => void
```

## 10. UI / UX

**Highlight treatment**

- A 2 px outline in the accent colour plus a 4 px offset ring, meeting 3:1 against adjacent colours in both
  themes.
- A small "Here" chip anchored to the control's top-start corner (mirrored in RTL) — the non-colour,
  non-motion signal.
- Persists 4 s, then fades over 200 ms (instantly under reduced motion), or clears immediately on
  interaction.
- Never pulses, never loops, never flashes.

**Arrival sequence (numbered)**

1. Route change completes.
2. Runtime reads `?focus` / `?focusEntity`, validates against the registry.
3. Container opened (accordion/tab/section) if declared.
4. Entity revealed if `kind: entity`.
5. Element observed until present (≤ 5 s).
6. `scrollIntoView({ block: 'center' })`, instant under reduced motion.
7. `element.focus({ preventScroll: true })`.
8. Ring + chip applied; polite live-region announcement.
9. URL params stripped via `history.replaceState`.
10. Ring cleared on timeout or first interaction.

**Failure states** — anchor unknown → plain navigation. Element never appears → plain page, no announcement.
Entity missing (deleted since the checklist was computed) → land on the list with a subtle inline notice
"That item no longer exists" and a prompt to re-check the checklist.

**Responsive** — `block: 'center'` accounts for sticky headers via `scroll-margin-top` on anchored elements;
on narrow widths, the "Here" chip renders above the control rather than beside it.

## 11. AI / ML Considerations

None.

## 12. Integration Points

- New: `clients/web/src/lib/focus-anchors.ts`, `clients/web/src/lib/use-focus-anchor.ts`,
  `clients/web/src/components/ui/anchor.tsx`, highlight styles in `index.css`.
- Modified (to attach anchors): `pages/lms/course-settings.tsx` and its section components,
  `pages/lms/course-features-section.tsx`, `pages/lms/course-grading-settings.tsx`, the outcomes settings
  section, `pages/lms/course-modules.tsx`, `components/syllabus/syllabus-block-editor.tsx`,
  `pages/lms/course-feed-page.tsx`, the enrollments page, discussions, office hours, groups,
  standards coverage, files.
- Reuses: PS.1 `settings-registry.ts` + `SettingRow` (which already positions controls addressably) and the
  PS.3 pin/scroll behaviour where it overlaps.
- Consumed by: CC.7 (web), CC.9 (native table), help-centre deep links.

## 13. Dependencies & Sequencing

- Must ship after: CC.1 (target shape in the catalog), and alongside/just after CC.7 (its only consumer at
  first).
- Must ship before: CC.9's native highlight work, and before promoting any rule pack to `essential` — a
  badge that sends people to the wrong place is worse than no badge.
- Cross-team: rule-pack authors (CC.3–CC.6) must not invent anchors; they pick from this registry or request
  an addition in the same PR.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Anchors rot as pages are refactored | **H** | H | FR-18 generated-fixture integrity test + FR-19 parity test, both blocking in CI; `resolved: false` telemetry as a runtime backstop |
| Programmatic focus disorients screen-reader users | M | M | Focus move is user-initiated; single polite announcement; no focus trap; verified in the SR script |
| A pulsing highlight triggers vestibular symptoms | M | H | FR-12: no pulsing at all, ever; reduced-motion path is instant |
| Anchor IDs fork from PS.1 setting IDs | M | M | FR-3 forbids minting a parallel ID; integrity test resolves through PS.1 first |
| Element never renders (conditional control) and the user is left confused | M | M | 5 s bounded observer then silent give-up; item copy still explains what to change |
| `?focus=` becomes an XSS vector | L | H | Registry validation before any DOM query, `CSS.escape`, never rendered as HTML (AC-13) |
| Two registries (anchors + PS.1) confuse contributors | M | L | One documented rule: "editor settings → PS.1; everything else → focus-anchors"; documented in §17 |

## 15. Rollout Plan

**No feature flag.** Behaviour is inert unless a `?focus=` param is present, so shipping it has no effect
until CC.7 emits targets.

1. Ship the registry, hook and highlight styles with anchors attached to the highest-traffic surfaces
   (course settings General/Features/Grading/Outcomes, modules, syllabus, enrollments).
2. Ship the integrity fixture and turn the CI test blocking in the same release.
3. Extend anchor coverage to the remaining surfaces as CC.4–CC.6 rule packs land; each rule pack PR must
   land its anchors with it.
4. GA criteria: `resolved: false` rate < 1% over a week; axe clean with a highlight active; SR script signed
   off.
5. Rollback: the runtime is a single hook mounted in the course layout — removing the mount disables all
   highlighting while leaving navigation intact.

## 16. Test Plan

- **Unit** — Registry integrity (unique, regex-conformant, aliases resolve, no anchor missing a route).
  `resolveFocusAnchor` including PS.1 delegation and aliases. `hrefForTarget` param substitution and
  encoding. Param stripping. Unknown-anchor no-op.
- **Integration** (RTL) — Container opening for accordion, tab and section variants; entity revealer;
  bounded observer timeout; highlight cleared on keypress/scroll; reduced-motion branch.
- **End-to-end** (Playwright) — From the checklist: item → settings control focused and ringed; evidence row
  → assignment editor with Outcomes expanded; pasted cold-load link; deleted-entity fallback notice.
- **Security** — Injection attempts through `focus`/`focusEntity`; assert no unescaped selector query and no
  HTML rendering of the value.
- **Accessibility** — axe with a highlight active in both themes and RTL; contrast measurement of the ring;
  keyboard walkthrough asserting focus order is preserved after the jump; VoiceOver + NVDA scripts for the
  arrival announcement; reduced-motion verification that no transition runs.
- **Performance / load** — Assert zero observers and zero listeners registered when no `focus` param is
  present; measure time-to-highlight on a 300-item modules page.
- **Manual exploratory** — Every anchor in the registry walked by hand once before GA; sticky-header overlap
  checked at three viewport heights; RTL mirroring of the chip.

## 17. Documentation & Training

- `docs/dev/focus-anchors.md` — the grammar, the two-registry rule ("editor settings live in PS.1; every
  other addressable spot lives in focus-anchors"), how to add an anchor, and how the integrity fixture works.
- Help-centre note that checklist links are shareable URLs support can paste.
- `docs/accessibility/` addendum documenting the highlight's conformance argument (2.4.3, 2.4.7, 1.4.1,
  2.3.1, 4.1.3) so a future auditor does not have to reverse-engineer it.
- Contributor checklist entry: "If you delete or rename a control, check `focus-anchors.ts`."

## 18. Open Questions

1. Should `?focus=` be a query param (proposed, shareable and hydration-safe) or a URL fragment? Fragments
   collide with the existing `#item-{id}` usage on the checklist page itself.
2. Should the highlight persist until interaction rather than timing out after 4 s? Proposed: timeout, so a
   forgotten tab does not keep a ring on screen.
3. Do we need a "highlight without focus" mode for regions where moving focus would be jarring (e.g. a whole
   table)? Proposed: yes — that is what `kind: region` with a `tabindex="-1"` wrapper does; confirm the SR
   experience.
4. Should help-centre articles be allowed to link to anchors, making the registry a public contract?
   Proposed: yes, which raises the bar on retiring an anchor (alias, never delete).
5. Where does the shared native target table live given the current `clients/packages/` layout? Needs a
   concrete path decision with the mobile team before CC.9 starts.

## 19. References

- Existing files this work touches: `clients/web/src/lib/settings-registry.ts` (PS.1),
  `clients/web/src/pages/lms/course-settings.tsx`, `.../course-features-section.tsx`,
  `.../course-grading-settings.tsx`, `.../course-modules.tsx`,
  `clients/web/src/components/syllabus/syllabus-block-editor.tsx`, `clients/web/src/index.css`.
- Precedent: [PS.1](../settings/PS.1-settings-registry-and-addressable-controls.md) (addressable
  controls) and [PS.3](../settings/PS.3-pin-and-reorder-ux-in-editor-panels.md) (revealing a
  control in a collapsed panel).
- Standards: WCAG 2.1 — 1.4.1 Use of Colour, 2.3.1 Three Flashes, 2.4.3 Focus Order, 2.4.7 Focus Visible,
  4.1.3 Status Messages.
- Related plans: [CC.7](CC.7-web-checklist-page-and-nav-badge.md),
  [CC.9](../../plan/checklist/CC.9-mobile-checklist-ios-and-android.md),
  [CC.4](CC.4-rule-pack-structure-outcomes-alignment.md).
- Dev docs: [focus-anchors.md](../../dev/focus-anchors.md),
  [focus-anchor-highlight a11y](../../accessibility/focus-anchor-highlight.md).
