# Information Security Management System (ISMS)

Lextures maintains an ISO/IEC 27001:2022 Information Security Management System (ISMS) scoped to the **Lextures LMS SaaS service**, extended with an ISO/IEC 27701:2019 Privacy Information Management System (PIMS) for PII processing.

## Documents

| Document | Purpose |
|----------|---------|
| [scope-statement.md](./scope-statement.md) | ISMS scope boundary (FR-1) |
| [statement-of-applicability.md](./statement-of-applicability.md) | Annex A control applicability (FR-2) |
| [risk-management-policy.md](./risk-management-policy.md) | Risk assessment and treatment (FR-5) |
| [supplier-security-policy.md](./supplier-security-policy.md) | Sub-processor review process (FR-7) |
| [incident-response.md](./incident-response.md) | Links to incident workflow (AC-3; shared with SOC 2 plan 10.9) |

## Engineering evidence

Technical controls map to:

- `server/internal/authz/authz.go` — RBAC (Annex A.8.2, A.8.3)
- `server/internal/auth/jwt.go` — Authentication (Annex A.8.5)
- `iac/production/` — Configuration management (Annex A.8.9)
- `docs/SECURITY.md` — Security posture baseline

## Operational tracking

Audit findings, risk register, SoA status, supplier reviews, and training completions are tracked in the compliance admin API (`/api/v1/compliance/iso/*`) when `ISO_ISMS_ENABLED=true`.
