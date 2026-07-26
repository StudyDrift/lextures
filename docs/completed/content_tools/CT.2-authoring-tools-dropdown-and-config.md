# CT.2 — Content Tools: Authoring — the Tools Dropdown, Insert Flow & Config Panel

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.2 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web / authoring team |
| **Depends on** | CT.1 |
| **Unblocks** | CT.3, and the authoring half of every tool story CT.10–CT.23 |

---

## 1. Problem Statement

CT.1 gives the platform a tool registry and a place to store a placed tool, but an author has no way
to place one. The section toolbar today offers only formatting affordances (lists, bold, code, link,
table, equation, image) plus **Generate**; the `/` slash menu offers structural blocks. An instructor
who wants a comprehension check inside a paragraph must leave the editor, build a whole quiz item,
and hope the student navigates back. This story adds the **Tools** dropdown to the section toolbar,
the matching `/` slash entries, the generic config panel driven by each tool's JSON Schema, and the
Markdown serialization that makes a placed tool survive save/reload/copy/export.

## 2. Goals

- Add a **Tools ▾** dropdown to the section toolbar (between the image button and *Generate*), listing
  the tools available in this course, searchable, grouped by category.
- Serialize a placed tool as a ` ```lex-tool ` fenced block carrying only `{instanceId, toolId, v}`,
  following the shipped ` ```board ` precedent exactly.
- Render a placed tool in the editor as a compact, non-editable **card** with title, tool icon,
  summary of its config, and Configure / Preview / Duplicate / Delete actions.
- Generate the configuration form **from the manifest's `configSchema`**, so a new tool needs no
  bespoke authoring UI — while allowing a tool to ship a custom editor when the generic form is not
  good enough.
- Keep instance lifecycle transactional with the body save: no dangling fences, no orphan configs.

## 3. Non-Goals

- The student-facing renderer and state persistence (CT.3).
- Tool-specific config UX beyond the generic schema form plus an opt-in custom editor slot (each tool
  story owns its own custom editor if it needs one).
- AI-assisted tool authoring ("generate three inline questions from this section") — deliberately
  deferred to each tool story, which owns its own generation prompt.
- Third-party tools appearing in the palette (CT.9).
- Mobile *authoring* (iOS/Android editors) — read-only rendering of tool blocks in mobile clients is
  covered in CT.3; authoring stays web-first.

## 4. Personas & User Stories

- **As an instructor**, I want a Tools dropdown in the section toolbar so that I can drop an
  interactive element exactly where the student needs it, without leaving the editor.
- **As an instructor**, I want to type `/ask` and get the Ask Questions tool so that I never take my
  hands off the keyboard.
- **As an instructor**, I want to preview a tool as a student sees it so that I can check the wording
  of a question before publishing.
- **As a TA**, I want to duplicate an existing tool and edit its config so that building the fifth
  check on a page takes ten seconds.
- **As an instructor**, I want deleting a tool block to warn me that student work exists so that I do
  not destroy evidence by accident.
- **As a course designer**, I want copied courses to keep their tools working so that our template
  course is genuinely reusable.

## 5. Functional Requirements

- **FR-1.** The section toolbar MUST show a **Tools ▾** dropdown when the course has
  `contentToolsEnabled`; when the flag is off the control MUST NOT render at all.
- **FR-2.** The dropdown MUST list the course catalog from `GET .../content-tools/catalog`, grouped by
  manifest `category`, each row showing icon, localized name and one-line description, with a search
  field that filters on name, description and keywords.
- **FR-3.** The `/` slash menu MUST expose the same tools as entries (`SlashCommandId` gains a
  `tool:{tool_id}` form) so both affordances share one code path.
