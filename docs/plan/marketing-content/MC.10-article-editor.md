# MC.10 — Article Editor: Authoring, Metadata, Preview & Revisions

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.10 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal staff surface) |
| **Status (today)** | MISSING — articles are written in a code editor and reviewed as a pull request |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Web platform + Docs/Content |
| **Depends on** | MC.4, MC.5, MC.9 |
| **Unblocks** | MC.11, MC.14, MC.15 |

---

## 1. Problem Statement

This is the screen the whole program exists to deliver: the place a content expert writes. It has to
be good enough that someone who has never opened a terminal can produce an article that satisfies the
answer-first content contract — key takeaways, a 40–60 word answer, question headings, self-contained
passages, cited claims, an FAQ block — and see exactly how it will look before it goes live. Today
that workflow requires markdown files, front matter, a local build and a pull request. If the editor
is mediocre, the content team goes back to asking engineers, and every other plan in this program
becomes shelfware.

## 2. Goals

- Provide a focused writing surface for markdown + content directives with formatting affordances
  that do not require knowing the syntax.
- Make the content contract visible and actionable while writing: live score, inline findings, and
  structure hints — never a wall of red after the fact.
- Show a true preview: the same rendering the public site will produce, in the site's content styles.
- Make metadata (the old front matter) a structured, validated form rather than free text.
- Keep work safe: autosave, revision history, diffs, restore, and a clear conflict experience.
- Make publishing obvious and reversible from the same screen.

## 3. Non-Goals

- No WYSIWYG/rich-text editing. Markdown remains the stored format (MC.4 decision) and the editor is
  a markdown editor with helpers — a rich-text layer would introduce a lossy conversion.
- No real-time multi-user collaborative editing (no CRDT/presence). Conflicts are handled optimistically
  (MC.2 FR-4). Deliberate: the collaboration stack exists for boards, but content editing is
  low-concurrency and the complexity is not justified.
- No AI drafting, rewriting or alt-text generation in this plan.
- No translation UI (MC.14) and no editorial calendar (MC.11) — the editor links to both.
- No custom page-builder / block layout system. Articles are documents.

## 4. Personas & User Stories

- **As a content expert**, I want to write and format without memorising directive syntax, so I can
  focus on the content.
- **As a content expert**, I want to know *before* I ask for review whether my article meets the
  contract, so review is about substance.
- **As a content expert**, I want to insert a screenshot with alt text in two clicks.
- **As a content expert**, I want autosave, so a closed laptop does not cost me an hour.
- **As an editor**, I want to see what changed between revisions, so review is a diff, not a re-read.
- **As a publisher**, I want to preview the exact public page, then publish or schedule from the same
  screen.
- **As a screen-reader user on the content team**, I want the editor to be operable and to announce
  findings, so authoring is not gated on sight.

## 5. Functional Requirements

- **FR-1.** The editor MUST be a two-pane layout (source ↔ preview) with a third collapsible metadata
  panel, and MUST support source-only and preview-only modes.
- **FR-2.** The source pane MUST be a markdown editor with syntax highlighting, line numbers, soft
  wrap, and gutter markers for findings; it MUST work with plain keyboard input and MUST NOT trap
  Tab (Tab moves focus; indentation uses an explicit control or Ctrl+] per the editor's documented
  keymap).
- **FR-3.** A toolbar MUST provide: heading levels, bold/italic/code, link, list, table, quote, plus
  **Insert block** for each content directive (`key-takeaways`, `answer`, `definition`,
  `comparison-table`, `steps`, `faq`, `callout`, `stat`, `sources`) inserting a valid skeleton with
  guidance comments.
- **FR-4.** The preview pane MUST render via the MC.4 sanitized renderer, styled with the site's
  content CSS, and MUST update within 300 ms of typing pause (debounced), preserving scroll position
  and offering scroll sync.
