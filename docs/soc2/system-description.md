# Lextures — SOC 2 System Description

> SSAE 18 / AT-C Section 320 System Description. Covers the Security, Availability, and Privacy Trust Services Criteria (TSC) for the Lextures SaaS platform.

## 1. Service Overview

Lextures is a cloud-native learning management system (LMS) delivered as a multi-tenant SaaS platform. It enables higher-education institutions (universities, colleges) and K-12 districts to deliver asynchronous and synchronous courses, manage student data, and integrate with institutional identity providers.

## 2. Infrastructure Components

| Component | Technology | Location |
|-----------|-----------|----------|
| Application servers | Go 1.25, Docker | AWS (us-east-1 primary) |
| Database | PostgreSQL 16 (RDS) | AWS (us-east-1, Multi-AZ) |
| Object storage | AWS S3 | AWS (us-east-1) |
| CDN | AWS CloudFront | Global edge |
| AI routing | OpenRouter API | Sub-processor (see §7) |
| Email delivery | AWS SES | AWS (us-east-1) |

## 3. Trust Services Criteria in Scope

**Phase 1 (this report):** Security (CC1–CC9), Availability (A1), Privacy (P1–P8).

**Phase 2 (future):** Confidentiality, Processing Integrity.

## 4. Control Environment (CC1)

- Board-level oversight: CTO owns the security program; reports quarterly to executive team.
- Code of conduct and acceptable-use policies required for all staff.
- Annual security awareness training required for all personnel with system access.
- Background checks conducted for all employees with production access.

## 5. Communication (CC2)

- Security policy library maintained in `docs/soc2/controls/`.
- Internal: Slack #security-alerts channel for security events.
- External: security advisories published to [trust center]; critical vulnerabilities reported within 72 hours per responsible disclosure policy.

## 6. Risk Assessment (CC3)

- Annual risk assessment identifies threats to the Security, Availability, and Privacy TSC.
- Risk register maintained in the compliance admin portal.
- Material risks reviewed quarterly; mitigation owners assigned.

## 7. Monitoring (CC7)

- Application and infrastructure logs aggregated and retained for 180 days (FR-5).
- Authentication events, privilege changes, and configuration changes flow through `admin_audit_log` (plan 10.11).
- Automated vulnerability scans run weekly; results stored in evidence bucket.
- Security incident response plan tested via annual tabletop exercise (AC-3).

## 8. Change Management (CC8)

- All production code and IaC changes require a reviewed pull request (FR-3, AC-1).
- Direct commits to `main` are rejected by GitHub branch protection rules.
- Deployments are gated by passing CI checks (unit, integration, e2e, lint, SAST).
- Emergency change process: break-glass access requires manager approval and is logged.

## 9. Logical and Physical Access (CC6)

- Access to production systems requires MFA.
- Privileged access reviewed quarterly; all production access reviewed semi-annually (FR-2, AC-2).
- Role-based access control (RBAC) enforced at the application layer (`user.app_roles`, `user.permissions`).
- Access reviews recorded in `compliance.access_reviews`.
- Physical access to AWS data centers managed by AWS (shared responsibility).

## 10. Vendor Risk Management (CC9)

- Sub-processors reviewed annually against SOC 2 or equivalent security reports (FR-6, AC-6).
- Vendor risk register maintained in `compliance.vendor_risk`.
- Critical vendors: OpenRouter (AI model routing), AWS (infrastructure, storage, email).

## 11. Availability (A1)

- Monthly availability target: 99.9% uptime.
- Status page at status.lextures.com.
- Database: Multi-AZ RDS with automated failover (RTO ≤ 15 min, RPO ≤ 5 min per plan 10.15).
- Backups: automated daily snapshots retained 30 days; tested quarterly.

## 12. Privacy (P1–P8)

- Privacy notice published at lextures.com/privacy.
- Student data processed under FERPA (plan 10.1), COPPA (plan 10.2), CCPA/CPRA (plan 10.4), state laws (plan 10.6).
- Data subject rights requests handled within regulatory deadlines.
- PII redacted from logs per plan 10.14.
- Data retention and deletion schedule documented in privacy notice.

## 13. Complementary User Entity Controls (CUECs)

Customer institutions are responsible for:
- Managing their own identity provider (IdP) and SSO configuration.
- Assigning appropriate roles to their users within Lextures.
- Conducting their own access reviews for institution-managed accounts.
- Reporting suspected security incidents to Lextures within 24 hours.
