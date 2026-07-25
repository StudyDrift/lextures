# TD.2 — Convention Charter & Automated Enforcement

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.2 |
| **Section** | Technical Debt Remediation |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | THIN |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Platform team (Go + web leads jointly) |
| **Depends on** | — |
| **Unblocks** | TD.4, TD.6, TD.8, TD.11 |

---

## 1. Problem Statement

The repository has real linting (`golangci-lint` with `errcheck`/`govet`/`ineffassign`/`staticcheck`/`unused`; `oxlint` on the web) and the hygiene results show it works — 2 `any` casts in 1,324 TS files, 1 `TODO` in 1,709 Go files. But nothing enforces *structure*: no rule caps file or package size, none prevents the HTTP layer from reaching into SQL, none enforces the kebab-case file convention that **51 files** already violate, and none stops a seventh copy of `apiJson<T>` appearing. Consequently every debt this programme pays down can silently re-accumulate, and reviewers must police structure by hand across a 1,700-file Go tree. Without a ratchet, TD.4–TD.14 are one-time cleanups rather than durable improvements.

## 2. Goals

- Write down the target architecture as a **short, opinionated charter** engineers can actually read.
- Add **automated enforcement** for every rule in the charter — a rule with no check is a suggestion, not a convention.
- Apply enforcement as a **ratchet**: new and modified code must comply immediately; the existing backlog is burned down by later TD stories and tracked by a shrinking allowlist.
- Make violations fail fast and locally (pre-commit / `make lint`), not only in CI.
- Ensure no rule can be silently loosened — threshold changes require an explicit, reviewed file edit.

## 3. Non-Goals

- Actually fixing the existing violations — TD.4 (dead code), TD.6 (package split), TD.9 (layering), TD.11–TD.14 (web) own the burn-down.
- Reformatting the repository or changing `gofmt`/`oxlint` base configuration.
- Adopting new frameworks or libraries (TD.13 owns that decision).
- Rewriting CI infrastructure.

## 4. Personas & User Stories

- **As a new engineer**, I want one page telling me where code belongs, so that my first PR lands in the right package.
- **As a reviewer**, I want structure enforced by CI, so that I can spend review on correctness instead of on file placement.
- **As a tech lead**, I want a metric that only moves down, so that I can show debt is actually shrinking rather than being relabelled.
- **As an engineer mid-refactor**, I want the allowlist to tell me exactly what remains, so that I know when a story is finished.

## 5. Functional Requirements

- **FR-1.** The repository MUST contain `docs/ARCHITECTURE_CONVENTIONS.md` — the charter — covering: package layout, layering rules, file-size budgets, naming, error handling, and test placement, for both Go and TypeScript.
- **FR-2.** CI MUST enforce a **file-size budget**: Go non-test files ≤ 600 LOC, TS/TSX files ≤ 500 LOC, with a checked-in allowlist of current violators.
- **FR-3.** CI MUST enforce a **package-size budget**: no Go package may exceed 40 non-test files, with an allowlist (today: `internal/httpserver` at 390).
- **FR-4.** CI MUST enforce **layering**: `internal/httpserver` MUST NOT import `pgx` query APIs directly, MUST NOT contain raw SQL string literals, and MUST reach the database only through `internal/repos/*`. Current violators (58 files) go on an allowlist that TD.9 empties.
- **FR-5.** CI MUST enforce **file naming**: all new web source files kebab-case; the 51 existing violations go on an allowlist.
- **FR-6.** CI MUST fail on **`deadcode` regressions**: the count of unreachable functions may not exceed the checked-in baseline (223), which later stories lower.
- **FR-7.** Every allowlist MUST be a checked-in plain-text file, sorted, one entry per line, with a header comment naming the TD story that will empty it.
- **FR-8.** Enforcement scripts MUST run from `make lint` locally and in CI, with identical results.
- **FR-9.** Allowlists MUST be **shrink-only**: CI MUST fail if an entry is added, unless the PR is explicitly labelled with a documented override.
- **FR-10.** The charter MUST state the **prime directive** — no user-visible behaviour change during remediation — and link the TD.1 harness as the proof mechanism.
- **FR-11.** Enforcement SHOULD report, per run, the remaining count per allowlist so progress is visible in CI output.