- **FR-5.** Validation MUST run on the same debounce, rendering findings as (a) gutter markers at the
  reported line, (b) a findings list grouped by severity, and (c) a score meter with the publish
  floor marked.
- **FR-6.** The metadata panel MUST expose, with validation and inline help: kind (immutable after
  create), slug (with live URL preview and a warning that changing a published slug creates a
  redirect), category (docs only), title, description (with character counter and the 160-char SEO
  limit), primary question, author, reviewer, cluster/pillar, keywords, related-to (path picker with
  autocomplete over known paths), roles, segments, verified-against, citations, hero image, review
  due date, `noindex`, canonical override (advanced, collapsed).
- **FR-7.** Autosave MUST persist to the server every 30 s when dirty and on blur, updating
  `expectedRevisionNo`, with a visible "Saved 12:04" indicator and an explicit **Save** action.
- **FR-8.** On a `409` conflict the editor MUST show a blocking dialog naming who saved, when, and
  offering **View their version**, **Keep mine (copy to clipboard)** and **Reload** — never a silent
  overwrite, never a lost buffer.
- **FR-9.** A revision drawer MUST list revisions (number, actor, time, status, change note), allow
  viewing any revision, show a side-by-side or unified **diff** against the current draft, and allow
  **Restore** (which creates a new revision per MC.2 FR-10).
- **FR-10.** Image insertion MUST open the MC.5 media dialog (upload or library) and insert the
  reference at the cursor with alt text already required at upload.
- **FR-11.** Link insertion MUST offer internal-path autocomplete from `known-paths` so
  `link.internal-resolves` findings are rare by construction.
- **FR-12.** The header MUST show status, live status (MC.8) and actions appropriate to the viewer's
  permission: Save, Preview, Submit for review, Approve / Request changes, Publish, Schedule,
  Unpublish, Archive.
- **FR-13.** **Preview** MUST open the article as it will appear publicly, using an MC.2 preview
  token; for `WWW_CONTENT_SOURCE=api` deployments this opens `lextures.com/<path>?preview_token=…`,
  otherwise it opens an in-app preview route rendering the same HTML in site styles.
- **FR-14.** Publish and Schedule MUST surface blocking findings in the confirmation dialog and MUST
  disable the action until they are resolved, with an **Override** path for `…:publish` holders that
  requires a typed justification (recorded — MC.11 FR-9).
- **FR-15.** The editor MUST warn on navigation away with unsaved changes (`beforeunload` + router
  guard).
- **FR-16.** Keyboard shortcuts MUST include Save (Ctrl/Cmd+S), Preview toggle, Bold/Italic/Link, and
  MUST be discoverable through a shortcuts dialog; they MUST NOT conflict with screen-reader or
  browser shortcuts.
- **FR-17.** Drafts MUST be creatable in one step from MC.9's New article action, with slug
  auto-derived from the title and editable before first save.
- **FR-18.** The editor MUST function at 320 px width (single pane with a mode switcher) and at
  400% zoom without horizontal scrolling of the page (WCAG 1.4.10).

## 6. Non-Functional Requirements

- **Performance** — Typing latency < 16 ms per keystroke at 40 KB documents; preview+validate
  round trip < 300 ms local (validation runs client-side against a WASM/JS port? **No** — see §18;
  v1 calls `POST /lint` debounced at 800 ms and renders preview client-side with the JS renderer for
  instant feedback). Editor chunk lazy-loaded, < 250 KB gzip incremental.
- **Security** — Preview HTML is produced by the sanitizing renderer; the preview pane renders it in
  the same document (not an iframe) only because the sanitizer is the same one the public site
  trusts — if that assumption is ever weakened, the pane must move to a sandboxed iframe. Preview
  tokens are never logged and expire in 30 minutes. All actions are server-authorized.
- **Privacy & Compliance** — Autosave stores drafts server-side; drafts are covered by the same audit
  trail. Screenshot guidance (no real learner data) is shown in the media dialog.
