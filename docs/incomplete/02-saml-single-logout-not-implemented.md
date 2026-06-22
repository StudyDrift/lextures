# 02 — SAML Single Logout (SLO) endpoint returns 501

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised SSO capability is stubbed)
- **Area:** Identity / SAML 2.0 SSO (plan 4.1)

## Summary

SAML 2.0 SSO is implemented for metadata, login redirect, and ACS (assertion consumer
service). **Single Logout (SLO)** is registered as a route but always returns
`501 Not Implemented`, so a SAML logout initiated by the IdP (or a Lextures-initiated SLO)
does nothing.

## Evidence

`server/internal/httpserver/saml_lti.go`:

```go
// line 14-19 — SLO route is wired
func (d Deps) registerSAMLBrowserRoutes(r chi.Router) {
    r.Get("/auth/saml/metadata", d.handleSAMLMetadata())
    r.Get("/auth/saml/login", d.handleSAMLLoginGet())
    r.Post("/auth/saml/acs", d.handleSAMLACS())
    r.Post("/auth/saml/slo", d.handleSAMLSLO())   // <-- always 501
}
```

```go
// line 77-85
func (d Deps) handleSAMLSLO() http.HandlerFunc {
    _ = d
    return func(w http.ResponseWriter, r *http.Request) {
        // Parity: server/src/routes/saml.rs saml_slo_unimplemented
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.WriteHeader(http.StatusNotImplemented)
        _, _ = w.Write([]byte("SAML Single Logout is not implemented yet."))
    }
}
```

## Impact

- IdP-initiated logout does not terminate the Lextures session; the local session lives
  until its own expiry. This is a security/compliance gap for enterprise SAML deployments
  that rely on SLO to cut access on the IdP side.
- Lextures cannot propagate a logout back to the IdP.

## Suggested fix

- Implement SLO request/response handling (parse `LogoutRequest`, validate signature,
  invalidate the local session/refresh token, and emit a signed `LogoutResponse`), reusing
  the SAML plumbing in `server/internal/browsersaml` and the session/refresh-revocation
  machinery (plan 4.8/4.9).
- If SLO is intentionally out of scope, remove the route and document the limitation in the
  SAML section of the self-hosting docs rather than returning a 501 from a wired endpoint.

## Acceptance criteria

- An IdP-initiated `LogoutRequest` invalidates the corresponding Lextures session and
  returns a valid `LogoutResponse`.
- `POST /auth/saml/slo` no longer returns 501 when SAML is enabled.
