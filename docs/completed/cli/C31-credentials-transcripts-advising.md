# C31 — Credentials, transcripts, advising & degree progress

> CLI parity plan. Source: `credentials_http.go` (`credentials`, 7), `registerTranscriptsRoutes` (`transcripts`, `admin/transcripts`), `registerCCRRoutes` (`me/ccr`), `advising_http.go` (`advisor`, `admin/advising`), `me/degree-progress`, `/api/v1/verify`, `registerOnboardingGoalsRoutes`. Baseline: `clients/cli/cmd/credentials_transcripts_advising.go`, `credentials_transcripts_advising_logic.go`, `credentials_transcripts_advising_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C31 |
| **Section** | Academic records |
| **Severity** | MAJOR |
| **Markets** | HE (primary) / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Registrar / CLI |
| **Depends on** | C06, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Credentials (badges/certificates), transcripts, comprehensive learner records (CCR), advising notes and degree-progress audits are UI-only. Registrars cannot batch-issue credentials, export official transcripts, or run degree-audit reports for a cohort via automation.

## 2. Goals

- Batch-issue and verify credentials/badges.
- Generate and export transcripts (and CCR) per student/cohort.
- Pull degree-progress/advising data for reporting and advisor tooling.

## 3. Non-Goals

- Designing credential templates in a WYSIWYG editor.
- Advising case-management workflow beyond notes read/append.

## 4. Personas & User Stories

- **As a registrar**, I want `credentials issue --file recipients.csv --template T` to batch-award.
- **As a registrar**, I want `transcripts export --user U --format pdf` for an official transcript.
- **As an advisor**, I want `degree-progress get --user U` and `advising notes list --user U`.
- **As a verifier**, I want `credentials verify <code>` to check authenticity.

## 5. Functional Requirements

- **FR-1.** MUST add `credentials list|issue|revoke|verify` (`credentials_http.go`; batch `--file`, public `/verify`).
- **FR-2.** MUST add `transcripts get|export <user>` (`--format pdf|json`) and `transcripts batch --section S` (`registerTranscriptsRoutes`).
- **FR-3.** MUST add `ccr export --user <u>` (comprehensive learner record).
- **FR-4.** SHOULD add `degree-progress get --user <u>` and `advising notes list|add --user <u>` (`advising_http.go`).
- **FR-5.** MAY add `onboarding-goals get|set` (`registerOnboardingGoalsRoutes`).

## 6. Non-Functional Requirements

- **Performance** — batch issue/transcript export async; `--wait` for large cohorts.
- **Security** — registrar/advisor scope; credential signing keys server-side only.
- **Privacy & Compliance** — transcripts/CCR are FERPA official records; export gated by `--yes`; verify endpoint is public but returns minimal data.
- **Reliability** — issue idempotent by (recipient, template); re-issue detected.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a recipients CSV, *When* `credentials issue --file`, *Then* credentials mint with a summary; re-run issues 0 dups.
- **AC-2.** *Given* a user, *When* `transcripts export --format pdf`, *Then* a PDF transcript is written.
- **AC-3.** *Given* a credential code, *When* `credentials verify`, *Then* validity + minimal metadata print.

## 8. Data Model

- None client-side. Document recipients CSV schema.

## 9. API Surface

- `credentials_http.go` (issue/revoke/verify); `registerTranscriptsRoutes`; `registerCCRRoutes`; `advising_http.go`; `me/degree-progress`; `/api/v1/verify`.

## 10. UI / UX

- `lextures credentials ...`, `lextures transcripts ...`, `lextures advising ...`, `lextures degree-progress ...`.

## 11. AI / ML Considerations

- Advising may include AI recommendations server-side; CLI reads them. No CLI model calls.

## 12. Integration Points

- Server credential/transcript/advising handlers; final grades (C06); open-badge/Comprehensive Learner Record standards.

## 13. Dependencies & Sequencing

- After: C06 (grades → transcript), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Transcript is an official/legal doc | M | H | Server generates + signs; CLI only requests/downloads |
| Batch double-issue | M | M | Idempotency key; `--skip-existing` |

## 15. Rollout Plan

- Ship credentials issue/verify + transcript export first, then CCR/advising/degree-progress.
- Rollback: additive.

## 16. Test Plan

- **Unit** — recipients CSV parse; idempotency.
- **Integration** — issue/verify; transcript export.
- **E2E** — batch-issue → verify a code.

## 17. Documentation & Training

- "Batch-issue completion certificates" recipe.

## 18. Open Questions

1. Which credential standard (Open Badges v2/v3, CLR)?
2. Is transcript export synchronous or job-backed?

## 19. References

- `credentials_http.go`, `registerTranscriptsRoutes`, `advising_http.go`.
- Related: [C06](C06-gradebook-final-grades.md), [C07](C07-outcomes-standards-sbg.md).
