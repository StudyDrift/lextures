# HS.6 — Docs, ISMS scope & e2e coverage metadata

> Implementation plan. Source: product rebrand of the **self-learner** segment to **Homeschool**.
> Closing plan for the HS series — it updates the remaining non-product references and flips the
> [HS.1](HS.1-terminology-copy-deck-and-guardrails.md) terminology guard from warn to fail.
> Code references: `docs/plan/{README.md,_TEMPLATE.md}`, `docs/isms/scope-statement.md`,
> `e2e/scripts/bootstrap-completed-coverage-manifest.ts`, `e2e/coverage/**`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.6 |
| **Section** | Documentation, compliance & test metadata |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — the planning template, the plan index, the ISMS scope statement, and the e2e coverage owner/market metadata all still use "Self-Learner" and the `SL` market token |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Platform + Compliance |
| **Depends on** | HS.1, HS.2, HS.3, HS.4, HS.5 (all must land first, or the guard flip breaks `main`) |
| **Unblocks** | — (closes the series) |

---

## 1. Problem Statement

After the four product surfaces are rebranded, the term survives in the places that describe the
product rather than being the product: the plan template's `Markets` row still offers `SL`, the plan
index still lists "15 — Self-Learner Specific" as a live section, the ISMS scope statement names the
"Hosted self-learner app", and the e2e coverage manifest attributes a whole feature section to
"Self-Learner Product" with market `SL`. Left alone, these seed the old term into every new plan and
every regenerated coverage report — the rename would decay within a quarter. This plan cleans them
up, decides explicitly what stays frozen (the shipped `docs/completed/15-self-learner-specific/`
archive), and turns on the guard that keeps the term from coming back.

## 2. Goals

- Retire the `SL` market token in favour of `HS` in the plan template, index, and e2e metadata.
- Update the ISMS scope statement's **label** while leaving the host `self.lextures.com` untouched.
- Rename the e2e coverage owner `Self-Learner Product` → `Homeschool Product` and regenerate the
  derived coverage artefacts.
- Freeze — and explicitly document — the shipped archive folder rather than renaming it.
- Flip `scripts/check-homeschool-terminology.sh` from `--warn` to failing, and prune the allowlist to
  only the entries that are genuinely permanent.

## 3. Non-Goals

- **No rename of `docs/completed/15-self-learner-specific/`.** It has 51 inbound references
  (plan cross-links, the coverage manifest, `docs/plan/README.md`) and its name encodes historical
  plan IDs `15.1`–`15.13` that shipped under that number. Renaming rewrites history for no user
  benefit.
- No rewriting of prose inside `docs/completed/**` — those are records of what was planned at the
  time.
- No change to `self.lextures.com` in the ISMS host column, the deploy workflow, or any DNS/TLS
  record.
- No renumbering of plan IDs and no change to the `_TEMPLATE.md` structure beyond the two lines named
  in §5.
- No new compliance control, no ISMS re-certification event — this is a label edit within existing
  scope.

## 4. Personas & User Stories

- **As an engineer writing the next plan**, I want `_TEMPLATE.md` to offer `HS`, so I do not
  reintroduce `SL` by copy-paste.
- **As a compliance reviewer**, I want the ISMS scope statement to name the hosted app the way the
  product does, while the in-scope host is provably unchanged.
- **As a QA owner**, I want coverage reports attributed to "Homeschool Product" so ownership routing
  matches the org.
- **As anyone reading `docs/completed/15-self-learner-specific/`**, I want a one-line note telling me
  the segment was renamed, so I do not think I have found a live inconsistency.
- **As a reviewer**, I want CI to reject a reintroduced "self-learner" instead of relying on me to
  catch it.

## 5. Functional Requirements

- **FR-1.** `docs/plan/_TEMPLATE.md:12` MUST read `| **Markets** | K12 / HE / HS |`.
- **FR-2.** `docs/plan/_TEMPLATE.md:36` MUST replace "and self-learner perspectives" with "and
  homeschool perspectives".
- **FR-3.** `docs/plan/README.md:34` MUST relabel the section link to
  `15 — Homeschool Specific (formerly Self-Learner)` while keeping both hrefs pointing at the
  unrenamed `15-self-learner-specific/` paths.
- **FR-4.** `docs/plan/README.md:55` MUST say "Homeschool" in the market list, and line 63's
  `SL · HE (CE)` MUST become `HS · HE (CE)`.
- **FR-5.** `docs/plan/README.md` MUST gain an `HS — Homeschool rebrand` row in the Sections list,
  linking to [`homeschool/`](README.md).