## 6. Non-Functional Requirements

- **Performance** — all structural checks complete in < 20 s; they run on every PR.
- **Security** — checks are read-only static analysis; no network access, no code execution of the analysed tree beyond the existing toolchain.
- **Privacy & Compliance** — n/a.
- **Accessibility** — n/a (no UI); the charter itself MUST restate the WCAG 2.1 AA obligation for any UI work so it is not lost during refactors.
- **Scalability** — checks are O(files); must not degrade as the tree grows.
- **Reliability** — deterministic. No check may depend on network, clock, or machine-specific paths.
- **Observability** — CI job prints a per-rule summary table.
- **Maintainability** — enforcement scripts live in `scripts/` beside the existing `check-*.sh`/`check-*.mjs` family and follow their conventions.
- **Internationalization** — n/a.
- **Backward compatibility** — no production code changes; allowlists guarantee a green build on day one.

## 7. Acceptance Criteria

- **AC-1.** *Given* the tree as of `4f8a82b1`, *When* CI runs the new checks, *Then* the build is **green** — every existing violation is allowlisted, nothing is newly broken.
- **AC-2.** *Given* an engineer adds a 700-line Go file, *When* CI runs, *Then* the build fails naming the file and the budget.
- **AC-3.** *Given* an engineer adds raw SQL to a file in `internal/httpserver`, *When* CI runs, *Then* the build fails citing the layering rule and linking TD.9.
- **AC-4.** *Given* an engineer removes a file from an allowlist by fixing it, *When* CI runs, *Then* the build passes and the summary shows a decremented count.
- **AC-5.** *Given* an engineer appends a new entry to an allowlist, *When* CI runs, *Then* the build **fails** — allowlists shrink only.
- **AC-6.** *Given* a new web component named `MyThing.tsx`, *When* CI runs, *Then* the build fails citing the kebab-case rule.
- **AC-7.** *Given* an engineer introduces an unreachable function, *When* CI runs, *Then* the `deadcode` baseline check fails.
- **AC-8.** *Given* `make lint` is run locally, *When* it completes, *Then* the result matches CI exactly.

## 8. Data Model

No schema change. New checked-in artefacts:

```
docs/ARCHITECTURE_CONVENTIONS.md
scripts/check-file-budgets.sh
scripts/check-package-budgets.sh
scripts/check-layering.sh
scripts/check-file-naming.mjs
scripts/check-deadcode-baseline.sh
scripts/allowlists/file-size.txt
scripts/allowlists/package-size.txt
scripts/allowlists/layering.txt
scripts/allowlists/file-naming.txt
scripts/allowlists/deadcode-baseline.txt
```

## 9. API Surface

No HTTP API change.

**Developer CLI surface:**

| Command | Behaviour |
|---|---|
| `make lint` | Runs existing linters **plus** all structural checks |
| `make lint-structure` | Structural checks only (fast) |
| `make lint-structure-report` | Prints remaining-violation counts per rule |

## 10. UI / UX

No product UI. Developer experience:

1. Violation output MUST name the file, the rule, the current value, the budget, and the owning TD story — e.g.
   `server/internal/httpserver/lms_dashboard.go: 1381 LOC exceeds budget 600 (rule: file-size; owner: TD.6)`.
2. Failure output MUST tell the engineer how to proceed: fix, or (for a legitimate exception) the documented override process.
3. Charter is a single markdown page, target under 400 lines — long enough to be complete, short enough to be read.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `Makefile` — new `lint-structure*` targets, wired into `lint`.
- `.github/workflows/` — add structural-check step to existing lint job.
- `.husky/pre-commit` — add the fast structural checks (naming, file size) to the existing `lint-staged` run.
- `scripts/` — new checks alongside `check-homeschool-terminology.sh`, `check-platform-feature-toggles.mjs`, `check-entity-labels.mjs`, `check-i18n-locales.mjs`.
- `server/.golangci.yml` — consider enabling additional linters (see Open Questions).

## 13. Dependencies & Sequencing

