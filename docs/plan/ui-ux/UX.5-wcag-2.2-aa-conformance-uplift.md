# UX.5 — WCAG 2.2 AA Conformance Uplift

> Implementation plan. Source: [audit.md](audit.md) §3 G-5b.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.5 |
| **Section** | UI/UX — Accessibility |
| **Severity** | MAJOR (BLOCKER for EU sales) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — product targets WCAG 2.1 AA; 2.2 criteria unaudited |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Accessibility |
| **Depends on** | UX.2, UX.4 |
| **Unblocks** | VPAT 2.5 → WCAG 2.2 attestation; EAA/EN 301 549 contractual claims |

---

## 1. Problem Statement

Lextures conforms to **WCAG 2.1 AA** (per `docs/vpat/` and the completed 12.x
programme). WCAG 2.2 added nine criteria in October 2023, of which six are A/AA
and therefore in scope for any AA claim (**R-35**). None has been audited. The
audit found concrete exposure on at least three: 177 uses of `h-6`/`h-7`/`h-8`
controls against a 24×24 minimum (**2.5.8**); `@dnd-kit` drag interactions in
module reordering, gradebook, kanban and boards with no single-pointer
alternative (**2.5.7**); and four sticky bars plus a toast stack with no
focus-scroll offsetting (**2.4.11**). Meanwhile the **European Accessibility
Act** deadline passed on 28 June 2025 and enforcement is live, with EN 301 549 as
the presumed-conformance standard (**R-36**) — making this a sales gate, not a
quality improvement.

## 2. Goals

- Achieve and evidence **WCAG 2.2 Level AA** conformance across the web client.
- Close the three criteria with known concrete exposure: target size, dragging
  alternatives, focus not obscured.
- Audit and close the three procedural criteria: consistent help, redundant entry,
  accessible authentication.
- Re-issue the VPAT against WCAG 2.2 / EN 301 549 with reproducible evidence.

## 3. Non-Goals

- WCAG 2.2 **AAA** criteria (2.4.12 Focus Not Obscured Enhanced, 2.4.13 Focus
  Appearance, 3.3.9 Accessible Authentication Enhanced). Documented as
  aspirational only.
- Re-litigating WCAG 2.1 criteria — that is [UX.4](UX.4-aria-widget-and-focus-management-remediation.md)
  and [UX.1](UX.1-semantic-design-token-system.md).
- Native iOS/Android clients (tracked separately in
  `docs/accessibility/mobile-audit-checklist.md`).
- Authored *course content* accessibility — instructors' responsibility, supported
  by the existing course checklist (CC) plan.

## 4. Personas & User Stories

- **As a learner with a motor impairment**, I want to reorder my learning path
  without a drag gesture, so that I can use the feature at all.
- **As an instructor on a touch device**, I want toolbar buttons large enough to
  hit reliably.
- **As a keyboard user**, I want the focused element never hidden behind the
  sticky header when I Tab down a long form.
- **As a learner with a cognitive disability**, I want to sign in without solving
  a puzzle or transcribing a code from memory.
- **As a learner enrolling in a course**, I want not to retype information I gave
  two steps ago.
- **As anyone who needs help**, I want the help control in the same place on every
  page.
- **As a compliance owner**, I want to answer an EU RFP's accessibility section
  with a current, evidenced VPAT.

## 5. Functional Requirements

### 2.5.8 Target Size (Minimum), AA

- **FR-1.** Every pointer target MUST be ≥**24×24 CSS px**, or satisfy a defined
  exception (inline in a sentence, user-agent default, essential, or with ≥24px
  spacing to neighbouring targets).
- **FR-2.** UX.2 components MUST enforce this by construction; the smallest `size`
  variant MUST still satisfy FR-1.
- **FR-3.** An automated check MUST measure rendered target sizes on the top 40
  routes and fail on violations, with an explicit, justified exception list.
- **FR-4.** Touch-primary surfaces SHOULD target **44×44** (the AAA/platform
  convention), not merely 24.

