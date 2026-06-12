# SEC-16 — No audit trail for failed logins, authz denials, or admin mutations

- **Severity:** Medium
- **Status:** Partially present (some domain audit tables exist; security-event logging does not)
- **Area:** Server / observability
- **Files:** [server/internal/service/authservice/credentials.go](../../server/internal/service/authservice/credentials.go), [server/internal/httpserver/admin.go](../../server/internal/httpserver/admin.go), all `apierr.WriteJSON(..., http.StatusForbidden, ...)` sites

## Problem

Failed credential checks return `ErrInvalidCredentials` silently. There is no `slog.Warn` for a failed login, no record of an authz denial, and no consolidated audit log for sensitive admin mutations. Some feature areas have their own audit tables (e.g. accommodation audit), but there is no SIEM-friendly trail answering "who tried what, and when" for authentication and privilege events.

## Risk

Credential stuffing (SEC-04) and privilege probing are invisible. After an intrusion you cannot reconstruct what the attacker attempted, which accounts they targeted, or when escalation began — the forensic blind spot that turns a contained incident into an unbounded one. Detection and response are the controls that most directly limit ShinyHunters-style dwell time.

## Fix

1. Structured security logs at minimum:
   - `slog.Warn("auth.failed_login", "email", email, "ip", ip)`
   - `slog.Warn("authz.denied", "user", uid, "perm", required, "route", r.URL.Path)`
   - `slog.Info("auth.login_success", "user", uid, "ip", ip)`
2. A DB-backed `audit_events` table for sensitive mutations: admin/RBAC role changes, SCIM/OneRoster token issuance, SAML/OIDC config edits, transcript webhook config changes (SEC-09), and branding uploads (SEC-05). Capture actor, action, target, IP, and timestamp.
3. Ship these to whatever log sink production uses, and alert on bursts of `auth.failed_login` / `authz.denied`.

## Verification

- A failed login emits a structured log line with the email and source IP.
- An RBAC role change writes an `audit_events` row.
- A burst of failed logins is queryable/alertable in the log sink.
