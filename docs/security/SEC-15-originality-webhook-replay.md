# SEC-15 — No replay protection on originality webhook

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server / webhooks
- **File:** [server/internal/httpserver/webhooks_originality.go](../../server/internal/httpserver/webhooks_originality.go)

## Problem

The inbound originality webhook verifies an HMAC-SHA256 over the body, which authenticates integrity and source. But the signed body carries no timestamp and no nonce, and the handler keeps no record of previously-seen deliveries. A captured valid request can be replayed indefinitely.

## Risk

`MarkDoneByProviderReport` is largely idempotent, but a replayed callback can resurrect a deleted or superseded report row depending on the current schema state, and at minimum lets an attacker who captured one valid signed request repeatedly drive state transitions. Replay of a validly-signed message is a recognized webhook weakness.

## Fix

1. Require an `X-Originality-Timestamp` header and include it in the HMAC input. Reject deliveries with a clock skew greater than ~5 minutes.
2. Persist `(provider, providerReportId, timestamp)` (or a hash of the body) and reject duplicates within a retention window.

## Verification

- Replaying a previously-accepted webhook request returns `409`/`400`, not `200`.
- A request with a timestamp older than the skew window is rejected.
- A legitimate first-delivery still succeeds.
