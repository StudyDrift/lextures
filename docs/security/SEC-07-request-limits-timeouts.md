# SEC-07 — Unbounded request bodies + missing server timeouts

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Server
- **Files:** [server/internal/app/app.go:123](../../server/internal/app/app.go) (`http.Server` construction), many handlers using `json.NewDecoder(r.Body)` / `io.ReadAll(r.Body)`

## Problem

The HTTP server is constructed with no timeouts at all:

```go
srv := &http.Server{
    Addr:    cfg.HTTPAddr,
    Handler: httpserver.NewHandler(deps),
}
```

There is no `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, or `MaxHeaderBytes`. Separately, most JSON handlers read the body with `io.ReadAll(r.Body)` / `json.NewDecoder(r.Body).Decode(...)` with no size cap — only a few endpoints (SCIM, the originality webhook) wrap the body in an `io.LimitReader`. The new transcript handlers also use uncapped `io.ReadAll(r.Body)`.

## Risk

- **Slowloris**: a client that dribbles request headers ties up a connection indefinitely (no `ReadHeaderTimeout`).
- **Memory exhaustion**: a multi-gigabyte JSON POST is read fully into memory before the handler can reject it.
- **Connection exhaustion**: no `IdleTimeout` lets idle keep-alive connections accumulate.

A single unauthenticated client can degrade or take down the API. Denial of availability is in scope for a hardening review even though it is not data exfiltration.

## Fix

1. Set server timeouts in `app.go`:
   ```go
   srv := &http.Server{
       Addr:              cfg.HTTPAddr,
       Handler:           httpserver.NewHandler(deps),
       ReadHeaderTimeout: 10 * time.Second,
       IdleTimeout:       60 * time.Second,
       MaxHeaderBytes:    1 << 20,
   }
   ```
   Set `ReadTimeout`/`WriteTimeout` to bound normal requests. The Canvas-import WebSocket and any long-poll endpoints need a separate `http.Server` or per-handler deadline so they are not killed by a global `WriteTimeout`.
2. Add a `bodyLimit(maxBytes)` middleware wrapping `r.Body` in `http.MaxBytesReader` (default 1 MiB) applied globally, with explicit larger opt-ins for known-large endpoints (file/H5P upload).

## Verification

- A slow-header client is dropped after `ReadHeaderTimeout`.
- A 50 MB JSON POST to `/api/v1/auth/login` returns `413`, not an OOM.
- File/H5P uploads still succeed at their documented size limits.
