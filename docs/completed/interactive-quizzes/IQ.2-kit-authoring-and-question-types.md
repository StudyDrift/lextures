# IQ.2 — Quiz-Kit Authoring & Question Types

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md). Reuses the question bank (`course.questions`, migration `075`) as an item source and the storage-object/media pipeline used elsewhere in the app.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.2 |
| **Section** | Interactive Quizzes |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Assessment squad |
| **Depends on** | IQ.1 |
| **Unblocks** | IQ.3, IQ.8, IQ.10 |

---

## 1. Problem Statement

A live quiz is only as good as its questions. IQ.2 turns the empty kit shell from IQ.1 into a full **authoring
experience**: an ordered list of game questions, each with a prompt, media, answer options, correct answer(s),
a per-question **time limit**, and a **points style** — plus the game-native question types (fast multiple
choice, true/false, type-the-answer, poll/opinion, ordering, numeric, "find the answer" word cloud) that make
a live quiz fun. Crucially it **reuses the question bank** so instructors can pull existing items into a kit
rather than re-authoring them.

## 2. Goals

- Let instructors add, edit, reorder (drag or keyboard), duplicate, and delete questions within a kit.
- Support the core game question types with per-question configuration (time limit, points style, media,
  answer shuffle).
- Reuse `course.questions`: import bank items into a kit (copy-with-link) and, optionally, push kit questions
  back to the bank.
- Attach media (image/audio/video/GIF) to prompts and answer options via the existing storage pipeline, with
  alt text and captions.
- Validate kits for "ready to host" (every question has a prompt, ≥1 correct answer where required, valid
  timer) and surface actionable errors.
- Autosave drafts so authoring never loses work.

## 3. Non-Goals

- Live hosting / gameplay (IQ.3/IQ.4) — IQ.2 only authors content.
- Scoring maths (IQ.5) — IQ.2 stores the *points style* selector; IQ.5 computes points.
- AI generation of questions (IQ.10) — IQ.2 exposes the insertion API IQ.10 calls.
- Full QTI import (referenced via `qtiimport`; a stretch mapping is noted, not required here).

## 4. Personas & User Stories

- **As an instructor**, I want to add a multiple-choice question with an image and a 20-second timer, so my
  class can race to answer.
- **As an instructor**, I want to reorder questions by dragging, so the quiz flows the way I teach.
- **As an instructor**, I want to pull ten questions I already wrote in the question bank into this kit, so I
  don't retype them.
- **As an instructor**, I want a "poll" question with no right answer, so I can take the room's opinion.
- **As an instructor**, I want the editor to tell me a question is missing a correct answer before I try to
  host, so I don't get caught mid-class.
- **As a self-learner building study games**, I want type-the-answer and ordering questions, so recall is
  active, not just recognition.

## 5. Functional Requirements

- **FR-1.** The system MUST store ordered questions in `quizgame.questions` (kit_id, position, type, prompt,
  media, options JSONB, correct JSONB, time_limit_seconds, points_style, answer_shuffle, source_question_id).
- **FR-2.** The system MUST support these game question types at launch:
  `mc_single`, `mc_multiple`, `true_false`, `type_answer` (short text, fuzzy/exact match list), `numeric`
  (value ± tolerance), `poll` (no correct answer), `ordering` (sequence), and `word_cloud` (open short text,
  aggregated). Types map onto/borrow from `course.question_type` where equivalents exist.
- **FR-3.** Each question MUST carry a `time_limit_seconds` (5–240, default 20) and a `points_style`
  (`standard` | `double` | `no_points`); IQ.5 consumes these.
- **FR-4.** MC options MUST support 2–6 answers, each with optional media and an `is_correct` flag; the editor
  MUST enforce ≥1 correct for graded MC and exactly the poll rule (0 correct) for polls.
- **FR-5.** `type_answer` MUST store an ordered list of accepted answers with a per-answer match mode
  (`exact` | `case_insensitive` | `trim` | `fuzzy≤N`); `numeric` MUST store value + tolerance + optional unit.
