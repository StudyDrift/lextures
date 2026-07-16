# Runbook: Google Vertex ADC / service account setup

Configure Lextures for Vertex AI Gemini with API key, service account JSON, or Application Default Credentials (AP.8).

## Auth modes

| `auth_mode` | Secrets needed | When to use |
| --- | --- | --- |
| `api_key` | `api_key` | Simple key auth |
| `service_account` | `service_account_json` | Uploaded SA key (encrypted, ≤ 64 KiB) |
| `adc` | none | Workload identity / GCE / Cloud Run metadata |

Also set `gcp_project` and `gcp_location` (or `vertex_base_url`).

## Service account

1. Create a service account with Vertex AI User (or a tighter custom role that can call `aiplatform.endpoints.predict` / generateContent).
2. Download JSON key **once**; paste or upload in the admin UI.
3. Lextures stores ciphertext only — GET never returns the JSON (AC-4).

## Application Default Credentials

On GCE/GKE/Cloud Run with a attached SA:

1. Set `auth_mode=adc`, `gcp_project`, `gcp_location`.
2. Ensure the runtime identity can call Vertex in that project/location.
3. **Test connection** mints an OAuth token via ADC and calls generateContent.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| Config: invalid service_account_json | Malformed JSON or wrong key type |
| Auth: ADC unavailable | Not running on GCP / no metadata SA |
| 403 | SA lacks Vertex permissions or API not enabled |

## Security

- Rotate SA keys via Clear + re-upload; avoid logging JSON (FR-8).
- Prefer ADC / workload identity over long-lived JSON keys when possible.