### 2.5.7 Dragging Movements, AA

- **FR-5.** Every drag interaction MUST have a single-pointer alternative. The
  standard pattern is **click-source → click-target**, plus an explicit
  "Move to…" menu item. Inventory: module reordering, module-item reordering,
  gradebook column layout, course catalog kanban, visual collaboration boards,
  syllabus block editor, portfolio editor, live-quiz kit editor.
- **FR-6.** Each drag surface MUST also expose keyboard reordering (`Space` to
  lift, arrows to move, `Space` to drop, `Escape` to cancel) — `@dnd-kit` supports
  this and it MUST be enabled and tested.
- **FR-7.** Reorder operations MUST announce their result via a live region
  ("Module 3 moved to position 1 of 7").

### 2.4.11 Focus Not Obscured (Minimum), AA

- **FR-8.** When any element receives focus it MUST NOT be **entirely** hidden by
  author content. Scroll-into-view MUST offset by the combined height of sticky
  chrome.
- **FR-9.** A `--lx-sticky-offset` token MUST be maintained by the layout shell
  and consumed via `scroll-margin-block-start` on focusable elements.
- **FR-10.** Toasts, the reading focus bar, the quiz focus bar and the Canvas
  import widget MUST NOT overlap the focused element; toasts MUST reposition or
  the focused element MUST scroll clear.

### 3.2.6 Consistent Help, A

- **FR-11.** Help affordances (help widget, contact, feature help, checklist help)
  MUST appear in a **consistent relative order** within the page chrome across all
  pages that offer them.
- **FR-12.** A single canonical help entry point MUST exist in the top bar on
  every authenticated route.

### 3.3.7 Redundant Entry, A

- **FR-13.** Within a multi-step process (signup, onboarding, enrollment,
  checkout, conference booking, accessibility intake), information already
  supplied MUST be auto-populated or selectable, not re-entered — except where
  re-entry is essential (password confirmation, security re-verification).
- **FR-14.** An inventory of all multi-step flows MUST be produced and each
  assessed against FR-13.

### 3.3.8 Accessible Authentication (Minimum), AA

- **FR-15.** No authentication step may require a **cognitive function test**
  (memorisation, transcription, puzzle) without an alternative.
- **FR-16.** All authentication fields MUST support paste and password-manager
  autofill (`autocomplete="username"`, `"current-password"`, `"one-time-code"`).
- **FR-17.** WebAuthn/passkey (already available via `@simplewebauthn/browser`)
  MUST be surfaced as a first-class alternative on the login screen.
- **FR-18.** MFA and magic-link flows MUST NOT block paste and MUST allow the code
  to remain visible for re-reading.

## 6. Non-Functional Requirements

- **Performance** — Target-size increases MUST NOT force layout reflow on dense
  toolbars; spacing exceptions may be used instead of enlarging. Drag alternatives
  MUST NOT add bundle weight beyond 3 KB gzip.
- **Security** — FR-16 (allow paste) is deliberate: blocking paste weakens
  security by discouraging password managers. FR-18 MUST NOT weaken OTP entropy or
  expiry.
- **Privacy & Compliance** — Delivers WCAG 2.2 AA; EN 301 549; EAA; ADA Title
  II/III and Section 508 as covered in `../standards/S20-accessibility-legal-mandates.md`.
- **Accessibility** — This plan *is* the requirement.
- **Scalability** — New drag surfaces MUST inherit the alternative from a shared
  `Reorderable` abstraction, so the criterion cannot regress.
- **Reliability** — Click-to-move MUST be cancellable and MUST not lose state if
  the target is invalid.
- **Observability** — CI emits `target_size_violations`,
  `drag_surfaces_without_alternative`, `focus_obscured_violations`.
- **Maintainability** — One `Reorderable` abstraction wrapping `@dnd-kit`.
- **Internationalization** — Reorder announcements MUST be i18n keys with correct
  pluralisation; arrow semantics MUST invert in RTL.