- **FR-4.** Inserting a tool MUST (a) `POST` an instance with the tool's schema defaults, (b) insert a
  ` ```lex-tool ` block carrying the returned `instanceId` at the caret, and (c) open the config panel.
  If (a) fails, nothing is inserted.
- **FR-5.** The Markdown serialization MUST be exactly:
  ````
  ```lex-tool
  {"instanceId":"<uuid>","toolId":"<tool_id>","v":1}
  ```
  ````
  — one JSON object, stable key order, no trailing whitespace, so diffs stay readable.
- **FR-6.** The editor MUST render the fence as an atomic, draggable, selectable TipTap node
  (`content_tool_block`) with a React node view; the raw JSON MUST never be directly editable.
- **FR-7.** The config panel MUST be generated from `configSchema`, supporting at minimum: string,
  multiline string (Markdown-aware), number, integer, boolean, enum (radio/select), array of objects
  (repeatable rows with add/remove/reorder), and nested objects.
- **FR-8.** A tool MAY declare `ui.customEditor`; when present the framework mounts that component
  instead of the generic form and passes the same `{config, onChange, errors}` contract.
- **FR-9.** Config edits MUST validate client-side against the schema before `PATCH`, and MUST surface
  server 422 field errors inline on the offending control.
- **FR-10.** Deleting a tool block MUST check for existing learner state; if any exists, a confirm
  dialog MUST state how many students have work and offer *Archive* (keep state, stop rendering) as
  the default action, with *Delete permanently* as the destructive secondary.
- **FR-11.** Duplicating a tool MUST create a new instance with copied config and a new `instanceId`,
  never sharing state with the original.
- **FR-12.** Saving the body MUST reconcile fences to instances: fences referencing unknown or
  cross-course instance ids MUST be stripped on save, and instances no longer referenced MUST be
  marked `archived` (never hard-deleted implicitly).
- **FR-13.** The editor MUST offer **Preview as student**, rendering the CT.3 runtime in a disposable
  sandbox whose state is written to a throwaway preview scope and never to a real enrollment.
- **FR-14.** Course copy, course export/import and template instantiation MUST clone instances and
  rewrite the fence ids in the copied bodies (server-side, in the same transaction).
- **FR-15.** The toolbar MUST enforce `max_instances_per_item`; when reached, the dropdown shows a
  disabled state with an explanatory tooltip.
- **FR-16.** Copy/paste of a tool block **within the same course** MUST duplicate the instance; paste
  into a *different* course MUST re-create the instance in the target course when the tool is allowed
  there, and otherwise paste as a plain informational note.

## 6. Non-Functional Requirements

- **Performance** — Dropdown opens in ≤ 100 ms with a cached catalog; the tool card renders without
  fetching (config arrives with the page's instance list). Adding 200 tools to the catalog must not
  regress editor mount time — the palette is data, not code, and renderers are lazily imported.
- **Security** — Instance mutations require the same permission as editing the host item. The editor
  never receives another course's instances. Config values are treated as untrusted when rendered
  (Markdown sanitised through the existing pipeline).
- **Privacy & Compliance** — Preview-as-student writes to a preview scope, so an instructor's
  exploration never creates a record attributable to a real learner.
- **Accessibility** — The dropdown is a WAI-ARIA menu button: roving focus, `Esc` closes, type-ahead,
  `aria-expanded`. The tool card is a labelled group with reachable actions and a clear focus ring.
  The config panel is a standard form with `<label>` association and `aria-describedby` error text.
  Drag-reorder of array rows has keyboard equivalents (move up/down buttons).
- **Scalability** — Catalog responses cached 60 s per course; palette virtualised beyond 50 entries.
- **Reliability** — Insert is atomic (FR-4); a failed config save keeps local edits and retries.
  Body save and instance reconciliation share one server transaction.
- **Observability** — `lextures_content_tool_insert_total{tool_id,surface}` where surface is
  `toolbar|slash|paste|duplicate`; `…_config_save_total{tool_id,outcome}`; time-to-first-configure
  histogram to detect confusing config forms.
- **Maintainability** — One generic form renderer, no per-tool authoring code by default. The palette
  entry, the slash entry and the config form all derive from the same manifest.
- **Internationalization** — Palette names/descriptions and generic form labels resolve from
  `contentTools.tools.{toolId}.*`; schema `title`/`description` are i18n keys, not literals.
- **Backward compatibility** — Bodies without fences are unchanged. A body containing a fence for an
  unknown tool renders a neutral "unavailable tool" card in the editor and a placeholder for students,
  never an error boundary.

## 7. Acceptance Criteria

- **AC-1.** *Given* `contentToolsEnabled=false`, *When* the section editor mounts, *Then* no Tools
  control is present in the DOM.
- **AC-2.** *Given* the flag is on, *When* the author picks a tool from the dropdown, *Then* an
  instance is created, a ` ```lex-tool ` fence is inserted at the caret, and the config panel opens
  focused on the first field.
