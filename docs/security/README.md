# Lextures Security Audit — Findings Index

_Audit date: 2026-06-12. Scope: Go API server, web SPA, iOS/Android/CLI clients, marketing site, and infrastructure (Terraform, Docker Compose, CI)._

This audit was triggered by the ShinyHunters / Salesloft-Drift breach pattern: **stolen long-lived tokens replayed out-of-band to pivot through a SaaS tenant.** The themes that matter most for that threat model are token-theft surfaces, stored XSS in shared content, integration secrets, self-elevating accounts, and anything reachable without a session.

Each finding below has its own file with a full problem statement, exact file/line references, risk, a concrete fix, and a verification step for the development team. Work the **Critical** and **High** items before the next production deploy.

## How to use this folder

- One file per finding, named `SEC-NN-short-slug.md`.
- Severity reflects exploitability × blast radius in *this* codebase, not a generic CVSS.
- "Status: confirmed present" means the issue was verified against the working tree on the audit date.

## Findings

| ID | Severity | Title | Area |
|----|----------|-------|------|
| [SEC-01](SEC-01-default-jwt-secret.md) | Critical | Default `JWT_SECRET` committed in `docker-compose.yml` | Server / secrets |
| [SEC-02](SEC-02-tokens-in-localstorage.md) | High | Access + refresh tokens stored in `localStorage` | Web client |
| [SEC-03](SEC-03-cors-and-security-headers.md) | High | CORS `*` and no security headers (server + nginx) | Server / infra |
| [SEC-04](SEC-04-auth-rate-limiting.md) | High | No rate limiting on password / session endpoints | Server / auth |
| [SEC-05](SEC-05-svg-upload-xss.md) | High | SVG branding upload served same-origin → stored XSS | Server / uploads |
| [SEC-06](SEC-06-course-file-mime.md) | High | Course-file content served with DB-controlled MIME, no `nosniff` | Server / uploads |
| [SEC-07](SEC-07-request-limits-timeouts.md) | High | Unbounded request bodies + missing server timeouts | Server |
| [SEC-08](SEC-08-authz-wildcard.md) | High | Permission wildcard matches the *required* side | Server / authz |
| [SEC-09](SEC-09-transcript-webhook-ssrf.md) | High | Transcript webhook is an unrestricted SSRF primitive | Server / transcripts |
| [SEC-10](SEC-10-course-code-path-traversal.md) | Medium | `course_code` flows into filesystem path with weak sanitization | Server / files |
| [SEC-11](SEC-11-saml-tokens-in-fragment.md) | Medium | Tokens delivered via URL fragment in SAML callback | Server / SSO |
| [SEC-12](SEC-12-open-redirect.md) | Medium | Open redirect via protocol-relative `next` / `RelayState` | Server + web |
| [SEC-13](SEC-13-jwt-hardening.md) | Medium | JWT: HS256 single key, no `kid`/`iss`/`aud`, no rotation | Server / auth |
| [SEC-14](SEC-14-argon2-parameters.md) | Medium | Argon2id parameters below current OWASP guidance | Server / auth |
| [SEC-15](SEC-15-originality-webhook-replay.md) | Medium | No replay protection on originality webhook | Server / webhooks |
| [SEC-16](SEC-16-audit-logging.md) | Medium | No audit trail for failed logins / authz denials / admin mutations | Server |
| [SEC-17](SEC-17-bearer-token-hashing.md) | Medium | SCIM / OneRoster bearer tokens stored as unsalted SHA-256 | Server / provisioning |
| [SEC-18](SEC-18-katex-dompurify.md) | Medium | KaTeX HTML rendered via `dangerouslySetInnerHTML` without sanitization | Web client |
| [SEC-19](SEC-19-dev-db-credentials.md) | Medium | Hardcoded dev DB credentials + host-exposed Postgres port | Infra |
| [SEC-21](SEC-21-hardcoded-debug-path.md) | Low | Hardcoded developer-machine debug path in shipped binary | Server |
| [SEC-22](SEC-22-sso-jit-teacher-role.md) | Low | SSO JIT provisioning trusts self-asserted Teacher role | Server / SSO |
| [SEC-23](SEC-23-login-timing-oracle.md) | Low | Login timing oracle enables user enumeration | Server / auth |
| [SEC-24](SEC-24-ios-keychain-accessibility.md) | Low | iOS Keychain items use device-backup-eligible accessibility | iOS client |
| [SEC-25](SEC-25-magic-link-ip-limit.md) | Low | Magic-link rate limit is per-user only, no IP throttle | Server / auth |
| [SEC-26](SEC-26-ci-secret-scanning.md) | Informational | No secret scanning / SAST in CI | CI |
| [SEC-27](SEC-27-client-jwt-decode.md) | Informational | Client decodes unverified JWT payload | Web client |

## Suggested fix order

**Block production launch:** SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06, SEC-07, SEC-09.

**Next sprint:** SEC-08, SEC-10, SEC-11, SEC-12, SEC-15, SEC-16, SEC-17, SEC-18.

**Following quarter:** SEC-13, SEC-14, SEC-19, SEC-21–SEC-27 as capacity allows.

## What was checked and found acceptable

- **Mobile token storage** — iOS uses Keychain, Android uses `EncryptedSharedPreferences` (AES-256-GCM). Good. (One hardening note: SEC-24.)
- **Android cleartext traffic** — `network_security_config.xml` permits cleartext only for `localhost`/`10.0.2.2`/`127.0.0.1` (emulator/dev). Correctly scoped.
- **SQL access** — repositories use parameterized queries (`$N` placeholders). Dynamic table/column names in the Canvas link-rewriter and migration runner are hardcoded constants, not user input. No SQL injection found.
- **AWS production Terraform** — RDS `publicly_accessible = false`, DB/Redis/RabbitMQ ingress scoped to EKS node security groups, `sslmode=require`, secrets in AWS Secrets Manager. Well-architected.
- **CI dependency scanning** — `govulncheck` (Go) and `npm audit --audit-level=high` are present. Secret scanning is the remaining gap (SEC-26).
- **Transcript request authorization** — student listing is scoped by `user_id`; the admin failed-requests endpoint is gated by `global:app:rbac:manage` and scoped by `org_id`. No IDOR found in the new transcript-delivery code; pickup instructions render as React text (no XSS).
