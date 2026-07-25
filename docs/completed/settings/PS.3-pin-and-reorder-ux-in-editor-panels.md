# PS.3 — Pin, Unpin & Reorder UX in the Assignment and Quiz Editors

> Implementation plan (completed). Source: authoring-UX gap — important settings are hidden inside collapsed accordions; instructors need their own settings at the top. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | PS.3 |
| **Section** | Pinned Editor Settings |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web client team |
| **Depends on** | PS.1, PS.2 |
| **Unblocks** | PS.4 |

---

## 1. Problem Statement

An instructor authoring a quiz has to remember that "lockdown delivery" lives under *Presentation*,
that "never drop this score" lives under *Grading*, and that "release grades at" only appears after
choosing manual posting. Ten accordions collapse by default, so the settings a given instructor uses
every week are three clicks and a scroll away, every time. The settings that matter most are also the
easiest to forget entirely — instructors ship quizzes without an access code or a late policy because
they never saw the control. PS.3 lets each instructor promote the settings *they* use to a **Pinned**
group at the top of the panel, persisted to their account by [PS.2](PS.2-pinned-settings-data-model-and-api.md).

## 2. Goals

- Let an instructor pin or unpin any individual setting in the assignment and quiz settings panels,
  in one click or one keystroke, from where the setting already lives.
- Render pinned settings in a **Pinned** group at the very top of the panel, in a user-defined order.
- Make pins persist across sessions, devices, and every assignment/quiz the instructor opens.
- Support reordering by both pointer drag and keyboard, meeting WCAG 2.1 AA.
- Degrade gracefully: flag off, request failure, stale pin keys, and zero pins all produce a working
  panel identical to today's.

## 3. Non-Goals

- No per-course or per-item pins — pins are per user, per surface, and apply to every item they open.
- No sharing, templating, or admin-pushed pin sets.
- No pinning of *sections* as a whole beyond the composite editors already registered as single
  entries in PS.1 (Rubric, Assign to, Outcomes).
- No new settings, no changes to what any setting does, no changes to save semantics.
- No surfaces beyond the two editor panels.
- No suggested/starter pins — that is PS.4.

## 4. Personas & User Stories

- **As an instructor**, I want to pin "Lockdown delivery" so that it is the first thing I see when I
  open any quiz, because proctored exams are most of what I author.
- **As an instructor**, I want to unpin a setting I no longer use, so that my pinned group stays short.
- **As an instructor**, I want to drag my pinned settings into the order I use them, so that the panel
  matches my workflow rather than the product's information architecture.
- **As a keyboard-only instructor**, I want to pin and reorder without a mouse, so that the feature is
  usable to me at all.
- **As a screen-reader user**, I want to hear when a setting is pinned and where it landed, so that I
  can trust the panel changed the way I intended.
- **As a K-12 teacher on a school-managed device**, I want my pins to be there on Monday morning even
  though the district wipes browser storage nightly.
- **As an instructor whose network drops**, I want a failed pin save to tell me and revert, so that I
  never believe a preference stuck when it did not.

## 5. Functional Requirements

### Pinning

- **FR-1.** Every registry-declared control with `pinnable: true` MUST expose a pin toggle in its row,
  in both panels.
- **FR-2.** The pin toggle MUST be a `<button>` with `aria-pressed` reflecting pinned state and an
  accessible name of the form `Pin {label} to top` / `Unpin {label}`.
- **FR-3.** The pin toggle MUST be visible on hover, on focus, and whenever the setting is already
  pinned; it MUST be reachable by keyboard at all times (never `display:none` when unfocused — use
  opacity so it stays in the tab order and is discoverable on focus).
- **FR-4.** Activating the toggle on an unpinned setting MUST append it to the end of that surface's
  pinned list; activating it on a pinned setting MUST remove it.
