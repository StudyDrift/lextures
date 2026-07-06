# C02 — Modules & course structure

> CLI parity plan. Source: `courses/{id}/structure` (25 routes), `modules`, `items`, `content-pages`, `external-links`, `registerConditionalReleaseRoutes`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C02 |
| **Section** | Course & content authoring |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / CLI |
| **Depends on** | C01, C40 |
| **Unblocks** | C03, C04, C05 |

---

## 1. Problem Statement

A course's learning structure — modules, ordered items, content pages, prerequisites and conditional-release rules — is the backbone of a course, yet the CLI cannot read or author any of it. Teams that want to define courses as code (git → CI → `lextures`) cannot lay down module skeletons or reorder content programmatically, blocking course templating and mass authoring.

## 2. Goals

- Read the full course structure tree and diff it against a local definition.
- Create/reorder/delete modules and items non-interactively.
- Author content pages from Markdown/HTML files.
- Set prerequisites and conditional-release requirements.

## 3. Non-Goals

- Rich media transcoding (see C33) and file upload internals (existing `files`).
- Quiz/assignment bodies (C03/C04) — this plan manages their placement in modules only.

## 4. Personas & User Stories

- **As a course designer**, I want `structure get <course>` to export the module tree as JSON.
- **As a designer**, I want `modules create/reorder` so I can script a course skeleton.
- **As an author**, I want `pages create --file lesson.md` to publish content pages from disk.
- **As an instructor**, I want `modules set-requirements` to gate a module behind a prerequisite.

## 5. Functional Requirements

- **FR-1.** MUST add `structure get <course> [--tree]` → `GET /api/v1/courses/{id}/structure`.
- **FR-2.** MUST add `modules list|create|update|delete <course>` and `modules reorder <course> --order id,id,...`.
- **FR-3.** MUST add `modules items add|remove|reorder` to place items (assignments, quizzes, pages, files, links) into a module.
- **FR-4.** MUST add `pages create|update|publish|list|get <course>` mapping to `content-pages` (`--file`, `--title`, `--publish`).
- **FR-5.** SHOULD add `links add|list <course>` for `external-links`.
- **FR-6.** SHOULD add `modules set-requirements <course> <module>` and `structure apply --file structure.json` (declarative sync).
- **FR-7.** MAY support `structure apply --dry-run` to preview diffs.

## 6. Non-Functional Requirements

- **Performance** — `structure get` for a large course p95 < 1 s.
- **Security** — server enforces course author scope; 403 → exit 2.
- **Privacy & Compliance** — content pages may hold PII in examples; no special handling beyond auth.
- **Reliability** — `reorder` and `apply` MUST be idempotent given identical input.
- **Observability** — `apply` prints a change summary (created/updated/deleted counts).
- **Maintainability** — new `cmd/modules.go`, `cmd/pages.go`.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course, *When* `structure get --json`, *Then* the module→item tree is emitted.
- **AC-2.** *Given* three modules, *When* `modules reorder --order c,a,b`, *Then* `get` reflects new order.
- **AC-3.** *Given* `lesson.md`, *When* `pages create --file lesson.md --publish`, *Then* a published page id is returned.
- **AC-4.** *Given* a JSON structure, *When* `structure apply --dry-run`, *Then* a diff is printed and nothing changes server-side.

## 8. Data Model

- No client persistence. Optional local `structure.json` schema documented for `apply`.

## 9. API Surface

- `GET /api/v1/courses/{id}/structure`; `.../modules` CRUD + reorder; `.../items` add/remove/reorder; `.../content-pages` CRUD/publish; `.../external-links`; conditional-release requirement endpoints.
- Bodies: JSON; `--file` reads Markdown/HTML for page content.

## 10. UI / UX

- `lextures modules ...`, `lextures pages ...`, `lextures structure ...`.
- `structure apply` shows `+ created / ~ updated / - deleted` summary; `--dry-run` for preview.
- Errors: 409 on reorder conflict → exit 2 with hint to re-fetch.

## 11. AI / ML Considerations

- None directly (page authoring may embed AI-generated content authored elsewhere).

## 12. Integration Points

- Server structure/modules/content-pages handlers.
- Internal: new command files under `clients/cli/cmd/`.

## 13. Dependencies & Sequencing

- After: C01 (course addressing), C40 (`--file`, diff/table helpers).
- Before: C03/C04/C05 use `modules items add` to place their objects.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Structure schema is deeply nested | H | M | Model `structure apply` as thin passthrough; document JSON shape |
| Item polymorphism (page vs quiz vs file) | M | M | `--type` flag + validation |

## 15. Rollout Plan

- Ship read verbs (`get`, `list`) first, then mutating verbs, then `apply`.
- Rollback: additive; revert command files.

## 16. Test Plan

- **Unit** — order flag parsing; type validation.
- **Integration** — httptest asserting reorder body and page create multipart/JSON.
- **E2E** — build a full course skeleton via CLI, verify in web UI.

## 17. Documentation & Training

- "Courses as code" guide showing `structure get > structure.json` → edit → `structure apply`.

## 18. Open Questions

1. Does the server accept a full declarative structure, or only per-object mutations? (Determines `apply` shape.)
2. Are content pages Markdown or HTML on the wire?

## 19. References

- `clients/cli/cmd/structure.go`, `modules.go`, `pages.go`, `links.go`, `structure_test.go`.
- `server/internal/httpserver` structure/modules/content-pages handlers; `registerConditionalReleaseRoutes`.
- Related: [C01](C01-courses.md), [C03](C03-assignments.md), [C05](C05-content-extras.md), [C40](../plan/cli/C40-cli-framework.md).
