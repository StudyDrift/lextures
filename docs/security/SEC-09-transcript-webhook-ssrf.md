# SEC-09 — Transcript webhook is an unrestricted SSRF primitive

- **Severity:** High
- **Status:** Confirmed present (new in the transcript-delivery change set)
- **Area:** Server / transcripts
- **Files:** [server/internal/httpserver/transcripts_http.go](../../server/internal/httpserver/transcripts_http.go) (`handlePutAdminTranscriptsConfig`, `deliverTranscriptWebhook`)

## Problem

An admin (`global:app:rbac:manage`) configures a transcript webhook URL. Validation only checks the scheme prefix:

```go
if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
    apierr.WriteJSON(w, http.StatusBadRequest, ...)
    return
}
```

When a student submits a transcript request, the server makes a server-side `POST` to that URL with the student's PII (name, email, student ID, delivery address) in the body:

```go
httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(*cfg.WebhookURL), bytes.NewReader(body))
...
client := &http.Client{Timeout: 30 * time.Second}
resp, err := client.Do(httpReq)
```

There is no allowlist, no DNS/IP validation, no block on private/link-local ranges, and `http://` (plaintext) is permitted. The default `http.Client` follows redirects, so even an external-looking URL can 302 to an internal target.

## Risk

This is a classic SSRF primitive, and it carries student PII as the payload:

- **Cloud metadata theft**: point the webhook at `http://169.254.169.254/latest/meta-data/...` (the demo droplet and any IMDSv1 host) to harvest instance credentials — a direct pivot to cloud-account compromise, the ShinyHunters end-goal.
- **Internal network probing**: target `http://postgres:8080`, `http://rabbitmq:15672`, or other service-mesh hosts. The response status code is recorded on the request row (`WebhookResponseCode`), giving a blind-to-semi-blind oracle.
- **PII exfiltration over plaintext** `http://`.

It requires an admin account, but a single compromised or malicious admin (or anyone who lands a forged admin token via SEC-01) turns this into infrastructure access. SSRF that reaches the metadata endpoint is the highest-leverage post-compromise step.

## Fix

1. Require `https://` only; reject `http://`.
2. Resolve the hostname and reject the request if **any** resolved IP is in a private, loopback, link-local, or unique-local range (`10/8`, `172.16/12`, `192.168/16`, `127/8`, `169.254/16`, `::1`, `fc00::/7`, `fe80::/10`). Re-validate after each redirect, or disable redirects entirely (`client.CheckRedirect = func(...) error { return http.ErrUseLastResponse }`).
3. Use a dialer with `Control` that re-checks the *connected* IP at dial time to defeat DNS-rebinding (resolve-then-connect TOCTOU).
4. Consider an explicit operator allowlist of webhook hostnames for institutional integrations.
5. Sign the payload (already done via HMAC) and document that the receiver must verify it.

## Verification

- Saving a webhook URL of `http://169.254.169.254/...` or `http://10.0.0.5/...` is rejected.
- A webhook that 302-redirects to a private IP fails the delivery rather than connecting.
- Plaintext `http://` URLs are rejected at config save.
