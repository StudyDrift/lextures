# Runbook — Adaptive Content Governance (AC.8)

## Oversight console

Admin Settings → AI governance → **Adaptive content oversight**.

Shows kill-switch, org enable/disable, queue, open contests, disparity flags, quarantined/regressing units, gate blocks, 30-day cost, and links to DPIA / EU AI Act checklist.

## Incident response

1. **Quarantine a unit** (serving → base immediately):

   `POST /api/v1/admin/adaptive-content/quarantine` `{ "unitId": "…", "reason": "…" }`

2. **Quarantine a course**:

   `POST /api/v1/admin/adaptive-content/quarantine` `{ "courseId": "…", "reason": "…" }`

3. **Engage kill-switch** (generation + serving halt):

   `POST /api/v1/admin/adaptive-content/kill-switch` `{ "engage": true }`  
   (also `ADAPTIVE_CONTENT_KILL_SWITCH=true` env)

4. If PII may have been exposed in generated content, follow the platform breach process (S03).

## Fairness audit

- Scheduled daily (`scheduled.adaptive_content_fairness`).
- On-demand: `POST /api/v1/admin/adaptive-content/fairness/refresh?course=<code|id>`
- Read cells: `GET /api/v1/admin/adaptive-content/fairness?course=…`
- Small cells (n < 5) suppress means; disparity flags require n ≥ 10 and material gap vs cohort mean.

## Contests

Students use **Report this adaptation** on the Adapted banner. Instructors resolve via Adaptive content settings → Contests tab.
