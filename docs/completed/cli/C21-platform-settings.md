# C21 — Platform settings & configuration

> CLI parity plan. Source: `settings_platform.go` (`settings/platform`, `locale`, `timezone`, `system-prompts`), `admin_password_policy.go`, `ai_provider_settings_http.go`, `registerDataResidencyRoutes`, `registerStorageQuotaRoutes` (`admin/storage-quotas`, `courses/{id}/storage-usage`), `admin_console.go` settings. Baseline: `clients/cli/cmd/settings.go`, `storage_quotas.go`, `platform_settings_logic.go`, `files usage`, `platform_settings_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C21 |
| **Section** | Admin & governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C40 |
| **Unblocks** | C09, C29 |

---

## 1. Problem Statement

Platform/tenant configuration — locale/timezone defaults, system prompts, password policy, AI provider settings, data residency, and storage quotas — is UI-only. Ops teams cannot codify tenant configuration or enforce policy (e.g. password rules, AI provider keys) via IaC, and can't monitor storage quotas from scripts.

## 2. Goals

- Read/write platform and tenant settings as version-controlled config.
- Manage password policy, AI provider settings, and data-residency config.
- Manage storage quotas and inspect usage; extend `files` with quota/usage.

## 3. Non-Goals

- Secret management systems integration (keys passed via `--file`/stdin only).
- Per-course settings (see C01).

## 4. Personas & User Stories

- **As a platform admin**, I want `settings get|set platform` from git.
- **As a security admin**, I want `settings password-policy set --file policy.json`.
- **As an AI admin**, I want `settings ai-provider set --file provider.json` (keys via stdin).
- **As an ops admin**, I want `storage-quotas list` and `files usage --course C`.

## 5. Functional Requirements

- **FR-1.** MUST add `settings get|set <scope>` for platform/locale/timezone/system-prompts (`--file`).
- **FR-2.** MUST add `settings password-policy get|set` (`admin_password_policy.go`).
- **FR-3.** MUST add `settings ai-provider get|set|test` (`ai_provider_settings_http.go`; keys never echoed).
- **FR-4.** SHOULD add `settings data-residency get|set` (`registerDataResidencyRoutes`).
- **FR-5.** MUST add `storage-quotas list|set` and `files usage [--course]` (`storage-usage`).
- **FR-6.** SHOULD add `settings export|apply --file settings.json` (declarative tenant config).

## 6. Non-Functional Requirements

- **Performance** — settings ops p95 < 500 ms.
- **Security** — platform-admin scope; provider keys/secrets read from file/stdin, redacted in all output, never in shell history via flags.
- **Privacy & Compliance** — data-residency changes are compliance-sensitive; require `--yes` and are audited.
- **Reliability** — set/apply idempotent; `--dry-run` diff.
- **Backward compatibility** — additive; existing `files` verbs unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a settings file, *When* `settings apply --dry-run`, *Then* a diff prints, no change.
- **AC-2.** *Given* an AI provider config, *When* `settings ai-provider set` then `... test`, *Then* connectivity is validated without printing the key.
- **AC-3.** *Given* a course over quota, *When* `files usage --course C`, *Then* usage vs limit prints.

## 8. Data Model

- None client-side. Document settings.json schema.

## 9. API Surface

- `settings/platform|locale|timezone|system-prompts`; `admin/password-policy`; `ai_provider_settings`; `data-residency`; `admin/storage-quotas`; `courses/{id}/storage-usage`.

## 10. UI / UX

- `lextures settings ...`, `lextures storage-quotas ...`, extend `files usage`.

## 11. AI / ML Considerations

- AI provider settings feed C09 grading and other AI features; `test` validates provider connectivity server-side.

## 12. Integration Points

- Server settings/policy/provider/residency/quota handlers; audit log (C19).

## 13. Dependencies & Sequencing

- After: C40.
- Before: C09 (needs provider config), C29 (residency ties to compliance).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provider key leakage | M | H | file/stdin only; redact everywhere; `test` never returns key |
| Residency change data movement | L | H | `--yes` + audit; document irreversibility |

## 15. Rollout Plan

- Ship settings get/set + storage first, then policy/provider/residency, then declarative apply.
- Rollback: additive.

## 16. Test Plan

- **Unit** — settings parse; secret redaction; diff.
- **Integration** — provider test; quota list.
- **Security** — key never in output/history.
- **E2E** — export→apply→verify.

## 17. Documentation & Training

- "Manage tenant config as code" recipe.

## 18. Open Questions

1. Which settings are platform-global vs org-scoped?
2. Does `ai-provider test` exist server-side, or must the CLI infer validity?

## 19. References

- `settings_platform.go`, `admin_password_policy.go`, `ai_provider_settings_http.go`, `registerStorageQuotaRoutes`.
- Related: [C01](C01-courses.md), [C09](C09-ai-grading-agents.md), [C19](C19-audit-impersonation-search.md), [C29](C29-compliance-privacy.md).
