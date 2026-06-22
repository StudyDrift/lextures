# 01 — LTI 1.3 Deep Linking 2.0 and platform-initiated launch return 501

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised capability is stubbed)
- **Area:** Integrations / LTI 1.3 consumer side (plan 2.12, 16.x)

## Summary

`README.md` advertises **"LTI 1.3 provider/consumer support for Canvas, Moodle, and
Blackboard."** The **provider** side (Lextures acting as a tool: OIDC login, launch, NRPS,
AGS scores, JWKS) is implemented, and there is a working "consumer frame" embed path. But
the standard **LTI 1.3 consumer / platform-initiated flow** is stubbed — three endpoints
return `501 Not Implemented`:

| Endpoint | Purpose | Status |
|----------|---------|--------|
| `POST /api/v1/lti/deep-link` | Deep Linking 2.0 (instructor picks content items from an external tool) | **501** |
| `GET  /api/v1/lti/callback` | LTI consumer OIDC callback | **501** |
| `POST /api/v1/lti/launch/{registration_id}` | Platform-initiated launch by registration | **501** |

## Evidence

`server/internal/httpserver/saml_lti.go`:

```go
// line 105-107 — routes wired to 501 stubs
r.Post("/api/v1/lti/deep-link", d.lti501DeepLink())
r.Get("/api/v1/lti/callback", d.lti501Callback())
r.Post("/api/v1/lti/launch/{registration_id}", d.lti501LaunchReg())
```

```go
// line 118-155
func (d Deps) lti501DeepLink() http.HandlerFunc { /* ... */ lti501JSON("Deep Linking 2.0 handler not yet implemented.")(w, r) }
func (d Deps) lti501Callback()  http.HandlerFunc { /* ... */ lti501JSON("LTI consumer OIDC callback not yet implemented.")(w, r) }
func (d Deps) lti501LaunchReg() http.HandlerFunc { /* ... */ lti501JSON("LTI platform launch initiation not yet implemented.")(w, r) }
```

The comment on the route registration is explicit (`server/internal/httpserver/saml_lti.go:96`):

```go
// LTI: JWKS, provider OIDC, NRPS, AGS, consumer frame; 501 for Rust-stub LTI subroutes.
```

## UI exists — and points at the stubs

The admin/instructor UI lets you **register external LTI tools and add them to modules**,
which sets up an expectation that they can be launched:

- `clients/web/src/components/settings/lti-tools-settings-panel.tsx` — register external tools
- `clients/web/src/pages/lms/module-lti-link-modal.tsx` — attach an LTI tool to a module
- `clients/web/src/pages/lms/course-module-lti-page.tsx` — renders the launch frame

The module LTI page launches via `GET /api/v1/lti/consumer/frame` (an embed-ticket path
that **is** implemented at `server/internal/httpserver/lti_http.go:524`), so the simplest
embed works. But **Deep Linking** (the normal way an instructor selects specific content
from the remote tool) and the **registration-based platform launch / OIDC callback** flow
are not available — any client that drives them gets a 501.

## Impact

- Deep Linking content selection — the headline LTI 1.3 instructor workflow — is
  unavailable. Instructors must hand-configure launch URLs instead of picking items.
- Interop with platforms that require the standard OIDC third-party-initiated login +
  callback handshake will fail.
- `README.md`'s "consumer support" claim is only partially true.

## Suggested fix

1. Implement `deep-link` (Deep Linking 2.0 response signing / content-item return),
   `callback` (consumer OIDC state validation + id_token verification), and
   `launch/{registration_id}` (third-party-initiated login → auth redirect) per the
   IMS LTI 1.3 + Deep Linking 2.0 spec, reusing the existing key/JWKS plumbing in
   `server/internal/lti` and `server/internal/httpserver/lti_http.go`.
2. Until implemented, either hide the Deep Linking affordances in the web UI or surface a
   clear "not available on this server" message instead of letting the request 501.
3. Update `README.md` to scope the consumer claim to "consumer frame embed" if Deep
   Linking will not ship soon.

## Acceptance criteria

- An instructor can run a Deep Linking round-trip against a test tool (e.g. Moodle) and
  receive selected content items back.
- A platform-initiated OIDC login → callback → launch completes end to end.
- No `/api/v1/lti/*` route returns 501 when LTI is enabled.