- **FR-6.** `docs/isms/scope-statement.md:18` MUST read `| Hosted homeschool app | self.lextures.com |`
  — label changed, host **unchanged**. The PR description MUST state that the technical scope
  boundary did not move.
- **FR-7.** `e2e/scripts/bootstrap-completed-coverage-manifest.ts:291-296` MUST keep the map key
  `'15-self-learner-specific'` (it is derived from the on-disk path) and MUST change
  `owner: 'Self-Learner Product'` → `'Homeschool Product'` and `markets: ['SL']` → `['HS']`.
- **FR-8.** `e2e/coverage/completed-feature-manifest.json` and `e2e/coverage/REPORT.md` MUST be
  regenerated via `npm run e2e:coverage:bootstrap` in `e2e/`, not hand-edited; the `path` fields will
  still contain `15-self-learner-specific` and that is correct.
- **FR-9.** A new `docs/completed/15-self-learner-specific/README.md` MUST state: the folder name is
  frozen for historical plan IDs; the segment is now called Homeschool; link to
  `docs/brand/homeschool-terminology.md` (created by [HS.1](HS.1-terminology-copy-deck-and-guardrails.md)) and to this
  plan folder.
- **FR-10.** `scripts/check-homeschool-terminology.sh` MUST run without `--warn` in
  `.github/workflows/ci.yml`, and the step MUST be required for merge.
- **FR-11.** The allowlist MUST be pruned to exactly the permanent entries, each with a `#` rationale:
  `docs/completed/**`, `server/migrations/[0-9]*_*.sql` (applied, immutable),
  the two `EnvironmentStore` persisted raw values, the `'self-learner'` literal in
  `onboarding_http.go` and migration 433, `www/dist/self-learner/**` (the redirect stub),
  `e2e/coverage/**` (generated), and `docs/brand/homeschool-terminology.md` (defines the banned
  terms).
- **FR-12.** Any allowlist entry not on the FR-11 list MUST be removed — if it still has hits, the
  owning plan is not finished.

## 6. Non-Functional Requirements

