# C01 — Courses (expand)

> CLI parity plan. Source: `server/internal/httpserver/courses_routes.go`, `course_syllabus.go`, `course_put_nodb_test.go`. Baseline: `clients/cli/cmd/courses.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C01 |
| **Section** | Course & content authoring |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (list, get, create, delete only) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / CLI |
| **Depends on** | C40 (framework) |
| **Unblocks** | C02, C03, C06, C27 |

---

## 1. Problem Statement

The CLI can list, get, create and archive courses but cannot **update** them, toggle publish state, manage the syllabus/settings, clone, or drive blueprint sync. Instructors and platform teams automating course provisioning (bulk term rollover, CI-managed course templates) must fall back to the web UI, breaking scripted workflows and CI/CD of course content.

## 2. Goals

- Full lifecycle: create → configure → publish → clone/rollover → archive/restore from the terminal.
- Idempotent, scriptable course updates suitable for git-tracked course definitions.
- Expose syllabus and course settings as get/set for infra-as-code.
- Support blueprint (master course) associations and sync.

## 3. Non-Goals

- Authoring module/page content (see C02, C05).
- Enrollment management (see C11).
- Grading configuration internals beyond course-level scheme pointer (see C06/C07).

## 4. Personas & User Stories

- **As an admin**, I want to `courses update` scheme/dates in bulk so that term rollover is scripted.
- **As an instructor**, I want to `courses publish`/`unpublish` so I can gate visibility from CI.
- **As a course designer**, I want to `courses clone` a template into a new term.
- **As a platform team**, I want `courses syllabus set --file` so syllabi are version-controlled.

## 5. Functional Requirements

- **FR-1.** MUST add `courses update <code>` mapping to `PUT /api/v1/courses/{id}` with flags for title, description, dates, term, grading-scheme, visibility.
- **FR-2.** MUST add `courses publish <code>` / `courses unpublish <code>`.
- **FR-3.** MUST add `courses restore <code>` (un-archive) complementing existing `delete`.
- **FR-4.** MUST add `courses clone <code> --to-term <id> [--name]` mapping to the copy/blueprint route.
- **FR-5.** SHOULD add `courses syllabus get|set <code>` (`--file`/stdin) via `course_syllabus.go`.
- **FR-6.** SHOULD add `courses settings get|set <code>` (general settings, features/tools toggles).
- **FR-7.** SHOULD add `courses hero-image set <code> <path>` and `courses catalog-listing set`.
- **FR-8.** MAY add `courses blueprint sync <code>` and `courses storage-usage <code>`.

## 6. Non-Functional Requirements

- **Performance** — single-course ops p95 < 500 ms server round-trip.
- **Security** — reuse Bearer auth; server enforces `course:manage`/instructor scope. 403 → exit 2 with message.
- **Privacy & Compliance** — no PII beyond what web UI exposes.
- **Accessibility** — N/A (CLI); ensure `--json` and table output both stable.
- **Reliability** — update/publish MUST be idempotent (re-publish a published course = success no-op).
- **Observability** — commands set a `User-Agent: lextures-cli/<ver>` for server-side attribution.
- **Backward compatibility** — existing `list/get/create/delete` output unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course code, *When* `courses update <code> --title X`, *Then* `PUT` succeeds and `get` reflects the change.
- **AC-2.** *Given* an unpublished course, *When* `courses publish`, *Then* published=true; re-running exits 0.
- **AC-3.** *Given* `--json`, *When* any verb runs, *Then* output is valid JSON on stdout only.
- **AC-4.** *Given* a template course, *When* `courses clone --to-term`, *Then* a new course code is printed.

## 8. Data Model

- No new client-side schema. Server tables unchanged; CLI serializes to existing course DTOs.
- Add a `coursePublic` field parity check test so new flags map to real JSON keys.

## 9. API Surface

- `PUT /api/v1/courses/{id}`, `POST /api/v1/courses/{id}/publish` (or PUT with published flag), `POST /api/v1/courses/{id}/blueprint` (clone/associate), `GET|PUT /api/v1/courses/{id}/syllabus`, `GET|PUT` settings, `PUT /api/v1/courses/{id}/hero-image`, `GET /api/v1/courses/{id}/storage-usage`.
- Reuse `client.NewRequest` + `doWithRetry`; 4xx → `apiError(resp, 2)`.

## 10. UI / UX

- `lextures courses <verb>` under existing `coursesCmd`. Table default; `--json` raw.
- Empty/error states: 404 → "course not found"; 403 → permission message; both exit 2.
- `set` verbs accept `--file -` for stdin piping.

## 11. AI / ML Considerations

- None (course metadata only).

## 12. Integration Points

- Internal: `clients/cli/cmd/courses.go`, `internal/client/client.go`.
- Server: `courses_routes.go`, `course_syllabus.go`.

## 13. Dependencies & Sequencing

- After: C40 (shared table/JSON/`--file` helpers).
- Before: C02, C06 rely on stable course addressing by code.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Publish route shape differs from expectation | M | M | Verify against `courses_routes.go` before coding; add nodb test |
| Clone is async (job) | M | M | Return job id; integrate with C40 `--wait` |

## 15. Rollout Plan

- Feature flag: none needed (additive verbs). Ship behind CLI minor version bump.
- Sequence: add verbs → tests → docs → release notes.
- Rollback: verbs are additive; revert command file.

## 16. Test Plan

- **Unit** — flag parsing, path building (mirror `courses_test.go`).
- **Integration** — httptest server asserting method/path/body per verb.
- **E2E** — create→update→publish→clone→archive→restore against dev stack.
- **Security** — 403 path returns exit 2.

## 17. Documentation & Training

- Update CLI README command table; add `docs/guides` recipe for term rollover scripting.

## 18. Open Questions

1. Is publish a dedicated route or a `PUT` field? Confirm in `courses_routes.go`.
2. Is clone synchronous or job-backed?

## 19. References

- `clients/cli/cmd/courses.go`, `courses_test.go`; `server/internal/httpserver/courses_routes.go`, `course_syllabus.go`.
- Related: [C02](C02-modules-course-structure.md), [C06](C06-gradebook-final-grades.md), [C40](C40-cli-framework.md).
