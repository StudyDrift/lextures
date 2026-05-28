# Vulnerability Triage Runbook

Plan reference: **10.16 — Bug Bounty / Responsible Disclosure**

## Intake

1. Reports arrive at **security@lextures.io** (encrypted preferred).
2. Auto-responder sends acknowledgment within 1 hour; human acknowledgment within **2 business days** with ticket ID.
3. Log the report in **Settings → Compliance → Security reports** (or `POST /api/v1/compliance/security-reports`) with `report_date`, summary, and optional `reporter_handle` / CVSS.

## Triage

| Step | Action |
|------|--------|
| Validate | Reproduce or confirm the issue; reject spam / out-of-scope per `SECURITY.md` |
| Severity | Assign CVSS 3.1 score → `critical` / `high` / `medium` / `low` / `informational` |
| Status | `triaging` → `accepted` (sets `triaged_at`) or `disputed` / `wont_fix` |
| Owner | Assign engineer; link internal incident if customer impact |

## Patch SLAs

| Severity | Calendar days from `report_date` |
|----------|----------------------------------|
| Critical | 7 |
| High | 30 |
| Medium | 90 |
| Low | Next release (no automated SLA flag) |

When status moves to **`patched`**, set `patch_date`. The system computes `sla_met` for critical/high/medium.

## Reporter communication

- Acknowledge receipt (2 business days max).
- Notify on severity assignment and expected fix window.
- After deploy, send patch summary and coordinate public disclosure (90-day default).

## Evidence for auditors

Export CSV: `GET /api/v1/compliance/security-reports/export` (requires `compliance:security:admin:*`).

Fields: severity, `triaged_at`, `patch_date`, `sla_met`, status.

## Escalation

- **Critical / active exploitation:** page security on-call; consider expedited disclosure timeline with reporter.
- **Legal / law enforcement:** involve Legal before responding.
