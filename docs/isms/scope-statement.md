# ISMS Scope Statement

**Version:** 1.1  
**Effective:** 2026-05-01  
**Owner:** CTO / Compliance Lead

## In scope

The Information Security Management System (ISMS) covers the **Lextures Learning Management System (LMS) SaaS product** delivered to higher-education and K-12 customers, including:

| Component | Location |
|-----------|----------|
| Web application (SPA) | `clients/web/` |
| REST API | `server/` |
| Relational database | PostgreSQL (customer data) |
| Object storage | Course files and media |
| Production infrastructure | `iac/self-aws/`, `iac/modules/aws/` |
| Hosted homeschool app | `self.lextures.com` |

## Out of scope (v1)

- Corporate back-office systems not connected to customer data processing
- Physical office facilities (cloud-only operations; physical controls satisfied via AWS shared responsibility)
- Customer-managed identity providers (covered by integration controls, not operated by Lextures)

## Boundaries

Data flows from end users and institutional SSO IdPs into the Lextures API, persisted in PostgreSQL and object storage within AWS **us-east-1**, fronted by Cloudflare for edge security.

## Document control history

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.0 | 2026-05-01 | CTO / Compliance Lead | Initial scope statement |
| 1.1 | 2026-07-22 | Platform + Compliance | Editorial: row label → "Hosted homeschool app" (segment rename); **no scope boundary change** — host cell remains `self.lextures.com` (HS.6) |