- **Backward compatibility** — Drag behaviour is preserved; alternatives are
  additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* the top 40 routes, *When* target sizes are measured, *Then*
  every target is ≥24×24 CSS px or appears on the justified exception list.
- **AC-2.** *Given* the module list, *When* a user clicks a module's "Move" action
  and then clicks a target position, *Then* the module moves — with no drag, using
  a single pointer.
- **AC-3.** *Given* the module list, *When* a keyboard user presses `Space`,
  arrows, `Space`, *Then* the module reorders; `Escape` cancels and restores the
  original order.
- **AC-4.** *Given* any reorder, *When* it completes, *Then* a live region
  announces the new position.
- **AC-5.** *Given* every drag surface in the FR-5 inventory, *When* audited,
  *Then* each has a documented single-pointer alternative.
- **AC-6.** *Given* a long form with the sticky top bar present, *When* the user
  Tabs to a field near the top of the viewport, *Then* the field is fully visible.
- **AC-7.** *Given* a toast is showing, *When* focus moves to an element beneath
  it, *Then* the focused element is not entirely obscured.
- **AC-8.** *Given* any authenticated route, *When* inspected, *Then* the help
  control is present and in the same relative position.
- **AC-9.** *Given* the enrollment/onboarding/checkout flows, *When* a user
  reaches a later step, *Then* no previously-supplied value must be retyped.
- **AC-10.** *Given* the login, MFA and magic-link screens, *When* a user pastes a
  password or OTP, *Then* the paste succeeds and autofill attributes are present.
- **AC-11.** *Given* the login screen, *When* a passkey is registered, *Then*
  passkey sign-in is offered as a visible primary alternative.
- **AC-12.** *Given* the completed work, *When* a third-party accessibility audit
  is commissioned, *Then* it certifies WCAG 2.2 AA with no Level A or AA
  non-conformance.

## 8. Data Model

None required by the criteria themselves. FR-13 (Redundant Entry) may require
persisting partial multi-step state; if so it reuses the existing onboarding
progress store rather than adding tables. No new tables, columns, enums, indexes,
migrations or backfill are planned. *(If the enrollment flow proves to need
server-side draft state, that is a follow-up captured in §18 Q3.)*

## 9. API Surface

No new routes. Two behavioural requirements on existing routes:

- Reorder endpoints (modules, module items, gradebook layout, board items) MUST
  accept an **absolute target index**, not only a relative swap, so click-to-move
  and keyboard-move can express "move to position N" atomically.
- Multi-step flows MUST expose already-supplied values on their `GET` step
  endpoints to satisfy FR-13.

```ts
// Existing reorder endpoints gain an explicit index form:
// PATCH /api/v1/courses/{code}/modules/{id}/position
type ReorderRequest = { toIndex: number }   // 0-based, absolute
type ReorderResponse = { fromIndex: number; toIndex: number; total: number }
```

- No WebSocket changes. Existing rate limits apply.
- **OpenAPI** — the `toIndex` form MUST be documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none.
- **Modified pages** — every drag surface (8 inventoried), login/MFA/magic-link,
  all multi-step flows, the app shell (sticky offset), dense toolbars.
- **Key user flows**
  1. Reorder a module by drag (unchanged), by click-to-move, or by keyboard.
  2. Tab down a long settings form with the sticky bar present.
  3. Sign in with a passkey.
  4. Complete enrollment without retyping.
- **States** — click-to-move introduces a "move mode": source selected → valid
  targets highlighted → click target commits / `Escape` cancels. Invalid targets
  are visibly and programmatically disabled.
- **Mobile/responsive** — move mode is the *primary* interaction on touch, where
  drag is least reliable. Targets ≥44px on touch (FR-4).
