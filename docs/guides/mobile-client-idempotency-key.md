# Mobile client idempotency key

Mobile apps queue offline writes in an ordered outbox and replay them on reconnect.
Each queued mutation carries a stable client-generated UUID sent to the API as:

```http
X-Idempotency-Key: <uuid>
```

## Client behavior

- Generated once when the mutation is first created (same key on every retry).
- Stored locally with the outbox item; successful replays are recorded in an applied-key set so the client never double-submits after a partial success.
- Replays are processed in enqueue order per device.

## Server guidance

Handlers that create resources or side effects SHOULD treat `X-Idempotency-Key` as a deduplication hint:

1. If the key was already processed for the authenticated user (and relevant scope, e.g. course/thread), return the original success response instead of applying again.
2. Return **409 Conflict** when the key matches a prior attempt but the payload differs, or the write is no longer valid (stale state). The mobile UI surfaces this as a conflict requiring review — it does not silently drop the item.

Some endpoints already accept an idempotency key in the JSON body (for example discussion posts use `idempotencyKey`). Mobile may send both the header and body field where the handler expects the body field; the header is the cross-feature convention for new outbox writes.

## Related code

- iOS: `clients/ios/Lextures/Core/Offline/`, `APIClient.requestRaw`
- Android: `clients/android/.../core/offline/`, `ApiClient.requestRaw`
