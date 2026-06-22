# 01 — LTI 1.3 Deep Linking 2.0 and platform-initiated launch return 501

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised capability is stubbed)
- **Area:** Integrations / LTI 1.3 consumer side (plan 2.12, 16.x)
- **Status:** Fixed (2026-06-22)

## Summary

`README.md` advertises **"LTI 1.3 provider/consumer support for Canvas, Moodle, and
Blackboard."** The **provider** side (Lextures acting as a tool: OIDC login, launch, NRPS,
AGS scores, JWKS) is implemented, and there is a working "consumer frame" embed path. The
standard **LTI 1.3 consumer / platform-initiated flow** was previously stubbed — three
endpoints returned `501 Not Implemented`.

## Fix

Implemented the LTI 1.3 platform (consumer) OIDC and Deep Linking 2.0 handlers:

| Endpoint | Purpose | Implementation |
|----------|---------|----------------|
| `POST /api/v1/lti/launch/{registration_id}` | Third-party-initiated OIDC login to an external tool | `handleLtiPlatformLaunch` in `lti_consumer_http.go` |
| `GET  /api/v1/lti/callback` | Platform OIDC authentication response (issues signed `id_token`) | `handleLtiConsumerCallback` in `lti_consumer_http.go` |
| `POST /api/v1/lti/deep-link` | Deep Linking 2.0 response handler (verifies tool JWT, returns/creates content items) | `handleLtiDeepLink` in `lti_consumer_http.go` |
| `GET  /api/v1/lti/consumer/target` | Resource-link `target_link_uri` landing page | `handleLtiConsumerTarget` in `lti_consumer_http.go` |

JWT claim builders and signing helpers live in `server/internal/lti/consumer.go`, reusing
the existing platform RSA key pair (`server/internal/lti/keys.go`).

**Launch flow:** authenticated instructor POST → signed `login_hint` JWT → redirect to the
tool's OIDC initiation URL with `iss`, `target_link_uri`, `client_id`, and optional
`lti_message_hint` for deep linking.

**Callback flow:** tool redirects to `/api/v1/lti/callback` with standard OIDC auth-request
params → platform validates `login_hint`, enforces nonce replay protection, signs an LTI
`id_token` (`LtiResourceLinkRequest` or `LtiDeepLinkingRequest`), and auto-POSTs it to the
tool's `redirect_uri`.

**Deep linking flow:** tool POSTs `JWT` (LtiDeepLinkingResponse) → signature verified
against the tool's JWKS → `content_items` parsed; when `courseId`/`moduleId` are present in
the opaque `data` claim, module items are created automatically (`lti_link` or
`external_link`).

Unit tests: `server/internal/lti/consumer_test.go`,
`server/internal/httpserver/lti_consumer_nodb_test.go`,
`server/internal/httpserver/lti_consumer_callback_test.go`.

## Acceptance criteria

- An instructor can run a Deep Linking round-trip against a test tool (e.g. Moodle) and
  receive selected content items back.
- A platform-initiated OIDC login → callback → launch completes end to end.
- No `/api/v1/lti/*` route returns 501 when LTI is enabled.