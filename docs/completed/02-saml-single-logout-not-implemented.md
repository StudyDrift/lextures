# 02 — SAML Single Logout (SLO) endpoint returns 501

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised SSO capability is stubbed)
- **Area:** Identity / SAML 2.0 SSO (plan 4.1)
- **Status:** Fixed (2026-06-22)

## Summary

SAML 2.0 SSO is implemented for metadata, login redirect, and ACS (assertion consumer
service). **Single Logout (SLO)** was registered as a route but always returned
`501 Not Implemented`.

## Fix

Implemented SAML 2.0 Single Logout in `server/internal/browsersaml/slo.go` and wired it
through `handleSAMLSLO` in `server/internal/httpserver/saml_lti.go`.

`GET` and `POST /auth/saml/slo` now handle:

| Flow | Behaviour |
|------|-----------|
| **IdP-initiated LogoutRequest** (`SAMLRequest`) | Validates IdP signature, revokes all refresh tokens and bumps `jwt_session_version` for the user identified by `NameID`, returns a signed `LogoutResponse` via HTTP-POST binding |
| **IdP LogoutResponse** (`SAMLResponse`) | Validates response signature (redirect or POST binding) and redirects the browser to the app origin |
| **SP-initiated logout** (`GET ?idpId=&nameId=`) | Revokes local sessions, then redirects to the IdP SLO URL with a signed `LogoutRequest` |

Supporting changes:

- `IDPMetadataXMLFromRow` now emits `SingleLogoutService` endpoints when `slo_url` is configured
- `samlidp.GetIDPByEntityID` resolves IdP rows by `entity_id` for logout request validation

Session revocation uses `authservice.RevokeAllSessionsForUser` (refresh token revocation +
`jwt_session_version` bump), matching plan 4.8/4.9 session invalidation.

Tests: `server/internal/browsersaml/slo_test.go`, updated `saml_lti_nodb_test.go`.

## Acceptance criteria

- An IdP-initiated `LogoutRequest` invalidates the corresponding Lextures session and
  returns a valid `LogoutResponse`.
- `POST /auth/saml/slo` no longer returns 501 when SAML is enabled.