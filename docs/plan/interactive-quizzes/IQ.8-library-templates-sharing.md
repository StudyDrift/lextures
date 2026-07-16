# IQ.8 — Content Library, Templates, Sharing & Discovery

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](README.md). Mirrors the templates/duplication pattern from Collaboration Boards ([VC.8](../visual-collaboration/VC.8-templates-and-duplication.md)) and reuses the org/marketplace sharing surfaces already in the platform.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.8 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad |
| **Depends on** | IQ.1, IQ.2 |
| **Unblocks** | — |

---

## 1. Problem Statement

Teachers rarely start from scratch, and the incumbent tools' biggest lock-in is a huge shared library of
ready-made quizzes. IQ.8 gives Lextures the same gravity: **duplicate** a kit, save/apply **templates**,
**share** a kit within a course or org (or a curated public catalog), and **discover** and import kits others
have made — turning one teacher's work into the whole institution's reusable bank. It also completes the
"duplicate" stub from IQ.1 with a real deep copy.

## 2. Goals

- **Deep duplicate** a kit (questions, media refs, settings) within a course or into another course the user
  teaches.
- **Templates:** save a kit as a reusable template (starter kits, blank structures) and create new kits from
  templates, including a small set of built-in starters.
- **Sharing:** grant other instructors access to a kit at course/org scope (view/copy/edit), respecting roles.
- **Discovery:** a searchable library of shareable kits (org-internal and, where enabled, a curated public
  catalog) with subject/grade/tag filters, preview, and one-click import.
- **Import paths:** from the question bank (IQ.2), from another kit, and (stretch) from QTI (`qtiimport`).

## 3. Non-Goals

- Monetised marketplace mechanics (pricing, payouts) — if desired, reuse the existing course marketplace
  (`marketplacecourses`/billing); IQ.8 ships free org/public sharing only.
- Real-time co-editing of a kit (single-editor with the IQ.2 version stamp is sufficient).
- Cross-tenant public federation beyond the platform's existing catalog boundaries.

## 4. Personas & User Stories

- **As an instructor**, I want to duplicate last term's kit into this term's course, so I reuse it.
- **As an instructor**, I want to save a kit as a template, so my department starts from a common structure.
- **As a department lead**, I want to share a vetted kit with all our instructors, so everyone uses the same
  review game.
- **As an instructor**, I want to search a library for "Grade 7 fractions" and import a ready kit, so I save
  hours.
- **As a self-learner**, I want to browse public study kits and copy one to practice.

## 5. Functional Requirements

- **FR-1.** The system MUST deep-**duplicate** a kit: copy `quizgame.kits` + all `quizgame.questions`
  (including options, correct answers, timers, points styles, and media **references**; media blobs are
  shared/copied per storage policy), producing an independent editable kit owned by the actor.
- **FR-2.** Duplication MUST support a target course the actor has authoring permission in; cross-course copy
  MUST re-scope `course_id` and drop any course-specific bank links (`source_question_id` becomes null or is
  re-resolved).
- **FR-3.** The system MUST support **templates**: `is_template` kits (org/system scope) that appear in a
  "New from template" picker; creating from a template performs a duplicate into the target course.
- **FR-4.** The platform SHOULD ship a few **built-in starter templates** (e.g. "Exit ticket", "Team review",
  "Vocabulary race") seeded at system scope.
- **FR-5.** The system MUST support **sharing** a kit via `quizgame.kit_shares`: grantee = specific user, a
  course, an org unit, or org-wide; permission = `view` | `copy` | `edit`. Sharing respects the kit's
  `visibility` and the actor's role.
- **FR-6.** A **library/discovery** surface MUST let instructors search shared kits by title, subject, grade
  band, language, and tags, preview a read-only kit, and import (copy) it.
- **FR-7.** A curated **public catalog** MAY be enabled per platform flag; public kits require moderation/
  approval (reuse `contentfilter` + an admin review queue) before listing (IQ.9/IQ.11 own the policy).
- **FR-8.** Import MUST preserve accessibility metadata (alt text/captions) and re-validate the imported kit
  (IQ.2 `validate`) so imported content meets the same "ready" bar.
- **FR-9.** Attribution MUST be recorded: an imported kit stores `derived_from_kit_id` and original author
  attribution (non-authorization-bearing, for provenance).
- **FR-10.** Unsharing/removing a kit from the library MUST NOT delete copies already made (copies are
  independent).
- **FR-11.** All sharing/visibility changes MUST be audited.

## 6. Non-Functional Requirements

- **Performance** — duplicate of a 60-question kit < 1 s; library search p95 < 300 ms with filters.
- **Security** — sharing honours roles/scope; `edit` grants are explicit; public listing gated by moderation;
  no cross-tenant leakage.
