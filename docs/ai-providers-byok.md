# AI providers & bring-your-own-key (BYOK)

Admin and operator guide for configuring optional AI in Lextures. **OpenRouter is one provider**, not the only path.

## Glossary

| Term | Meaning |
| --- | --- |
| **AI provider** | Backend that runs models: `openrouter`, `anthropic`, `openai`, `azure_openai`, `bedrock`, `vertex` |
| **BYOK** | Bring-your-own-key — platform or org credentials you supply; secrets are write-only |
| **OpenRouter (provider)** | Optional routing gateway listed alongside direct providers |

## Configure (web)

1. Sign in as a **global admin**.
2. Open **Settings → Intelligence → Models**.
3. With multi-provider AI enabled, use **AI providers** to add credentials for one or more backends.
4. Set feature models (course setup, flashcards, vibe, grading, image) from the provider catalog.
5. Org admins can override provider / BYOK under organization AI settings when allowed.

Multi-provider AI is **on by default** (`AI_PROVIDER_ABSTRACTION_ENABLED`, AP.9). Set the env var to `0` only for emergency rollback — see [runbooks/ai-provider-rollback.md](runbooks/ai-provider-rollback.md). Legacy mode (flag off) still exposes a single OpenRouter API key field.

## Supported providers (GA surface)

| Provider | Typical credentials |
| --- | --- |
| OpenRouter | API key |
| Anthropic | API key (optional base URL) |
| OpenAI | API key (optional base URL) |
| Azure OpenAI | API key + `azure_base_url`; optional `azure_api_version`, `default_deployment`, `deployments` map — see [Azure runbook](runbooks/azure-openai-setup.md) |
| Amazon Bedrock | `auth_mode`: `api_key` \| `access_key` \| `iam_role` + `aws_region` — see [Bedrock IAM runbook](runbooks/bedrock-iam-setup.md) |
| Google Vertex AI | `auth_mode`: `api_key` \| `service_account` \| `adc` + project/location — see [Vertex ADC runbook](runbooks/vertex-adc-setup.md) |

Do **not** paste real keys into tickets, docs, or git. Use placeholders in examples.

## CLI

```bash
lextures settings ai-provider get
lextures settings ai-provider set --file provider.json
lextures settings ai-provider test
```

Example `provider.json` (org scope):

```json
{
  "provider": "anthropic",
  "byokApiKey": "REPLACE_ME"
}
```

`openRouterApiKey` on settings payloads is **deprecated**; use provider + `byokApiKey` (or platform provider credentials in the web UI). See [api-changelog-ai-providers.md](api-changelog-ai-providers.md).

## GA / ops

- Manual soak checklist: [runbooks/ai-provider-ga-checklist.md](runbooks/ai-provider-ga-checklist.md)
- Rollback (flag or image): [runbooks/ai-provider-rollback.md](runbooks/ai-provider-rollback.md)
- Alert: `AIProviderElevatedErrors` on `lextures_ai_provider_calls_total{outcome="error"}`

## Disclosure & trust

- In-app disclosure and banners reflect **configured** providers (see `/ai-disclosure`).
- Trust Center lists AI vendors as **when configured**; customer BYOK to the customer’s own cloud account is not automatically a Lextures sub-processor. See the note on `/trust`.

## Developer notes

- Product AI calls go through `server/internal/service/aiprovider` (resolver / gateway).
- Epic plans: [docs/plan/ai-providers/README.md](plan/ai-providers/README.md) (completed stories under `docs/completed/ai-providers/`).
- Adding a new provider: start from the provider capability interface (AP.1) and credential field matrix (AP.2 / AP.5).

## Support macros

**How do I use Azure OpenAI?**  
Global admin → Settings → Intelligence → Models → add **Azure OpenAI** with endpoint, API version, deployment map, and API key. Optionally set org BYOK under the org AI provider panel. Confirm with Test connection. See [docs/runbooks/azure-openai-setup.md](runbooks/azure-openai-setup.md). OpenRouter is not required.

**How do I use Bedrock without storing AWS keys?**  
Set Bedrock `auth_mode` to **IAM role / instance profile**, set `aws_region`, and run the API on a role that can call `bedrock:Converse`. See [docs/runbooks/bedrock-iam-setup.md](runbooks/bedrock-iam-setup.md).

**How do I use Vertex with a service account?**  
Set Vertex `auth_mode` to **Service account JSON**, upload the JSON (never returned later), set project/location, Save, Test connection. See [docs/runbooks/vertex-adc-setup.md](runbooks/vertex-adc-setup.md).

**We don’t want OpenRouter.**  
Configure Anthropic, OpenAI, Azure, Bedrock, or Vertex only. Trust/disclosure copy will not claim OpenRouter processes data unless that provider is configured.
