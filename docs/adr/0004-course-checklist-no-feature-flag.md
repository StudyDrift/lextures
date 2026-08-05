# ADR 0004 — Course Checklist: no feature flag, tiered promotion, structural levers

- **Status:** Accepted
- **Date:** 2026-08-04
- **Plan:** [CC.10 Guidance, assisted fixes, analytics & rollout](../completed/checklist/CC.10-analytics-guidance-and-rollout.md)
- **Related:** [ADR 0003](0003-course-checklist-code-registry.md)

## Context

The product decision for the course checklist is that it is **always on** for staff
(teachers-and-higher). A traditional feature flag would hide the entire surface, but the
risk we care about is not "ship the page" — it is "ship a bad rule into the badge" and
"cannot reverse without a client deploy."

## Decision

1. **No product feature flag** for the checklist surface.
2. **Safety valves are structural** (server-only where possible):
   - `RETIRED_ITEM_IDS` — remove a rule without client release
   - Tier demotion (`essential` → `recommended`) — drop from nav badge within one snapshot TTL
   - `CHECKLIST_SNAPSHOT_TTL` tuning
   - Link-check kill switch env (CC.6)
   - `EngineVersionConst` bump to invalidate warm snapshots
3. **All rules ship `recommended`.** Promotion to `essential` follows objective per-rule gates
   in [checklist-promotion-programme.md](../completed/checklist/checklist-promotion-programme.md)
   (sample size, dismissal rates, manual review, a11y sign-off).
4. **Promotions are batched** (≤ 8 rules, at most one batch / two weeks) with an in-product banner.
5. **Assisted fixes** reuse existing AI paths, honour opt-out, write only on human confirm, and
   are optional registry actions (unknown kinds render nothing).

## Consequences

- Ops can retire/demote a misbehaving rule with a server release alone — see
  [course-checklist runbook](../runbooks/course-checklist.md).
- Analytics (pass rates, dismissals, target resolution, assist acceptance) are required before
  large-scale promotion; without them the badge would encode unvalidated heuristics.
- Support answers "the checklist is wrong" with rule fix / demotion, not a flag flip.
