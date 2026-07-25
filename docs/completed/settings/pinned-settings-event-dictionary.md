# Pinned settings — client event dictionary (PS.4)

Staff-facing product analytics only. No student data, course IDs, item IDs, item titles, setting values, or free-text search queries.

**Transport:** in-process listener bus (`clients/web/src/lib/settings-telemetry.ts`); fire-and-forget. Suppressed when `ffPinnedSettings` is off or when the user has opted out (`navigator.doNotTrack` or `localStorage['lextures.analytics.opt-out'] === '1'`).

**Retention:** inherits platform product-analytics retention (confirm with privacy before warehouse export). Open question: see PS.4 §18.

## Events

| Event | When | Fields |
|---|---|---|
| `settings_pin_added` | User pins a setting (manual or from suggestion) | `surface`, `setting_id`, `role`, `pin_count?` |
| `settings_pin_removed` | User unpins | `surface`, `setting_id`, `role`, `pin_count?` |
| `settings_pin_reordered` | Drag or keyboard reorder | `surface`, `role`, `setting_id?`, `position?`, `pin_count?` |
| `settings_pin_save_failed` | Debounced PUT fails | `surface`, `role` |
| `settings_suggestion_accepted` | User pins from the suggestion strip | `surface`, `setting_id`, `role`, `position?`, `pin_count?` |
| `settings_suggestion_dismissed` | User clicks “Not now” | `surface`, `role` |
| `settings_search_performed` | Search idle 1 s after typing | `surface`, `role`, `query_hash`, `result_count` |
| `settings_search_zero_results` | Same flush when `result_count === 0` | `surface`, `role`, `query_hash`, `result_count` |
| `settings_control_changed` | Control interaction, max 1 / setting / 2 s | `surface`, `setting_id`, `role` |

## Field definitions

| Field | Type | Notes |
|---|---|---|
| `surface` | `assignment` \| `quiz` | Required |
| `setting_id` | registry key | Canonical ID when present |
| `role` | `instructor` \| `admin` \| `other` | Coarse; derived from permission strings |
| `query_hash` | hex string | NFKC + lowercase then salted digest — **never** the raw query |
| `result_count` | number | Match count for the surface |
| `position` | number | 1-based pin position when relevant |
| `pin_count` | number | List length after the mutation |

## Query hash

- Normalisation: Unicode NFKC, lowercase, collapse whitespace.
- Salt: `VITE_SETTINGS_QUERY_HASH_SALT` when set, else build default `lextures-settings-query-v1`.
- Prefer SHA-256 (first 32 hex chars) via SubtleCrypto; FNV-1a fallback for non-browser tests.
- Pre-registered allowlist of safe setting-related terms can be hashed at build time for zero-result analysis (PS.4 open question 4).

## Schema enforcement

`validateSettingsTelemetryEvent` rejects unknown event names, missing required fields, and any attempt to pass `query` / `raw_query` / `search`. Extra keys are stripped.
