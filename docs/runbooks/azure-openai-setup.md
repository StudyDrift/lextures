# Runbook: Azure OpenAI setup

Configure Lextures to call Azure OpenAI with deployment-based routing (AP.8).

## Prerequisites

- An Azure OpenAI resource with at least one deployment
- An API key for that resource
- Platform secrets encryption: `PLATFORM_SECRETS_KEY` set on the API

## Settings

| Field | Required | Notes |
| --- | --- | --- |
| `azure_base_url` | yes | e.g. `https://contoso.openai.azure.com` |
| `azure_api_version` | no | Defaults to `2024-10-21` |
| `default_deployment` | no | Used when the model alias is not in `deployments` |
| `deployments` | no | JSON map of alias/model id → deployment name |
| API key | yes | Stored encrypted; never returned by GET |

Example `deployments`:

```json
{
  "gpt-4o": "gpt4o-prod",
  "text-fast": "gpt4o-mini"
}
```

Requests go to:

`POST {azure_base_url}/openai/deployments/{deployment}/chat/completions?api-version=...`

## Admin UI

Settings → Intelligence → Models → Azure OpenAI: set endpoint, optional API version / deployment map, paste API key, Save, then **Test connection**.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| 400 / config error on Test | Missing `azure_base_url` or invalid endpoint |
| 401 / 403 | Wrong API key or key for a different resource |
| 404 on deployment | Alias not mapped; set `deployments` or `default_deployment` to a real deployment name |

## Security

- Do not put API keys in `settings` JSON — only in the credential secret store.
- Prefer private networking / Private Link at the Azure layer; Lextures does not automate VNet peering.
