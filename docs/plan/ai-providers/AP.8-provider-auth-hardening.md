# AP.8 — Provider Auth Hardening (Azure, Bedrock IAM, Vertex ADC)

> Implementation plan. Source: multi-provider BYOK epic ([README](README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.8 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE (enterprise) |
| **Status (today)** | THIN — Azure/Bedrock/Vertex use API-key-style HTTP wrappers; limited enterprise auth |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / Cloud |
| **Depends on** | AP.2, AP.4 |
| **Unblocks** | Enterprise GA for AWS/GCP-native customers |

---

## 1. Problem Statement

Enterprise customers rarely want long-lived raw API keys for cloud AI. Azure OpenAI needs correct resource URLs, API versions, and deployment names. AWS Bedrock is typically accessed with **IAM roles** (instance profile / IRSA), not a static key. Google Vertex commonly uses **ADC / service accounts**. Today’s `aiprovider` factory only models API keys plus a few base URL settings, which is enough for demos but not for production cloud-native deployments.

## 2. Goals

- Production-grade configuration for Azure OpenAI (deployments, api-version, endpoints).
- Bedrock auth via static keys **or** default AWS credential chain (role-based).
- Vertex auth via API key **or** Google ADC / service account JSON (encrypted).
- Clear admin validation errors when cloud settings are incomplete.
- Documented runbooks for each cloud.

## 3. Non-Goals

- Multi-cloud automatic failover beyond existing provider fallback.
- Implementing every Bedrock/Vertex regional specialty.
- Customer-managed VPC private link automation (document only).

## 4. Personas & User Stories

- **As an AWS-hosted university**, I want Bedrock via the node’s IAM role so that we do not store long-lived keys in Lextures.
- **As a GCP customer**, I want to upload a service account JSON (encrypted) or use workload identity so that Vertex calls succeed.
- **As an Azure admin**, I want to map aliases to deployment names on our OpenAI resource.

## 5. Functional Requirements

- **FR-1.** Azure OpenAI provider MUST require `azure_base_url` and support `azure_api_version` + per-alias deployment mapping in settings.
- **FR-2.** Bedrock provider MUST support `auth_mode=api_key|access_key|iam_role` (names illustrative); `iam_role` uses AWS SDK default chain and region.
- **FR-3.** Vertex provider MUST support `auth_mode=api_key|service_account|adc`; service account JSON stored encrypted as a secret material type.
- **FR-4.** Factory MUST refuse to build providers with incomplete settings (actionable error).
- **FR-5.** Test Connection MUST exercise the real auth path (not only key presence).
- **FR-6.** Credential store (AP.2) MUST allow multiple secret_keys per provider when needed (`api_key`, `aws_secret_access_key`, `service_account_json`).
- **FR-7.** OpenRouter and Anthropic/OpenAI direct paths MUST remain unchanged (still API key).
- **FR-8.** Logs MUST never print service account JSON or AWS secret keys.

## 6. Non-Functional Requirements

- **Performance** — AWS/GCP SDK clients reused; avoid per-request session create.
- **Security** — Least privilege IAM examples; encrypt SA JSON; memory wipe best-effort on rotate.
- **Privacy** — Regional endpoints honored for data residency claims.
- **Accessibility** — Admin fields in AP.5 must expose new auth modes.
- **Scalability** — Connection pooling per provider instance.
- **Reliability** — Token refresh handled by SDKs; surface auth errors distinctly (non-retryable for fallback).
- **Observability** — `ai_provider_errors_total{provider,error_type=auth|quota|server}`.
- **Maintainability** — Prefer official SDKs over hand-rolled SigV4 if complexity warrants (`go.mod` additions).
- **Internationalization** — Error messages localized later; English first.
- **Backward compatibility** — Existing API-key Azure/Bedrock/Vertex configs keep working.

## 7. Acceptance Criteria

- **AC-1.** *Given* Azure settings with deployment map, *When* alias `gpt-4o` is resolved, *Then* requests hit the deployment path, not a generic model id only.
- **AC-2.** *Given* Bedrock `iam_role` mode without stored key, *When* test connection runs on a machine with valid AWS role, *Then* complete succeeds (manual/integration env).
- **AC-3.** *Given* invalid Azure endpoint, *When* test connection runs, *Then* error is auth/config class, not opaque 502 only.
- **AC-4.** *Given* service account JSON stored, *When* GET credentials API, *Then* JSON never returned.
- **AC-5.** *Given* unit tests with mocked cloud endpoints, *When* CI runs, *Then* no real cloud credentials required.

## 8. Data Model

- Extends AP.2 secrets: additional `secret_key` values per provider.
- `settings` JSON schema per provider versioned in code validation.

Example settings:

```json
{
  "auth_mode": "iam_role",
  "aws_region": "us-west-2"
}
```

```json
{
  "azure_base_url": "https://contoso.openai.azure.com",
  "azure_api_version": "2024-10-21",
  "deployments": { "gpt-4o": "gpt4o-prod", "text-fast": "gpt4o-mini" }
}
```

## 9. API Surface

- No new routes; validation on existing PUT credential/settings endpoints.
- Test endpoint returns `authMode` in response metadata.

## 10. UI / UX

- AP.5 forms gain auth mode selectors and conditional fields.
- Help text + links to cloud IAM setup docs.
- File upload for service account JSON (client reads → PUT secret; not stored in browser).

## 11. AI / ML Considerations

- Model availability varies by region; catalog (AP.3) should filter by region when known.

## 12. Integration Points

- `aiprovider/{azure path in openai.go,bedrock.go,vertex.go,factory.go}`
- `go.mod` — optional `aws-sdk-go-v2`, `google.golang.org/api` / cloud aiplatform
- AP.2 credential store multi-secret
- AP.5 admin forms
- Runbooks under `docs/runbooks/`

## 13. Dependencies & Sequencing

- After AP.2 (storage) and enough of AP.4 that test connection is meaningful.
- Can parallel late AP.5 UI fields.
- Not a hard blocker for OpenRouter/Anthropic/OpenAI GA.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SDK dependency weight | M | M | Build tags or optional modules; keep HTTP path for simple modes |
| IRSA/local dev complexity | H | M | Document `auth_mode=access_key` for dev |
| SigV4 bugs | M | H | Prefer AWS SDK Converse API |
| SA JSON size / rotation | L | M | Size limits; rotation runbook |

## 15. Rollout Plan

- Ship API-key modes first (already partial); add iam/adc behind provider settings validation.
- Pilot with one AWS and one Azure customer.
- Feature flag optional: `AI_CLOUD_AUTH_MODES_ENABLED`.
- Rollback: force `auth_mode=api_key`.

## 16. Test Plan

- **Unit** — settings validation; deployment resolution; auth mode switch.
- **Integration** — httptest for Azure; SDK interfaces mocked for Bedrock/Vertex.
- **Manual** — real sandbox accounts in staging.
- **Security** — secret redaction tests; file upload size limits.

## 17. Documentation & Training

- Runbooks: Azure OpenAI, Bedrock IAM, Vertex ADC.
- IAM policy least-privilege examples.
- Troubleshooting matrix (401/403/404 deployment).

## 18. Open Questions

1. Is workload identity federation in-scope for v1 or phase 2?
2. Do we support multiple Azure deployments per alias set for blue/green?
3. Bedrock model access is account-gated — how do we surface “model not enabled” in UI?

## 19. References

- `server/internal/service/aiprovider/openai.go` (Azure)
- `server/internal/service/aiprovider/bedrock.go`
- `server/internal/service/aiprovider/vertex.go`
- 16.7 open questions on IAM
- Related: [AP.2](AP.2-credential-store-and-byok.md), [AP.5](AP.5-admin-intelligence-ui.md)
