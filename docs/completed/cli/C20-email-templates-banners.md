# C20 — Email templates & banners

> CLI parity plan. Source: `admin_email_templates.go` (`admin-console/email-templates`, 8), `banner_http.go` (`admin/banners`, 5). Baseline: `clients/cli/cmd/email_templates.go`, `banners.go`, `email_templates_banners_logic.go`, `email_templates_banners_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C20 |
| **Section** | Admin & governance |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Transactional email templates and system banners (maintenance notices, announcements) are UI-only. Admins cannot version-control template copy, localize it in bulk, or script scheduled maintenance banners.

## 2. Goals

- Manage email templates as version-controlled files (get/set/preview).
- Schedule/publish/expire system banners from CLI (e.g. from a maintenance runbook).

## 3. Non-Goals

- Actually sending emails (server transactional pipeline).
- Rich WYSIWYG editing.

## 4. Personas & User Stories

- **As a comms admin**, I want `email-templates set welcome --file welcome.html` in git.
- **As an SRE**, I want `banners create --message "Maintenance 2am" --from --until` in a runbook.
- **As a localizer**, I want `email-templates set --locale es` for translated copy.

## 5. Functional Requirements

- **FR-1.** MUST add `email-templates list|get|set|preview <key>` (`--file`, `--locale`).
- **FR-2.** MUST add `banners list|create|update|delete` (`--message`, `--severity`, `--from`, `--until`, `--audience`).
- **FR-3.** SHOULD add `email-templates test-send --to <email>` for validation.
- **FR-4.** MAY add `banners publish|expire <id>` shortcuts.

## 6. Non-Functional Requirements

- **Performance** — trivial payloads; p95 < 300 ms.
- **Security** — comms/admin scope.
- **Privacy & Compliance** — test-send uses a caller-supplied address only; no bulk sends from CLI.
- **Reliability** — set/create idempotent by key/id.
- **Internationalization** — templates keyed by locale; `--locale` supported.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an HTML file, *When* `email-templates set --file`, *Then* `get` returns it.
- **AC-2.** *Given* a window, *When* `banners create`, *Then* it's active within the window.
- **AC-3.** *Given* `--locale es`, *Then* the Spanish variant is stored separately.

## 8. Data Model

- None client-side.

## 9. API Surface

- `admin-console/email-templates` CRUD/preview/test-send; `admin/banners` CRUD.

## 10. UI / UX

- `lextures email-templates ...`, `lextures banners ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server email-template + banner handlers.

## 13. Dependencies & Sequencing

- After: C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Template variables mis-rendered | M | L | `preview` renders with sample data before set |

## 15. Rollout Plan

- Ship both groups together (small surface).
- Rollback: additive.

## 16. Test Plan

- **Unit** — locale routing; time-window validation.
- **Integration** — template CRUD; banner window.
- **E2E** — set template → preview → test-send.

## 17. Documentation & Training

- "Version-control transactional emails" recipe; maintenance-banner runbook.

## 18. Open Questions

1. What is the template variable/interpolation syntax?

## 19. References

- `admin_email_templates.go`, `banner_http.go`.
- Related: [C21](C21-platform-settings.md), [C33](C33-accessibility-media-localization.md).