- **AC-3.** *Given* an inserted tool, *When* the body is saved and the page reloaded, *Then* the same
  card renders with the same config (round-trip through Markdown is lossless).
- **AC-4.** *Given* a config missing a required field, *When* the author saves, *Then* an inline error
  appears on that field and no `PATCH` is sent.
- **AC-5.** *Given* the server rejects a config with 422, *When* the response arrives, *Then* the
  field-level errors are shown on the matching controls and the panel stays open with edits intact.
- **AC-6.** *Given* three students have state for a tool, *When* the author deletes the block, *Then*
  the dialog names the count and defaults to Archive; choosing Archive leaves all three state rows.
- **AC-7.** *Given* a tool block is duplicated, *When* both are rendered for a student, *Then* they
  have independent state rows.
- **AC-8.** *Given* a body whose fence references an instance from another course, *When* it is saved,
  *Then* the fence is stripped and the save succeeds.
- **AC-9.** *Given* a course is copied, *When* the copy is opened, *Then* every tool block renders with
  cloned config, new instance ids, and zero learner state.
- **AC-10.** *Given* the author types `/inline`, *When* the slash menu filters, *Then* the Inline
  Questions tool is listed and `Enter` inserts it identically to the dropdown path.
- **AC-11.** *Given* the item already holds `max_instances_per_item` tools, *When* the dropdown opens,
  *Then* every entry is disabled with an explanatory tooltip and no request is sent.
- **AC-12.** *Given* Preview as student is used and answers are submitted, *When* the instructor exits
  preview, *Then* no `content_tool_states` row exists for any real enrollment.

## 8. Data Model

No new tables. CT.2 uses `course.content_tool_instances` and `course.content_tool_settings` from CT.1.

Migration `server/migrations/450_content_tool_preview_scope.sql` (+ `.down.sql`) adds the preview
escape hatch so previews never pollute real state:

```sql
-- 450_content_tool_preview_scope.sql
ALTER TABLE course.content_tool_states
    ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'enrollment'
        CHECK (scope IN ('enrollment', 'preview'));
COMMENT ON COLUMN course.content_tool_states.scope IS
    'enrollment = real learner work; preview = instructor preview-as-student, purged nightly (plan CT.2).';

-- Real state stays unique per (instance, enrollment); preview rows are excluded from that uniqueness.
DROP INDEX IF EXISTS course.content_tool_states_instance_id_enrollment_id_key;
ALTER TABLE course.content_tool_states
    DROP CONSTRAINT IF EXISTS content_tool_states_instance_id_enrollment_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cts_instance_enrollment_real
    ON course.content_tool_states (instance_id, enrollment_id)
    WHERE scope = 'enrollment';
CREATE INDEX IF NOT EXISTS idx_cts_preview_created
    ON course.content_tool_states (created_at) WHERE scope = 'preview';
```

**Backfill** — every existing row (there are none at this point in the sequence) defaults to
`enrollment`. **Purge** — preview rows older than 24 h are deleted by the nightly sweeper.

## 9. API Surface

Reuses CT.1's instance CRUD; adds two authoring-only routes:

| Verb | Path | Auth scope |
|---|---|---|
| `POST` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/duplicate` | instructor |
| `GET` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/usage` | instructor |

```ts
// GET …/usage — drives the delete confirmation dialog
type ToolInstanceUsage = {
  instanceId: string
  learnersWithState: number
  learnersCompleted: number
  lastInteractionAt: string | null
  referencedInBody: boolean
}
```

Body save (`PATCH /content-pages/{item_id}`, syllabus PUT, assignment PATCH) gains server-side fence
reconciliation: parse fences → validate ownership → archive unreferenced instances — all inside the
existing save transaction. **Rate limits**: duplicate 30/min/user. **OpenAPI**: both routes documented.

