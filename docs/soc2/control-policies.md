# SOC 2 Control Policies

> Control policy library for Lextures SOC 2 Type II engagement. Maps AICPA Trust Services Criteria (2017) to implemented controls.

## Security (CC1–CC9)

### CC1 — Control Environment

| Control | Implementation |
|---------|---------------|
| CC1.1 — COSO Principle 1: Commitment to integrity | Code of conduct required; annual acknowledgement tracked in HR system. |
| CC1.2 — COSO Principle 2: Board oversight | CTO security program; quarterly executive review. |
| CC1.3 — COSO Principle 3: Structures, authorities | Org chart and RACI documented; approval thresholds enforced via RBAC. |
| CC1.4 — COSO Principle 4: Commitment to competence | Annual security training required; completion tracked. |
| CC1.5 — COSO Principle 5: Accountability | Performance reviews include security objectives. |

### CC2 — Communication and Information

| Control | Implementation |
|---------|---------------|
| CC2.1 — COSO Principle 13: Internal information | `admin_audit_log` for all security-relevant events; queryable by compliance team. |
| CC2.2 — COSO Principle 14: External communication | Trust center; responsible disclosure policy at docs/SECURITY.md. |
| CC2.3 — COSO Principle 15: Reporting to external parties | SOC 2 report made available under NDA via trust center (plan 20.2). |

### CC3 — Risk Assessment

| Control | Implementation |
|---------|---------------|
| CC3.1 — COSO Principle 6: Objectives | Security objectives aligned to TSC; documented in system description. |
| CC3.2 — COSO Principle 7: Risk identification | Annual risk assessment; threats catalogued per asset. |
| CC3.3 — COSO Principle 8: Fraud risk | Insider-threat controls: least privilege, quarterly access reviews, audit log. |
| CC3.4 — COSO Principle 9: Change analysis | Change management process (CC8); post-deploy monitoring. |

### CC6 — Logical and Physical Access Controls

| Control | Implementation |
|---------|---------------|
| CC6.1 — Logical access security | RBAC; JWT-based session tokens; MFA enforced for production access. |
| CC6.2 — Authentication | Argon2id password hashing; HIBP breach check on new passwords (plan 4.x). |
| CC6.3 — Access removal and review | Quarterly privileged access reviews; semi-annual all-production reviews stored in `compliance.access_reviews`. |
| CC6.6 — Logical access security for external parties | Vendor SSO integrations reviewed; API keys rotated quarterly. |
| CC6.7 — Transmission protection | TLS 1.2+ enforced on all endpoints; HSTS enabled. |
| CC6.8 — Malware protection | Dependency scanning in CI; ClamAV malware scanning on uploads (plan 8.6). |

### CC7 — System Operations

| Control | Implementation |
|---------|---------------|
| CC7.1 — Configuration management | IaC in `iac/production/`; all changes via PR review; Terraform state in encrypted S3. |
| CC7.2 — Monitoring | Authentication logs 180-day retention; anomaly alerts in CloudWatch. |
| CC7.3 — Incident identification | Automated anomaly detection; security alerts → PagerDuty → on-call engineer. |
| CC7.4 — Incident response | Incident response plan; incidents logged in `compliance.incidents`; post-mortem required within 5 business days. |
| CC7.5 — Incident recovery | Backup and restore tested quarterly (plan 10.15); RTO ≤ 15 min, RPO ≤ 5 min. |

### CC8 — Change Management

| Control | Implementation |
|---------|---------------|
| CC8.1 — Change management process | All changes via PR; branch protection blocks direct push to `main` (AC-1); CI gates (tests, SAST, lint). |

### CC9 — Risk Mitigation

| Control | Implementation |
|---------|---------------|
| CC9.1 — Risk mitigation | Residual risk acceptance documented in risk register; mitigations tracked quarterly. |
| CC9.2 — Vendor and business partner management | Vendor risk register in `compliance.vendor_risk`; sub-processors reviewed annually. |

## Availability (A1)

| Control | Implementation |
|---------|---------------|
| A1.1 — Current processing capacity | Auto-scaling ECS tasks; capacity planning reviewed quarterly. |
| A1.2 — Environmental protections | AWS Multi-AZ; redundant network paths; automated failover (RTO ≤ 15 min). |
| A1.3 — Recovery plan testing | DR tested quarterly; backup restore tested; results in evidence bucket. |

## Privacy (P1–P8)

| Control | Implementation |
|---------|---------------|
| P1 — Privacy notice | lextures.com/privacy updated on material changes; FERPA, COPPA, CCPA, state laws documented. |
| P2 — Choice and consent | CCPA opt-out (plan 10.4); COPPA parental consent (plan 10.2); FERPA directory opt-out (plan 10.1). |
| P3 — Collection | Data minimisation policy; only data necessary for service delivery collected. |
| P4 — Use, retention, disposal | Retention schedule in privacy notice; deletion enforced after retention period. |
| P5 — Access | Data subject rights requests handled within regulatory deadlines (CCPA 45 days, GDPR 30 days). |
| P6 — Disclosure | Disclosure log maintained (FERPA); third-party disclosures require documented legal basis. |
| P7 — Quality | Data correction workflow (FERPA amendment, GDPR rectification). |
| P8 — Monitoring and enforcement | DPA violations reported to compliance officer; annual privacy impact assessment. |
