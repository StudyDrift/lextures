# Statement of Applicability (SoA)

**Standard:** ISO/IEC 27001:2022 Annex A (93 controls)  
**PIMS extension:** ISO/IEC 27701:2019 (mapped to GDPR Art. 25 via plan 10.3)

## Summary

| Status | Count |
|--------|------:|
| Implemented | Tracked in `compliance.iso_soa_controls` |
| Planned | In progress for certification target Q1 2027 |
| Excluded | Justified per control in admin SoA register |

The authoritative SoA register is maintained in the database (`compliance.iso_soa_controls`) and exposed via:

- Admin UI: `/admin/compliance/iso`
- API: `GET /api/v1/compliance/iso/soa`

## Control themes (2022 structure)

1. **Organizational** (37 controls) — policies, supplier relationships, incident management, BCM
2. **People** (8 controls) — screening, awareness training, remote working
3. **Physical** (14 controls) — largely **excluded** with AWS/datacenter shared-responsibility justification
4. **Technological** (34 controls) — authentication, logging, cryptography, secure SDLC

## Key implemented technological controls

| Control | Implementation |
|---------|----------------|
| A.8.2 Privileged access | RBAC; Global Admin role; least privilege |
| A.8.5 Secure authentication | JWT, MFA (TOTP/WebAuthn), HIBP password checks |
| A.8.15 Logging | Admin audit log (plan 10.11); application request logs |
| A.8.24 Cryptography | TLS 1.2+, AES-256 at rest (plan 10.13) |

Full control titles and statuses are seeded from `server/internal/service/iso/soa_controls.go` (93 entries).
