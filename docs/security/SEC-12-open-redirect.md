# SEC-12 — Open redirect via protocol-relative `next` / `RelayState`

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server + web client
- **Files:** [clients/web/src/pages/saml-callback.tsx:19](../../clients/web/src/pages/saml-callback.tsx), [server/internal/browsersaml/acs.go:265](../../server/internal/browsersaml/acs.go)

## Problem

Both redirect-validation sites accept any value beginning with `/`:

```ts
// saml-callback.tsx
const to = nextRaw && nextRaw.startsWith('/') ? decodeURIComponent(nextRaw) : '/'
```

```go
// acs.go
if rs := strings.TrimSpace(r.PostFormValue("RelayState")); rs != "" && strings.HasPrefix(rs, "/") {
```

A protocol-relative URL like `//attacker.example/phish` **also** starts with `/`, but browsers treat it as an absolute off-origin navigation. Notably, `magicLinkSanitizeRedirect` (`server/internal/service/authservice/magic_link.go`) already rejects `//` — so the codebase knows the correct check; these two sites just don't apply it.

## Risk

Phishing-grade open redirect. A user clicks a trusted SAML/login link and is silently bounced to an attacker-controlled clone that harvests credentials. Combined with the SSO flow, the victim has every reason to trust the initial URL.

## Fix

At every redirect-validation site, reject protocol-relative and absolute URLs:

```go
if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") {
    return "/"
}
```

Better, parse and require an empty host:

```go
u, err := url.Parse(s)
if err != nil || u.IsAbs() || u.Host != "" { return "/" }
```

Apply the equivalent in `saml-callback.tsx`. Centralize this in one helper shared by all three sites so future redirect targets inherit it.

## Verification

- `next=//evil.example` and `RelayState=//evil.example` both redirect to `/`, not off-origin.
- `next=/dashboard` still works.