## 10. UI / UX

**New components** under `clients/web/src/components/content-tools/authoring/`:
`tools-dropdown.tsx`, `tool-palette-list.tsx`, `tool-block-card.tsx`, `tool-config-panel.tsx`,
`schema-form/` (field renderers), `tool-preview-modal.tsx`; TipTap node
`clients/web/src/components/editor/extensions/content-tool-tip-tap.ts` + `content-tool-node-view.tsx`.

**Modified**: `syllabus-block-editor.tsx` (toolbar slot), `markdown-body-slash.ts` (`tool:` entries),
`markdown-body-editor.tsx` (register the node), `markdown-format-toolbar.tsx` (button placement).

**Flows**

1. *Insert from toolbar* — click **Tools ▾** → search/scan → click a tool → card appears at caret →
   config panel opens in the right sidebar → author fills fields → autosave on blur.
2. *Insert from slash* — type `/` → tools appear under a "Tools" group → `Enter` → same as above.
3. *Reconfigure* — click the card's **Configure** → sidebar panel → changes autosave.
4. *Preview* — card **Preview** → modal renders the CT.3 runtime in preview scope with a persistent
   "Preview — nothing is saved to students" banner and a *Reset preview* button.
5. *Delete* — card **Delete** → usage-aware dialog (FR-10) → Archive (default) or Delete permanently.

**States** — *Empty*: dropdown shows "No tools are enabled for this course" with a link to settings
for instructors. *Loading*: skeleton rows. *Error*: inline retry, editor stays usable. *Offline*:
insertion disabled with a tooltip (an instance id must come from the server); existing cards render.

**Mobile / responsive** — the sidebar config panel becomes a full-height bottom sheet under 768 px;
the palette becomes a full-screen searchable list.

**Accessibility annotations** — menu button pattern (`aria-haspopup="menu"`, `aria-expanded`), focus
returns to the trigger on close; the node view is `contenteditable=false` with
`aria-label="{tool name} interactive element"` and a documented keyboard path (select with arrows,
`Enter` to configure, `Delete` to remove); config panel is a `<form>` with a labelled `<fieldset>`
per schema object.

**Copy & i18n** — `contentTools.authoring.*` in a new `contentTools.json` namespace, tool names under
`contentTools.tools.{toolId}.name|description`.

## 11. AI / ML Considerations

CT.2 makes no model calls. It reserves a `ui.aiAssist` manifest hint that renders a "Generate with
AI" button inside a tool's config panel; the button is inert until the owning tool story implements
its own prompt through `aigateway`. This keeps generation prompts with the tool that understands its
own config shape instead of centralising them here.

## 12. Integration Points

- **Internal** — `clients/web/src/components/syllabus/syllabus-block-editor.tsx`,
  `components/editor/block-editor/*`, `lib/courses-api.ts` (typed client),
  `server/internal/httpserver/content_tools_instances.go`, `service/coursecopy`,
  `service/courseexportimport`, `service/introcourse` (template course ships example tools).
- **Markdown pipeline** — `syllabus-section-markdown.ts` must round-trip the fence untouched;
  `strip-pasted-html-colors.ts` and `markdown-clipboard-paste.ts` must not mangle it.
