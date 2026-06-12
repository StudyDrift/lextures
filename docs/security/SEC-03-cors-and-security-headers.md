# SEC-03 — CORS `*` and missing security headers

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Server middleware / nginx
- **Files:** [server/internal/httpserver/cors.go](../../server/internal/httpserver/cors.go), wired at [server/internal/httpserver/server.go:79](../../server/internal/httpserver/server.go) (`r.Use(corsAll)`), [clients/web/nginx.conf](../../clients/web/nginx.conf)

## Problem

`corsAll` mirrors the legacy "allow everything" policy:

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "*")
```

It is wired into every route at `server.go:79`. Separately, **no** response — from the Go server or from the nginx layer — sets any of:

- `Strict-Transport-Security`
- `Content-Security-Policy`
- `X-Frame-Options` / `frame-ancestors`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy`

(`grep -rn` across `server/internal` and `clients/web/nginx.conf` returns none of these.)

## Risk

- **No CSP** means the stored-XSS surfaces (SEC-05, SEC-06) execute unconstrained — a script can read tokens (SEC-02) and exfiltrate to any origin.
- **No `X-Frame-Options`/`frame-ancestors`** allows clickjacking of `/admin/*` pages.
- **No `nosniff`** lets browsers MIME-sniff uploaded files into executable HTML (compounds SEC-06).
- **No HSTS** leaves a downgrade window.
- `Access-Control-Allow-Origin: *` is contained *today* because tokens are bearer-header, not cookies — but it becomes an immediate credential-leak/CSRF problem the moment SEC-02's cookie migration lands. Fix both together.

## Fix

1. Replace `corsAll` with an explicit allowlist. Echo the request `Origin` only when it matches `PUBLIC_WEB_ORIGIN` (plus any configured tenant subdomains). Drop `Access-Control-Allow-Headers: *` and name the headers actually used (`Authorization, Content-Type`).
2. Add a `secureHeaders` middleware in `NewHandler`, applied to all responses:
   ```
   Strict-Transport-Security: max-age=31536000; includeSubDomains
   X-Frame-Options: DENY
   X-Content-Type-Options: nosniff
   Referrer-Policy: strict-origin-when-cross-origin
   Content-Security-Policy: default-src 'self'; script-src 'self'; img-src 'self' data: blob: https:; style-src 'self' 'unsafe-inline'; object-src 'none'; frame-ancestors 'none'
   ```
3. Also set these in `nginx.conf` (`add_header ... always;`) so static SPA assets are covered even when the request never reaches the Go server.

## Verification

- A cross-origin `fetch` from `https://evil.example` to `/api/v1/courses` is rejected by the browser (no `Access-Control-Allow-Origin: evil.example`).
- `curl -I https://<host>/` shows all five headers.
- Loading the app inside an `<iframe>` on a foreign origin is blocked.
