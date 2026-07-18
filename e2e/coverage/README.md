# Completed feature coverage (E2E.4)

Machine-readable mapping from every eligible `docs/completed/**` story to its E2E disposition.

| Artifact | Purpose |
|---|---|
| `completed-feature-manifest.json` | Reviewed source of truth (do not hand-edit generated report) |
| `REPORT.md` | Generated summary (`npm run e2e:coverage:report`) |
| `../lib/completed-feature-coverage.ts` | Schema, exclusions, validation, report helpers |
| `../scripts/coverage-check.ts` | CI gate (`npm run e2e:coverage:check`) |
| `../scripts/coverage-self-test.ts` | Fixture unit/integration tests |

## Coverage levels

| Level | Meaning |
|---|---|
| `journey` | Full user-journey Playwright coverage linked by exact spec path(s) |
| `smoke` | Partial automated coverage (not a full journey) |
| `api-contract` | API/contract assertions without a UI journey |
| `covered-by-parent` | Explicitly covered by a parent story/spec (`parentStoryId` required) |
| `manual` | Controlled manual evidence (`manualEvidence` owner + cadence) |
| `not-applicable` | Internal, mobile-only, CLI, ops, or non-journey docs |
| `missing` | Owned gap (`severity`, `owner`, `targetMilestone` required) |

## Registering a story moved into `docs/completed`

1. Add a disposition row to `completed-feature-manifest.json` (same path as the Markdown file).
2. Choose a coverage level and write a short rationale + owner.
3. For automated levels, link exact `e2e/tests/*.spec.ts` paths (titles optional).
4. For flagged product stories, fill all six `flags.*` dimensions (`true` / `false` / `"n/a"`).
5. Run `npm run e2e:coverage:check` from `e2e/` (or `make e2e-coverage-check`).

## Exclusions (FR-1)

- `README.md` section indexes
- `docs/completed/assets/**`
- Non-Markdown files

## Commands

```bash
cd e2e
npm run e2e:coverage:check    # fail on missing/broken/unknown/unowned missing
npm run e2e:coverage:report   # write REPORT.md
npm run e2e:coverage:test     # fixture self-tests (no browser)
```

Regenerate the baseline only with intent: `npm run e2e:coverage:bootstrap`.