- **Events** — instance create/update/archive rows written to `content_tool_events` (CT.1).

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1 (registry, instances, settings).
- **Must ship before:** CT.3 (runtime needs placed instances to render) and every tool story.
- **Shared infra needed:** none beyond CT.1.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Generic schema form produces poor UX for complex tools | H | M | `ui.customEditor` escape hatch from day one; the two example tools (CT.10/CT.11) exercise both paths |
| Markdown round-trip corrupts the fence (paste, AI rewrite, import) | M | H | Golden round-trip tests over every editor path; server strips only unrecognised fences and never rewrites valid ones |
| Authors delete blocks and destroy student work | M | H | Usage-aware dialog defaults to Archive; permanent delete is a second, explicit action |
| ACE rewrites a body and drops the fence | M | H | ACE fidelity check treats ` ```lex-tool ` as a protected span (CT.7 §12 + AC.3 fidelity rules); regression test in the ACE suite |
| Palette becomes unusable at 200+ tools | M | M | Search-first design, category grouping, "recently used" section, virtualised list |
| Toolbar crowding on small screens | M | L | Tools collapses into the existing overflow menu below 1024 px |

## 15. Rollout Plan

- **Feature flag** — inherits `content_tools_enabled` (CT.1). No separate flag.
- **Sequencing** — migration `450_*` → server duplicate/usage routes + reconciliation → TipTap node →
  toolbar/slash UI → config forms → preview.
- **Dogfood** — internal course authors build one page per tool as the acceptance walkthrough.
- **GA criteria** — round-trip tests green across all five host kinds; zero body-corruption reports
  over 7 dogfood days.
- **Rollback** — remove the toolbar entry (config flag) while leaving the node registered, so existing
  blocks keep rendering; schema rollback via `450_*.down.sql`.

## 16. Test Plan

- **Unit** — fence parse/serialize round-trip; schema-form field renderers per JSON Schema type;
  validation error mapping; palette filtering/search; slash-command id parsing.
- **Integration** — insert → save → reload → identical Markdown; reconciliation archives unreferenced
  instances; duplicate creates independent instance; cross-course fence stripped; course-copy clone.
- **End-to-end** — Playwright: full insert-configure-preview-delete loop on a content page, an
  assignment body and the syllabus; keyboard-only insert via slash; copy/paste within and across
  courses.
- **Security** — student and TA attempts at instance mutation; instance ids from a foreign course;
  XSS payloads in config strings rendered in the card and preview.
- **Accessibility** — axe on dropdown, card and config panel; NVDA/VoiceOver scripts for insert and
  configure; keyboard-only deletion path.
- **Performance** — editor mount with 20 tool blocks ≤ 150 ms added; palette open ≤ 100 ms with 200
  catalog entries.
- **Manual exploratory** — paste from Word/Google Docs around a tool block; undo/redo across insert
  and delete; ACE-rewritten page still holds its fences.

## 17. Documentation & Training

- **End-user** — "Add an interactive tool to a page" walkthrough with screenshots.
- **Instructor** — archive vs. delete; what students see in preview vs. reality.
- **Developer** — how a tool declares `configSchema` for a good generic form; when to ship a
  `customEditor`; the fence contract and why the body carries only an id.
- **API reference** — duplicate + usage routes.
- **Runbook** — repairing a body whose fences were mangled by an import.

## 18. Open Questions

1. Should the fence embed a human-readable comment line (e.g. the tool name) for Git-diff legibility
   at the cost of a second parse rule? Proposed: no — the card is the human surface.
2. Should archived instances still render for students who already have state (read-only), or vanish?
   Proposed: render read-only with an "closed" badge; confirm with instructors during dogfood.
3. Does the assignment/quiz editor need the palette restricted (e.g. no discussion tools inside a
   quiz body)? Proposed: yes, via `manifest.ui.allowedHostKinds`; add if a tool story needs it.
4. Should "recently used" and "pinned tools" reuse the shipped pinned-settings mechanism (PS.1–PS.4)?
   Proposed: yes; scoped as a fast follow, not a blocker.

## 19. References

- Existing files this work touches: `clients/web/src/components/syllabus/syllabus-block-editor.tsx`,
  `clients/web/src/components/editor/block-editor/markdown-body-slash.ts`,
  `clients/web/src/components/editor/block-editor/markdown-body-editor.tsx`,
  `clients/web/src/components/editor/extensions/board-tip-tap.ts` (pattern),
  `server/internal/httpserver/courses_routes.go`, `server/migrations/450_content_tool_preview_scope.sql`.
- External standards: WAI-ARIA Authoring Practices — Menu Button; JSON Schema draft 2020-12;
  CommonMark fenced code blocks.
- Related plans: [CT.1](CT.1-foundations-registry-and-data-model.md),
  [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.5](CT.5-tool-sdk-sandboxing-and-versioning.md),
  [PS.1–PS.4 pinned settings](../../completed/settings/).
