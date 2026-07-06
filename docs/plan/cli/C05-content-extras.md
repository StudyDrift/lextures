# C05 — Content extras (pages, glossary, H5P, SCORM, external tools, resources)

> CLI parity plan. Source: `courses/{id}` sub-resources `glossary`, `h5p`/`h5p-items`, `scorm`/`scorm-items`, `lti-external-tools`, `external-links`, `textbook-resources`, `library-resources`, `collab-docs`, `whiteboards`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C05 |
| **Section** | Course & content authoring |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Content / CLI |
| **Depends on** | C02, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Rich course content — interactive H5P, SCORM packages, glossaries, embedded external (LTI) tools, textbook/library resource links, collaborative docs and whiteboards — cannot be managed from the CLI. Content teams migrating or bulk-loading interactive packages (e.g. dozens of SCORM zips) must click through the UI package by package.

## 2. Goals

- Bulk-import SCORM/H5P packages and link them into modules.
- Manage course glossary terms as data (get/set from file).
- Register external LTI tools and resource links per course.
- Read (and where sensible, create) collab-docs/whiteboards for backup/migration.

## 3. Non-Goals

- Live editing of whiteboards/collab-docs (real-time browser experience).
- Media transcoding (see C33) beyond triggering it via existing upload.

## 4. Personas & User Stories

- **As a content engineer**, I want `scorm import <zip>` to load a package and get its item id.
- **As a designer**, I want `glossary set --file glossary.csv` to bulk-load terms.
- **As an admin**, I want `tools add --url ... --key ...` to register an LTI tool in a course.
- **As an archivist**, I want `collab-docs export <course>` for backup.

## 5. Functional Requirements

- **FR-1.** MUST add `scorm import <course> <zip>` and `scorm list|get|delete` (+ `scorm-items`).
- **FR-2.** MUST add `h5p import <course> <package>` and `h5p list|delete` (+ `h5p-items`).
- **FR-3.** MUST add `glossary list|get|set <course>` (`--file` CSV/JSON) and per-term add/delete.
- **FR-4.** SHOULD add `tools add|list|remove <course>` for `lti-external-tools` and `external-links`.
- **FR-5.** SHOULD add `resources link|list <course>` for `textbook-resources`/`library-resources`.
- **FR-6.** MAY add `collab-docs list|export` and `whiteboards list|export` (read/backup only).

## 6. Non-Functional Requirements

- **Performance** — package import streams uploads with progress (reuse `files upload`).
- **Security** — content author scope; LTI tool secrets never echoed back in output.
- **Privacy & Compliance** — collab-doc/whiteboard exports may contain student content → FERPA `--yes` gate on bulk export.
- **Reliability** — import is resumable/skips-existing by content hash where the server supports it.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a SCORM zip, *When* `scorm import`, *Then* an item id is returned and it appears in `structure get`.
- **AC-2.** *Given* `glossary.csv`, *When* `glossary set --file`, *Then* terms are created and `glossary list` shows them.
- **AC-3.** *Given* an LTI tool, *When* `tools add`, *Then* the secret is not printed in `--json` output.

## 8. Data Model

- None client-side. Document CSV/JSON glossary schema.

## 9. API Surface

- SCORM/H5P import + item endpoints; `glossary` CRUD; `lti-external-tools`/`external-links`; `textbook-resources`/`library-resources`; `collab-docs`/`whiteboards` read/export.

## 10. UI / UX

- `lextures scorm|h5p|glossary|tools|resources ...`, all `--course` scoped.
- Import shows upload progress; `--json` returns created ids.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server content handlers; TUS/upload pipeline (existing `files upload`).
- Internal: new command files.

## 13. Dependencies & Sequencing

- After: C02 (to link items into modules), C40 (upload/progress/`--file`).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SCORM/H5P import is async (transcode/unpack job) | M | M | Return job id; integrate C40 `--wait` |
| Whiteboard/collab-doc export format proprietary | M | L | Export as opaque JSON blob; document limits |

## 15. Rollout Plan

- Ship SCORM/H5P import + glossary first (highest bulk value), then tools/resources, then read-only exports.
- Rollback: additive.

## 16. Test Plan

- **Unit** — glossary CSV parsing; secret redaction.
- **Integration** — package import multipart; item listing.
- **E2E** — import SCORM → verify in module.

## 17. Documentation & Training

- "Bulk-import SCORM packages" recipe.

## 18. Open Questions

1. Are SCORM/H5P imports synchronous or job-backed?
2. Is there a stable export format for collab-docs/whiteboards?

## 19. References

- Content sub-resource handlers in `server/internal/httpserver`.
- Related: [C02](C02-modules-course-structure.md), [C23](C23-lti-developer-keys.md), [C33](C33-accessibility-media-localization.md), [C40](C40-cli-framework.md).
