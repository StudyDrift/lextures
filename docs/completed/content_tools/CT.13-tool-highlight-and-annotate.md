# CT.13 — Tool: Highlight & Annotate (prompted close reading with a class heat map)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.13 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | CT.1, CT.2, CT.3 |
| **Unblocks** | Close-reading workflows in humanities and science-literacy courses |

---

## 1. Problem Statement

Lextures already lets a learner highlight a content page for themselves
(`course.content_page_user_markups`) — private, unprompted, ungraded, and invisible to the teacher. That
is a study aid, not an instructional activity. What instructors actually assign is *directed* annotation:
"highlight every claim the author does not support", "tag each step where energy is conserved". That
task has a prompt, a category set, a target passage, a visible-to-teacher result, and a class-level
pattern worth discussing. None of that is expressible today. This tool makes prompted, categorised,
reviewable annotation a placeable interaction — and gives the instructor a heat map of where the class
looked.

## 2. Goals

- Let an author define a target passage, a prompt, and a small set of tag categories.
- Let a student select text within the target passage, assign a tag, and optionally add a note.
- Persist annotations per enrollment as anchored ranges that survive re-render and modest content edits.
- Show the instructor a **heat map** of the passage plus a filterable list of annotations.
- Keep the shipped personal-highlight feature untouched and clearly distinct.

## 3. Non-Goals

