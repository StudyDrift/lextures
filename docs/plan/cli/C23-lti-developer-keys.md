# C23 — LTI & developer keys

> CLI parity plan. Source: `registerLTIHTTPRoutes` (`lti`, 10), `saml_lti.go`, `courses/{id}/lti-external-tools`, `developer` (2 — developer keys), `admin/lti` (7). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C23 |
| **Section** | Integrations & interoperability |
| **Severity** | MAJOR |
| **Markets** | K12 / HE |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Integrations / CLI |
| **Depends on** | C40 |
| **Unblocks** | C05 |

---

## 1. Problem Statement

LTI 1.3 tool registration, deployment configuration, and developer API keys are UI-only. Integration teams cannot register/rotate LTI tools or issue/rotate developer keys via automation, which is essential when standing up many tenants or rotating credentials on a schedule.

## 2. Goals

- Register/configure/rotate LTI tools at platform and course scope.
- Issue, list, and rotate/revoke developer API keys.
- Export LTI platform config (issuer, JWKS, deployment ids) for tool vendors.

## 3. Non-Goals

- Performing an LTI launch (browser flow).
- SAML SSO config beyond what overlaps in `saml_lti.go` (SSO config could be a follow-up).

## 4. Personas & User Stories

- **As an integration admin**, I want `lti tools register --file tool.json` to add an LTI tool.
- **As a security admin**, I want `dev-keys rotate <id>` on a schedule.
- **As a vendor liaison**, I want `lti platform-config` to hand the vendor our issuer/JWKS/deployment ids.

## 5. Functional Requirements

- **FR-1.** MUST add `lti tools list|register|update|delete` (admin + course `lti-external-tools`).
- **FR-2.** MUST add `lti deployments list|create` and `lti platform-config` (issuer/JWKS/keys export).
- **FR-3.** MUST add `dev-keys list|create|rotate|revoke` (`developer`).
- **FR-4.** SHOULD add `lti tools test <id>` (validate config / OIDC login round-trip where server supports).

## 6. Non-Functional Requirements

- **Performance** — trivial payloads; p95 < 500 ms.
- **Security** — integration-admin scope; client secrets/private keys shown once on create, then redacted; rotation invalidates old.
- **Privacy & Compliance** — LTI deployments may transmit PII to tools; command surfaces the tool's data-sharing scope where available.
- **Reliability** — register/update idempotent by client id.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a tool config, *When* `lti tools register`, *Then* a client id + one-time secret print; re-fetch redacts the secret.
- **AC-2.** *Given* a dev key, *When* `dev-keys rotate`, *Then* a new key prints and the old stops working.
- **AC-3.** *Given* `lti platform-config --json`, *Then* issuer/JWKS URL/deployment ids are emitted.

## 8. Data Model

- None client-side. Document tool config JSON.

## 9. API Surface

- `registerLTIHTTPRoutes` + `admin/lti`; `courses/{c}/lti-external-tools`; `developer` key endpoints; `saml_lti.go`.

## 10. UI / UX

- `lextures lti ...`, `lextures dev-keys ...`.
- One-time secret display with an explicit "save now" notice.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server LTI/dev-key handlers; course tool registration (C05).

## 13. Dependencies & Sequencing

- After: C40.
- Before: C05 (course tool add references platform tools).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Secret shown only once, user misses it | M | M | Explicit notice + optional `--secret-out file` |
| Rotation breaks live integrations | M | H | `--grace` window if server supports; warn before revoke |

## 15. Rollout Plan

- Ship dev-keys + LTI tool CRUD first, then deployments/platform-config/test.
- Rollback: additive.

## 16. Test Plan

- **Unit** — config parse; secret one-time display.
- **Integration** — tool register; dev-key rotate invalidation.
- **Security** — secret redaction on re-fetch.
- **E2E** — register tool → platform-config → (mock) launch validate.

## 17. Documentation & Training

- "Register an LTI 1.3 tool" and "Rotate developer keys" recipes.

## 18. Open Questions

1. Does the platform support LTI Advantage services config via API?
2. Grace period on dev-key rotation?

## 19. References

- `registerLTIHTTPRoutes`, `saml_lti.go`, developer-key handlers.
- Related: [C05](C05-content-extras.md), [C25](C25-integrations-webhooks-bots.md).
