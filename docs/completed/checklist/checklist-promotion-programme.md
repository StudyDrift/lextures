# Course checklist — tier promotion programme (CC.10 FR-20–FR-22)

Every rule ships as **`recommended`**. Promotion to **`essential`** is what makes the nav badge
show a non-zero count. This document is the objective gate and process.

## Promotion gates (per rule)

A rule may be promoted only when **all** of the following hold:

1. **≥ 200 courses evaluated** with the rule in the catalog (full evaluations; status samples present).
2. **Manual plausibility review** of a 20-finding sample by instructional design / product.
3. **`disagree` dismissal rate < 10%** over the review window (7–30 days of dismissals).
4. **`done_elsewhere` dismissal rate < 15%** over the same window.
5. For **accessibility rules** (`a11y.*`, `udl.*`, `links.external-health`): accessibility-owner sign-off.

Additionally for `outcomes.assessment-mapping`: the mapping assist must have shipped and
acceptance rate ≥ 40% (prefer ≥ 60%) before promotion (CC.4 / CC.10 sequencing).

## Batching (FR-21)

- At most **one promotion release every two weeks**.
- Batch size ≤ **8** rules.
- Announce in-product via the existing banner system (one show per user).
- Promotion is a **server-only** descriptor `Tier` change — no client deploy required for badge math.

## Demotion / retirement (FR-22)

| Lever | Effect | Client release? |
|---|---|---|
| Demote `essential` → `recommended` | Leaves badge within one snapshot TTL | No |
| Add to `RETIRED_ITEM_IDS` | Item disappears from API | No |
| Bump `EngineVersionConst` | Force-invalidate snapshots | No |

**Staging exercise:** demote one rule in staging, confirm badge drop after refresh/TTL, re-promote.
Record the date of the last rehearsal in the release notes.

## Tracking spreadsheet columns (suggested)

`item_id | courses_eval | disagree_rate | done_elsewhere_rate | sample_review | a11y_signoff | assist_ready | decision | release`

## GA criteria for the section (from CC.10 §15)

- All four rule packs shipped
- ≥ 20 rules promoted to `essential`
- Target resolution failure < 1%
- No open privacy findings
- Runbooks published