- **FR-6.** The system MUST allow **import from the question bank**: given `course.questions` ids, create kit
  questions that copy presentation and set `source_question_id` (a link, not a hard FK dependency, so bank
  edits don't silently mutate a hosted kit). Available only when `question_bank_enabled`.
- **FR-7.** The system SHOULD allow **push to bank**: create a `course.questions` row from a kit question
  (respecting the bank's type mapping in `questionbank/sync_editor.go`).
- **FR-8.** Media attachments MUST go through the existing storage-object upload + AV-scan pipeline
  (`storageobjects`, `avscanjobs`), and each media MUST have alt text (images) or captions (audio/video)
  fields for accessibility.
- **FR-9.** Reordering MUST persist a stable integer/`position` ordering; the API MUST accept a bulk reorder.
- **FR-10.** The editor MUST **autosave** (debounced) and support optimistic edits; a kit MUST expose a
  computed `is_ready` validation with a list of blocking issues per question.
- **FR-11.** Deleting/duplicating a question MUST keep `quizgame.kits.question_count` in sync (trigger or
  repo-maintained).
- **FR-12.** Question prompts and answers MUST be length-capped and sanitised (no active HTML; markdown-lite
  where the app already renders it) to keep the projector view safe.

## 6. Non-Functional Requirements

- **Performance** — editor load for a 60-question kit < 500 ms; autosave round-trip < 200 ms p95.
- **Security** — course-scoped; media served via signed URLs; imported bank items respect bank sharing scope.
- **Privacy & Compliance** — kit content is instructor IP; media inherits DRM/retention policies.
- **Accessibility** — full keyboard authoring incl. reorder (move-up/down + drag); every media field requires
  alt/caption; colour is never the only differentiator of an answer.
- **Scalability** — a kit realistically ≤ 200 questions; JSONB options bounded; media offloaded to storage.
- **Reliability** — autosave is idempotent and last-write-wins per field with a version stamp to catch
  conflicting concurrent edits.
- **Observability** — counters per question type created, bank imports, media attach failures.
- **Maintainability** — one canonical question-type registry (server enum + shared TS union) drives editor,
  validation, and hosting.
- **Internationalization** — question content is authored in any language; UI chrome localised; RTL-safe.
- **Backward compatibility** — additive; new tables only.

## 7. Acceptance Criteria

- **AC-1.** *Given* a kit, *when* the instructor adds an MC question with 4 options and marks one correct and
  sets a 15s timer, *then* it persists and appears in the ordered list.
- **AC-2.** *Given* a kit with 5 questions, *when* the instructor drags Q4 above Q2, *then* the new order
  persists and reloads identically.
- **AC-3.** *Given* the question bank flag is on and 3 bank items selected, *when* the instructor imports,
  *then* 3 kit questions are created with `source_question_id` set and editable independently.
- **AC-4.** *Given* a poll question, *when* the instructor saves it with no correct answer, *then* it is valid
  (polls are exempt from the ≥1-correct rule).
- **AC-5.** *Given* an MC question with no correct answer marked, *when* the instructor tries to mark the kit
  ready, *then* validation blocks it and points to that question.
- **AC-6.** *Given* an image attached without alt text, *when* saving, *then* the editor requires alt text
  before the kit is "ready".
- **AC-7.** *Given* concurrent edits to the same question from two tabs, *when* both save, *then* the version
  stamp detects the conflict and the user is warned rather than silently overwritten.

## 8. Data Model

Migration `387_interactive_quizzes_questions.sql`:

```sql
CREATE TYPE quizgame.question_type AS ENUM (
  'mc_single','mc_multiple','true_false','type_answer','numeric','poll','ordering','word_cloud'
);
CREATE TYPE quizgame.points_style AS ENUM ('standard','double','no_points');

CREATE TABLE quizgame.questions (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kit_id             UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
  position           INTEGER NOT NULL,
  question_type      quizgame.question_type NOT NULL DEFAULT 'mc_single',
  prompt             TEXT NOT NULL,
  prompt_media_ref   TEXT,                       -- storage object key
  prompt_media_alt   TEXT,                       -- alt text / caption ref
  options            JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{id,text,mediaRef,mediaAlt,isCorrect}]
  correct_answer     JSONB,                      -- type-specific (accepted answers, numeric+tolerance, order)
  time_limit_seconds INTEGER NOT NULL DEFAULT 20 CHECK (time_limit_seconds BETWEEN 5 AND 240),
  points_style       quizgame.points_style NOT NULL DEFAULT 'standard',
  answer_shuffle     BOOLEAN NOT NULL DEFAULT TRUE,
  explanation        TEXT,                       -- shown on reveal
  source_question_id UUID REFERENCES course.questions (id) ON DELETE SET NULL, -- bank link, non-blocking
  version            INTEGER NOT NULL DEFAULT 1, -- optimistic-concurrency stamp
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (kit_id, position)
);
CREATE INDEX idx_quizgame_questions_kit ON quizgame.questions (kit_id, position);
```

- **Constraints:** unique `(kit_id, position)`; timer bounds via CHECK; `question_count` on `quizgame.kits`
  maintained by trigger `AFTER INSERT/DELETE`.
- **Backfill:** none.
- **Down:** drop table + enums.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET | `/live-quizzes/kits/{kit_id}/questions` | course access |
| POST | `/live-quizzes/kits/{kit_id}/questions` | `item:create` |
| PATCH | `/live-quizzes/kits/{kit_id}/questions/{qid}` (If-Match version) | `item:create` |
| DELETE | `/live-quizzes/kits/{kit_id}/questions/{qid}` | `item:create` |
| POST | `/live-quizzes/kits/{kit_id}/questions/reorder` (`[{id,position}]`) | `item:create` |
| POST | `/live-quizzes/kits/{kit_id}/questions/import-bank` (`{questionIds:[]}`) | `item:create` |
| POST | `/live-quizzes/kits/{kit_id}/questions/{qid}/push-to-bank` | `item:create` |
| GET | `/live-quizzes/kits/{kit_id}/validate` → `{ isReady, issues:[{questionId,code,message}] }` | course access |

- **Media** upload reuses the standard `POST /storage/objects` presign flow; the returned key is stored as
  `prompt_media_ref` / option `mediaRef`.
- **Concurrency:** PATCH takes an `If-Match: <version>`; mismatch → `409`.
- **OpenAPI:** register all shapes; a shared TS union `LiveQuizQuestion` mirrors the server enum.

## 10. UI / UX

- **Kit editor page** `clients/web/src/pages/lms/live-quiz-kit-editor-page.tsx`: left rail = ordered question
  list (drag handles, type icons, timer badges), center = question editor, right = per-question settings
  (timer, points style, shuffle, explanation).
- **Components** in `components/live-quiz/`: `question-type-picker`, `mc-option-list`, `type-answer-editor`,
  `numeric-editor`, `ordering-editor`, `poll-editor`, `word-cloud-editor`, `media-attach`, `bank-import-drawer`.
- **Flows:** (1) add question → pick type → (2) fill prompt/options → (3) set timer/points → (4) autosave →
  (5) reorder → (6) "Import from bank" drawer → (7) "Check kit" surfaces validation issues.
- **States:** empty kit ("Add your first question"), autosaving/saved indicator, validation error list,
  media-uploading, media-scan-pending, offline (queue edits).
- **Responsive:** authoring is desktop-first but usable on tablet; small screens collapse the settings rail.
- **Accessibility:** reorder via keyboard (move up/down buttons alongside drag); each media requires alt text;
  answer options labelled by index; focus management on add/delete.
- **Copy & i18n:** `liveQuiz.editor.*`, `liveQuiz.qtype.*`, `liveQuiz.validate.*`.

## 11. AI / ML Considerations

Not AI-touching directly, but IQ.2 defines the **insertion contract** (question shapes + validation) that
IQ.10's AI generation writes through, so generated questions are indistinguishable from hand-authored ones and
pass the same validation.

## 12. Integration Points

- **Reuse:** `course.questions` + `questionbank/` (import/push, type mapping in `sync_editor.go` /
  `delivery_resolve.go`), `storageobjects` + `avscanjobs` (media), `imagealtrepo`/captions pipeline for a11y,
  markdown renderer already used for prompts.
- **Server new:** `repos/quizgame/questions.go`, `httpserver/quizgame_questions.go`.
- **Web new:** editor page + `components/live-quiz/*`; extend `live-quiz-api.ts`.

## 13. Dependencies & Sequencing

- Must ship after: IQ.1 (kits exist).
- Must ship before: IQ.3 (host needs questions), IQ.8 (templates copy questions), IQ.10 (AI writes questions).
- Shared infra: storage/AV-scan, question bank.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Bank import couples kit to later bank edits | M | M | Copy-with-link (`source_question_id` non-blocking); never live-join at host time |
| Question-type sprawl | M | M | Ship the 8 core types; gate exotic types (hotspot, code) behind later stories |
| Media without alt text ships to projector | M | M | Validation blocks "ready" until alt/caption present |
| Reorder race / position gaps | M | L | Bulk reorder endpoint recomputes contiguous positions in a tx |
| Concurrent-edit clobber | M | M | Version stamp + `If-Match` → 409 |

## 15. Rollout Plan

- **Flag:** same `interactive_quizzes_enabled`; bank-import sub-affordance gated by `question_bank_enabled`.
- **Sequencing:** migration `387` → server question CRUD → editor UI → enable for pilot instructors from IQ.1.
- **Dogfood:** author a 15-question kit end-to-end incl. media + bank import; run validation.
- **GA criteria:** all 8 types author/validate/persist; bank import round-trips; a11y checks pass.
- **Rollback:** feature flag off; questions retained.

## 16. Test Plan

- **Unit** — per-type validation, points-style/timer bounds, position reordering, version-stamp conflict.
- **Integration** — CRUD + reorder + bank import/push authz; media attach + scan-pending gate; `validate`.
- **End-to-end** — Playwright: author each type, attach image with alt, reorder, import bank, fix validation.
- **Security** — cross-course kit/question probing; imported bank items respect sharing scope.
- **Accessibility** — axe on editor; keyboard reorder; alt-text enforcement.
- **Performance** — 60-question editor load + autosave latency.
- **Manual** — RTL prompt authoring; long-answer wrapping on projector preview.

## 17. Documentation & Training

- End-user: "Author a quiz kit"; per-type authoring tips; "Import from your question bank".
- Instructor: accessibility requirements (alt text/captions), validation checklist.
- API reference: question + import endpoints.
- Runbook: question-type registry location; media pipeline touchpoints.

## 18. Open Questions

1. Do we support **QTI import** into kits now, or defer to IQ.8's import surface? (Recommendation: defer;
   expose the internal insert API so `qtiimport` can target it later.)
2. Should `type_answer` fuzzy matching be Levenshtein-N or token-based? (Recommendation: Levenshtein-N with a
   small default, configurable per answer.)
3. Do bank edits ever propagate into kits? (Recommendation: no — copy-with-link only; offer a manual
   "re-sync from bank" action instead of silent propagation.)

## 19. References

- Existing files: `server/migrations/075_question_bank.sql`,
  `server/internal/repos/questionbank/sync_editor.go`, `server/internal/repos/questionbank/delivery_resolve.go`,
  `server/internal/repos/storageobjects/`, `server/internal/repos/avscanjobs/`.
- Related plans: [IQ.1](IQ.1-foundation-and-feature-flag.md),
  [IQ.3 (completed)](IQ.3-live-game-hosting-engine.md),
  [IQ.10](IQ.10-ai-assisted-generation.md).