- **Accessibility** — WCAG 2.1 AA is a hard requirement for an authoring tool used daily:
  the editor exposes a labelled `textarea`-equivalent with `aria-describedby` pointing at the
  findings summary; findings update announces politely; the toolbar is a `role="toolbar"` with arrow
  navigation; diff view provides a text summary ("12 lines added, 3 removed") not just colour;
  contrast of all editor chrome meets 4.5:1; no keyboard trap (FR-2); focus is visible everywhere;
  the two-pane split is resizable by keyboard.
- **Scalability** — Handles documents up to the 1 MB cap; findings list virtualized above 200 items.
- **Reliability** — Autosave failures surface immediately and retry with backoff; the local buffer is
  never discarded on failure; a crash recovery banner offers the last locally cached buffer
  (sessionStorage) when the server draft is older.
- **Observability** — `marketing_content.editor_opened`, `.autosave_result`, `.conflict_shown`,
  `.publish_blocked{rule}`, `.override_used`; no content bodies in analytics.
- **Maintainability** — Editor composed from `components/marketing-content/editor/*` with the
  markdown editing library isolated behind one adapter component so it can be replaced;
  design tokens only; file budgets enforced.
- **Internationalization** — UI strings via i18n keys; editor supports RTL body content (the source
  pane sets `dir="auto"` on the preview, keeps LTR for markdown source); date/time in viewer locale.
- **Backward compatibility** — N/A (new surface). Articles remain plain markdown, so a future editor
  change cannot strand content.

## 7. Acceptance Criteria

- **AC-1.** *Given* a new blog draft, *when* the author types a title, *then* the slug is derived,
  the URL preview shows `/blog/<slug>`, and the first save creates the article.
- **AC-2.** *Given* typing in the source pane, *when* 300 ms pass, *then* the preview updates and
  matches the published rendering for the same markdown (asserted against the MC.4 golden corpus).
- **AC-3.** *Given* an article missing the answer block, *when* validation runs, *then* a gutter
  marker and a findings entry appear, the score meter shows below-floor, and Publish is disabled with
  an explanatory tooltip and accessible text.
- **AC-4.** *Given* a `…:publish` holder and a blocking finding, *when* they choose Override, *then*
  a justification of ≥ 20 characters is required, the publish proceeds, and the justification is
  recorded on the publish event.
- **AC-5.** *Given* an open editor and a concurrent save by another user, *when* autosave fires,
  *then* the conflict dialog appears with the other editor's name and time, and the local buffer is
  preserved.
- **AC-6.** *Given* 7 revisions, *when* the revision drawer opens, *then* all are listed and a diff of
  any two renders with an accessible added/removed summary.
- **AC-7.** *Given* Restore on revision 2, *when* confirmed, *then* the editor content becomes
  revision 2's body, a new revision is created, and the status is unchanged.
- **AC-8.** *Given* Insert → Key takeaways, *when* chosen, *then* a valid `:::key-takeaways` block is
  inserted at the cursor with placeholder bullets and the cursor lands inside it.
- **AC-9.** *Given* the media dialog, *when* an image is uploaded with alt text and inserted, *then*
  the preview shows it and the published page renders `<picture>` with dimensions (verified in MC.7's
  build test).
- **AC-10.** *Given* unsaved changes, *when* the user navigates away, *then* a confirmation appears
  and cancelling keeps the buffer.
- **AC-11.** *Given* keyboard-only use, *when* the editor is operated, *then* every action is
  reachable, Tab never traps in the source pane, and axe reports zero serious/critical violations.
- **AC-12.** *Given* a 320 px viewport, *when* the editor loads, *then* it renders a single pane with
  a mode switcher and no horizontal page scrolling; at 400% zoom the same holds.
- **AC-13.** *Given* Preview, *when* clicked on a draft, *then* a tokenised preview opens showing the
  draft, and the same URL without the token returns 404.
