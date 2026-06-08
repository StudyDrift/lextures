# Requirements quality checklist

Run before delivering a requirements document.

## Structure

- [ ] Title uses `{Feature ID} — {Feature Name}` format
- [ ] Source line present under the title
- [ ] Metadata table complete (all 9 fields)
- [ ] Sections 1–19 present in order
- [ ] No `…` or stray `TBD` outside §18 Open Questions

## Content quality

- [ ] Problem statement is 2–4 sentences with gap, affected users, and outcome
- [ ] Goals are outcomes (3–5), not implementation tasks
- [ ] Non-goals explicitly bound scope and cross-reference deferred Feature IDs
- [ ] User stories cover all relevant personas
- [ ] FRs use MUST / SHOULD / MAY and are individually testable
- [ ] NFRs include concrete targets where applicable (latency, availability, WCAG level)
- [ ] ACs use Given/When/Then and map to FRs
- [ ] Data model includes migration path and backfill strategy if schema changes
- [ ] API surface lists auth scope for every route
- [ ] UI section covers empty, loading, error, and responsive states
- [ ] §11 present — either filled or explicitly "N/A — not AI-touching."
- [ ] Integration points cite real or proposed file paths from codebase exploration
- [ ] Dependencies use Feature ID cross-references with relative plan links
- [ ] Risks table has at least 3 rows with mitigations
- [ ] Rollout plan names a feature flag when rollout is gradual
- [ ] Test plan covers unit, integration, and e2e at minimum
- [ ] Open questions are numbered and actionable
- [ ] References link to existing code paths and related plans

## Lextures alignment

- [ ] Authz model specified for new endpoints and UI surfaces
- [ ] Student-data features note FERPA / privacy obligations
- [ ] New UI claims WCAG 2.1 AA conformance
- [ ] Migration files follow `server/migrations/NNN_*.sql` convention
- [ ] Async work specifies job queue vs synchronous path

## If saving to docs/plan

- [ ] Filename: `{section}.{number}-{kebab-slug}.md`
- [ ] Folder matches top-level section in [docs/plan/README.md](../../../../docs/plan/README.md)
- [ ] Cross-references use relative markdown links between plans
