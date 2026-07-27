# CT.14 — Tool: Sort & Sequence (drag items into categories or into order)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.14 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | CT.1, CT.2, CT.3 |
| **Unblocks** | The drag-interaction pattern reused by CT.15 |

---

## 1. Problem Statement

Two of the most common learning tasks in every subject — *classify these things* and *put these in
order* — cannot be authored inline. The question bank has `matching` and `ordering` types, but only
inside a formal quiz, and the platform has no way to place a two-minute sorting exercise between two
paragraphs. Sorting is how learners build category schemas (acid vs base, primary vs secondary source,
mitosis phases); sequencing is how they encode procedures and causality. Both are also the classic
accessibility trap: drag-and-drop implemented pointer-only excludes exactly the learners who most need
concrete manipulation. This tool ships both modes with keyboard equivalence as a design constraint.

## 2. Goals

- Ship two modes on one engine: **categorize** (items → buckets) and **order** (items → sequence).
- Grade server-side with configurable strictness, and give per-item feedback on check.
- Make the keyboard path a first-class equivalent, not an afterthought.
- Give instructors an item-level error map: which item is most often misplaced, and where it lands.
- Keep authoring to a minute: paste a list, name the buckets, mark the answers.

## 3. Non-Goals

- Image-based placement (CT.15 owns diagrams/hotspots).
- Free-form spatial arrangement / concept mapping (a separate tool with graph semantics).
- Multi-dimensional sorting (matrix/2-axis) in v1 — noted as a future mode on the same engine.
- Timed or competitive sorting (that is the shipped live-quiz product's territory).

## 4. Personas & User Stories

- **As a biology teacher**, I want students to sort organelles by function so that classification is
  practised rather than read.
- **As a history teacher**, I want students to order events so that chronology becomes a task, not a list.
- **As a student**, I want to see which items I got wrong and try again so that I learn from the attempt.
- **As a student with motor impairments**, I want to place items using only the keyboard so that I can
  do the same activity as everyone else.
- **As an instructor**, I want to know that 18 students put "mitochondrion" in the wrong bucket so that
  I address that specific confusion.

## 5. Functional Requirements

- **FR-1.** The tool MUST support `mode: 'categorize' | 'order'` on a shared item model.
- **FR-2.** In `categorize`, the author defines 2–6 buckets and assigns each item a correct bucket;
  items MAY be allowed in multiple buckets when configured.
- **FR-3.** In `order`, the author defines the correct sequence; the author MAY mark groups of items as
  order-insensitive (ties).
- **FR-4.** Correct placements MUST be `x-lex-sensitive` and MUST NOT reach the client before checking.
- **FR-5.** Checking MUST occur in a server action returning per-item correctness plus the configured
  feedback, never a client-side comparison.
- **FR-6.** The tool MUST support a **keyboard path**: focus an item, press `Enter`/`Space` to pick it
  up, arrow keys to choose a target bucket or position, `Enter` to drop, `Esc` to cancel — with live
  announcements at each step.
- **FR-7.** The tool MUST support touch drag with an equally available tap-to-select-then-tap-to-place
  path on all breakpoints.
- **FR-8.** The author MUST configure attempts (1..5 or unlimited), whether to show correctness per item
  or only a total, and whether correctly-placed items lock on subsequent attempts.
- **FR-9.** State MUST record the placement after every attempt plus per-attempt correctness.
- **FR-10.** Scoring MUST be per item (default: fraction correct) with a configurable all-or-nothing
  mode; the score is reported to the framework for CT.7.
- **FR-11.** Items MUST support Markdown text, inline math and an optional small image (decorative or
  meaningful, with required alt text when meaningful).
- **FR-12.** Item order in the source tray MUST be shuffled per learner by default (deterministically,
  seeded by enrollment, so a reload is stable).
- **FR-13.** The instructor MUST see a confusion view: per item, the distribution of chosen buckets (or
  positions), with the most common error highlighted.
- **FR-14.** The tool MUST honour `prefers-reduced-motion` by disabling drag animations.
- **FR-15.** CT.4 reset MUST return all items to the tray and clear attempts.

## 6. Non-Functional Requirements

- **Performance** — Drag interaction at 60 fps with up to 30 items; check round-trip p95 ≤ 200 ms;
  renderer ≤ 34 KB gz.
- **Security** — Answer key server-side only; attempts enforced server-side.
- **Privacy & Compliance** — Placements are student work (DSAR, retention). No AI, no egress.
- **Accessibility** — WCAG 2.1 AA including 2.1.1 (keyboard), 2.5.7 (dragging movements — a
  single-pointer alternative is mandatory), and 4.1.3 (status messages). The a11y declaration states
  full keyboard equivalence with no known limitations; this is verified per release by a manual script.
- **Scalability** — One state row per learner; confusion aggregates from CT.7 facets.
- **Reliability** — Idempotent check; placements autosaved as drafts before checking so a refresh does
  not lose arrangement work.
- **Observability** — `lextures_content_tool_checks_total{tool_id="sort_sequence",mode,outcome}`,
  attempts histogram, keyboard-path usage rate (validates the investment).
- **Maintainability** — One placement engine, two mode adapters; CT.15 reuses the pick-up/drop model.
- **Internationalization** — RTL-correct drag semantics (order reverses visually, not logically);
  announcements localized with position phrasing that reads naturally in each locale.
- **Backward compatibility** — Additive; independent of quiz `matching`/`ordering` types.

## 7. Acceptance Criteria

- **AC-1.** *Given* an unchecked learner, *When* the payload is inspected, *Then* no correct bucket or
  correct sequence is present.
- **AC-2.** *Given* a learner places all items and checks, *Then* the server returns per-item correctness
  and the state records the attempt with a score.
- **AC-3.** *Given* keyboard-only operation, *When* the learner picks up, moves and drops an item,
  *Then* each step is announced and the resulting state is identical to a pointer drag.
- **AC-4.** *Given* touch input, *When* the learner taps an item then taps a bucket, *Then* the item is
  placed without requiring a drag gesture.
- **AC-5.** *Given* `lockCorrect = true` and attempt 2, *Then* previously-correct items are not
  draggable and this is conveyed to assistive technology.
- **AC-6.** *Given* order mode with a tie group, *When* items within the group are swapped, *Then* both
  arrangements are scored correct.
- **AC-7.** *Given* 30 learners, *When* the instructor opens the confusion view, *Then* per-item
  distributions match raw state and the most common error is highlighted.
- **AC-8.** *Given* `prefers-reduced-motion`, *Then* no drag animation plays and placement still works.
- **AC-9.** *Given* a reload mid-arrangement before checking, *Then* the draft placement is restored.
- **AC-10.** *Given* a CT.4 reset, *Then* all items return to the tray and attempts are cleared.

## 8. Data Model

**No migration.**

```ts
// configSchema
type SortSequenceConfig = {
  mode: 'categorize' | 'order'
  prompt: string
  items: Array<{ id: string; text: string; imageUrl?: string; imageAlt?: string }>
  buckets?: Array<{ id: string; label: string; description?: string }>     // categorize
  correctBucketByItem?: Record<string, string | string[]>                  // x-lex-sensitive
  correctOrder?: string[]                                                  // x-lex-sensitive (order)
  tieGroups?: string[][]                                                   // x-lex-sensitive
  itemFeedback?: Record<string, string>                                    // x-lex-sensitive
  attempts: number | 'unlimited'          // default 3
  showPerItemCorrectness: boolean         // default true
  lockCorrect: boolean                    // default true
  scoreMode: 'per_item' | 'all_or_nothing'   // default 'per_item'
  shuffleItems: boolean                   // default true
}

// stateSchema
type SortSequenceState = {
  v: 1
  placement: Record<string, string | null> | string[]   // itemId→bucketId, or ordered itemIds
  attempts: Array<{ at: string; correctItemIds: string[]; scorePct: number }>
  lockedItemIds: string[]
  completedAt?: string
}
```

`scoring.mode = 'auto'`; `capabilities = ['state','scoring']`; `maxStateBytes = 24000`;
conflict policy `server_wins`.

## 9. API Surface

**No new routes.**

- `PUT .../state` — draft placement (pre-check).
- `POST .../actions/check` — `{placement, idempotencyKey}` → `{perItem, scorePct, attemptsRemaining, state}`.
- `POST .../actions/reset-attempt` — returns unlocked items to the tray within the same attempt budget
  when the author allows it.
- Confusion view via CT.7 facets `itemId`, `placedIn`, `correct`.

## 10. UI / UX

**Categorize** — a source tray of item chips above, bucket columns below, each showing its label,
count and drop affordance. **Order** — a single ordered list with drag handles and up/down controls
always visible (not hover-revealed).

**Check** button appears once every item is placed (or immediately, if partial checking is allowed).
After check: per-item ✓/✗ with feedback popovers, a score line, and **Try again** while attempts remain.

**States** — *Unplaced*, *Arranging (draft saved)*, *Checked (with results)*, *Exhausted (review)*,
*Read-only*, *Error (retry, arrangement preserved)*.

**Mobile** — tap-to-place is the default interaction; drag still works; buckets stack vertically with
sticky headers.

**Accessibility** — items are buttons with `aria-roledescription="sortable item"`; the pick-up/drop
model uses `aria-grabbed`-equivalent semantics via live announcements ("Mitochondrion, picked up.
Bucket 2 of 3, Organelles, 4 items. Press Enter to drop."); buckets are labelled regions with counts;
`aria-live="polite"` for all movement; correctness conveyed by icon + text; visible focus at all times.

**Copy & i18n** — `contentTools.tools.sortSequence.*`; announcement templates are ICU messages so
position phrasing is translatable.

**Authoring** — custom editor: paste-a-list bulk import ("one item per line"), bucket builder,
assign-correct-bucket via dropdown per item (or drag in a preview), tie-group marking in order mode.

## 11. AI / ML Considerations

None in v1. Reserved: generate items and buckets from the section text via the CT.2 `ui.aiAssist` hook
(reusing quiz-generation prompts), and a distractor-quality check that flags items whose correct bucket
is ambiguous. Both deferred; the authoring bulk-import already removes most of the friction.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/sortsequence/` (grading), shared
  `clients/web/src/components/content-tools/shared/placement-engine/` (reused by CT.15),
  `clients/web/src/components/content-tools/tools/sort-sequence/`.
- **CT.7** — per-item facets and score reporting.
- **Outcomes** — optional outcome alignment per instance for mastery evidence.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** CT.15 (shares the placement engine).
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Drag-and-drop ships inaccessible | H | H | Keyboard path is a hard FR, verified by manual script per release; CT.8 gate blocks release without it |
| Touch drag is fiddly on small screens | H | M | Tap-to-place is the default on touch; large targets; sticky bucket headers |
| Ambiguous items make grading feel unfair | M | M | Multi-bucket allowance, tie groups, per-item feedback, instructor confusion view surfaces ambiguity |
| Draft placement lost on refresh | M | M | Draft autosave before check (CT.3 autosave) |
| Long item text breaks layout | M | L | Chip wrapping, max height with internal scroll, authoring guidance |
| RTL ordering confusion | M | M | Logical order preserved; visual mirroring tested in ar locale |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — placement engine + a11y model → grading functions → renderer (pointer, touch,
  keyboard) → authoring editor → confusion view → pilot.
- **Dogfood** — a biology classification unit and a history chronology unit.
- **GA criteria** — keyboard script validated with a screen-reader user; 60 fps with 30 items;
  a11y audit passed with no known limitations.
- **Rollback** — remove from the allowlist.

## 16. Test Plan

- **Unit** — grading in both modes (exact, partial, tie groups, multi-bucket); shuffle determinism per
  enrollment; attempt/lock accounting; score modes.
- **Integration** — check idempotency; answer-key redaction; draft persistence; reset.
- **End-to-end** — Playwright: pointer, touch and keyboard paths each complete the same activity and
  produce identical state; retry with locked items; instructor confusion view.
- **Security** — payload inspection; tampered item/bucket ids; over-large placements.
- **Accessibility** — axe; full screen-reader script for pick-up/move/drop; target sizes; reduced-motion;
  focus management after check.
- **Performance** — 30-item drag frame rate; check latency; chunk budget.
- **Manual exploratory** — RTL, long text, images with alt text, 6 buckets on a phone.

## 17. Documentation & Training

- **Instructor** — designing unambiguous categories; when to use ties; reading the confusion view.
- **Student** — the keyboard shortcuts (surfaced in-tool via a help affordance).
- **Developer** — the placement engine contract and its a11y model, for reuse in CT.15 and beyond.

## 18. Open Questions

1. Should a 2-axis matrix mode ship as a third mode on the same engine? Proposed: after v1 usage data —
   the engine is designed for it.
2. Should partial checking (check what is placed so far) be allowed by default? Proposed: no; it
   encourages guess-and-check over thinking.
3. Should items be reusable across instances (an item bank)? Proposed: not in v1; bulk paste covers it.

## 19. References

- Existing files this work touches: `clients/web/src/components/content-tools/`,
  `server/internal/service/contenttools/`.
- Precedents: quiz `matching` / `ordering` question types (`server/migrations/075_question_bank.sql`).
- External standards: WCAG 2.1 AA — 2.1.1 Keyboard, 2.5.7 Dragging Movements, 4.1.3 Status Messages;
  WAI-ARIA Authoring Practices — drag-and-drop alternatives.
- Related plans: [CT.15](CT.15-tool-labeled-diagram-and-hotspot.md),
  [CT.11](CT.11-tool-inline-questions.md), [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md).
