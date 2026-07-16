# Runbook: AWS Bedrock IAM setup

Configure Lextures for Bedrock with API key (gateway), static access keys, or IAM role (AP.8).

## Auth modes

| `auth_mode` | Secrets needed | When to use |
| --- | --- | --- |
| `api_key` | `api_key` | Bearer token / proxy gateways and local tests |
| `access_key` | `aws_access_key_id`, `aws_secret_access_key` | Local/dev or break-glass |
| `iam_role` | none | Production on EC2 / ECS / EKS (IRSA) / Lambda |

Also set `aws_region` (e.g. `us-west-2`). Optional `bedrock_base_url` for `api_key` mode only.

## Least-privilege IAM (example)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BedrockConverse",
      "Effect": "Allow",
      "Action": ["bedrock:InvokeModel", "bedrock:Converse"],
      "Resource": [
        "arn:aws:bedrock:REGION::foundation-model/*",
        "arn:aws:bedrock:REGION:ACCOUNT:inference-profile/*"
      ]
    }
  ]
}
```

Tighten `Resource` to the model IDs your tenants actually use. Enable model access in the Bedrock console per account/region.

## IRSA / instance profile

1. Attach a role with the policy above to the API workload.
2. In Lextures, create a Bedrock credential with `auth_mode=iam_role` and `aws_region`.
3. No long-lived keys are stored.
4. **Test connection** exercises the real Converse path.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| Auth error / AccessDenied | Role missing `bedrock:Converse` or model not enabled in region |
| Config error on access_key | Missing access key id or secret |
| Model not ready / ValidationException | Model access not granted for the account |

## Security

- Prefer `iam_role` in production; use `access_key` only for local development.
- Secrets are encrypted at rest and never returned by GET (only `*Configured` flags).