- **FR-5.** The pinned list MUST be capped at **8** entries in the UI (under PS.2's schema cap of 12);
  when full, further pin toggles MUST be disabled with a tooltip/`aria-describedby` explaining the cap.
- **FR-6.** Pinning MUST **move** the control into the Pinned group rather than duplicating it: the
  control is rendered exactly once in the DOM, preserving its existing `id` and label association.
- **FR-7.** The originating section MUST show a muted, non-interactive hint listing how many of its
  settings are pinned (e.g. "2 pinned to top"), so the setting does not appear to have vanished.

### Pinned group

- **FR-8.** The Pinned group MUST render above the search field's results and above all accordions,
  as an always-open group titled **Pinned**, with a count badge.
- **FR-9.** The Pinned group MUST NOT render at all when the list is empty (no empty shell), except
  during the first-run coach state defined in FR-20.
- **FR-10.** Pinned controls MUST render with the same component, props, disabled state, and
  conditional-visibility rules they have in their home section; a pinned control whose condition is
  false (e.g. `Max attempts` while unlimited attempts is on) MUST be omitted from the Pinned group
  without being unpinned.
- **FR-11.** When a search query is active, the Pinned group MUST be filtered by the same predicate as
  every other control, and MUST be hidden when none of its controls match.

### Reordering

- **FR-12.** Pinned entries MUST be reorderable by pointer drag using the repo's existing `@dnd-kit`
  setup, including `defaultKeyboardSensorOptions` from `clients/web/src/lib/dnd/keyboardSensorConfig.ts`.
- **FR-13.** Pinned entries MUST be reorderable by keyboard independently of drag: with focus on a
  pinned row's drag handle, <kbd>Alt</kbd>+<kbd>↑</kbd>/<kbd>↓</kbd> MUST move the entry one position.
- **FR-14.** Every reorder, pin, and unpin MUST be announced in a polite ARIA live region, e.g.
  "Due date pinned, position 3 of 3" / "Due date moved to position 1 of 3" / "Due date unpinned".

### Persistence

- **FR-15.** Pins MUST be loaded once per editor mount via `fetchPinnedSettings()` and MUST NOT block
  first paint: the panel renders unpinned, then reflows when pins arrive.
- **FR-16.** Pin/unpin/reorder MUST update local state optimistically and persist via
  `savePinnedSettings(surface, keys)` debounced at **500 ms**, coalescing rapid changes into one write.
- **FR-17.** A failed save MUST revert the optimistic change, surface a non-blocking toast
  ("Couldn't save your pinned settings"), and leave the panel usable.
- **FR-18.** Keys returned by the server that `resolveSettingId` cannot resolve MUST be dropped from the
  rendered list; the pruned list MUST be written back only when the user next changes their pins, not
  automatically on load.
- **FR-19.** The panel MUST refetch pins on window focus so a change made in another tab or device
  converges without a reload.

### Discovery & gating

- **FR-20.** On an instructor's first eligible panel render with zero pins, the panel MUST show a
  one-time, dismissible hint ("Pin the settings you use most — they'll show up here"), dismissed
  permanently via `localStorage`; it MUST NOT reappear after dismissal or after the first pin.
- **FR-21.** Every pin affordance, the Pinned group, and all pin API traffic MUST be gated on
  `ffPinnedSettings` from `platform-features-context`; when false the panels MUST render exactly as
  they do before PS.3.

## 6. Non-Functional Requirements

- **Performance** — Pin toggle → visual update < 100 ms (optimistic, no network wait). Reflow when
  pins load MUST not cause layout shift below the fold measurably worse than CLS 0.1. Panel keystroke
  render stays < 16 ms p95 (inherited from PS.1). At most one pin `GET` per editor mount and one
  debounced `PUT` per 500 ms burst.
- **Security** — No new data crosses a boundary beyond PS.2's API. Setting keys are used only as React
  keys and registry lookups — never interpolated into selectors, HTML, or URLs.
- **Privacy & Compliance** — Pins are per-user preferences covered by PS.2's data lifecycle. The
  first-run hint's dismissal flag is local-only and contains no personal data.
- **Accessibility** — WCAG 2.1 AA. Specifically: 2.1.1 Keyboard (pin + reorder without a pointer),
  2.4.3 Focus Order (focus follows a control when it moves between groups), 2.4.7 Focus Visible,
  4.1.2 Name/Role/Value (`aria-pressed` on the toggle), 4.1.3 Status Messages (live-region
  announcements), 1.4.11 Non-text Contrast (pin icon ≥ 3:1 in light and dark themes), 2.5.7 Dragging
  Movements (keyboard alternative to drag, FR-13), 2.5.5 target size ≥ 24×24 CSS px for the toggle.
- **Scalability** — Rendering cost is bounded by the 8-pin UI cap; no virtualisation needed.
- **Reliability** — Every failure mode (flag off, `GET` fails, `PUT` fails, unknown keys, offline)
  resolves to a fully functional panel. No pin state is required to author or save an item.
- **Observability** — Client-side counters emitted through the existing telemetry path for
  `pin_added`, `pin_removed`, `pin_reordered`, `pin_save_failed`, each with `surface` and
  `setting_id`; consumed by PS.4. No PII beyond the authenticated user context.
- **Maintainability** — Pin rendering lives in `clients/web/src/components/settings-panel/`; both
  panels consume the same `usePinnedSettings()` hook, so behaviour cannot diverge between them.
- **Internationalization** — All new strings (group title, hint, toggle labels, toast, live-region
  templates) MUST be externalised the same way the panels' existing copy is handled, with
  interpolation slots for the setting label and position numbers.
- **Backward compatibility** — Flag-gated and additive. With the flag off the panels are byte-identical
  to PS.1's output.

## 7. Acceptance Criteria

- **AC-1.** *Given* `ffPinnedSettings` is on and I have no pins, *When* I open a quiz editor, *Then* no
  Pinned group renders and the panel matches the pre-PS.3 layout apart from the one-time hint.
- **AC-2.** *Given* the quiz panel, *When* I click the pin toggle on "Lockdown delivery", *Then* a
  Pinned group appears at the top containing that control, the Presentation section shows
  "1 pinned to top", and the control appears exactly once in the DOM.
- **AC-3.** *Given* a pinned setting, *When* I reload the page, *Then* it is still pinned and in the
  same position.
- **AC-4.** *Given* pins created in the quiz editor, *When* I open the assignment editor, *Then* the
  assignment panel shows only assignment-surface pins.
- **AC-5.** *Given* three pinned settings, *When* I focus the second one's drag handle and press
  <kbd>Alt</kbd>+<kbd>↑</kbd>, *Then* it moves to position 1, focus stays on it, and the live region
  announces "… moved to position 1 of 3".
- **AC-6.** *Given* three pinned settings, *When* I drag the third above the first with a pointer,
  *Then* the new order persists after reload.
- **AC-7.** *Given* 8 pinned settings, *When* I try to pin a ninth, *Then* the toggle is disabled and
  its description explains the limit; nothing is written.
- **AC-8.** *Given* a pinned "Max attempts" and "Unlimited attempts" turned on, *Then* "Max attempts"
  is absent from the Pinned group but remains pinned, and reappears there when unlimited is turned off.
- **AC-9.** *Given* the pin `PUT` returns `500`, *When* I pin a setting, *Then* the pin reverts, a
  toast appears, and the panel remains fully usable.
- **AC-10.** *Given* the pin `GET` fails, *When* the editor loads, *Then* the panel renders with no
  Pinned group and no error dialog.
- **AC-11.** *Given* stored pins include a retired key, *When* the panel renders, *Then* that key is
  silently ignored and the remaining pins render in order.
- **AC-12.** *Given* `ffPinnedSettings` is off, *When* I open either editor, *Then* there are no pin
  toggles, no Pinned group, and no requests to `/api/v1/me/pinned-settings`.
- **AC-13.** *Given* I pin a setting in tab A, *When* I focus tab B's open editor, *Then* tab B's
  Pinned group converges to the same list without a reload.
- **AC-14.** *Given* a search query matching only a pinned control, *When* results render, *Then* the
  Pinned group shows that control and all accordions are hidden.
- **AC-15.** *Given* the assignment panel with pins, *When* `axe` runs in light and dark themes,
  *Then* there are zero violations, including contrast on the pin icon and drag handle.
- **AC-16.** *Given* a screen reader, *When* I pin a setting, *Then* the toggle's state change is
  announced via `aria-pressed` and the position is announced via the live region.

## 8. Data Model

No schema changes. PS.3 consumes `settings.user_pinned_settings` through PS.2's API.

Client state shape (per surface, held by `usePinnedSettings`):

```ts
type PinnedState = {
  status: 'loading' | 'ready' | 'unavailable'  // 'unavailable' = flag off or GET failed
  keys: string[]                                // server truth, unresolved keys included
  resolved: SettingDescriptor[]                 // keys → descriptors, unknown dropped (FR-18)
  saving: boolean
}
```

Local storage: `lextures.pinned-settings.hint-dismissed` (`'1'`), the only client-persisted value.

## 9. API Surface

No new endpoints. PS.3 consumes:

- `GET /api/v1/me/pinned-settings` — once per editor mount and on window focus (FR-19).
- `PUT /api/v1/me/pinned-settings/{surface}` — debounced 500 ms per change burst (FR-16).
- `GET /api/v1/platform/features` — existing call, reads `ffPinnedSettings`.

Client budget: ≤ 1 `GET` per mount plus 1 per focus event; ≤ 2 `PUT`/s sustained worst case, well
inside PS.2's 60/min per-user budget.

## 10. UI / UX

### New components (`clients/web/src/components/settings-panel/`)

- `pinned-settings-group.tsx` — the Pinned group: heading, count badge, dnd-kit `SortableContext`,
  live region, empty-hint state.
- `pin-toggle.tsx` — the per-row toggle button (pin/unpin, disabled-at-cap variant).
- `use-pinned-settings.ts` — load, optimistic mutate, debounce, save, refetch-on-focus, prune.

### Modified components

- `clients/web/src/components/settings-panel/setting-row.tsx` (from PS.1) — renders the pin toggle and
  cooperates with the layout context to place the control in either the Pinned group or its section.
- `assignment-page-settings-panel.tsx`, `quiz-page-settings-panel.tsx` — mount the Pinned group and the
  hook; add the per-section "N pinned to top" hint.

### Key user flows

1. **Pin** — hover (or tab to) a setting row → pin icon appears → click/<kbd>Enter</kbd> → the control
   animates to the Pinned group → live region announces → debounced save.
2. **Unpin** — click the filled pin in the Pinned group → the control returns to its home section,
   which force-opens briefly so the instructor sees where it went → announce → save.
3. **Reorder (pointer)** — grab the drag handle → drop at a new index → announce → save.
4. **Reorder (keyboard)** — tab to the handle → <kbd>Alt</kbd>+<kbd>↑</kbd>/<kbd>↓</kbd> → announce →
   save.
5. **Search with pins** — type a query → Pinned group filters alongside sections.
6. **First run** — zero pins → dismissible hint above the accordions → dismiss or pin once → hint gone
   for good.

### States

- **Loading** — panel renders immediately without the Pinned group; no skeleton (avoids a flash for
  users with no pins). If pins arrive, the group animates in with the repo's standard motion tokens
  and respects `prefers-reduced-motion`.
- **Empty** — no group; hint only on first run.
- **Error (save)** — toast + revert (FR-17). **Error (load)** — silent; `status: 'unavailable'`.
- **Offline** — behaves as save-error; pins revert and the panel stays editable.
- **Cap reached** — remaining pin toggles disabled with an explanatory description.

### Mobile / responsive

The settings sidebar's existing responsive behaviour is preserved. Below the sidebar breakpoint the
pin toggle is always visible (no hover state on touch), and drag reordering uses a long-press
activation constraint; keyboard reorder remains available on external keyboards.

### Accessibility annotations

- Pinned group: `<section aria-labelledby="pinned-settings-heading">`, heading level matching the
  accordions' summary level.
- Toggle: `<button aria-pressed={pinned} aria-label={pinned ? 'Unpin X' : 'Pin X to top'}>`; disabled
  variant adds `aria-describedby` pointing at the cap explanation.
- Drag handle: `<button aria-label="Reorder X" aria-describedby="pinned-reorder-help">` with help text
  naming the <kbd>Alt</kbd>+arrow shortcut.
- Live region: `<div role="status" aria-live="polite" class="sr-only">`, one per panel.
- Focus management: when a control moves between groups, focus moves with the pin toggle of that
  control so the keyboard user is never dumped at the top of the document.

### Copy & i18n keys

`settingsPanel.pinned.title` ("Pinned"), `.hint` ("Pin the settings you use most — they'll show up
here"), `.pinAction` ("Pin {label} to top"), `.unpinAction` ("Unpin {label}"), `.capReached`
("You can pin up to {max} settings. Unpin one to add another."), `.sectionHint` ("{count} pinned to
top"), `.saveFailed` ("Couldn't save your pinned settings"), `.announcePinned`
("{label} pinned, position {index} of {total}"), `.announceUnpinned` ("{label} unpinned"),
`.announceMoved` ("{label} moved to position {index} of {total}"), `.reorderHelp`
("Press Alt plus up or down arrow to reorder").

## 11. AI / ML Considerations

Not applicable. (PS.4 considers *suggested* pins; even there, ranking is frequency-based, not a model.)

## 12. Integration Points

- Internal modules touched: `clients/web/src/components/settings-panel/*` (new + PS.1 files),
  `clients/web/src/components/assignment/assignment-page-settings-panel.tsx`,
  `clients/web/src/components/quiz/quiz-page-settings-panel.tsx`,
  `clients/web/src/lib/pinned-settings-api.ts` (PS.2),
  `clients/web/src/lib/settings-registry.ts` (PS.1),
  `clients/web/src/lib/dnd/keyboardSensorConfig.ts`,
  `clients/web/src/context/platform-features-context.tsx`.
- External services: none. Webhooks/events: none.
- e2e: new spec `e2e/tests/pinned-settings.spec.ts`; the feature flag must be settable from the e2e
  harness the same way other `ff_*` flags are.

## 13. Dependencies & Sequencing

- Must ship after: **PS.1** (addressable controls, relocation wrapper) and **PS.2** (API + flag).
- Must ship before: **PS.4** (needs real pin interactions to measure).
- Shared infra needed: none beyond PS.2.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Instructors lose track of a setting after pinning it ("where did it go?") | M | M | Move-not-duplicate plus the per-section "N pinned to top" hint (FR-7); unpin force-opens the home section |
| Drag-and-drop is inaccessible or fails WCAG 2.5.7 | M | H | Keyboard reorder is a first-class path (FR-13), not a fallback; a11y test gate in AC-5/AC-15/AC-16 |
| Focus is lost when a control moves between groups | M | M | Explicit focus transfer to the moved control's toggle; covered by AC-5 and the focus-order test |
| Layout shift when pins arrive after first paint | M | M | Render unpinned first and animate the group in; measure CLS in the perf check; consider reserving no space rather than a skeleton |
| Optimistic state and refetch-on-focus fight each other (flicker) | M | M | Refetch is ignored while a save is in flight or debounced; server response is the reconciliation point |
| Duplicate DOM ids if a control is ever rendered twice | L | H | Move-not-duplicate is enforced by a test asserting each `settingId` appears once per panel render |
| Pin toggle clutters an already dense sidebar | M | M | Opacity-based reveal on hover/focus, always visible when pinned; dogfood before flag flip |
| Composite editors (Rubric, Assign to) remount when moved between groups and refetch | M | M | Keep composites' subtree identity stable via a portal or a shared React key; verified by a test asserting one fetch per mount |

## 15. Rollout Plan

- **Feature flag** — `ff_pinned_settings` (shared with PS.2), default **false**.
- **Sequencing** — (1) hook + Pinned group behind the off flag, (2) pin toggles in the quiz panel,
  (3) assignment panel, (4) e2e + a11y gates, (5) enable internally, (6) pilot orgs, (7) default on.
- **Dogfood** — Internal instructors for one sprint; collect qualitative feedback on the pin
  affordance's discoverability and on the 8-pin cap.
- **Pilot** — One K-12 and one HE org; success = ≥ 25 % of active authors pin at least one setting in
  the first two weeks and no increase in editor-related support tickets.
- **GA criteria** — Pilot success metric met, zero open a11y defects, `pin_save_failed` rate < 0.5 % of
  attempts, no CLS regression on the editor pages.
- **Rollback path** — Flip the flag off; panels revert to PS.1 behaviour and stored pins are retained
  for a later re-enable.

## 16. Test Plan

- **Unit** — `usePinnedSettings`: optimistic add/remove/reorder; debounce coalescing; revert on
  rejection; prune of unresolved keys; cap enforcement; refetch-on-focus suppressed during an in-flight
  save; `status: 'unavailable'` when the flag is off or the load fails.
- **Integration (component)** — Both panels: pin → group appears + section hint (AC-2); unpin → control
  returns; conditional pinned control hidden but retained (AC-8); search filters the Pinned group
  (AC-14); flag off renders no pin UI and issues no requests (AC-12); each `settingId` renders exactly
  once; save failure toast + revert (AC-9).
- **End-to-end (Playwright)** — `e2e/tests/pinned-settings.spec.ts`: pin in the quiz editor, reload,
  assert order; reorder by keyboard and assert persisted order; open the assignment editor and assert
  surface isolation; second browser context for the same user converges on focus (AC-13); flag-off
  variant asserts the legacy panel.
- **Security** — Confirm no pin request is issued for an unauthenticated session; confirm setting keys
  from the server are never used in `dangerouslySetInnerHTML`, `querySelector`, or URL construction
  (lint rule + review checklist).
- **Accessibility** — `axe` on both panels with 0, 1, and 8 pins, light and dark; keyboard-only script
  (tab to a setting, pin, reorder, unpin) with focus assertions; screen-reader script for NVDA +
  VoiceOver verifying `aria-pressed` and live-region text; reduced-motion check that the group
  transition is suppressed.
- **Performance / load** — React Profiler: pin toggle interaction < 100 ms; Lighthouse CLS on the quiz
  editor page with pins configured; assert ≤ 1 `GET` per mount via network interception.
- **Manual exploratory** — Pin every pinnable setting to hit the cap; pin a conditional setting and
  toggle its condition; pin, then have an admin disable the flag mid-session; slow-3G throttling to see
  the unpinned-first reflow; touch device long-press reorder.

## 17. Documentation & Training

- End-user (help center): "Pin the settings you use most" — how to pin, reorder, unpin; note that pins
  follow the user across devices and apply to every quiz/assignment they open.
- Instructor onboarding: add the pin affordance to the editor tour if one covers the settings sidebar.
- Admin docs: cross-reference `ff_pinned_settings` from PS.2.
- Internal runbook: extend PS.2's kill-switch runbook with the user-visible effect of flipping the flag
  off mid-session (pins disappear from the panel; data retained).
- API reference: no change beyond PS.2.

## 18. Open Questions

1. Is 8 the right UI cap, or should it scale with viewport height? **Proposed:** fixed 8; revisit with
   PS.4 data on how many pins users actually keep.
2. Should unpinning scroll/force-open the home section, or is the section hint enough? **Proposed:**
   force-open briefly; validate in dogfood — it may be more disorienting than helpful.
3. Should the Pinned group be collapsible like other sections? **Proposed:** no — a collapsed pinned
   group defeats the purpose; revisit if dogfood disagrees.
4. Do composite editors (Rubric especially) behave acceptably when relocated, or should they be
   pin-disabled? Decide during implementation using the remount test in §14.
5. Should pinning be offered to students anywhere? No student-facing settings panel exists today; out
   of scope.
6. Does the desktop app (if it wraps the same web client) need any separate handling for the drag
   interaction? Assumed no; verify during pilot.

## 19. References

- Existing files: `clients/web/src/components/quiz/quiz-page-settings-panel.tsx`,
  `clients/web/src/components/assignment/assignment-page-settings-panel.tsx`,
  `clients/web/src/lib/dnd/keyboardSensorConfig.ts`,
  `clients/web/src/context/platform-features-context.tsx`,
  `clients/web/src/pages/lms/course-module-quiz-page.tsx`,
  `clients/web/src/pages/lms/course-module-assignment-page.tsx`, `e2e/tests/`.
- Related plans: [PS.1](PS.1-settings-registry-and-addressable-controls.md),
  [PS.2](PS.2-pinned-settings-data-model-and-api.md),
  [PS.4](../../plan/settings/PS.4-suggested-pins-telemetry-and-rollout.md).
- External standards: WCAG 2.1 AA — 2.1.1, 2.4.3, 2.4.7, 2.5.5, 2.5.7, 4.1.2, 4.1.3; WAI-ARIA
  Authoring Practices (button `aria-pressed`, live regions); RFC 2119.