- **Privacy & Compliance** — shared kits are instructor content, not student data; public kits must not embed
  student PII (validated); attribution respects author preferences.
- **Accessibility** — library and preview are AA; imported kits carry a11y metadata forward.
- **Scalability** — library search indexed (GIN on tags + FTS on title/description); catalog paginated.
- **Reliability** — duplication transactional (all-or-nothing); media reference handling consistent.
- **Observability** — counters: duplicates, template uses, shares, imports, catalog views.
- **Maintainability** — one deep-copy routine reused by duplicate/template/import.
- **Internationalization** — library filters include language; localized subject/grade taxonomies.
- **Backward compatibility** — additive; IQ.1's duplicate stub is replaced by the real deep copy.

## 7. Acceptance Criteria

- **AC-1.** *Given* a 20-question kit, *when* the instructor duplicates it, *then* an independent copy with all
  20 questions, timers, and media appears and edits to the copy don't affect the original.
- **AC-2.** *Given* a kit, *when* saved as a template, *then* it appears in "New from template" and creating
  from it produces a fresh editable kit in the chosen course.
- **AC-3.** *Given* a department lead shares a kit org-wide as `copy`, *when* another instructor opens the
  library, *then* they can preview and import it but not edit the original.
- **AC-4.** *Given* the library, *when* an instructor filters "Grade 7 / fractions / English", *then* matching
  shared kits are returned and previewable.
- **AC-5.** *Given* a public catalog is enabled, *when* an instructor submits a kit, *then* it is not listed
  until it passes moderation.
- **AC-6.** *Given* an imported kit, *when* it loads, *then* alt text/captions are preserved and IQ.2
  validation runs.
- **AC-7.** *Given* an owner unshares a kit, *when* others had already copied it, *then* their copies remain
  intact.

## 8. Data Model

Migration `397_interactive_quizzes_sharing.sql`:

```sql
ALTER TABLE quizgame.kits
  ADD COLUMN IF NOT EXISTS is_template       BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS template_scope    TEXT,        -- system | org | course (when is_template)
  ADD COLUMN IF NOT EXISTS derived_from_kit_id UUID REFERENCES quizgame.kits (id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS subject           TEXT,
  ADD COLUMN IF NOT EXISTS grade_band        TEXT,
  ADD COLUMN IF NOT EXISTS language          TEXT,
  ADD COLUMN IF NOT EXISTS catalog_status    TEXT NOT NULL DEFAULT 'unlisted'; -- unlisted|pending|listed|rejected

CREATE TABLE quizgame.kit_shares (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kit_id        UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
  grantee_type  TEXT NOT NULL,               -- user | course | org_unit | org
  grantee_id    UUID,                        -- NULL for org-wide
  permission    TEXT NOT NULL DEFAULT 'copy',-- view | copy | edit
  created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (kit_id, grantee_type, grantee_id, permission)
);
CREATE INDEX idx_quizgame_kit_shares_kit ON quizgame.kit_shares (kit_id);

-- discovery indexes
CREATE INDEX idx_quizgame_kits_catalog ON quizgame.kits (catalog_status) WHERE catalog_status = 'listed';
CREATE INDEX idx_quizgame_kits_title_fts ON quizgame.kits USING gin (to_tsvector('english', title || ' ' || description));
```

- Deep-copy routine is transactional across `kits` + `questions`.
- Built-in starters seeded via a data migration at `system` template scope.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| POST | `/live-quizzes/kits/{kit_id}/duplicate` `{targetCourseCode?}` | `item:create` in target |
| POST | `/live-quizzes/kits/{kit_id}/save-as-template` `{scope}` | `item:create` (+ org role for org scope) |
| GET | `/live-quizzes/templates` | authenticated instructor |
| POST | `/live-quizzes/templates/{id}/create-kit` `{targetCourseCode}` | `item:create` |
| POST/DELETE | `/live-quizzes/kits/{kit_id}/shares` | kit owner / grade-role |
| GET | `/live-quizzes/library?q=&subject=&grade=&lang=&tag=` | authenticated instructor |
| GET | `/live-quizzes/library/{kit_id}/preview` | share-scoped |
| POST | `/live-quizzes/library/{kit_id}/import` `{targetCourseCode}` | `item:create` |
| POST | `/live-quizzes/kits/{kit_id}/submit-to-catalog` | owner (→ moderation) |

- **OpenAPI:** document share/template/library schemas and moderation status.
- **Rate-limit:** import/duplicate under the standard write limiter.

## 10. UI / UX

- **Kit gallery (IQ.1) additions:** "Duplicate", "Save as template", "Share", "Submit to catalog" on the kit
  menu; a "New from template" entry on Create.
- **Library page** `clients/web/src/pages/lms/live-quiz-library-page.tsx`: search + facet filters, kit cards
  with subject/grade/language/tags, read-only preview drawer, "Import to course" action.