- **Accessibility annotations** — move mode must announce entry ("Move mode. Use
  arrow keys or select a destination."), valid targets, and the result.
- **Copy & i18n** — new keys under `common.reorder.*` and `auth.passkey.*`,
  at parity in all four locales.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — `@dnd-kit/core` and `@dnd-kit/sortable` (keyboard sensor
  enabled); `@simplewebauthn/browser` (passkey surfacing).
- **Internal**
  - `clients/web/src/components/ui/` — `Reorderable` abstraction (UX.2)
  - `clients/web/src/pages/lms/course-modules.tsx` (primary drag surface)
  - `clients/web/src/pages/lms/course-catalog-kanban.tsx`
  - `clients/web/src/pages/lms/gradebook/**`
  - `clients/web/src/components/boards/**`, `components/syllabus/syllabus-block-editor.tsx`,
    `pages/lms/portfolio-editor-page.tsx`, `pages/lms/live-quiz-kit-editor-page.tsx`
  - `clients/web/src/pages/login.tsx`, `mfa-login.tsx`, `magic-link.tsx`
  - `clients/web/src/components/layout/app-shell.tsx` — sticky offset token
  - `clients/web/src/components/layout/help-widget.tsx` — consistent help
  - `server/internal/httpserver` — `toIndex` reorder form
- **Events** — none.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](UX.2-core-component-library-and-adoption-ratchet.md)
  (target size by construction), [UX.4](UX.4-aria-widget-and-focus-management-remediation.md)
  (focus management is a precondition for 2.4.11).
- **Must ship before** — VPAT re-attestation and any EU RFP response claiming
  2.2 AA.
- **Coordinates with** — [`../standards/S20-accessibility-legal-mandates.md`](../standards/S20-accessibility-legal-mandates.md),
  which owns the legal framing; UX.5 owns the implementation.
- **Shared infra** — none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Target-size fixes bloat dense toolbars and reduce information density | **H** | M | Prefer the *spacing* exception over enlargement in dense contexts; design review per toolbar |
| Click-to-move is discovered by nobody and drag remains the only used path | H | M | Expose "Move to…" in the item's overflow menu (discoverable), not only as a hidden mode; measure usage |
| Eight drag surfaces is a large surface to retrofit | **H** | **H** | Single `Reorderable` abstraction; retrofit once, apply eight times; prioritise modules (highest traffic) first |
| Sticky-offset token drifts as chrome changes | M | M | Offset is computed from the actual rendered chrome height at runtime, not hardcoded |
| Allowing paste is misread as a security regression | M | M | Document the rationale (password managers) in the security review; it is explicitly required by 3.3.8 |
| Third-party audit finds issues outside this plan's scope | M | M | Run the external audit *before* GA, not after; budget a remediation sprint |

## 15. Rollout Plan

- **Feature flag** — none for the criteria fixes. `ffPasskeyPrimary` may gate
  *promoting* passkeys to the primary login affordance, since that is a product
  decision rather than a conformance one.
- **Sequencing**
  1. Audit pass: produce the target-size exception list, the drag inventory, the
     multi-step flow inventory, and the auth review. **Deliverable: a written
     conformance gap report.**
  2. Sticky offset + focus-not-obscured (cheapest, broadest).
  3. `Reorderable` abstraction; modules first, then the other seven surfaces.
  4. Target-size sweep with design review.
  5. Consistent help + redundant entry.
  6. Authentication (autofill, paste, passkey surfacing).
  7. External third-party audit.
  8. VPAT re-issue.
- **Dogfood** — internal org; one week per batch.
- **GA criteria** — AC-1…AC-12 green, including the external audit (AC-12).
- **Rollback** — per-batch revert; requires accessibility-lead sign-off as it
  restores a known non-conformance.

## 16. Test Plan

- **Unit** — `Reorderable`: click-to-move state machine, keyboard sensor, cancel,
  invalid target, announcement strings; sticky-offset computation; autofill
  attribute presence.
- **Integration** — `toIndex` reorder endpoint: bounds, concurrency (two users
  reordering the same list), idempotency; multi-step flow prefill.
