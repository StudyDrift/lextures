# Security Policy

## Reporting a Vulnerability

We take the security of Lextures seriously. If you believe you have found a security vulnerability, please report it to us as described below.

**Contact:** [security@lextures.io](mailto:security@lextures.io)

**PGP fingerprint:** `E3F4 9A12 7B6C 8D01 4F2E 91A3 5C7D 0E8B 2A4F 6B9C`

**Public key:** [keys.openpgp.org — security@lextures.io](https://keys.openpgp.org/search?q=security%40lextures.io)

Encrypted reports are preferred for sensitive findings. Include steps to reproduce, affected URLs or components, and any proof-of-concept you can safely share.

We aim to acknowledge valid reports within **2 business days** and will provide a ticket reference for tracking.

## Safe Harbor

If you make a good-faith effort to comply with this policy during your security research, we will not initiate or support legal action against you for that research. This safe harbor applies when:

- You avoid privacy violations, destruction of data, and interruption or degradation of our service.
- You do not access data belonging to other users or exceed the minimum access needed to demonstrate a vulnerability.
- You give us reasonable time to remediate before any public disclosure.

This policy is intended to align with ISO/IEC 29147:2018 §6.2 (coordinated vulnerability disclosure).

## Scope

**In scope**

- The Lextures web application (`*.lextures.io`, `*.lextures.com`) and official API endpoints
- Authentication, authorization, session handling, and tenant isolation
- Student data confidentiality (FERPA-aligned handling)
- Infrastructure misconfigurations on systems we operate

**Out of scope**

- Denial-of-service attacks against production systems
- Social engineering of Lextures staff or customers
- Physical security of non-Lextures facilities
- Third-party services (report issues to the vendor; we may forward with your permission)
- Issues requiring physical access to a user's device
- Automated scanner output without a demonstrated exploit

## Coordinated Disclosure

We follow a **90-day** coordinated disclosure timeline from initial report acknowledgment, consistent with Google Project Zero and CERT/CC norms. We may agree to shorter timelines for critical issues or extensions when remediation requires complex changes.

## Patch SLAs (CVSS 3.1)

| Severity | Target patch |
|----------|----------------|
| Critical | 7 calendar days |
| High | 30 calendar days |
| Medium | 90 calendar days |
| Low | Next scheduled release |

Severity is assessed using [CVSS 3.1](https://www.first.org/cvss/). We will notify you when a fix is deployed and coordinate public disclosure after the patch is available.

## Bug Bounty

Lextures may offer rewards for valid critical and high findings through an invite-only program. Bounty eligibility and amounts are determined at our discretion after triage. Participation in third-party platforms (e.g. HackerOne) will be announced on our [security page](https://app.lextures.io/security) when available.

## Internal Documentation

Security engineers: see `docs/security/triage_runbook.md` for triage workflow and `compliance.security_reports` for audit evidence.
