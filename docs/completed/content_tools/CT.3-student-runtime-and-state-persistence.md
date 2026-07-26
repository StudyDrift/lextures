# CT.3 — Content Tools: Student Runtime Host & State Persistence

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.3 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + backend platform |
| **Depends on** | CT.1, CT.2 |
| **Unblocks** | CT.4, CT.5, CT.7, and the runtime half of every tool story CT.10–CT.23 |

---

## 1. Problem Statement

After CT.1 and CT.2 an author can place a tool and configure it, but a student sees nothing: the
reader renders ` ```lex-tool ` as an unstyled code block, there is no component that knows how to
mount a tool, and there is no write path for learner state. This story ships the **runtime host** —
the single component that resolves a fence to a renderer, hands it its config and state, autosaves
what it produces, dispatches its server actions, and does so with one accessibility, error and
offline story shared by every tool that will ever exist. Getting this contract right is what makes
tool #200 cost a week instead of a quarter.

## 2. Goals

- Render placed tools inside the reader for students, instructors and observers, on web first and
  degrade gracefully on iOS/Android/desktop.
- Ship the **state persistence contract**: load, autosave with optimistic concurrency, explicit
  submit, conflict resolution, size enforcement and schema validation.
- Ship the **action dispatch contract** — one route, registry-routed, so an AI-backed or graded tool
  needs no new endpoint.
- Guarantee failure isolation: a broken tool degrades to a labelled placeholder and never blanks the
  page around it.
- Establish the accessibility baseline every tool inherits (focus, live regions, status announcements)
  so per-tool a11y work is incremental, not from scratch.

## 3. Non-Goals

- Instructor visibility into other learners' state, and reset (CT.4).
- Third-party/iframe isolation and the postMessage bridge (CT.5) — CT.3 mounts first-party renderers
  in-process only.
- Grounded AI context and web-link ingestion (CT.6).
- Analytics dashboards and gradebook passback (CT.7).
- Native authoring or full native parity for every tool on mobile (CT.3 defines the degradation rule;
  each tool story declares its own mobile support level).

## 4. Personas & User Stories

- **As a student**, I want an interactive element to appear inline in the page so that I can act on it
  the moment I read the idea it belongs to.
- **As a student**, I want my work saved automatically so that closing the tab or losing signal does
  not lose my answers.
- **As a student**, I want to come back tomorrow and see exactly what I did so that I can build on it.
- **As a student using a screen reader**, I want the tool to announce its purpose, its state and the
  result of my actions so that I can complete it without sight.
- **As an instructor previewing my own page**, I want to interact with tools normally so that I can
  sanity-check them.
- **As a student on a slow 3G connection in a rural district**, I want the page to remain readable and
  my answers to queue and sync so that connectivity is not a barrier to participation.

## 5. Functional Requirements

- **FR-1.** The reader MUST render a ` ```lex-tool ` fence via a `ContentToolHost` component that
  resolves `toolId` → renderer from the client registry and lazily imports the renderer chunk.
- **FR-2.** The host MUST fetch config + the viewer's state for every instance on the page in **one**
  batched request (`GET .../content-tools/instances?itemId=…&withState=1`).
- **FR-3.** The host MUST pass each renderer a stable, versioned props contract:
  `{instanceId, toolId, config, state, status, readOnly, save(patch), submit(patch), runAction(name, input), t, announce(message)}`.
- **FR-4.** `save()` MUST autosave with debounce (default 1500 ms, tool-overridable 500–10000 ms) and
  MUST flush on blur, on route change and on `visibilitychange → hidden`.
- **FR-5.** Every write MUST carry the last known `revision`; a mismatch MUST return **409** with the
  server's current document, and the host MUST apply the tool's declared conflict policy
  (`server_wins` default, `client_wins`, or `merge` via a tool-provided reducer).
- **FR-6.** The server MUST validate every incoming `state_json` against the tool's `stateSchema`,
  rejecting non-conforming documents with 422 and leaving the stored state untouched.
- **FR-7.** The server MUST enforce `storage.maxStateBytes`, returning 413 without partial writes.
- **FR-8.** `status` transitions MUST be server-enforced: `not_started → in_progress → submitted →
  completed`; backwards transitions are only possible through a CT.4 reset.
- **FR-9.** `POST .../instances/{id}/actions/{action}` MUST dispatch to the manifest-declared handler,
  which receives `(ctx, instance, state, input, principal)` and returns `(statePatch, result, score?)`
  atomically — so a tool that must not trust the client (grading, AI, code execution) never does.