- **Share dialog:** grantee picker (user/course/org unit/org), permission selector, current-shares list.
- **Flows:** duplicate → land in editor; save-as-template → confirm scope; share → pick grantee/permission;
  library → search → preview → import → editor.
- **States:** empty library, moderation-pending badge, already-imported, share-list empty.
- **Accessibility:** facet filters keyboard-operable; preview is a read-only accessible view; cards are links.
- **Copy & i18n:** `liveQuiz.library.*`, `liveQuiz.share.*`, `liveQuiz.template.*`.

## 11. AI / ML Considerations

Optional: AI-suggested tags/subject/grade on save (reuse IQ.10/AP path) to improve discovery. Not required.

## 12. Integration Points

- **Reuse:** org/role model (`orgunit`, `orgroles`, `orgrolegrant`), `contentfilter` + an admin review queue
  for catalog moderation (IQ.9/IQ.11), storage media policy, FTS/GIN indexing patterns, audit
  (`adminaudit`/course audit).
- **Server new:** `repos/quizgame/{sharing,templates,library}.go`, deep-copy routine in `repos/quizgame`,
  `httpserver/quizgame_library.go`.
- **Web new:** library page, share/template dialogs, gallery menu additions.

## 13. Dependencies & Sequencing

- Must ship after: IQ.1 (kits), IQ.2 (questions to copy).
- Must ship before: nothing hard-depends; public catalog listing needs IQ.9 moderation and IQ.11 admin queue.
- Shared infra: org/role model, content moderation, storage, search indexing.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Public catalog hosts inappropriate/copyright content | M | H | Mandatory moderation queue before listing; report/takedown (IQ.9); attribution |
| Deep copy misses media / breaks references | M | M | Single transactional copy routine; media policy explicit; tests assert independence |
| Sharing scope bugs leak kits cross-tenant | L | H | Reuse proven org/role scoping; authz matrix tests |
| Library search slow at scale | M | M | GIN + FTS indexes; pagination; facet counts cached |
| `edit` shares cause surprise co-edits | M | M | Default `copy`; `edit` explicit + version stamp warns on conflict |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled`; public catalog behind a separate platform flag (default off), org
  sharing on by default within the feature.
- **Sequencing:** migration `397` → duplicate/template → org sharing + library → (later) public catalog + moderation.
- **Dogfood:** duplicate across courses, save/apply a template, share org-wide, import from library.
- **GA criteria:** AC-1..AC-7 pass; deep-copy independence verified; sharing authz matrix green.
- **Rollback:** disable public-catalog flag; org sharing/duplicate retained.

## 16. Test Plan

- **Unit** — deep-copy completeness/independence; template scope rules; share permission resolution.
- **Integration** — duplicate cross-course re-scoping; share grant/revoke; library search facets; import +
  re-validate.
- **End-to-end** — Playwright: duplicate, save template, create-from-template, share org-wide, import.
- **Security** — cross-tenant/cross-course share leakage; catalog moderation gate; `edit` vs `copy`.
- **Accessibility** — library/preview/share dialog axe + keyboard.
- **Performance** — 60-question duplicate; library search under target.
- **Manual** — unshare-after-copy independence; moderation approve/reject flow.

## 17. Documentation & Training

- Instructor: "Duplicate & reuse kits", "Templates", "Share with your department", "Find & import kits".
- Admin: enabling the public catalog; moderation workflow; attribution policy.
- API reference: sharing/template/library endpoints.
- Runbook: deep-copy routine, seeded starters, moderation queue.

## 18. Open Questions

1. Ship a public catalog at GA or org-only first? (Recommendation: org-only at GA; public catalog after IQ.9
   moderation + IQ.11 admin queue are proven.)
2. Do we reuse the course marketplace for *paid* kits, or keep IQ.8 free-only? (Recommendation: free-only;
   revisit paid via `marketplacecourses` if demand appears.)
3. Media on copy — reference-share or duplicate blobs? (Recommendation: reference-share within a tenant;
   duplicate on cross-tenant import to avoid dangling refs.)

## 19. References

- Existing files: `server/internal/repos/orgunit/`, `server/internal/repos/orgroles/`,
  `server/internal/repos/marketplacecourses/`, `server/internal/repos/contentfilter/`,
  `server/internal/repos/adminaudit/`.
- Related plans: [IQ.1 (completed)](../../completed/interactive-quizzes/IQ.1-foundation-and-feature-flag.md), [IQ.2 (completed)](../../completed/interactive-quizzes/IQ.2-kit-authoring-and-question-types.md),
  [IQ.9](IQ.9-moderation-safety-accessibility.md), [IQ.11](IQ.11-admin-governance-quotas-lifecycle.md),
  [VC.8](../visual-collaboration/VC.8-templates-and-duplication.md).
