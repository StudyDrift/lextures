# SEC-02 — Access + refresh tokens stored in `localStorage`

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Web client / session management
- **Files:** [clients/web/src/lib/auth.ts](../../clients/web/src/lib/auth.ts) (`ACCESS_TOKEN_KEY`), [clients/web/src/lib/session-tokens.ts](../../clients/web/src/lib/session-tokens.ts) (`REFRESH_TOKEN_KEY`)

## Problem

Both the access token (`studydrift_access_token`) and the refresh token (`studydrift_refresh_token`) are stored in `localStorage`:

```ts
localStorage.setItem(ACCESS_TOKEN_KEY, token)
localStorage.setItem(REFRESH_TOKEN_KEY, token)
```

`localStorage` is readable by **any** JavaScript running on the SPA origin. In production the SPA and the API are served from the *same* origin (nginx proxies `/api/` to the server container — see [clients/web/nginx.conf](../../clients/web/nginx.conf)), so any content served by the API origin shares this storage.

## Risk

This is the single most important finding for the ShinyHunters threat model. A single XSS anywhere on the origin — including the stored-XSS surfaces in SEC-05 (SVG branding) and SEC-06 (course-file MIME) — lets an attacker read the **refresh token** out of `localStorage` and exfiltrate it. Refresh tokens are long-lived and can be replayed out-of-band from attacker infrastructure to mint fresh access tokens indefinitely, surviving the victim closing the tab. That is precisely the Salesloft/Drift intrusion pattern: extract a long-lived token, replay it elsewhere.

## Fix

1. Issue the **refresh token** as an `HttpOnly; Secure; SameSite=Lax` cookie scoped to the refresh path (`/api/v1/auth/refresh`). It must never be readable from JavaScript.
2. Switch `authorizedFetch` (and the refresh flow) to `credentials: 'include'` so the cookie rides automatically.
3. If the access token must remain JS-readable for compatibility, keep it short-lived (≤ 15 min) and keep it in memory only (not `localStorage`), refreshing from the cookie. At minimum, the refresh token must move server-side.
4. Pair the cookie move with CSRF defenses (SameSite=Lax covers most; add a double-submit token or origin check on state-changing requests). Note CSRF is *not* a concern today because tokens are sent as bearer headers, but it becomes one the moment cookies are introduced — do both together.

## Verification

- After login, `localStorage.getItem('studydrift_refresh_token')` returns `null`; the refresh token is only visible as an `HttpOnly` cookie in DevTools → Application → Cookies.
- A scripted `fetch` from the page console cannot read the refresh token.
- The refresh endpoint succeeds with the cookie and 401s without it.