- **End-to-end** — Playwright per drag surface: drag path, click-to-move path,
  keyboard path all produce identical server state. Focus-not-obscured checks
  Tabbing through the 5 longest forms. Auth: paste into password/OTP, autofill,
  passkey sign-in.
- **Security** — authz on `toIndex` reorder; verify paste-enable does not weaken
  OTP validation; passkey flow reviewed against the existing auth threat model.
- **Accessibility**
  - *Automated*: target-size measurement harness on the top 40 routes;
    axe regression suite.
  - *Manual*: single-pointer-only session (no keyboard, no drag) completing every
    reorder task; keyboard-only session; screen-reader verification of reorder
    announcements.
  - *External*: commissioned third-party WCAG 2.2 AA audit (AC-12).
- **Performance / load** — reorder INP ≤200 ms p75 on a 50-item module list.
- **Manual exploratory** — QA checklist covering every surface in the FR-5
  inventory on touch, mouse and keyboard.

## 17. Documentation & Training

- **End-user** — help-centre: "Reordering without dragging", "Signing in with a
  passkey".
- **Admin / instructor** — note in the course-design guide that module ordering
  has three input methods.
- **Engineer** — `docs/guides/accessibility-patterns.md` gains the `Reorderable`
  contract and the sticky-offset rule.
- **API reference** — OpenAPI for the `toIndex` reorder form.
- **Runbook** — "Target-size check failed: enlarge or space?".
- **Compliance** — re-issue `docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md` against
  WCAG 2.2 / EN 301 549; update `docs/vpat/VPAT_UPDATE_CHECKLIST.md`;
  cross-reference from `../standards/S20-accessibility-legal-mandates.md`.

## 18. Open Questions

1. Do we claim WCAG 2.2 AA or **2.1 AA + 2.2 AA**? The EAA presumes EN 301 549,
   which currently incorporates 2.1. *Recommendation: claim 2.2 AA — it is a
   superset and future-proofs the next EN revision.*
2. Which third-party auditor? Requires procurement lead time; start early.
3. Does FR-13 (Redundant Entry) require server-side draft state for the enrollment
   flow, or is client-side sufficient? Needs a spike.
4. Is the `44px` touch target (FR-4) a hard requirement or a recommendation for
   the web client, given native apps handle most mobile traffic?
5. Should passkeys become the *default* offered method (a product decision beyond
   conformance)?

## 19. References

- Existing files: `clients/web/src/pages/lms/course-modules.tsx`,
  `clients/web/src/pages/lms/course-catalog-kanban.tsx`,
  `clients/web/src/pages/login.tsx`, `mfa-login.tsx`, `magic-link.tsx`,
  `clients/web/src/components/layout/app-shell.tsx`,
  `clients/web/src/components/layout/help-widget.tsx`,
  `clients/web/src/components/layout/quiz-focus-top-bar.tsx`,
  `clients/web/src/components/layout/reading-focus-top-bar.tsx`,
  `docs/vpat/VPAT_2.5_INT_Lextures_2026-05.md`,
  `docs/accessibility/mobile-audit-checklist.md`
- Research: [research.md](research.md) R-35, R-36, R-37
- Audit: [audit.md](audit.md) G-5b
- External: [WCAG 2.2](https://www.w3.org/TR/WCAG22/),
  [What's new in WCAG 2.2 — TetraLogical](https://tetralogical.com/blog/2023/10/05/whats-new-wcag-2.2/),
  [EAA June 2025 deadline](https://www.insideglobaltech.com/2025/06/10/european-accessibility-act-june-2025-deadline-has-arrived/),
  [EN 301 549](https://www.deque.com/en-301-549-compliance/)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.4](UX.4-aria-widget-and-focus-management-remediation.md),
  [`../standards/S20-accessibility-legal-mandates.md`](../standards/S20-accessibility-legal-mandates.md),
  `../../completed/12-accessibility/`
