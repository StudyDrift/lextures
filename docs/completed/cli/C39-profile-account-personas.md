# C39 — Profile, account, security & personas (me, parent)

> CLI parity plan. Source: `registerMeRoutes` + `registerMeProfileDepthRoutes` (`me`, `me/profile-fields`, `me/sessions`, `me/mfa`, `me/oidc-identities`, `me/access-keys`, `me/device-tokens`, `me/demographics`, `me/consent-studies`, `me/entitlements`, `me/onboarding`), `registerParentRoutes` (`parent`, `orgs/{orgId}/parent-links`), `me/calendar-token`. Baseline: `clients/cli/cmd/me_profile.go`, `me_profile_logic.go`, `me_profile_test.go`, `cli_framework.go` (`whoami`), existing `auth` + `access-keys`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C39 |
| **Section** | Student experience / personas |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Identity / CLI |
| **Depends on** | C40 |
| **Unblocks** | C25 |

---

## 1. Problem Statement

The CLI authenticates but exposes nothing about the current account: no `me` profile, session management, MFA, linked identities, personal access keys, or consent/demographics. Service accounts and users can't self-manage credentials, and the parent/guardian persona (parent-links to children, viewing their data) is entirely absent.

## 2. Goals

- Inspect and edit the current user's profile and custom fields.
- Manage security: sessions (list/revoke), MFA (enroll/disable), linked OIDC identities, personal access keys.
- Manage consent/demographics/onboarding self-service.
- Support the parent/guardian persona: link to children and view permitted data.

## 3. Non-Goals

- Admin user management (see C15).
- Browser SSO flows (existing `auth login` covers device/browser login).

## 4. Personas & User Stories

- **As any user**, I want `me get` and `me update --file profile.json`.
- **As a security-conscious user**, I want `me sessions list` + `me sessions revoke --all` after a device loss.
- **As a user**, I want `me mfa enroll` / `me access-keys create` for CI.
- **As a parent**, I want `parent children list` and `parent grades --child C` to follow my child's progress.

## 5. Functional Requirements

- **FR-1.** MUST add `me get|update` (`--file`), `me profile-fields get|set`.
- **FR-2.** MUST add `me sessions list|revoke [--all]`, `me mfa status|enroll|disable`, `me oidc-identities list|link|unlink`.
- **FR-3.** MUST add `me access-keys list|create|revoke` (personal tokens for CI; one-time display) — shared surface with C25.
- **FR-4.** SHOULD add `me demographics get|set`, `me consent-studies list|opt-in|opt-out`, `me onboarding status`, `me entitlements`.
- **FR-5.** SHOULD add `parent children list`, `parent link|unlink`, `parent grades|attendance --child <c>` (`registerParentRoutes`, `parent-links`).

## 6. Non-Functional Requirements

- **Performance** — trivial; p95 < 400 ms.
- **Security** — self-scope; MFA/session ops are sensitive → confirm on revoke-all; access keys one-time display; parent access strictly limited to linked children (server-enforced).
- **Privacy & Compliance** — demographics/consent are sensitive (GDPR/FERPA/COPPA); parent-link honors guardianship rules.
- **Reliability** — session revoke idempotent; the CLI's own session survives revoke-all unless `--include-current`.
- **Backward compatibility** — existing `auth` unchanged; `me access-keys` shared with C25.

## 7. Acceptance Criteria

- **AC-1.** *Given* a user, *When* `me get --json`, *Then* the profile emits.
- **AC-2.** *Given* multiple sessions, *When* `me sessions revoke --all`, *Then* others end; current stays unless `--include-current`.
- **AC-3.** *Given* a parent, *When* `parent grades --child C`, *Then* only that linked child's grades return.

## 8. Data Model

- None client-side beyond existing token store.

## 9. API Surface

- `me` + `me/*` (profile-fields, sessions, mfa, oidc-identities, access-keys, device-tokens, demographics, consent-studies, entitlements, onboarding, calendar-token); `parent` + `orgs/{orgId}/parent-links`.

## 10. UI / UX

- `lextures me ...`, `lextures parent ...`. Extends the existing `auth` group conceptually.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server me/parent handlers; auth token store (`internal/auth`); access keys shared with C25.

## 13. Dependencies & Sequencing

- After: C40.
- Before: C25 (access-keys), any command wanting `me` context.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Self-lockout via session revoke | M | M | Exclude current session by default; `--include-current` explicit |
| Parent over-scope | L | H | Server enforces linked-child scope; CLI passes child id, never enumerates others |

## 15. Rollout Plan

- Ship `me get/update` + sessions + access-keys first, then MFA/identities/consent, then parent persona.
- Rollback: additive.

## 16. Test Plan

- **Unit** — profile parse; revoke-all current-session exclusion.
- **Integration** — sessions/mfa/access-keys; parent child scope.
- **Security** — access-key one-time display; parent scope 403 on non-linked child.
- **E2E** — create access key → use it → revoke.

## 17. Documentation & Training

- "Manage your account and CI keys" recipe; parent-portal guide.

## 18. Open Questions

1. Does MFA enroll require an interactive TOTP step (QR/secret) the CLI must render?

## 19. References

- `registerMeRoutes`, `registerMeProfileDepthRoutes`, `registerParentRoutes`; `clients/cli/internal/auth`.
- Related: [C15](C15-people-provisioning.md), [C25](C25-integrations-webhooks-bots.md), [C29](C29-compliance-privacy.md), [C40](C40-cli-framework.md).
