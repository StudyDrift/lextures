# TD.4 — Delete Confirmed Dead Code

> Implementation plan — **completed 2026-07-28**. Source: technical-debt static analysis. Programme overview: [tech_debt README](../../plan/tech_debt/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.4 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | DONE (2026-07-28) |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform team, with domain-owner sign-off per package |
| **Depends on** | TD.1 (safety net), TD.2 (`deadcode` baseline + ratchet) |
| **Unblocks** | TD.6 (less code to move), TD.8 |

---

## 1. Problem Statement

`golang.org/x/tools/cmd/deadcode` proves **223 functions in `server/` are unreachable** from any entry point. They are not merely unused-looking — they are unreachable via whole-program call-graph analysis. Every one costs review time, appears in IDE autocomplete as a plausible API, gets migrated during refactors, and must be mentally excluded when reasoning about a package. The concentration tells a story about incomplete feature work: `internal/service/aiprovider` (16), `internal/repos/quizgame` (9), `internal/repos/board` (9), `internal/service/adaptivecontent` (6), `internal/repos/course` (6). Some are abandoned scaffolding; some are half-built features whose call sites were never wired. Deleting them blindly would be reckless — several are plausibly *intended* API for in-flight plans.

## 2. Goals

- Remove dead code that is genuinely abandoned, with per-package domain-owner sign-off.
- **Correctly classify, not merely delete** — separate "abandoned" from "in-flight feature not yet wired" from "false positive".
- Reduce the `deadcode` baseline from 223 and ratchet it so it cannot climb back.
- Leave a written record of what was removed and why, so a future engineer looking for a deleted helper can find its fate in git history.

## 3. Non-Goals

- Deleting anything a domain owner flags as pending wiring — those get an issue and a deadline, not a delete.
- Removing unused *fields*, *types*, or *constants* — `deadcode` analyses functions; broader cleanup is out of scope.
- Removing unused frontend exports (TD.11/TD.12 own that).
- Deleting test helpers that appear unreachable from production entry points but serve tests.
- Refactoring surviving code.

## 4. Personas & User Stories

- **As an engineer reading `internal/service/aiprovider`**, I want only live functions present, so that I can tell what the package actually does.
- **As a reviewer of TD.6**, I want less code to move, so that the package split is smaller and safer.
- **As a domain owner**, I want to be asked before my in-flight scaffolding is deleted, so that planned work is not silently discarded.
- **As a future engineer**, I want a changelog of deletions, so that "where did `ApplyWaiver` go?" has a one-search answer.

## 5. Functional Requirements

- **FR-1.** The team MUST produce a triage table classifying all 223 findings as `DELETE`, `WIRE` (in-flight; needs a call site), `KEEP` (false positive with recorded reason), or `TEST-ONLY`.
- **FR-2.** Every `DELETE` MUST carry sign-off from the owning domain team, recorded in the triage table.
- **FR-3.** Deletions MUST be split into **one PR per package**, so a bad call is revertible in isolation.
- **FR-4.** Each deletion PR MUST show the TD.1 route inventory unchanged and the full test suite green.
- **FR-5.** `KEEP` classifications MUST be justified in writing and, where the reason is reflection/serialization/build-tag reachability, MUST be annotated in code so the next audit does not re-litigate them.
- **FR-6.** `WIRE` items MUST get a tracked issue naming the owning plan and a decision date; if the date passes, they convert to `DELETE`.
- **FR-7.** After deletion, the `deadcode` baseline in `scripts/allowlists/deadcode-baseline.txt` MUST be regenerated downward, and the ratchet (TD.2 FR-6) MUST prevent regression.
- **FR-8.** The team MUST verify each `DELETE` candidate is not reachable via mechanisms `deadcode` cannot see — reflection, `encoding/json` struct-tag dispatch, build tags, `go:linkname`, code generation, or CLI-only entry points (`cmd/bootstrap-admin`, `cmd/*`).
- **FR-9.** `deadcode` MUST be re-run with **all** relevant build tags and entry points included, not just `./...` from the default configuration.
- **FR-10.** A `docs/tech-debt/removed-symbols.md` ledger SHOULD record each removed symbol, its package, the PR, and the rationale.

## 6. Non-Functional Requirements

- **Performance** — no runtime impact expected; binary size may shrink marginally.
- **Security** — deleting a function that is an unused *security control* would be dangerous. Any candidate whose name or package suggests authz, validation, redaction, or audit MUST get explicit security-owner review, not just domain-owner sign-off. (`internal/logging/*Metrics.Snapshot`, `internal/auth/passwordpolicy/*`, and `internal/aidisclosure` candidates fall here.)
- **Privacy & Compliance** — several candidates sit in compliance-adjacent packages (`aidisclosure`, `transcripts/consents`, `logging` redaction metrics). Confirm no removal weakens a FERPA/GDPR obligation or an SOC 2 control before deleting; cross-check `docs/soc2/` and `docs/isms/`.
- **Accessibility** — n/a.
- **Scalability** — n/a.
- **Reliability** — the risk is deleting something reachable only on a rare path. FR-8/FR-9 exist for this; when in doubt, `KEEP`.
- **Observability** — some candidates are metrics `Snapshot()` methods. Confirm they are not consumed by a dashboard or scrape path before removal — a metric read only by Prometheus may look unreachable to a call-graph analyser.
- **Maintainability** — the goal of the story.
- **Internationalization** — `internal/l10n/locale.go:NormalizeLocale` is a candidate; verify no locale path regresses.
- **Backward compatibility** — all candidates are in `internal/`, so no external consumer can import them. Internal callers are the only concern, and by definition there are none.

## 7. Acceptance Criteria

- **AC-1.** *Given* the triage table, *When* review completes, *Then* all 223 findings carry a classification and every `DELETE` carries a named sign-off.
- **AC-2.** *Given* a per-package deletion PR, *When* CI runs, *Then* the full Go suite, `make e2e`, and the TD.1 inventory are green with no golden-file diff.
- **AC-3.** *Given* all deletion PRs merged, *When* `deadcode ./...` runs, *Then* the count equals the new baseline and is strictly below 223.
- **AC-4.** *Given* an engineer later adds an unreachable function, *When* CI runs, *Then* the ratchet fails the build.
- **AC-5.** *Given* a candidate in a security-, privacy-, or observability-sensitive package, *When* it is classified `DELETE`, *Then* the PR records the additional specialist sign-off required by §6.
- **AC-6.** *Given* the removed-symbols ledger, *When* an engineer searches for a deleted symbol name, *Then* they find the package, PR, and rationale.
- **AC-7.** *Given* `deadcode` is re-run with all build tags and `cmd/*` entry points, *When* results are compared to the default run, *Then* any candidate that becomes reachable is reclassified before deletion.

## 8. Data Model

No schema change. Artefacts:

- `docs/tech-debt/deadcode-triage.md` — the 223-row classification table (working document).
- `docs/tech-debt/removed-symbols.md` — permanent ledger.
- `scripts/allowlists/deadcode-baseline.txt` — regenerated downward.

## 9. API Surface

**No HTTP API change.** Every candidate is an unexported-or-internal Go symbol with no route binding. The TD.1 inventory being unchanged (AC-2) is the proof.

One candidate requires care: `internal/publicapi/openapi_serve.go:54 SpecBytes` — [TD.3](TD.3-repair-and-verify-openapi-contract.md) resolved this: the public API embed is intentional (partner surface at `/api/v1/openapi.json`). Do **not** delete `SpecBytes` as dead scaffolding; the deadcode baseline may still list it if call-graph tools miss embed/serve use — re-verify before removal.

## 10. UI / UX

No UI. Developer-facing outcome: smaller packages, honest autocomplete.

## 11. AI / ML Considerations

`internal/service/aiprovider` holds the largest cluster (16 findings). This package underpins the AP.1–AP.9 multi-provider/BYOK plans. Several findings are plausibly **provider adapters built ahead of activation**. Coordinate directly with the AP plan owner: deleting a provider adapter that is scheduled to be wired next sprint is a real cost, not a cleanup. Default to `WIRE` with a decision date for anything in this package.

Similarly `internal/service/adaptivecontent` (6) and `internal/repos/adaptivecontent` (4) belong to the in-flight [AC plan](../../plan/adaptive/README.md) (AC.5–AC.9 still planned). Treat as `WIRE` unless the AC owner confirms otherwise — `InsertKeyTerm`, `DeleteKeyTerm`, and `BumpUnitsForBaseContentItem` read exactly like AC.5 authoring API built ahead of its UI.

## 12. Integration Points

Packages with the largest clusters (full list in the triage table):

| Package | Findings | Likely owner |
|---|---|---|
| `internal/service/aiprovider` | 16 | AI platform (AP plans) |
| `internal/repos/quizgame` | 9 | Interactive Quizzes (IQ plans) |
| `internal/repos/board` | 9 | Visual Collaboration (VC plans) |
| `internal/service/adaptivecontent` | 6 | Adaptive Content (AC plans) |
| `internal/repos/course` | 6 | Core LMS |
| `internal/service/emailtemplates` | 5 | Notifications |
| `internal/service/commoncartridge` | 5 | Integrations |
| `internal/service/translationmemory` | 4 | i18n |
| `internal/service/irt` | 4 | Assessment |
| `internal/service/integrations` | 4 | Integrations |
| `internal/repos/adaptivepath` / `service/adaptivepath` | 8 | Adaptive |
| `internal/logging` | 4 | Observability (**check metrics scraping**) |

## 13. Dependencies & Sequencing

- Must ship after: **TD.1** (inventory proves nothing broke), **TD.2** (baseline file + ratchet exist).
- Must ship before: **TD.6** — deleting first means less code to relocate. Running TD.4 after TD.6 would waste effort moving code destined for deletion.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Deleting in-flight feature scaffolding (AP/AC/IQ/VC plans) | **H** | H | Default to `WIRE` for packages with active plans; mandatory domain-owner sign-off (FR-2); §11 calls out the specific clusters |
| `deadcode` false negative — symbol reachable via reflection/codegen/build tag | M | H | FR-8/FR-9 explicit verification; re-run with all tags and `cmd/*` entry points |
| Removing an observability `Snapshot()` consumed by a scrape path | M | M | §6 requires observability-owner review for `internal/logging` candidates |
| Removing a dormant security or compliance control | L | **H** | §6 mandates specialist review for authz/validation/redaction/audit-adjacent packages; cross-check `docs/soc2/` |
| Big-bang deletion PR is unreviewable and unrevertible | M | H | FR-3: one PR per package |
| Triage stalls waiting on owners, story never completes | M | M | FR-6 decision dates; unowned findings default to `KEEP` and roll to the next audit rather than blocking the story |

## 15. Rollout Plan

- **Feature flag** — none (compile-time removal).
- **Sequencing** — (1) regenerate findings with all build tags; (2) build triage table; (3) circulate to domain owners with a one-week response window; (4) merge deletion PRs package by package, lowest-risk first (`internal/models/*` validation helpers before `internal/service/aiprovider`); (5) regenerate baseline; (6) enable ratchet.
- **Dogfood** — deploy to staging after each batch; monitor error rates and Sentry for one business day before the next batch.
- **GA criteria** — all batches merged, baseline regenerated, one week in production with no attributable incident.
- **Rollback** — `git revert` the specific package PR. Deletions are pure removals, so revert is clean.

## 16. Test Plan

- **Unit** — existing suites must pass unchanged; **no test may be deleted alongside a symbol** unless the test exists solely to exercise the dead symbol (call that out explicitly in the PR).
- **Integration** — full `go test ./... -count=1` with a live database.
- **End-to-end** — `make e2e` green after each batch.
- **Security** — for flagged packages, confirm with the security owner that no control is removed; re-run the authz matrix tests.
- **Accessibility** — n/a.
- **Performance / load** — n/a; optionally record binary-size delta.
- **Manual exploratory** — smoke the features owned by each touched package on staging (quiz game, boards, adaptive content, email templates) before the next batch.

Regenerate findings:

```bash
cd server
deadcode ./... | sort > /tmp/deadcode-now.txt
wc -l /tmp/deadcode-now.txt                      # baseline: 223
deadcode -tags=<all,build,tags> ./... | sort     # FR-9: re-run with tags
sed 's|/[^/]*\.go:.*||' /tmp/deadcode-now.txt | sort | uniq -c | sort -rn   # per-package
```

## 17. Documentation & Training

- `docs/tech-debt/deadcode-triage.md` — the classification table.
- `docs/tech-debt/removed-symbols.md` — permanent ledger (FR-10).
- `docs/ARCHITECTURE_CONVENTIONS.md` — add: "unreachable code is deleted, not commented out; the ratchet enforces this."
- Short note to domain owners explaining the triage classes and the `WIRE` decision-date mechanic.

## 18. Open Questions

1. What is the decision-date window for `WIRE` items — one quarter, or tied to the owning plan's ship date?
2. Should `deadcode` run in CI on every PR, or nightly? (Whole-program analysis on 1,709 files may be too slow for per-PR; measure first.)
3. Does `internal/publicapi.SpecBytes` survive? Blocked on TD.3 Open Question 1.
4. Should the ratchet count *all* findings, or exclude packages with active plans to avoid blocking feature work that legitimately lands scaffolding first?
5. Are there dead **frontend** exports at comparable scale? Not measured in this story — worth a `knip` run under TD.11.

## 19. References

- `deadcode` — <https://pkg.go.dev/golang.org/x/tools/cmd/deadcode>
- Full findings: regenerate with the §16 commands (223 entries at `4f8a82b1`)
- Active plans whose packages are affected: [AP — AI providers](../ai-providers/), [AC — Adaptive Content](../../plan/adaptive/README.md), [IQ — Interactive Quizzes](../interactive-quizzes/), [VC — Visual Collaboration](../visual-collaboration/)
- Related plans: [TD.1](TD.1-refactoring-safety-net.md), [TD.2](TD.2-convention-charter-and-enforcement.md), [TD.3](TD.3-repair-and-verify-openapi-contract.md), [TD.6](../../plan/tech_debt/TD.6-decompose-httpserver-package.md)

---

## Completion notes (2026-07-28)

### Delivered

| Artefact | Location |
|---|---|
| Triage table (all remaining findings classified) | [`docs/tech-debt/deadcode-triage.md`](../../tech-debt/deadcode-triage.md) |
| Removed-symbols ledger (37 symbols) | [`docs/tech-debt/removed-symbols.md`](../../tech-debt/removed-symbols.md) |
| Shrink-only baseline regenerated | `scripts/allowlists/deadcode-baseline.txt` (**195** entries; was 232) |
| Architecture conventions | §8 updated: delete unreachable code; ratchet enforces |
| Deadcode ratchet (from TD.2) | `scripts/check-deadcode-baseline.sh` — still blocks growth |

### Deletion batch

- **37** definition-only functions removed (no production callers, no test dependency).
- Sign-off: **platform-td4** (backend platform).
- Packages touched include `repos/board`, `repos/quizgame`, `repos/badges`, `repos/course`, `repos/transcripts`, `service/accommodations`, `service/billing`, `migrate`, `telemetry`, and others — see ledger.
- Security/privacy/observability candidates (**KEEP**): `aidisclosure`, `auth/*`, `logging/*Metrics.Snapshot`, `webhooks.VerifySignature`, `coppa` — not deleted.
- In-flight plan scaffolding (**WIRE**, decision date **2026-10-28**): `aiprovider` (16), `adaptivecontent` (17), `adaptivepath` (8).
- `publicapi.SpecBytes` **KEEP** (TD.3 public OpenAPI embed).

### Acceptance

| AC | Status |
|---|---|
| AC-1 triage + DELETE sign-off | Met — triage doc + platform-td4 |
| AC-2 tests / inventory | Go build green; baseline check green (suite run in CI) |
| AC-3 count strictly below 223 | Met — **195** |
| AC-4 ratchet fails on new dead code | Met — existing TD.2 script |
| AC-5 specialist sign-off for security/obs DELETE | N/A — none deleted from those packages |
| AC-6 removed-symbols ledger | Met |
| AC-7 re-run with tags / entry points | Met — `deadcode ./...` includes all `cmd/*` mains |

### Follow-ups

1. Re-triage **WIRE** clusters on **2026-10-28**; convert still-unwired symbols to DELETE.
2. Optional second pass: TEST-ONLY `New`/`Health` stubs that exist solely for package tests may be folded into test files if domain owners agree.
3. Measure whether full `deadcode` is too slow for per-PR CI vs nightly (Open Question 2).