- **FR-10.** Actions MUST be rate-limited per `(user, instance, action)` with per-manifest limits, and
  MUST be idempotent when an `idempotencyKey` is supplied.
- **FR-11.** A renderer that throws MUST be caught by an error boundary rendering a labelled fallback
  card with a retry; the rest of the page MUST continue to function.
- **FR-12.** An unknown `toolId`, an archived instance or a tool disabled by course/org policy MUST
  render an explanatory placeholder, never an error.
- **FR-13.** Offline: writes MUST queue in the existing service-worker/outbox mechanism and replay in
  order on reconnect; the tool MUST show an "unsynced" affordance while queued.
- **FR-14.** The host MUST record interaction telemetry (`first_interacted_at`, `last_interacted_at`,
  `interaction_count`) server-side on write — never trusting client-supplied timestamps.
- **FR-15.** Instructors and TAs MUST be able to interact with tools as themselves; their state rows
  are stored against their own enrollment and MUST be excluded from class aggregates by role.
- **FR-16.** Read-only rendering MUST be supported for: archived instances, closed activities, past
  due dates where the tool declares `respectsDueDate`, and observers/parents.
- **FR-17.** Mobile and desktop clients MUST render a first-class placeholder ("Open in browser to use
  this activity") for renderers they do not implement, and MUST render supported tools natively.
- **FR-18.** The host MUST expose `announce()` backed by a shared `aria-live` region so tools report
  saves, scores and errors without each shipping its own live region.

## 6. Non-Functional Requirements

- **Performance** — Host mount adds ≤ 30 ms per instance on a mid-range device; the batched instance
  request p95 ≤ 120 ms; each tool chunk ≤ 40 KB gzipped (CI budget); a page with 20 tools must stay
  under a 250 KB added transfer budget through lazy loading and viewport-based hydration.
- **Security** — Every read/write re-checks enrollment and course flag server-side; `instance_id` is
  the only client-supplied key and is always validated against the course. Students may only read and
  write **their own** state — enforced in SQL by joining on the caller's enrollment, never by trusting
  a body field. Config redaction from CT.1 applies to every runtime response.
- **Privacy & Compliance** — `state_json` is student work: exported by DSAR (S01), retained per policy
  (S02), and shown to parents only through existing observer permissions. Tools declaring `ai` are
  additionally gated by `aigateway` (CT.6/CT.8).
- **Accessibility** — WCAG 2.1 AA baseline provided by the host: labelled region per tool
  (`role="group"` + `aria-label`), consistent focus order matching visual order, shared polite live
  region, status/score changes announced, no keyboard trap, visible focus, prefers-reduced-motion
  honoured, colour never the sole carrier of correctness.
- **Scalability** — Writes are single-row upserts; expect ~5–20 writes per learner per page. Autosave
  debounce plus coalescing keeps write QPS ≈ page views. Hot instances (a poll during class) are
  handled by CT.21's aggregate cache, not by the state table.
- **Reliability** — Optimistic concurrency prevents lost updates; the offline outbox prevents lost
  work; actions are idempotent; a provider/AI outage degrades an AI tool to a "try again" state
  without losing the transcript.
- **Observability** — `lextures_content_tool_state_saves_total{tool_id,outcome}`,
  `…_state_conflicts_total{tool_id}`, `…_action_latency_seconds{tool_id,action}`,
  `…_render_errors_total{tool_id}`, `…_offline_replays_total`. Log fields `tool_id`, `instance_id`,
  `enrollment_id`, `revision`. Alert: render-error rate > 1% for any `tool_id` over 15 min.
- **Maintainability** — The props contract is versioned (`RUNTIME_CONTRACT_VERSION`); a renderer
  declares the contract range it supports and the host refuses to mount outside it.
- **Internationalization** — `t` is scoped to the tool's i18n namespace; RTL verified; numbers and
  dates through the shipped `formatNumber`/locale helpers.
- **Backward compatibility** — Bodies render identically for courses with the flag off (placeholder
  suppressed entirely). Contract changes are additive; a removed prop requires a major contract bump
  and a migration window (CT.5).

## 7. Acceptance Criteria

- **AC-1.** *Given* a page with a placed tool, *When* a student opens it, *Then* the tool renders with
  its config and any prior state, and the page issues exactly one instances request.
- **AC-2.** *Given* a student types into a tool, *When* 1.5 s elapse, *Then* one `PUT …/state` is sent
  and the tool shows a "Saved" indicator announced politely to screen readers.
- **AC-3.** *Given* two tabs edit the same instance, *When* the stale tab saves, *Then* it receives 409
  with the server document and resolves per the declared conflict policy without data loss for the
  winning writer.
- **AC-4.** *Given* a state document violating `stateSchema`, *When* it is submitted, *Then* the API
  returns 422 and `SELECT state_json` is unchanged.
- **AC-5.** *Given* a state document above `maxStateBytes`, *When* it is submitted, *Then* the API
  returns 413 and the stored document is unchanged.
- **AC-6.** *Given* a student is offline, *When* they interact and then reconnect, *Then* the queued
  writes replay in order and the final state matches the last local state.
- **AC-7.** *Given* a renderer throws during mount, *When* the page renders, *Then* a fallback card
  with a retry appears and every other tool on the page still works.
- **AC-8.** *Given* a student calls `PUT …/state` with another student's `enrollmentId`, *Then* the API
  returns 403 and writes nothing.
- **AC-9.** *Given* a tool declares `scoring.mode = "auto"`, *When* the grading action runs
  server-side, *Then* `score_raw`/`score_max` are set by the server and the client cannot override them.
- **AC-10.** *Given* an action is retried with the same `idempotencyKey`, *Then* the second call
  returns the first result and performs no additional side effects (including no second AI call).
- **AC-11.** *Given* the course flag is switched off while a student has the page open, *When* they
  next interact, *Then* the API returns 404 and the tool switches to a read-only placeholder without
  discarding what is already stored.
- **AC-12.** *Given* an iOS client encounters an unimplemented tool, *Then* it shows the "open in
  browser" placeholder and does not crash or blank the page.
- **AC-13.** *Given* keyboard-only navigation through a page with three tools, *Then* focus order
  matches visual order, every control is reachable, and no focus trap occurs (automated + manual).

## 8. Data Model

No new tables. CT.3 writes `course.content_tool_states` (CT.1) and appends to
`course.content_tool_events`.

Migration `server/migrations/451_content_tool_state_runtime.sql` (+ `.down.sql`) adds only what the
write path needs:

```sql
-- 451_content_tool_state_runtime.sql

-- Idempotency for server actions (AI calls, grading, code runs) — short-lived.
CREATE TABLE IF NOT EXISTS course.content_tool_action_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    instance_id     UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id   UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    action          TEXT NOT NULL,
    result_json     JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctai_created ON course.content_tool_action_idempotency (created_at);

-- Fast "what has this learner touched" lookups for the reader's batched load.
CREATE INDEX IF NOT EXISTS idx_cts_enrollment_instance
    ON course.content_tool_states (enrollment_id, instance_id) WHERE scope = 'enrollment';
```

Idempotency rows are purged after 24 h by the nightly sweeper. **Backfill** — none.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `/api/v1/courses/{course_code}/content-tools/instances?itemId=&withState=1` | course member |
| `GET` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state` | course member (own state) |
| `PUT` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state` | course member (own state) |
| `POST` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/submit` | course member (own state) |
| `POST` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/actions/{action}` | course member |

```ts
type ToolStateEnvelope = {
  instanceId: string
  revision: number                    // increment-on-write concurrency token
  status: 'not_started' | 'in_progress' | 'submitted' | 'completed'
  state: Record<string, unknown>      // validated against the tool's stateSchema
  score: { raw: number; max: number } | null
  updatedAt: string | null
  resetCount: number
  lastResetAt: string | null
}

// PUT …/state
type SaveStateRequest = { revision: number; state: Record<string, unknown>; status?: 'in_progress' | 'submitted' }
// 200 → ToolStateEnvelope · 409 → { error: 'revision_conflict', current: ToolStateEnvelope }
// 422 → { error: 'schema_invalid', errors: [{ path, message }] } · 413 → { error: 'state_too_large', maxBytes }

// POST …/actions/{action}
type RunActionRequest = { input: Record<string, unknown>; idempotencyKey?: string }
type RunActionResponse = { result: Record<string, unknown>; state: ToolStateEnvelope }
```

- **Rate limits** — state writes 120/min/user/instance; actions per-manifest (default 20/min/user,
  AI-backed default 10/min/user) on top of the existing `ratelimit` middleware.
- **WebSocket** — none in CT.3. Tools needing live aggregates (CT.21) poll a cached endpoint; a shared
  channel is considered only if a second tool needs it.
- **OpenAPI** — all five routes documented, including the 409/413/422 shapes.

## 10. UI / UX

**New components** under `clients/web/src/components/content-tools/host/`:
`content-tool-host.tsx` (resolve, load, mount), `tool-frame.tsx` (title bar, status chip, help,
"unsynced"/"saved" indicator), `tool-error-boundary.tsx`, `tool-placeholder.tsx`,
`use-tool-state.ts` (load/save/debounce/conflict), `use-tool-action.ts`, `tool-live-region.tsx`,
`registry/index.generated.ts` (lazy renderer map).

**Modified**: `syllabus-markdown-view.tsx` / `markdown-themed-components.tsx` (map
`code.language-lex-tool` → host), `content-page-reader.tsx`, plus the assignment, quiz-instructions,
syllabus and portfolio readers.

**Flow** — student opens page → host reads fences → one batched fetch → each tool mounts inside a
`ToolFrame` showing name, status and (when scored) score → student interacts → debounce → save →
"Saved ✓" chip and polite announcement → status advances on explicit submit.

**States** — *Loading*: frame with skeleton body, never layout shift. *Empty/not started*: the tool's
own idle UI. *Error*: labelled card + Retry. *Offline*: amber "Saved on this device — will sync" chip.
*Read-only*: dimmed controls + reason ("Archived by your instructor", "Past due", "Preview").
*Disabled tool*: neutral placeholder naming the tool.

**Mobile / responsive** — full-width frames, ≥ 44 px touch targets, no horizontal page scroll (wide
tool content scrolls inside its own container).

**Accessibility annotations** — `role="group"` + `aria-label="{tool name}"` per frame; shared
`aria-live="polite"` region for saves/scores and `aria-live="assertive"` reserved for errors; heading
level derived from surrounding content so the document outline stays valid; `aria-busy` while an
action runs.

**Copy & i18n** — `contentTools.runtime.*` (`saved`, `saving`, `unsynced`, `retry`, `unavailable`,
`readOnlyArchived`, `readOnlyPastDue`, `openInBrowser`).

## 11. AI / ML Considerations

CT.3 makes no model calls itself; it provides the *transport* AI tools use (`runAction`) and the
guarantees around it: server-side execution only, idempotency to prevent duplicate spend, per-tool
rate limits, and `aigateway` evaluation before any provider call (enforced in the action dispatcher,
so a tool cannot bypass disclosure). Cost attribution is written to `analytics.ai_usage_log` with
`feature = manifest.ai.featureId` and the `instance_id` in metadata. Fallback when a provider fails:
the action returns a typed `provider_unavailable` error, the tool keeps its state and shows a retry.

## 12. Integration Points

- **Internal** — `clients/web/src/components/syllabus/syllabus-markdown-view.tsx`,
  `components/content-page/content-page-reader.tsx`, `lib/courses-api.ts`, `sw.ts` (offline outbox),
  `server/internal/httpserver/content_tools_state.go` / `content_tools_actions.go`,
  `service/contenttools/`, `service/aigateway` (action gate), `internal/ratelimit`,
  `internal/telemetry/metrics.go`.
- **Learning events** — `service/learningevents` gains `ContentToolInteracted` /
  `ContentToolCompleted` xAPI statements (CT.7 formalises the verb map).
- **Seat time** (`service/seattime`) — tool interaction counts as active engagement on the page.
- **Mobile/desktop** — `clients/ios`, `clients/android`, `clients/desktop` implement the placeholder
  in this story; native renderers per tool are scoped by each tool story.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1 (model), CT.2 (instances exist to render).
- **Must ship before:** CT.4 (reset needs state), CT.5, CT.7, and every tool story.
- **Shared infra needed:** existing service worker outbox, rate limiter, telemetry, `aigateway`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A single tool's bug blanks a whole content page | M | H | Per-tool error boundary + render-error alerting + CI budget on renderer chunk size |
| Autosave storms on fast typing | M | M | Debounce + coalescing + per-instance rate limit; write path is a single-row upsert |
| Lost work from conflicting tabs | M | H | Revision-based 409 with explicit policies; `merge` reducers for accumulative tools (chat, annotations) |
| Offline replay applies stale documents over newer server state | M | M | Replay carries the original revision; conflicts surface as 409 and resolve by policy, with an "unsynced copy" recovery panel |
| Client-supplied scores trusted | L | H | `scoring.mode = auto` scores are written only by server actions; state schema forbids score fields |
| Page weight balloons as the shelf grows | M | M | Lazy chunks, viewport hydration, per-tool size budget enforced in CI |
| Instructor "interact as self" rows pollute class analytics | M | M | Aggregates filter by enrollment role; CT.7 asserts this in tests |

## 15. Rollout Plan

- **Feature flag** — inherits `content_tools_enabled`; additionally `CONTENT_TOOLS_RUNTIME_READONLY`
  (ops) forces every tool read-only during an incident without hiding stored work.
- **Sequencing** — migration `451_*` → state/action handlers → host + hooks on web → reader wiring →
  mobile placeholders → enable on the dogfood course.
- **Dogfood** — internal course with CT.10 and CT.11 placed on three pages; 2 weeks.
- **GA criteria** — save success ≥ 99.9%, render-error rate < 0.1%, zero data-loss reports, a11y audit
  passed (CT.8 gate).
- **Rollback** — `CONTENT_TOOLS_RUNTIME_READONLY=on` (work preserved, no writes) → flag off →
  migration down as a last resort.

## 16. Test Plan

- **Unit** — debounce/flush logic; conflict policies (`server_wins`, `client_wins`, `merge`); status
  transition validator; schema + size validation; idempotency key handling; registry resolution and
  contract-version gating.
- **Integration** — full save/load cycle against a real DB; 409 on stale revision; 403 on foreign
  enrollment; 413/422 boundaries; action dispatch with a stub tool including `aigateway` denial;
  cascade delete with enrollment.
- **End-to-end** — Playwright: interact → reload → state restored; two-tab conflict; offline
  interaction then reconnect (service-worker harness); error-boundary injection; read-only past-due.
- **Security** — authz matrix across roles and cross-course ids; attempts to write `score_raw` from
  the client; oversized/deeply-nested JSON; prototype-pollution payloads in state; rate-limit
  enforcement.
- **Accessibility** — axe on a page with three tools; NVDA + VoiceOver scripts covering announcement
  of save/score/error; keyboard-only completion of a stub tool; reduced-motion verification.
- **Performance / load** — k6: 500 concurrent learners saving on one instance; assert p95 write
  ≤ 150 ms and zero lost updates. Lighthouse budget on a 20-tool page.
- **Manual exploratory** — flaky-network throttling, mid-action tab close, browser back/forward, RTL
  locale, 200% zoom.

## 17. Documentation & Training

- **End-user** — "Your work in interactive elements is saved automatically" help article, including
  what the unsynced indicator means.
- **Instructor** — how preview differs from student reality; what read-only badges mean.
- **Developer** — the renderer props contract, conflict policies, when to use `save` vs `submit` vs
  `runAction`, and the rule that anything the student must not forge belongs in an action.
- **API reference** — state and action routes with error shapes.
- **Runbook** — read-only kill switch, diagnosing 409 storms, replaying stuck outbox items.

## 18. Open Questions

1. Should `merge` reducers run client-side, server-side, or both? Proposed: client-side for CT.3, with
   a server-side hook reserved for tools where merge order affects grading.
2. Is a shared WebSocket channel warranted for aggregate tools, or is a 5 s cached poll enough?
   Proposed: poll now; revisit when a second live-aggregate tool exists (CT.21 is the first).
3. Should instructor/TA self-interaction rows be stored in the same table with a role filter, or in
   the preview scope? Proposed: same table, role-filtered — instructors legitimately want to keep
   their own notes in tools like CT.13.
4. Default autosave debounce of 1.5 s — validate with real classroom typing telemetry during dogfood.

## 19. References

- Existing files this work touches: `clients/web/src/components/syllabus/syllabus-markdown-view.tsx`,
  `clients/web/src/components/content-page/content-page-reader.tsx`, `clients/web/src/sw.ts`,
  `server/internal/httpserver/courses_routes.go`, `server/internal/service/aigateway/service.go`,
  `server/internal/telemetry/metrics.go`, `server/migrations/451_content_tool_state_runtime.sql`.
- Precedents followed: `content.h5p_completions` per-user state (`server/migrations/165_h5p_packages.sql`);
  quiz attempt autosave semantics (`service/quizattempt`).
- External standards: WCAG 2.1 AA; RFC 7232 (optimistic concurrency semantics); RFC 2119.
- Related plans: [CT.1](CT.1-foundations-registry-and-data-model.md),
  [CT.2](CT.2-authoring-tools-dropdown-and-config.md),
  [CT.4](../../plan/content_tools/CT.4-instructor-state-console-and-reset.md),
  [CT.5](../../plan/content_tools/CT.5-tool-sdk-sandboxing-and-versioning.md),
  [CT.8](../../plan/content_tools/CT.8-governance-safety-privacy-accessibility.md).
