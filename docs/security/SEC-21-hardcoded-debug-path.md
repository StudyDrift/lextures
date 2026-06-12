# SEC-21 — Hardcoded developer-machine debug path in shipped binary

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** Server
- **File:** [server/internal/httpserver/canvas_agent_debug_log.go:13](../../server/internal/httpserver/canvas_agent_debug_log.go)

## Problem

```go
const canvasAgentDebugLogPath = "/Users/willdech/Documents/lextures/.cursor/debug-054d1d.log"
```

A developer's absolute home-directory path is compiled into the server binary. In production the path doesn't exist, so the debug write silently no-ops, but the constant is a leftover that should not ship.

## Risk

Low. It leaks a developer username/layout (minor information disclosure) and is a code-smell indicating debug instrumentation that was meant to be removed. If a similarly-constructed path were ever writable in an environment, it could become an unintended file-write sink.

## Fix

- Delete the debug-log file/feature, or
- Gate it behind a build tag (`//go:build dev`) and/or drive the path from an env var that is unset in production.

## Verification

- `grep -rn '/Users/' server/internal` returns nothing.
- The production binary contains no developer-specific filesystem paths.
