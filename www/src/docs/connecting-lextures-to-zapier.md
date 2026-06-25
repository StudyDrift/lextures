# Connecting Lextures to Zapier

Use the Lextures Zapier connector to automate workflows between Lextures and thousands of other apps.

## Prerequisites

1. Enable **Public API**, **API Tokens**, **Outbound Webhooks**, and **Zapier Connector** in Lextures **Settings → Global platform**.
2. Create a personal access token under **Settings → Integrations → API Keys** with scopes for your Zaps (for example `webhooks:manage`, `enrollments:write`, `grades:write`).

## Connect your account

1. In Zapier, search for **Lextures** and add the app.
2. Enter your Lextures API base URL (for example `https://app.lextures.com`).
3. Paste your personal access token.
4. Zapier calls `GET /api/v1/me` to verify the connection and shows your name and email when the token includes `pii:read`.

## Example: New enrollment → Slack

1. **Trigger:** Lextures → **New Enrollment** (REST hook).
2. **Action:** Slack → **Send Channel Message** mapping course and student fields.

When a student is enrolled, Lextures delivers a signed webhook to Zapier within seconds.

## Privacy

Trigger payloads omit student email unless your API token includes the `pii:read` scope. Data processed by Zapier is subject to Zapier's privacy policy.

## Local development

From the repo root:

```bash
cd integrations/zapier
npm ci
npm test
npm run validate
```

Use `zapier push` with a Zapier Platform account to publish updates.
