# Pinned editor settings — disable / re-enable

Plans: **PS.2** (API), **PS.3** (editor UI), **PS.4** (suggestions, telemetry, GA). Ops control for the per-user pinned assignment/quiz settings capability.

## Purpose

Turn off pinned settings API and the editor pin UI without dropping stored pin rows. Instructors see the unpinned panel layout; pins are restored when the flag is turned back on. Suggestions and client telemetry are inert when the flag is off.

## Control

| Mechanism | Location | Default |
|---|---|---|
| Platform flag `ff_pinned_settings` | `settings.platform_app_settings` / Global platform settings UI (`ffPinnedSettings`) | **true** (GA — PS.4) |

When **off**:

- `GET /api/v1/me/pinned-settings` → `404` `NOT_FOUND` — message: "Pinned settings are not enabled."
- `PUT /api/v1/me/pinned-settings/{surface}` → same `404`
- `GET /api/v1/platform/features` reports `ffPinnedSettings: false`
- Assignment/quiz editor settings panels render **without** pin toggles, Pinned group, suggestion strip, or pin API traffic
- Client pin/suggestion/search telemetry is **suppressed**
- Mid-session flip: open editors re-render without pin UI on the next features refresh; stored pin rows are **retained**
- Existing rows in `settings.user_pinned_settings` are **retained**

When **on**:

- Authenticated callers can read/write their own pin lists (max 12 keys per surface; surfaces `assignment` | `quiz`)
- Write rate limit: 60 PUTs/minute per user → `429` `RATE_LIMITED`
- Editors show pin toggles, Pinned group (when the user has pins), debounced saves (PS.3; UI cap 8)
- Users with **zero** pins who have not dismissed suggestions see the curated suggestion strip (PS.4)
- Product analytics events fire for pin, search, and control interactions (staff-facing only; see event dictionary)

### Effect of flipping off after users have pins (GA)

- Pins remain in the database; DSAR export still includes them.
- Editors look like pre-pin panels (no toggles, no Pinned group, no suggestions).
- Turning the flag back on restores each user's pin list without migration.
- No client-side localStorage cleanup is required; dismissal keys for suggestions are harmless when the flag is off.

## Procedure — disable

1. Open **Settings → Global platform** (or `PATCH` platform settings with `{ "ffPinnedSettings": false }`).
2. Confirm `GET /api/v1/platform/features` shows `ffPinnedSettings: false`.
3. Confirm `GET /api/v1/me/pinned-settings` returns 404 for a signed-in instructor.
4. Confirm assignment/quiz editor settings panels load without pin toggles, a Pinned group, or a suggestion strip.

## Procedure — re-enable

1. Set `ffPinnedSettings: true` the same way.
2. Confirm features endpoint and a `GET` of pinned-settings succeed; previously saved pins reappear.
3. Users with no pins (and who have not dismissed suggestions on that device) see the suggestion strip again.

## Metrics

| Metric | Meaning |
|---|---|
| `lextures_pinned_settings_writes_total{surface}` | Successful PUTs |
| `lextures_pinned_settings_rejects_total{reason}` | `shape` \| `too_many` \| `duplicate` \| `bad_surface` |
| `lextures_pinned_settings_pins_gauge` | Histogram of pin-list length observed on write (buckets 0–12) |

### Alert

Fire if `pinned_settings_rejects_total{reason="shape"}` is non-zero for **15 minutes** — indicates a client/server contract break (invalid key shape from a shipped client).

Example PromQL:

```promql
increase(lextures_pinned_settings_rejects_total{reason="shape"}[15m]) > 0
```

## Related

- Table: `settings.user_pinned_settings` (cascade-deleted with the user)
- DSAR export includes `pinnedSettings` with raw setting keys
- Reporting queries: [`docs/completed/settings/pinned-settings-reporting.md`](../completed/settings/pinned-settings-reporting.md)
- Event dictionary: [`docs/completed/settings/pinned-settings-event-dictionary.md`](../completed/settings/pinned-settings-event-dictionary.md)
- Plans: PS.2 (API), PS.3 (UI), PS.4 (suggestions + telemetry + GA)
