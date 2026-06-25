# Using Lextures with Make.com

The Lextures Make.com custom app exposes the same triggers and actions as the Zapier connector.

## Setup

1. Enable platform flags: **Public API**, **API Tokens**, **Outbound Webhooks**, and **Zapier Connector** (also gates Make REST hooks).
2. Create an API token with required scopes.
3. Import the bundle from `integrations/make/` using the Make Apps Editor (VS Code extension) or upload modules manually.

## Enroll User action

Provide **Course ID**, **User email**, and optional **Course role**. Make calls `POST /api/v1/courses/{id}/enrollments` and returns the new enrollment ID.

## Watch New Enrollments trigger

Registers a REST hook with `settings.source=make` so deliveries are tagged in Lextures logs.

## Validate locally

```bash
cd integrations/make
npm run validate
npm test
```