- **AC-14.** *Given* an autosave failure, *when* it occurs, *then* an inline error with retry appears,
  the indicator shows "Unsaved changes", and no content is lost.

## 8. Data Model

No new tables. Client-side state:

```ts
type EditorState = {
  article: Article                 // MC.2 DTO
  buffer: { bodyMd: string; metadata: ArticleMetadata }
  dirty: boolean
  lastSavedAt: string | null
  expectedRevisionNo: number
  findings: Finding[]; score: number | null; validating: boolean
  conflict: { currentRevisionNo: number; updatedBy: string; updatedAt: string } | null
  previewHtml: string
}
```

Local crash-recovery buffer in `sessionStorage` keyed `mc:draft:{articleId}`, cleared on successful
save; never contains anything not already sent to the server.

## 9. API Surface

Consumes MC.2, MC.4, MC.5, MC.8:

```
POST   /api/v1/admin/marketing/articles                       create
GET    /api/v1/admin/marketing/articles/{id}                  load (incl. bodyMd, qualityReport)
PATCH  /api/v1/admin/marketing/articles/{id}                  autosave / save
POST   /api/v1/admin/marketing/articles/{id}/transition       submit/approve/publish/schedule/…
GET    /api/v1/admin/marketing/articles/{id}/revisions        drawer
GET    /api/v1/admin/marketing/articles/{id}/revisions/{no}   diff source
POST   /api/v1/admin/marketing/articles/{id}/revisions/{no}/restore
POST   /api/v1/admin/marketing/articles/{id}/preview-token    preview
POST   /api/v1/admin/marketing/lint                           debounced validation
POST   /api/v1/admin/marketing/media                          image upload
GET    /api/v1/admin/marketing/media                          library
GET    /api/v1/admin/marketing/known-paths                    link autocomplete (read side)
```

No new endpoints are introduced by this plan except the `known-paths` **read** route
(`GET`, permission `…:view`), added here for autocomplete.

## 10. UI / UX

**Layout**

```
┌ ← Marketing Content   Finding your course        ○ Draft · Saved 12:04  [Preview] [Submit ▾] ┐
├───────────────────────────────┬──────────────────────────┬───────────────────────────────────┤
│ 1  ---                        │  Finding your course     │ METADATA                          │
│ 2  :::key-takeaways           │                          │ Slug  finding-your-course         │
│ 3  - Courses appear …         │  Key takeaways           │ URL   /docs/courses/finding-…     │
│ ⚠4  :::answer                 │  • Courses appear …      │ Category  Courses & content ▾     │
│ 5  Your courses are …         │                          │ Description  ▓▓▓▓▓░ 142/160       │
│                               │  Direct answer           │ Author  Chase Willden ▾           │
│                               │  Your courses are …      │ Review due  2026-11-01            │
├───────────────────────────────┴──────────────────────────┤ Hero image  [Choose…]             │
│ Quality 7.4 / 8.0 floor  ▓▓▓▓▓▓▓░░                       │ Keywords  [courses] [navigation]  │
│ ⚠ struct.faq-count — add at least 3 FAQ entries (line 4) │ ▸ Advanced                        │
└──────────────────────────────────────────────────────────┴───────────────────────────────────┘
```

**Key flows**

1. Create → title → slug derived → write → autosave → findings resolve → Submit for review.
2. Review → editor opens with the reviewer action set → diff against last published → Approve or
   Request changes with a note.
3. Publish → confirmation with findings summary and target URL → Publish now / Schedule → status
   becomes `Publishing…` (MC.8) → `Live`.
4. Fix a live page → edit → Save → Publish → build dispatched.
5. Recover → conflict dialog → view theirs → reload or copy mine.

**States** — loading skeleton; not-found; no-permission (read-only editor with actions absent);
saving/saved/unsaved-error; validating; preview error (renderer failure shows source with a warning);
offline (autosave paused, banner, buffer retained).