- Must ship after: nothing. Can run in parallel with TD.1.
- Must ship before: TD.4 (needs the `deadcode` baseline), TD.6 (needs package budget), TD.8/TD.9 (need layering rule), TD.11 (needs naming/duplication rules).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Budgets set arbitrarily, causing churn or gaming (files split badly just to pass) | M | M | Derive budgets from the current distribution (p90), not from taste; charter states that splitting must follow a real seam, and reviewers enforce it |
| Allowlists become permanent, rule becomes decorative | H | M | Each allowlist header names its owning TD story; README tracks counts; FR-9 makes them shrink-only |
| Pre-commit gets slow, engineers use `--no-verify` | M | M | Only the two fastest checks run pre-commit; the rest are CI-only |
| Layering check produces false positives on legitimate cases | M | M | Start as warn-only for one week; promote to blocking after tuning |
| Charter written but never read | M | M | Link from `AGENTS.md`, `README.md`, and the PR template; keep it under 400 lines |

## 15. Rollout Plan

- **Feature flag** — none.
- **Sequencing** — (1) land charter doc; (2) land checks in warn-only mode with generated allowlists; (3) after one week of tuning, flip to blocking; (4) enable FR-9 shrink-only enforcement last.
- **Dogfood** — run against the last 20 merged PRs; measure the false-positive rate before flipping to blocking. Target < 5%.
- **GA criteria** — two weeks blocking with no override requests attributable to false positives.
- **Rollback** — set checks back to warn-only via a single flag in the Makefile; no production impact.

## 16. Test Plan

- **Unit** — each check script tested against fixture trees: a compliant file, a violating file, an allowlisted violating file, and an allowlist with an illegal addition.
- **Integration** — full `make lint` on the real tree must be green at `4f8a82b1` (AC-1).
- **End-to-end** — n/a (no runtime behaviour).
- **Security** — confirm scripts do not execute analysed code and do not reach the network.
- **Accessibility** — n/a.
- **Performance** — assert `make lint-structure` < 20 s on the full tree.
- **Manual exploratory** — an engineer unfamiliar with the charter attempts a typical feature PR and reports whether messages were actionable.

Baseline commands to generate initial allowlists:

```bash
# file-size violators
find server -name '*.go' ! -name '*_test.go' -exec wc -l {} + | awk '$1>600{print $2}' | sort
find clients/web/src -name '*.ts*' -exec wc -l {} + | awk '$1>500{print $2}' | sort
# layering violators (expect 58)
grep -rlE 'd\.Pool\.(Query|QueryRow|Exec)' server/internal/httpserver/*.go | grep -v _test | sort
# naming violators (expect 51)
find clients/web/src \( -name '*.ts' -o -name '*.tsx' \) | grep -vE '/[a-z0-9.-]+\.(ts|tsx)$' | sort
# deadcode baseline (expect 223)
cd server && deadcode ./... | sort
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` — the charter (new, primary deliverable).
- `AGENTS.md` — add structural-check commands to the commands table; link the charter.
- PR template — add a "structure" checkbox linking the charter.
- Internal runbook — how to request a documented override and who approves.

## 18. Open Questions

1. Exact budgets: 600 LOC (Go) / 500 LOC (TS) are proposed from the current p90. Confirm against the distribution before locking.
2. Should `golangci-lint` gain `revive`, `gocyclo`, or `dupl`? `gocyclo` would catch the 455-line `registerCourseRoutes`; `dupl` may be noisy on handler boilerplate until TD.7 lands.
3. Who approves overrides — any two reviewers, or a named owner per rule?
4. Should the layering rule be expressed as a `depguard` config inside `golangci-lint` instead of a bespoke script? (Likely yes for the import half; the raw-SQL half needs a custom check.)
5. Does the web side need a duplication detector (e.g. `jscpd`) to prevent a seventh `apiJson`, or does TD.11's shared module plus review suffice?

## 19. References

- `server/.golangci.yml` — current Go lint config
- `clients/web/package.json` — `oxlint`, existing `check-*.mjs` scripts
- `scripts/check-homeschool-terminology.sh` — existing enforcement-script pattern to follow
- `.husky/pre-commit` — existing hook
- Related plans: [TD.1](TD.1-refactoring-safety-net.md), [TD.4](TD.4-delete-confirmed-dead-code.md), [TD.6](TD.6-decompose-httpserver-package.md), [TD.9](TD.9-enforce-repo-layering.md)
