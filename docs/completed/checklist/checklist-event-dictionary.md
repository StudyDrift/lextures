# Course checklist — event dictionary (CC.10)

Staff-facing product analytics only. **No PII, no evidence content, no course codes, no user IDs.**
Payloads carry item IDs, statuses, counts, and anchor IDs only (FR-16).

**Transport (web):** in-process listener bus (`clients/web/src/lib/checklist-telemetry.ts`); fire-and-forget.
Suppressed when the user has opted out (`navigator.doNotTrack` or `localStorage['lextures.analytics.opt-out'] === '1'`).

**Server:** Prometheus counters (aggregated, no course label on the hot path).

**Retention:** client events inherit platform product-analytics retention; `course.course_checklist_events` follows the 400-day policy (CC.2).

## Client events (FR-15)

| Event | When | Fields |
|---|---|---|
| `checklist_viewed` | Checklist page load succeeds | — |
| `checklist_item_expanded` | Evidence/title expand | `itemId` |
| `checklist_evidence_clicked` | Evidence table shown | `itemId` |
| `checklist_target_navigated` | User follows a target/evidence link | `itemId`, `anchorId?`, `resolved` |
| `checklist_item_dismissed` | Dismiss succeeds | `itemId`, `reason` |
| `checklist_item_restored` | Restore succeeds | `itemId` |
| `checklist_item_rechecked` | Single-item recheck succeeds | `itemId` |
| `checklist_refreshed` | Full recheck succeeds | — |
| `checklist_assist_started` | Assisted fix begins | `itemId`, `actionKind` |
| `checklist_assist_accepted` | Proposals applied | `itemId`, `acceptedCount`, `proposedCount`, `actionKind` |
| `checklist_help_opened` | About-this-check opened | `itemId` |

### Field definitions

| Field | Type | Notes |
|---|---|---|
| `itemId` | string | Registry item ID (e.g. `syllabus.late-policy`) |
| `reason` | enum | `not_applicable` \| `done_elsewhere` \| `disagree` \| `later` \| `other` |
| `anchorId` | string | Focus anchor token when present |
| `resolved` | boolean | Whether the target navigated successfully |
| `acceptedCount` | number | Links/actions written after human confirm |
| `proposedCount` | number | Proposals returned by the assist |
| `actionKind` | string | Registry action kind |
| `status` | string | Optional evaluation status |

### Hard exclusions (AC-8)

These item IDs **must never** appear on any client event:

- `accommodations.honored`
- `accommodations.reviewed`

Schema validation rejects them (and any forbidden PII keys).

## Server metrics (FR-14)

| Metric | Labels | Notes |
|---|---|---|
| `lextures_coursechecklist_item_status_total` | `item_id`, `status` | Incremented on full evaluations; accommodations excluded |
| `lextures_coursechecklist_dismissals_total` | `reason` | Existing (CC.2) |
| `lextures_coursechecklist_snapshot_hits_total` | `result=hit\|stale\|miss` | Existing (CC.2) |
| `lextures_coursechecklist_rule_errors_total` | `item_id`, `kind` | Existing (CC.1) |

## Schema enforcement

`validateChecklistTelemetryEvent` rejects unknown event names, forbidden keys
(`courseId`, `courseCode`, `userId`, `evidence`, `note`, …), and accommodation item IDs.