**Responsive** — `< md`: single pane with a segmented control (Write · Preview · Details); toolbar
collapses to an overflow menu; findings become a bottom sheet.

**Accessibility annotations** — editor labelled "Article body, markdown"; findings summary is
`aria-live="polite"` and referenced by `aria-describedby`; toolbar `role="toolbar"` with arrow-key
roving tabindex; diff has a visually hidden summary and per-line markers with text prefixes (+/−);
split handles are keyboard-resizable with `aria-valuenow`; all icon-only buttons have accessible
names; error and status text never rely on colour.

**Copy & i18n keys** — `marketingContent.editor.*` (`save`, `saved`, `unsaved`, `conflict.title`,
`conflict.body`, `publish.blocked`, `publish.override.prompt`, `insert.keyTakeaways`,
`preview.open`, `revisions.restore`, `metadata.description.counter`, `slug.changeWarning`).

## 11. AI / ML Considerations

Out of scope by decision. The editor is designed so that a later assistive feature (draft outline,
alt-text suggestion, FAQ suggestions) can be added as a *panel* that inserts text the author accepts,
without changing the storage format or the validation gate. Any such feature must:
route through `internal/service/aigateway`, be visibly labelled, be off by default, be disclosed via
`internal/aidisclosure`, and never satisfy a content-contract rule on the author's behalf.

## 12. Integration Points

- **Client modules:** `pages/admin/marketing-content/editor.tsx`,
  `components/marketing-content/editor/*` (source pane adapter, toolbar, findings, metadata panel,
  revision drawer, media dialog integration, publish dialog), `lib/marketing-content-api.ts`,
  `components/ui/*`, `components/editor/*` (reuse where it fits — the block editor is a different
  model and is **not** reused for article bodies).
- **Shared rendering:** the preview uses the same renderer module `www` uses. To avoid duplicating
  it, the markdown renderer is extracted into a small shared package (`clients/packages/content-render`,
  source of truth `www/src/lib/markdown.ts`) imported by both apps — the same pattern as
  `@lextures/tool-sdk`.
- **Server:** MC.2/MC.4/MC.5/MC.8 endpoints.
- **External libraries:** a markdown source editor (CodeMirror 6 proposed) plus a diff library; both
  isolated behind adapters.

## 13. Dependencies & Sequencing

- Must ship after: MC.9 (shell + routing), MC.4 (validation + renderer), MC.5 (media), MC.2 (write
  API).
