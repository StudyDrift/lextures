# SEC-11 — Tokens delivered via URL fragment in SAML callback

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server / SSO
- **File:** [server/internal/browsersaml/acs.go](../../server/internal/browsersaml/acs.go) (~L251 fragment build, ~L270 `location.replace`)

## Problem

After a successful SAML assertion, `HandleACS` returns an inline-script page that redirects the browser to the SPA with the tokens in the URL **fragment**:

```go
frag := "access_token=" + url.QueryEscape(res.AccessToken) + "&token_type=" + ...
// ...
`<script>location.replace("%s/saml-callback#%s%s");</script>`
```

The fragment carries the access token, refresh token, and any MFA-pending token. Fragments are not sent to servers, but they persist in browser history, `window.history`, browser extensions that read page URLs, and devtools/network logging on the client.

## Risk

Token leakage to anything with read access to the browser's URL state: extensions, a second user browsing history on a shared machine, screen-recording/observability tooling. Refresh tokens in particular are long-lived (SEC-02), so a leak here is replayable. Auditors flag URL-borne credentials independently of XSS.

## Fix

Replace the token-in-fragment handoff with a one-time exchange code:

1. ACS issues a random 128-bit single-use correlation code in the fragment and stores `(code → tokens)` server-side with a short TTL (≤ 60 s, single redemption).
2. The SPA `POST`s the code to `/api/v1/auth/saml/exchange`, which returns the tokens (ideally as the `HttpOnly` cookie from SEC-02) and deletes the code.

## Verification

- The SAML callback URL fragment contains only an opaque code, never a JWT.
- Replaying the same code a second time fails.
- The code expires after its TTL.
