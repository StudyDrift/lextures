# C32 — Catalog, library, OER & bookstore

> CLI parity plan. Source: `registerPublicCatalogRoutes` + `registerCatalogRoutes` (`catalog`, `courses/{id}/catalog-listing`), `registerLibraryRoutes` + `registerHELibraryRoutes` (`library`, `orgs/{orgId}/library`), `registerOERRoutes` (`oer`, `admin/oer-providers`), `bookstore_textbook.go` (`admin/bookstore`, `courses/{id}/textbook-resources`, `inclusive-access`). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C32 |
| **Section** | Catalog & materials |
| **Severity** | MINOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Content / CLI |
| **Depends on** | C01, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

The public course catalog, institutional library, OER provider integrations and bookstore/textbook resources are UI-only. Catalog managers cannot bulk-publish course listings, sync library resources, or manage textbook/inclusive-access assignments programmatically.

## 2. Goals

- Manage public catalog listings (publish/unpublish, pricing/visibility) in bulk.
- Search/link library and OER resources into courses.
- Configure bookstore/textbook resources and inclusive-access.

## 3. Non-Goals

- Building the storefront UX.
- Payment (see C30).

## 4. Personas & User Stories

- **As a catalog manager**, I want `catalog publish --course C` and `catalog list --org O`.
- **As a librarian**, I want `library search "..."` and `library link --course C --resource R`.
- **As a course designer**, I want `oer search` to find open resources and attach them.
- **As a bookstore admin**, I want `textbooks set --course C --file textbooks.json`.

## 5. Functional Requirements

- **FR-1.** MUST add `catalog list|get|publish|unpublish` (`catalog-listing`, public catalog read).
- **FR-2.** MUST add `library search|list|link|unlink` (`registerLibraryRoutes`, org library) and `library-resources` per course.
- **FR-3.** SHOULD add `oer search|providers list|link` (`registerOERRoutes`, `admin/oer-providers`).
- **FR-4.** SHOULD add `textbooks list|set` and `inclusive-access get|set` (`bookstore_textbook.go`).

## 6. Non-Functional Requirements

- **Performance** — search paginated; catalog list streamed.
- **Security** — catalog/library admin scope; public catalog read needs no auth (skip-auth annotation).
- **Privacy & Compliance** — inclusive-access ties to student billing consent; respected server-side.
- **Reliability** — publish/link idempotent.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course, *When* `catalog publish`, *Then* it appears in `catalog list`.
- **AC-2.** *Given* a query, *When* `library search`, *Then* matching resources print.
- **AC-3.** *Given* an OER resource, *When* `oer link --course C`, *Then* it is attached to the course.

## 8. Data Model

- None client-side.

## 9. API Surface

- `catalog` public + admin; `library`/org library; `oer` + providers; `bookstore`/`textbook-resources`/`inclusive-access`.

## 10. UI / UX

- `lextures catalog ...`, `lextures library ...`, `lextures oer ...`, `lextures textbooks ...`.

## 11. AI / ML Considerations

- OER/library search may be AI-ranked server-side; CLI reads results.

## 12. Integration Points

- Server catalog/library/OER/bookstore handlers; course linking (C01/C05).

## 13. Dependencies & Sequencing

- After: C01 (course listings), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Public catalog auth model differs | M | L | Mark read commands skip-auth; verify against `registerPublicCatalogRoutes` |

## 15. Rollout Plan

- Ship catalog + library first, then OER + bookstore.
- Rollback: additive.

## 16. Test Plan

- **Unit** — search params; publish idempotency.
- **Integration** — catalog list; library link.
- **E2E** — publish a course to the catalog → verify public read.

## 17. Documentation & Training

- "Bulk-publish courses to your catalog" recipe.

## 18. Open Questions

1. Is the public catalog per-org or global?

## 19. References

- `registerCatalogRoutes`, `registerLibraryRoutes`, `registerOERRoutes`, `bookstore_textbook.go`.
- Related: [C01](C01-courses.md), [C05](C05-content-extras.md), [C30](C30-billing-payments-tax.md).