- Replacing or migrating personal page markups (`content_page_user_markups`) — different purpose, stays.
- Collaborative/social annotation (peers seeing each other's annotations) — a peer-visible variant is a
  future option requiring CT.8 moderation; v1 is student→instructor only.
- Annotating PDFs, images or video (the shipped PDF annotation and CT.15/CT.19 cover those surfaces).
- Auto-grading annotation quality (a rubric-based AI variant is noted as future work).

## 4. Personas & User Stories

- **As an English teacher**, I want students to tag evidence, claim and warrant in a paragraph so that
  they practise argument analysis on the text itself.
- **As a science teacher**, I want students to highlight the independent variable in a procedure so that
  I can see who is reading carefully.
- **As an instructor**, I want to see which sentence most of the class marked so that I can start
  discussion where the attention already is.
- **As a student**, I want my annotations to still be there next week so that they are useful for review.
- **As a student using a screen reader**, I want to annotate without a mouse so that the assignment is
  actually assigned to me too.

## 5. Functional Requirements

- **FR-1.** The author MUST define the **target passage**: either the Markdown block(s) immediately
  preceding the tool, an explicit passage authored in the config, or a named section anchor.
- **FR-2.** The author MUST define 1–6 tag categories (label, colour, optional description) and a prompt.
- **FR-3.** A student MUST be able to select text in the target passage and apply a tag; an annotation
  MUST store the tag, the quoted text, an anchor, and an optional note.
- **FR-4.** Anchoring MUST be resilient: store `{quote, prefix, suffix, approxOffset}` (a
  quote-plus-context anchor) rather than raw offsets, and re-resolve on render.
- **FR-5.** When an anchor cannot be resolved (the author edited the passage), the annotation MUST be
  shown as **orphaned** with its quoted text preserved, never silently deleted.
- **FR-6.** A **keyboard-and-screen-reader path** MUST exist: the passage is segmented into addressable
  units (sentences by default, configurable to paragraph or line), navigable with arrow keys, taggable
  with a menu — no pointer selection required. This is a hard requirement, not a fallback.
- **FR-7.** The author MUST be able to set a minimum and maximum number of annotations and require a
  note per annotation.
- **FR-8.** Completion MUST be defined as meeting the configured minimum; the tool reports
  `status='completed'` accordingly.
- **FR-9.** The instructor MUST see a heat map: the passage rendered with per-unit intensity by
  annotation count, filterable by tag, with counts on hover/focus and an accessible table alternative.
- **FR-10.** The instructor MUST see a list of annotations grouped by tag with student attribution, and
  MUST be able to jump from a list item to its position in the passage.
- **FR-11.** Notes MUST pass the CT.8 content filter before storage.
- **FR-12.** The tool MUST support `readOnly` (view annotations, no editing) and CT.4 reset.
- **FR-13.** The author MUST be able to mark "expected" regions (optional) so the instructor view can
  show agreement/miss rates; expected regions are `x-lex-sensitive` and never sent to students.

## 6. Non-Functional Requirements

- **Performance** — Anchor resolution for 50 annotations ≤ 30 ms; heat-map aggregation server-side and
  cached (CT.7); renderer ≤ 32 KB gz.
- **Security** — Annotations are per-enrollment; instructor reads are permission-gated and FERPA-logged
  (CT.4 pattern). Expected regions redacted for students.
- **Privacy & Compliance** — Notes are student work: DSAR export, retention, filter, reset snapshots.
- **Accessibility** — WCAG 2.1 AA with an explicit statement: pointer selection is *one* input path;
  the unit-navigation path is fully equivalent. Tag colour is always paired with a text label and a
  pattern; annotations are announced on creation; the heat map has a table alternative. Known limitation
  documented: precise sub-sentence selection is pointer-only, so authors are guided to sentence-level
  tasks when equivalence matters.
- **Scalability** — Annotation arrays capped (default 50 per learner per instance) inside the state cap.
- **Reliability** — Append/edit merges cleanly across tabs (conflict policy `merge` on annotation id).
- **Observability** — `lextures_content_tool_annotations_total{tool_id="highlight_annotate"}`,
  orphaned-anchor rate (an authoring-quality signal), completion rate.
- **Maintainability** — Anchoring is a standalone, well-tested module reusable by future annotation tools.
- **Internationalization** — Works with RTL text and CJK segmentation (sentence splitting is
  locale-aware; documented limitations for languages without clear sentence delimiters).
- **Backward compatibility** — No change to personal markups.

## 7. Acceptance Criteria

- **AC-1.** *Given* a target passage, *When* a student selects a sentence and applies a tag, *Then* the
  annotation is stored with quote, anchor and tag, and re-renders after reload in the same position.
- **AC-2.** *Given* the author edits the passage so a quote no longer matches, *When* the student
  returns, *Then* the annotation appears in an "orphaned" list with its original quote intact.
- **AC-3.** *Given* keyboard-only navigation, *When* the student moves through units and applies a tag,
  *Then* the annotation is created identically to the pointer path and is announced.
- **AC-4.** *Given* `minAnnotations = 3`, *When* the student has 2, *Then* status is `in_progress`; at 3
  it becomes `completed`.
- **AC-5.** *Given* 25 students annotated, *When* the instructor opens the heat map, *Then* per-unit
  counts match raw state and a table alternative presents the same data.
- **AC-6.** *Given* a note containing filtered content, *When* it is saved, *Then* the configured CT.8
  action applies and the student's text is preserved in the editor on block.
- **AC-7.** *Given* expected regions are configured, *When* a student loads the tool, *Then* they are
  absent from the payload, while the instructor view shows agreement rates.
- **AC-8.** *Given* two tabs annotate simultaneously, *When* both save, *Then* both annotations survive
  (merge by id) with no lost work.
- **AC-9.** *Given* a CT.4 reset, *Then* all annotations are cleared and snapshotted.

## 8. Data Model

**No migration.**

```ts
// configSchema
type HighlightAnnotateConfig = {
  prompt: string
  passageSource: 'preceding_block' | 'inline' | 'section_anchor'
  passageMarkdown?: string                 // when 'inline'
  sectionAnchor?: string
  unitGranularity: 'sentence' | 'paragraph' | 'line'   // default 'sentence'
  tags: Array<{ id: string; label: string; color: string; description?: string }>
  minAnnotations: number                   // default 1
  maxAnnotations: number                   // default 20
  requireNote: boolean                     // default false
  expectedRegions?: Array<{ tagId: string; quote: string }>   // x-lex-sensitive
}

// stateSchema
type HighlightAnnotateState = {
  v: 1
  annotations: Array<{
    id: string
    tagId: string
    quote: string
    anchor: { prefix: string; suffix: string; approxOffset: number; unitIndex?: number }
    note?: string
    createdAt: string
    orphaned?: boolean
  }>
  completedAt?: string
}
```

`scoring.mode = 'none'` (optionally `manual`); `capabilities = ['state','aggregate']`;
`maxStateBytes = 48000`; conflict policy `merge` keyed on annotation id.

## 9. API Surface

**No new routes.**

- `PUT .../state` — annotation add/edit/delete (merge policy).
- `POST .../actions/filter_note` — runs the CT.8 filter server-side before accepting a note.
- Heat map via `GET .../instances/{id}/analytics` (CT.7) with facets `tagId`, `unitIndex`.

## 10. UI / UX

1. Prompt banner with the tag legend (colour chip + label + count-so-far).
2. Target passage rendered with existing annotations underlined/highlighted in tag colour and a small
   tag chip at the end of each annotated unit.
3. Selecting text (or focusing a unit) opens a compact tag menu; choosing a tag creates the annotation;
   an optional note field appears inline when required.
4. Sidebar/below: **My annotations** list, grouped by tag, each with quote, note, edit and delete.
5. Progress line: "3 of 3 required annotations."

**States** — *Empty*: prompt plus a hint for both input paths. *Orphaned*: a collapsible "annotations
that no longer match the text" list. *Read-only*. *Error*: retry, selection preserved.

**Mobile** — long-press selection with a bottom-sheet tag menu; unit-tap mode offered as the primary
path on touch, since precise selection is painful on phones.

**Accessibility** — units are focusable elements with `aria-label` including position and existing
tags; the tag menu is a proper menu; creation/deletion announced politely; colour always paired with a
label and an underline style; heat map has a table alternative.

**Copy & i18n** — `contentTools.tools.highlightAnnotate.*`.

**Authoring** — custom editor: tag builder with colour picker (contrast-checked), passage source
picker with live preview, and optional expected-region marking done by annotating the preview.

## 11. AI / ML Considerations

None in v1. Reserved: AI-assisted review that clusters annotations per unit and drafts a summary of what
the class noticed ("18 students tagged sentence 4 as *claim*; 6 tagged it *evidence*"), and rubric-based
formative feedback on notes. Both would run as CT.6-grounded actions with disclosure, and both are
explicitly out of v1 because the heat map already delivers most of the instructional value.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/highlightannotate/`, `service/contentfilter`,
  new shared `clients/web/src/lib/text-anchoring/` module,
  `clients/web/src/components/content-tools/tools/highlight-annotate/`.
- **Adjacent shipped features** — personal markups (`content_page_user_markups`) remain independent;
  the reader must render both without visual confusion (distinct styling, documented).
- **CT.7** — heat-map facets and completion reporting.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** any future annotation-based tool (shares the anchoring module).
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Anchors break when authors edit content | H | M | Quote+context anchoring, orphan preservation, orphan-rate metric surfaced to authors, warning when editing a passage with annotations |
| Selection UX is inaccessible or unusable on touch | H | H | Unit-navigation path is a hard requirement (FR-6), touch-first unit tapping, documented limitation for sub-sentence tasks |
| Visual clash with personal highlights | M | M | Distinct styling and a legend; user setting to hide personal highlights while a tool is active |
| Tag colours fail contrast | M | M | Contrast-checked palette in the authoring picker; colour never the sole signal |
| Large annotation sets bloat state | L | M | Per-learner cap plus the framework size cap |
| CJK/RTL segmentation errors | M | M | Locale-aware segmentation with tests; paragraph granularity fallback documented |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — anchoring module + tests → manifest → renderer (pointer + keyboard paths) → heat map
  → authoring editor → pilot in a humanities course.
- **Dogfood** — one literature unit and one lab-procedure unit (very different text shapes).
- **GA criteria** — keyboard path verified by a screen-reader user; orphan rate < 5% in dogfood; a11y
  audit passed with limitations documented.
- **Rollback** — remove from the allowlist; annotations preserved.

## 16. Test Plan

- **Unit** — anchoring resolve/re-resolve across edits (insert before, insert inside, delete, reorder);
  segmentation for en/de/ar/zh; completion thresholds; merge conflict resolution.
- **Integration** — save/load/merge; orphan detection; filter on notes; expected-region redaction; reset.
- **End-to-end** — Playwright: pointer path and keyboard path produce identical state; heat map matches;
  orphan flow after an author edit.
- **Security** — payload inspection for expected regions; cross-enrollment writes; XSS in notes and quotes.
- **Accessibility** — axe; full screen-reader script for the keyboard path; contrast checks on every
  built-in tag colour; touch target sizes.
- **Performance** — 50 annotations render/resolve budget; heat map for 300 learners.
- **Manual exploratory** — annotating math-heavy text, tables, lists; RTL passages; very long passages.

## 17. Documentation & Training

- **Instructor** — designing an annotation task; choosing granularity; why editing the passage orphans
  annotations; reading the heat map.
- **Student** — how to annotate with mouse, touch and keyboard.
- **Developer** — the anchoring module contract for reuse.

## 18. Open Questions

1. Should peers eventually see each other's annotations (social annotation)? Proposed: yes as a
   configurable variant after CT.8 moderation is proven — it is the most-requested extension of this
   pattern.
2. Should orphaned annotations be auto-rematched with fuzzy search? Proposed: offer a one-click
   "reattach" suggestion, never automatic.
3. Should sub-sentence selection be disabled entirely when the author needs strict a11y equivalence?
   Proposed: add an authoring switch "sentence-level only" that guarantees equivalence.

## 19. References

- Existing files this work touches: `server/migrations/051_content_page_user_markups.sql` (adjacent
  feature), `clients/web/src/components/syllabus/syllabus-markdown-view.tsx`,
  `server/internal/service/contentfilter/`.
- External standards: W3C Web Annotation Data Model (anchor design informed by TextQuoteSelector);
  WCAG 2.1 AA.
- Related plans: [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md), [CT.22](../../plan/content_tools/CT.22-tool-inline-discussion.md).
