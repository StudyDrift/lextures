# Adaptive Content Engine emergency kill-switch

Plan: **AC.1**. Ops-only control for incident response.

## Purpose

Halt all Adaptive Content Engine (ACE) **write / generate / serve** traffic without editing every course. Course instructors still own the normal per-course flag `adaptiveContentEnabled`.

## Env var

| Variable | Default | Engaged values |
|---|---|---|
| `ADAPTIVE_CONTENT_KILL_SWITCH` | *disengaged* (unset / empty / anything else) | `on`, `true`, `1`, `yes` (case-insensitive) |

When **disengaged**, the kill-switch never blocks a course. When **engaged**:

- `PUT .../adaptive-content/settings` → `503` `SERVICE_UNAVAILABLE`
- `POST` / `PATCH` / `DELETE .../adaptive-content/units*` → `503`
- `POST .../units/{id}/pre-check/generate` → `503`
- Profile *reads* (`GET .../profile`, `GET .../profiles`) still succeed; submit-hook profile writes are skipped when kill-switch is engaged
- Generate and serve endpoints (AC.3 / AC.6) honor the same gate via `adaptivecontent.ActiveForCourse` / `KillSwitchEngaged`
- `GET .../adaptive-content/settings` still returns `200` (config reads succeed)

## Metric

`lextures_adaptive_content_kill_switch_engaged` gauge: `1` engaged, `0` disengaged.

## Procedure

1. Set `ADAPTIVE_CONTENT_KILL_SWITCH=on` on all API instances and restart (or roll deploy).
2. Confirm `GET .../adaptive-content/settings` still works; a mutation returns 503.
3. Resolve the incident.
4. Unset the variable (or set to empty) and restart.

## Related

- Per-course flag: `course.courses.adaptive_content_enabled` / `PATCH .../features` field `adaptiveContentEnabled`
- Helper: `server/internal/service/adaptivecontent.ActiveForCourse(courseFlag)`
- **Generation pause (AC.4)** — distinct from this kill-switch: pauses *new generation* only; serving existing cache and base content continues.
  - Platform: `PATCH /api/v1/admin/adaptive-content` `{ "generationPaused": true }` (stores `settings.platform_app_settings.adaptive_content_generation_paused`)
  - Per-course: `PATCH .../adaptive-content/settings` `{ "generationPaused": true }`
  - Pre-warm: `POST .../adaptive-content/units/{id}/prewarm`
  - Budget: `GET .../adaptive-content/budget`
