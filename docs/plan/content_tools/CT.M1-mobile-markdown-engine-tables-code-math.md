# CT.M1 — Mobile Markdown Engine: Tables, Code, Math & Media

> Implementation plan. Source: mobile parity for the rich content the web editor now emits. Folder overview: [README](README.md). Mirrors the web reader stack (`react-markdown` + `remark-gfm` + `remark-math` + `rehype-katex` + `normalizeMarkdownTables`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M1 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | — |
| **Unblocks** | CT.M2, CT.M3 |

---

## 1. Problem Statement

The web block editor now emits GFM pipe tables (8.13), fenced code, `$…$` / `$$…$$` math, images and
` ```lex-tool ` fences, and the web reader renders all of them. The mobile apps do not: a repo-wide
search for table handling in `clients/ios` and `clients/android` returns **zero** matches, so a table
an instructor authored on the web appears on a phone as a wall of `|` characters. Worse, the apps run
**three divergent renderers** — iOS `CourseMarkdownContentView` (rich-ish), iOS `MarkdownTextView`
(`ItemDetailView.swift:372`, headings/bullets/paragraphs only), and Android `MarkdownText`
(`ItemDetailScreen.kt:412`, which *strips* bold/italic/code/link syntax rather than rendering it) — so
the same paragraph looks different depending on which screen a student opened. CT.M1 replaces all
three with one tested engine per platform that renders what the authoring tools actually produce.

## 2. Goals

- One markdown block model and one renderer per platform, used by every reading surface (CT.M2 does
  the adoption).
- Render **GFM tables** legibly on a phone: header row, horizontal scroll inside the block, no page
  scroll, and a full-screen expand for wide tables.
- Render **fenced code** with language label, monospace, horizontal scroll and copy; render **inline
  code**, bold, italic, strikethrough and links without stripping them.
- Render **math** (`$…$`, `$$…$$`) as typeset math, not raw LaTeX, with a speech-friendly
  accessibility label.
- Parse — and safely *hide* — the ` ```lex-tool ` fence so students never see raw instance JSON, with
  a labelled placeholder that CT.M3 upgrades into the live tool host.
- Match the web's parse semantics exactly, including the blank-line table healing in
  `normalizeMarkdownTables`, verified by a shared golden-fixture corpus.

## 3. Non-Goals

- Rendering or running Content Tools (CT.M3 host, CT.M4 sandbox, CT.M5–CT.M8 renderers). CT.M1 stops
  at "don't show raw JSON".
- Markdown **authoring** on mobile — the notebook editor's own composer is untouched beyond consuming
  the shared block model.
- Adopting the new renderer across screens — that is CT.M2. CT.M1 ships the engine plus one pilot
  surface (content pages) to prove it.
- HTML passthrough. Bodies are markdown; raw HTML is not rendered on mobile (parity with the web
  reader's sanitized pipeline).
- Table *editing* on mobile.

## 4. Personas & User Stories

- **As a student**, I want a comparison table in my reading to look like a table on my phone so I can
  actually use it while commuting.
- **As a student**, I want a code sample to keep its indentation and be copyable, not reflowed into a
  paragraph.
- **As a math student**, I want a fraction to look like a fraction, not `\frac{a}{b}`.
- **As a blind student**, I want VoiceOver/TalkBack to tell me which column and row a cell belongs to.
- **As an instructor**, I want to author once on the web and trust that a phone shows the same thing.
- **As a homeschool parent**, I want the lesson my child reads on a tablet to be the lesson I wrote.

## 5. Functional Requirements

- **FR-1.** The apps MUST expose one parser (iOS `Core/Notebook/NotebookMarkdown.swift`, Android
  `core/notebook/NotebookMarkdown.kt`) whose block model adds `table`, `codeBlock(language)`,
  `math(display:)` and `toolFence(instanceId, toolId, version)` to the existing kinds.
- **FR-2.** The parser MUST recognise GFM pipe tables: a header row, a separator row
  (`| --- | :---: | ---: |`, edge pipes optional), and zero or more body rows, honouring per-column
  left/center/right alignment.
- **FR-3.** The parser MUST heal blank lines inside a table exactly as
  `clients/web/src/components/syllabus/normalize-markdown-tables.ts` does, and MUST fall back to
  emitting the original lines verbatim when the candidate is not a valid table.
- **FR-4.** Table cells MUST render inline markdown (bold, italic, code, links) rather than raw text.
- **FR-5.** A table MUST scroll horizontally **inside its own container**; the page MUST NOT scroll
  horizontally. Column widths derive from content with a minimum touch-legible width, and the header
  row MUST stay visible while scrolling vertically within an expanded table.
- **FR-6.** Tapping a table MUST open a full-screen table viewer (pinch-to-zoom optional, scroll both
  axes) for tables wider than the viewport; the entry point MUST be discoverable and labelled.
- **FR-7.** Fenced code MUST render monospaced, preserve leading whitespace, scroll horizontally, show
  the declared language when present, and offer copy-to-clipboard. Code MUST NOT be word-wrapped by
  default.
- **FR-8.** Inline formatting — `**bold**`, `*italic*`, `~~strike~~`, `` `code` ``, `[text](url)` —
  MUST render as formatting on **both** platforms; the Android `stripInline` behaviour MUST be deleted.
- **FR-9.** Links MUST route through the existing in-app link handling (iOS `ContentLinkRouter`,
  Android equivalent) so course-internal links deep-link rather than bounce to a browser.
- **FR-10.** Math MUST render typeset. `$$…$$` renders as a centred display block; `$…$` renders
  inline within its paragraph. Unparseable LaTeX MUST fall back to monospace source text, never a
  crash or blank.
- **FR-11.** Every math node MUST carry an accessibility label derived from its LaTeX source so screen
  readers announce something meaningful.
- **FR-12.** A ` ```lex-tool ` fence MUST parse into a `toolFence` block and MUST NOT render as a code
  block. Until CT.M3 lands it renders a neutral, labelled placeholder; the raw JSON MUST never be
  shown to a learner.
- **FR-13.** Unknown fence languages MUST render as plain code blocks (safe default), and any parser
  failure MUST degrade to rendering the source text rather than dropping content.
- **FR-14.** Images MUST keep the existing authorized-fetch behaviour (iOS `AuthorizedNotebookImage`),
  honour `alt` text, constrain to the content width and offer full-screen view on tap.
- **FR-15.** Task-list items (`- [ ]` / `- [x]`) and nested lists (up to 3 levels) MUST render with
  correct indentation and checkbox state (read-only in reader contexts).
- **FR-16.** Parsing MUST be pure, synchronous and side-effect free, and MUST live in files covered by
  the existing logic-test conventions (`LexturesTests/*LogicTests.swift`,
  `app/src/test/kotlin/.../core/notebook/*Test.kt`).

## 6. Non-Functional Requirements

- **Performance** — Parse of a 50 KB body ≤ 30 ms on a mid-range device; blocks parsed once and
  memoised per body (iOS: computed once per `markdown` value, not per `body` evaluation — the current
  `CourseMarkdownContentView.blocks` computed property re-parses on every render and MUST be fixed);
  a 20-row × 8-column table renders in ≤ 100 ms; lists virtualise beyond 200 blocks.
- **Security** — No HTML execution, no remote code, no arbitrary URL schemes (only `http`, `https`,
  `mailto`, and in-app deep links); images fetched only through the authenticated image loader.
- **Privacy & Compliance** — Bodies are education records; nothing is logged verbatim to analytics or
  crash reports. Copy-to-clipboard is user-initiated only.
- **Accessibility** — WCAG 2.1 AA. Tables expose row/column header association (iOS
  `accessibilityLabel` per cell of the form "{column header}, {row header}: {value}"; Android
  `semantics` with equivalent text); code blocks are a single readable element with a language
  announcement; Dynamic Type / font-scale honoured up to 200% without clipping; contrast ≥ 4.5:1 in
  both themes; horizontal scroll containers are reachable by switch control and keyboard.
- **Scalability** — Engine is client-local; no server load added.
- **Reliability** — Every block kind has a fallback rendering; a malformed table, fence or math span
  degrades to plain text and never blanks the surrounding page.
- **Observability** — Count parse fallbacks (`markdown_parse_fallback{kind}`) through the existing
  client error logging so we learn which authored constructs mobile still misses.
- **Maintainability** — One parser, one renderer, one block enum per platform. Adding a block kind is
  a case in two switches plus a fixture.
- **Internationalization** — All chrome strings (`copy`, `expand table`, `code`, language names) come
  from `clients/mobile/locales/*.json` under `mobile.markdown.*`; tables and lists lay out correctly
  in RTL (column order mirrors; cell text direction follows content).
- **Backward compatibility** — Additive. Bodies that render today MUST render at least as well after;
  a golden-fixture snapshot suite guards against regressions on existing content.

## 7. Acceptance Criteria

- **AC-1.** *Given* a body containing a GFM table, *When* it renders on iOS and Android, *Then* a
  bordered grid with a distinct header row appears, alignment is honoured, and the page does not
  scroll horizontally.
- **AC-2.** *Given* a table with blank lines between rows, *When* it renders, *Then* it is healed into
  a single table, matching the output of `normalizeMarkdownTables` on the same input.
- **AC-3.** *Given* pipe text that is *not* a valid table (no separator row), *When* it renders,
  *Then* the original lines appear as paragraph text with no content lost.
- **AC-4.** *Given* a table wider than the screen, *When* the student swipes across it, *Then* only
  the table scrolls; *when* they tap it, *Then* a full-screen viewer opens with the header row pinned.
- **AC-5.** *Given* a fenced ```python block, *When* it renders, *Then* indentation is preserved, the
  language is labelled, the block scrolls horizontally, and Copy places the exact source on the
  clipboard.
- **AC-6.** *Given* `**bold**`, `*italic*`, `` `code` `` and `[link](url)` in one paragraph, *When* it
  renders on Android, *Then* all four render as formatting (not stripped, not literal asterisks).
- **AC-7.** *Given* `$$\frac{a}{b}$$`, *When* it renders, *Then* a typeset fraction is displayed and
  the raw LaTeX string is not visible.
- **AC-8.** *Given* invalid LaTeX, *When* it renders, *Then* monospace source text is shown and the
  screen does not crash.
- **AC-9.** *Given* a body containing a ` ```lex-tool ` fence, *When* a student views it, *Then* a
  labelled placeholder appears and `instanceId` / `toolId` JSON is nowhere on screen.
- **AC-10.** *Given* VoiceOver/TalkBack on a table, *When* the student moves cell to cell, *Then* each
  cell announces its column header and value.
- **AC-11.** *Given* the largest Dynamic Type / font-scale setting, *When* a table and code block
  render, *Then* no text is clipped and both remain scrollable.
- **AC-12.** *Given* the shared fixture corpus, *When* the iOS and Android parser tests run, *Then*
  both produce the same block sequence for every fixture.
- **AC-13.** *Given* CI, *When* it runs, *Then* the iOS build, Android compile, and both unit suites
  are green.

## 8. Data Model

**No server schema change, no migration.** CT.M1 is a client-rendering story.

Client block model (Kotlin shown; iOS mirrors it as a Swift enum in `NotebookBlock.Kind`):

```kotlin
sealed interface MdBlock {
  data class Heading(val level: Int, val spans: List<MdSpan>) : MdBlock
  data class Paragraph(val spans: List<MdSpan>) : MdBlock
  data class Bullet(val depth: Int, val spans: List<MdSpan>) : MdBlock
  data class Ordered(val depth: Int, val marker: String, val spans: List<MdSpan>) : MdBlock
  data class TaskItem(val checked: Boolean, val spans: List<MdSpan>) : MdBlock
  data class Quote(val spans: List<MdSpan>) : MdBlock
  data class CodeBlock(val language: String?, val source: String) : MdBlock
  data class MathBlock(val latex: String) : MdBlock
  data class Image(val alt: String, val url: String) : MdBlock
  data class Table(val align: List<MdAlign>, val header: List<List<MdSpan>>,
                   val rows: List<List<List<MdSpan>>>) : MdBlock
  data class ToolFence(val instanceId: String, val toolId: String, val version: Int) : MdBlock
  data object Divider : MdBlock
}

sealed interface MdSpan {                       // inline
  data class Text(val value: String) : MdSpan
  data class Styled(val bold: Boolean, val italic: Boolean, val strike: Boolean,
                    val code: Boolean, val children: List<MdSpan>) : MdSpan
  data class Link(val href: String, val children: List<MdSpan>) : MdSpan
  data class InlineMath(val latex: String) : MdSpan
}

enum class MdAlign { Default, Left, Center, Right }
```

## 9. API Surface

**No new or changed endpoints.** CT.M1 consumes bodies already returned by existing item, assignment,
quiz, syllabus and portfolio responses. No WebSocket, no rate-limit, no OpenAPI change.

## 10. UI / UX

- **New (iOS)** — `Core/Notebook/MarkdownTableLogic.swift` (pure table parse/heal, unit-tested),
  `Features/Courses/MarkdownTableView.swift`, `MarkdownTableFullScreenView.swift`,
  `MarkdownCodeBlockView.swift`; `MathLatexView` replaced with a real typesetting view.
- **New (Android)** — `core/notebook/MarkdownTableLogic.kt`,
  `features/courses/markdown/{MarkdownTable,MarkdownTableFullScreen,MarkdownCodeBlock,MarkdownMath}.kt`.
- **Modified (iOS)** — `Core/Notebook/NotebookMarkdown.swift` (block kinds + table/math/fence parse),
  `Features/Courses/CourseMarkdownContentView.swift` (new cases, memoised blocks),
  `Features/Courses/ItemDetailView.swift` (delete `MarkdownTextView`).
- **Modified (Android)** — `core/notebook/NotebookMarkdown.kt`,
  `features/courses/ItemDetailScreen.kt` (delete `MarkdownText` + `stripInline`, re-export a shim so
  CT.M2 can migrate call sites incrementally).
- **Key flows** — (1) Student opens a content page → engine parses once → blocks render. (2) Student
  meets a wide table → swipes within it → taps → full-screen viewer → dismiss returns focus to the
  table. (3) Student long-presses a code block → Copy → toast confirms.
- **States** — *Loading*: existing skeleton, unchanged. *Empty*: nothing rendered for an empty body.
  *Malformed block*: plain-text fallback with no visible error. *Offline*: images show the cached or
  placeholder state already used by `AuthorizedNotebookImage`; text always renders from the cached body.
- **Accessibility annotations** — table = `accessibilityElement(children: .contain)` with per-cell
  labels; full-screen viewer traps focus and returns it on dismiss; code block exposes a single label
  "{language} code block" plus a Copy action; math exposes its LaTeX-derived label.
- **Copy & i18n** — `mobile.markdown.table.expand`, `.table.close`, `.code.copy`, `.code.copied`,
  `.code.language`, `.tool.placeholder`, `.math.inline`, `.math.display` in
  `clients/mobile/locales/{en,es,fr,ar,en-XA}.json`.

## 11. AI / ML Considerations

Not AI-touching. (Bodies may have been AI-generated upstream; CT.M1 only renders them.)

## 12. Integration Points

- **External** — a native math typesetting dependency per platform (candidates: `iosMath`/SwiftMath on
  iOS, `JLaTeXMath-android` on Android). Both are additive, offline, and must be size-audited against
  the app budget; see Open Question 1.
- **Internal (iOS)** — `Core/Notebook/NotebookMarkdown.swift`, `Core/Routing/ContentLinkRouter.swift`,
  `Core/Accessibility/{ReadAloud,AccessibilitySupport}.swift` (read-aloud must receive a sensible
  plain-text projection of tables/code/math), `Core/I18n`, `Core/Offline` (unchanged), Xcode project
  regeneration for new files.
- **Internal (Android)** — `core/notebook/NotebookMarkdown.kt`, `core/routing`, `core/accessibility`,
  `core/i18n`.
- **Web (reference, unchanged)** — `clients/web/src/components/syllabus/normalize-markdown-tables.ts`
  and its tests are the normative spec for FR-3.
- **Events** — none emitted.

## 13. Dependencies & Sequencing

- Must ship after: nothing (this is the mobile entry point).
- Must ship before: **CT.M2** (adoption across readers) and **CT.M3** (the tool host mounts on the
  `toolFence` block this story introduces).
- Shared infra: none beyond the two client build pipelines.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Native math libraries are stale, heavy, or LGPL-awkward | M | M | Size/licence audit in week 1; fall back to a styled monospace + accessible-label rendering that still beats today's raw LaTeX (FR-10 is satisfiable without a library) |
| iOS and Android parsers drift from each other and from web | H | M | One shared golden-fixture corpus checked into the repo, run by both mobile unit suites and a web test that asserts the same block sequence |
| Wide tables are unusable on a 4" phone regardless of scrolling | M | M | Full-screen viewer (FR-6) plus a card-per-row fallback layout evaluated in usability testing (Open Question 2) |
| Deleting the two legacy renderers regresses a screen nobody tested | M | H | CT.M1 keeps a thin shim; CT.M2 migrates call sites one screen at a time with snapshot tests |
| Re-parsing on every render tanks scroll performance | M | M | Memoise blocks by body hash (explicit NFR); scroll-performance check on a 50 KB body in the manual pass |
| Fixture corpus does not cover author reality | M | M | Seed fixtures from real course bodies in the dev database, including AI-generated tables |

## 15. Rollout Plan

- **Feature flag** — client-side `mobileRichMarkdownEnabled`, default **on** in internal builds, on at
  GA. It gates the *new* renderer path only; flipping it off restores the legacy renderers (which is
  why CT.M1 keeps the shim rather than deleting them outright).
- **Sequencing** — engine + fixtures → tables → code/inline → math → `lex-tool` placeholder → pilot on
  content pages → flip default on.
- **Dogfood** — internal courses that already contain tables and math; a QA course seeded with the
  fixture corpus.
- **GA criteria** — all ACs green, no P1 rendering bug open for 1 week of dogfood, parse-fallback rate
  < 0.5% of rendered bodies.
- **Rollback** — flip `mobileRichMarkdownEnabled` off; no server or data change to revert.

## 16. Test Plan

- **Unit** — parser: table detection, separator/alignment parsing, blank-line healing, non-table pipe
  text, nested lists, task items, fence classification (`lex-tool` vs unknown vs known language),
  inline span tree, math segmentation across `$`/`$$` boundaries and unmatched delimiters.
- **Integration** — golden-fixture corpus: every fixture parsed on iOS, Android and web produces the
  same block sequence; snapshot tests for table, code, math and mixed bodies in light and dark themes.
- **End-to-end (device)** — open a seeded content page containing a table, code block and equation on
  a phone and a tablet, both orientations; expand the table; copy the code.
- **Security** — link scheme allowlist; no HTML execution; image loads stay authenticated.
- **Accessibility** — automated scanner on the pilot screen; scripted VoiceOver and TalkBack passes
  over a table, a code block and an equation; 200% font-scale pass; RTL (`ar`) and pseudo-locale
  (`en-XA`) passes.
- **Performance** — parse benchmark on 10/50/200 KB bodies; scroll FPS on a body with 5 tables.
- **Manual exploratory** — pasted-from-Word tables, AI-generated tables with blank lines, tables with
  empty cells and ragged column counts, code containing pipes, math containing `$` in prose.

## 17. Documentation & Training

- End-user: "Reading course content on mobile" — note table expand and code copy gestures.
- Instructor: a short "what renders on mobile" matrix in the authoring help so authors know tables and
  math are safe to use.
- Internal: `clients/ios/README.md` and `clients/android/README.md` gain a "markdown engine" section
  naming the single renderer and the fixture corpus.
- No API reference change.

## 18. Open Questions

1. Which math typesetting dependency per platform, and does either exceed the app-size budget? (Owner:
   mobile squad, week 1. Fallback is the FR-10 monospace path.)
2. For very wide tables, is a full-screen viewer enough, or do we also need a "card per row" reading
   mode on small phones? (Decide from usability testing before GA.)
3. Should read-aloud (`ReadAloud.swift` / `ReadAloud.kt`) linearise tables cell-by-cell with headers,
   or skip them with a "table, N rows" summary? (Recommendation: header-prefixed linearisation, capped.)
4. Does the fixture corpus live in `clients/mobile/` (shared, new) or duplicate per platform?
   (Recommendation: `clients/mobile/fixtures/markdown/` alongside `locales/`, read by all three suites.)

## 19. References

- Existing mobile: `clients/ios/Lextures/Core/Notebook/NotebookMarkdown.swift`,
  `clients/ios/Lextures/Features/Courses/CourseMarkdownContentView.swift`,
  `clients/ios/Lextures/Features/Courses/ItemDetailView.swift:372` (`MarkdownTextView`),
  `clients/android/app/src/main/kotlin/com/lextures/android/core/notebook/NotebookMarkdown.kt`,
  `clients/android/app/src/main/kotlin/com/lextures/android/features/courses/ItemDetailScreen.kt:412`
  (`MarkdownText`, `stripInline`).
- Web reference: `clients/web/src/components/syllabus/normalize-markdown-tables.ts` (+ tests),
  `clients/web/src/components/syllabus/syllabus-markdown-view.tsx`, `clients/web/package.json`
  (`remark-gfm`, `remark-math`, `rehype-katex`).
- Related plans: [CT.M2](CT.M2-mobile-rich-content-parity-assignments-quizzes.md),
  [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.3 (web runtime)](../../completed/content_tools/CT.3-student-runtime-and-state-persistence.md).
- Standards: WCAG 2.1 AA §1.3.1 (info & relationships — table headers), §1.4.4 (resize text),
  §1.4.10 (reflow — no two-axis page scrolling).