- **Performance** — the guard step MUST stay under 5 s (HS.1 §6).
- **Security** — none.
- **Privacy & Compliance** — the ISMS edit is a documentation change inside existing scope. It MUST
  be recorded in the ISMS document-control history (author, date, "editorial: segment renamed; no
  scope boundary change") so an auditor can see the host did not move. If the scope statement is
  covered by an approval workflow, route it through that workflow rather than merging directly.
- **Accessibility** — n/a.
- **Scalability** — n/a.
- **Reliability** — the guard is the last line of defence, not the only one; each product plan keeps
  its own assertions.
- **Observability** — CI step named `Terminology guard`, failing loudly with `file:line`.
- **Maintainability** — allowlist entries without a rationale comment fail review (HS.1 FR-6).
- **Internationalization** — n/a (English-only docs).
- **Backward compatibility** — every link into `docs/completed/15-self-learner-specific/**` keeps
  resolving; no path moves.

## 7. Acceptance Criteria

- **AC-1.** *Given* `docs/plan/_TEMPLATE.md`, *Then* the Markets row offers `K12 / HE / HS` and no
  `SL` token remains in the file.
- **AC-2.** *Given* `docs/plan/README.md`, *Then* it links to `homeschool/`, uses `HS` in the market
  column, and every existing link to `15-self-learner-specific/` still resolves.
- **AC-3.** *Given* `docs/isms/scope-statement.md`, *Then* the row label reads "Hosted homeschool app"
  and the host cell is byte-identical to before (`self.lextures.com`).
- **AC-4.** *Given* `npm run e2e:coverage:bootstrap` in `e2e/`, *Then* the regenerated
  `completed-feature-manifest.json` shows `"owner": "Homeschool Product"` and `"markets": ["HS"]` for
  every `15-*` entry, and the diff contains no unrelated churn.
- **AC-5.** *Given* `docs/completed/15-self-learner-specific/README.md`, *Then* it exists and explains
  the freeze.
- **AC-6.** *Given* CI on a branch that adds `selfLearner` to `clients/web/src/`, *Then* the
  `Terminology guard` step fails.
- **AC-7.** *Given* CI on `main` after this plan merges, *Then* the guard passes with the pruned
  allowlist.
- **AC-8.** *Given* a link checker over `docs/`, *Then* there are no broken relative links.

## 8. Data Model

None.

## 9. API Surface

None.

## 10. UI / UX

No product UI. The only reader-facing change is documentation structure:

- `docs/plan/README.md` gains an `HS` row and relabels section 15.
- `docs/completed/15-self-learner-specific/README.md` is new.
- `docs/isms/scope-statement.md` relabels one table row.

Copy comes from the [HS.1 deck](HS.1-terminology-copy-deck-and-guardrails.md#10-ui--ux).

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- Internal: `docs/plan/`, `docs/completed/`, `docs/isms/`, `docs/brand/`, `e2e/scripts/`,
  `e2e/coverage/`, `.github/workflows/ci.yml`.
- External: whatever ISMS document-control process governs `docs/isms/**` (see §6).
- Emissions: none.

## 13. Dependencies & Sequencing

- Must ship **after** HS.1–HS.5, all of them. FR-10 turns the guard into a merge gate; flipping it
  while any surface still carries the old term red-lights `main`.
- Must ship before: nothing.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Guard flipped before a product plan lands → `main` is red | M | M | §13 ordering; the flip PR runs the guard in fail mode on its own branch first and must be green before merge |
| ISMS edit read as a scope change by an auditor | L | **H** | FR-6 + §6: label-only, host cell unchanged, recorded in document-control history with an explicit "no scope boundary change" note |
| Coverage manifest hand-edited instead of regenerated → drifts from the bootstrap script | M | L | FR-8 + AC-4 (diff must contain no unrelated churn) |
| Someone later renames the frozen archive folder to satisfy a future cleanup | M | M | FR-9 README states the freeze in the folder itself, where a would-be renamer will see it |
| Allowlist becomes a dumping ground and the guard stops meaning anything | M | M | FR-11 fixes the exact permitted set; FR-12 removes everything else; each entry needs a rationale comment |

## 15. Rollout Plan

- Feature flag: none.
- Sequencing: confirm HS.1–HS.5 are all merged → run the guard in fail mode locally → docs + e2e
  metadata edits → regenerate coverage artefacts → flip the CI step → merge.
- Dogfood: run the guard against the branch before opening the PR; paste the "0 findings" output in
  the description next to HS.1's baseline count.
- GA criteria: AC-1…AC-8 green; the HS series is then complete and the folder can move to
  `docs/completed/homeschool/`.
- Rollback: revert the CI step edit (one line) to return the guard to warn mode; the doc edits are
  inert and can stay.

## 16. Test Plan

- **Unit** — HS.1's script fixture tests still pass with the pruned allowlist.
- **Integration** — `npm run e2e:coverage:bootstrap` regenerates cleanly; a second run is a no-op
  (idempotent).
- **End-to-end** — n/a.
- **Security** — n/a.
- **Accessibility** — n/a.
- **Performance / load** — guard step under 5 s (timed in CI).
- **Manual exploratory** — markdown link check across `docs/`; read the plan index and confirm the
  `15-*` links still open; compliance reviewer reads the ISMS diff.

## 17. Documentation & Training

- End-user docs: none.
- Admin/instructor docs: none.
- API reference: none.
- Internal runbook: record in the ISMS document-control history (§6); add the `Terminology guard`
  row to the `AGENTS.md` commands table if HS.1 did not; note in the plan-authoring conventions that
  `HS` replaces `SL`.

## 18. Open Questions

1. Does `docs/isms/scope-statement.md` require formal re-approval for an editorial change, or is a
   document-control history entry sufficient? Compliance owner's call — this is the only gate in the
   plan that is not purely mechanical.
2. Should `docs/plan/README.md` keep listing section 15 as active at all, given every `15.x` plan
   except `15.13` is in `docs/completed/`?
3. Should the HS folder move to `docs/completed/homeschool/` as part of this PR or as a follow-up
   once all six plans are verified in production?
4. Is there any external artefact (investor deck, App Store listing, business plan under
   `docs/marketing/`) that still says "self-learner" and needs an owner? `business-plan.docx` was not
   inspected — it is a binary and outside the guard's reach.
5. Do other repos, dashboards, or BI tools reference the `SL` market token or the
   `Self-Learner Product` owner string?

## 19. References

- Existing files this work touches: `docs/plan/_TEMPLATE.md`, `docs/plan/README.md`,
  `docs/isms/scope-statement.md`, `docs/completed/15-self-learner-specific/README.md` (new),
  `e2e/scripts/bootstrap-completed-coverage-manifest.ts`,
  `e2e/coverage/{completed-feature-manifest.json,REPORT.md}` (generated),
  `.github/workflows/ci.yml`, `scripts/homeschool-terminology-allow.txt`.
- External standards: ISO/IEC 27001 A.5.1 (documented information / document control) for the ISMS
  scope edit.
- Related plans: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md),
  [HS.2](HS.2-www-marketing-site-rebrand.md), [HS.3](HS.3-web-client-rebrand.md),
  [HS.4](HS.4-mobile-clients-rebrand.md), [HS.5](HS.5-server-copy-and-onboarding-program.md).