- Must ship before: MC.11 (workflow UI builds on the editor's action set), MC.14, MC.15.
- Shared infra: none beyond the above.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Markdown is too technical for non-technical authors | **H** | **H** | Toolbar + insert-block menus mean the syntax is never required; live preview; templates per kind (MC.11); training. Reassess after two weeks of real use — if authors still struggle, evaluate a rich-text layer as a follow-on, accepting the conversion risk knowingly |
| Preview drifts from the published page | M | H | Shared renderer package (§12) + MC.4 golden corpus; a preview banner appears if the renderer version differs from the last build |
| CodeMirror accessibility gaps | M | H | Explicit AC-11; screen-reader testing in the definition of done; fall back to a plain `textarea` mode toggle ("simple editor") that is always available |
| Autosave conflicts frustrate paired authors | M | M | 30 s autosave keeps tokens fresh; conflict dialog preserves work; document the "one author at a time" norm |
| Editor bundle bloat | M | M | Lazy route, dynamic import of the editor library, bundle budget in CI |
| Scope creep into a page builder | M | M | Explicit non-goal; directives cover layout needs |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content`. Additionally the editor ships behind
  `marketing_content_editor_enabled` (platform setting, default on when the flag is on) so it can be
  disabled without hiding the read-only workspace if a severe bug appears.
- **Sequencing:** shared renderer package → editor shell + source pane + preview → validation panel →
  metadata panel → media + link pickers → revisions/diff → publish dialogs → shortcuts + polish.
- **Dogfood:** the content team writes three real help articles in staging before production enable;
  every friction point is logged and triaged before GA.
- **GA criteria:** all ACs; axe clean; three articles authored end-to-end by a non-engineer without
  assistance; p95 typing latency verified.
- **Rollback:** disable the editor setting (workspace stays read-only) or flag off entirely; content
  in the database is unaffected.

## 16. Test Plan

- **Unit (vitest)** — slug derivation; directive insertion at cursor; debounce logic; dirty tracking;
  conflict state machine; findings→gutter mapping; metadata validation (description length, date
  formats, path autocomplete filtering); crash-recovery buffer lifecycle.
- **Integration** — editor against a mocked API: create → autosave → conflict → resolve; validation
  round trip; revision list/diff/restore; publish blocked → override → published.
- **End-to-end (Playwright)** — `e2e/tests/marketing-content-editor.spec.ts`: author a full article
  including an image and every directive, submit, approve, publish; verify preview token page;
  verify the article appears in the MC.7 build test fixture.
- **Security** — preview token handling (not logged, expires); sanitizer applied to preview HTML
  (inject a script through the body and assert it does not execute in the editor); authz for each
  header action; override justification required.
- **Accessibility** — axe on the editor route (both themes, both pane modes); full keyboard script;
  screen-reader script covering writing, findings, and publishing; 400% zoom and 320 px checks;
  reduced-motion respected.
- **Performance** — typing latency benchmark at 10/40/200 KB; preview render timing; bundle size gate.
- **Manual exploratory** — QA checklist: paste from Word/Google Docs, paste an image (should route to
  upload), very long lines, tables, RTL content, network loss mid-autosave, two tabs open on the same
  article.

## 17. Documentation & Training

- Help article: "Writing a help article in Lextures" — the content contract explained for authors,
  with the directive palette.
- Help article: "Publishing and scheduling marketing content".
- Internal: shortcuts reference; the "simple editor" fallback; conflict resolution guidance.
- A recorded 20-minute walkthrough for the content team, plus a live session at GA.

## 18. Open Questions

1. Client-side validation (port the rules to TS) vs server `POST /lint`? (Proposed: server for v1 —
   one implementation, already required by the publish gate; if latency is annoying, port the cheap
   structural rules client-side and keep the server authoritative.)
2. CodeMirror 6 vs a plain textarea with a formatting toolbar? (Proposed: CodeMirror with a
   guaranteed plain-textarea fallback mode; decide after an a11y spike.)
3. Do we extract `markdown.ts` into `clients/packages/content-render` or duplicate it? (Proposed:
   extract — duplication is exactly the drift risk MC.4 exists to prevent.)
4. Should approve/request-changes live in the editor only, or also as a lightweight review view with
   comments? (Comments are an MC.11 question; the editor exposes the actions either way.)
5. Do we need per-block comments for review? (Deferred; MC.11 uses change notes and a review note.)

## 19. References

- Files this work touches: `clients/web/src/pages/admin/marketing-content/editor.tsx`,
  `clients/web/src/components/marketing-content/editor/*`,
  `clients/web/src/lib/marketing-content-api.ts`, `clients/packages/content-render/*` (new),
  `www/src/lib/markdown.ts` (becomes the package source).
- Precedents: `clients/web/src/components/editor/block-editor/*` (editor shell patterns),
  `clients/packages/tool-sdk` (shared package wiring), `components/ui/*`.
- Standards: WCAG 2.1 AA (1.4.10, 1.4.12, 2.1.1, 2.1.2, 2.4.7, 4.1.2, 4.1.3), CommonMark.
- Related plans: [MC.4](MC.4-content-rendering-and-validation-service.md),
  [MC.5](MC.5-marketing-media-library.md), [MC.9](MC.9-marketing-content-workspace-shell.md),
  [MC.11](MC.11-editorial-workflow-and-governance.md).
