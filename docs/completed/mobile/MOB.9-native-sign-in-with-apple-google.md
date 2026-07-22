# MOB.9 — Native Sign in with Apple (& Google Sign-In on Android)

> Implementation plan. Source: App Store Review Guideline 4.8 (Login Services) compliance gap +
> mobile ↔ web auth parity. Related shipped plan: [`M1.1-sso-mfa-magic-link`](../../completed/mobile/M1.1-sso-mfa-magic-link.md).
> Code references: `clients/ios/Lextures/Features/Auth/*`, `clients/android/app/.../features/auth/*`,
> `server/internal/service/oidcauth/*`, `server/internal/httpserver/{auth.go,oidc_routes.go}`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.9 |
| **Section** | Mobile parity / Identity & SSO |
| **Severity** | BLOCKER (iOS App Store) / MINOR (Android parity) |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team + Identity/Backend |
| **Depends on** | Existing OIDC service (`oidcauth`), Apple key material config (`OIDC_APPLE_*`) — both shipped |
| **Unblocks** | iOS App Store submission / re-submission without 4.8 rejection |

---


## Implementation notes (2026-07-21)

Native Sign in with Apple (iOS) and Continue with Google (Android) for App Store Guideline 4.8 and mobile parity.

- **Server**: `POST /api/v1/auth/oidc/apple/native`, `POST /api/v1/auth/oidc/google/native`; shared `finishProviderIdentityLogin` in `oidcauth`; audiences via `OIDC_APPLE_NATIVE_AUDIENCE` / `OIDC_GOOGLE_NATIVE_AUDIENCE` (fallback web Google client ID).
- **Availability**: always on when audiences resolve — Apple defaults to `com.lextures.ios`; Google when `OIDC_GOOGLE_CLIENT_ID` / `OIDC_GOOGLE_NATIVE_AUDIENCE` is set. No feature flag. Status fields `appleNative` / `googleNative` on `GET /api/v1/auth/oidc/status`.
- **iOS**: `SignInWithAppleButton` on login/signup, entitlement `com.apple.developer.applesignin`, nonce SHA-256, `AuthAPI.nativeAppleSignIn`.
- **Android**: Credential Manager `GetSignInWithGoogleOption`, `GOOGLE_SERVER_CLIENT_ID` in `local.properties`, `AuthApi.nativeGoogleSignIn`.
- **Rate limit**: native paths on the auth-sensitive limiter (same as login).
- **Tests**: Go unit tests (`oidcauth`, config, httpserver status); iOS `AppleSignInLogicTests`; Android `GoogleSignInLogicTest`.

## 1. Problem Statement

The iOS and Android apps already offer third-party/social login (Google and Microsoft via OIDC) — but only when a school tenant has explicitly enabled those providers, and only as a generic web-redirect button through `ASWebAuthenticationSession` / Custom Tabs. Apple **App Store Review Guideline 4.8** requires that any app offering a third-party or social login to set up or authenticate the primary account **must also offer Sign in with Apple as an equivalent, equally prominent option**, implemented with the **native** `AuthenticationServices` flow — not a web redirect. Today we ship neither the native flow nor an always-present button, so the app is exposed to a 4.8 rejection on submission. The fix is to add a native Sign in with Apple button to the login and account-creation screens (with an equivalent native "Continue with Google" on Android for parity), backed by a server endpoint that verifies the provider identity token and issues Lextures tokens.

## 2. Goals

- Ship a **native Sign in with Apple** button on the iOS login and signup screens using `AuthenticationServices` (`SignInWithAppleButton` / `ASAuthorizationController`), present on the default self-learner path regardless of tenant OIDC config.
- Add a server endpoint that verifies a natively obtained Apple identity token (audience = app bundle ID, not the web Services ID) and issues Lextures access/refresh tokens by reusing the existing OIDC identity-linking logic.
- Provide an equivalent **native "Continue with Google"** button on Android via Credential Manager, verified server-side against the same account model.
- Satisfy Guideline 4.8: name/email-only data collection, private-relay email support, no ad tracking, button prominence ≥ other social buttons.
- Reuse — not fork — the existing find-or-create/link identity path so native and web sign-in resolve to the same account.

## 3. Non-Goals

- No native Sign in with Apple on Android (Apple's native SDK is Apple-platform only; Android keeps the existing web-redirect Apple button for parity where a tenant enables it).
- No change to school SSO paths (SAML, Clever, ClassLink) or the forced-SAML tenant behavior.
- No migration of existing password/magic-link users to social login; native social login is additive.
- No web (browser) client changes — the web app's OIDC redirect flow is unchanged.
- No new account *types*; native social sign-up creates a standard individual (self-learner) account, consistent with today's `/signup` default.

## 4. Personas & User Stories

- **As a self-learner on iPhone**, I want to tap "Sign in with Apple" on the login or create-account screen so I can start without typing a password or exposing my real email.
- **As a returning Apple user**, I want a subsequent tap to log me straight back into my existing Lextures account (Apple `sub` already linked).
- **As an Android self-learner**, I want a one-tap "Continue with Google" so account setup is equally fast on my platform.
- **As a student with a school-provisioned account**, I want the school SSO buttons to still take priority when my tenant forces SSO, and social buttons to not create a shadow duplicate account.
- **As an admin / compliance owner**, I want assurance that native social sign-in collects only name + email and supports Apple private relay, so we pass App Review and privacy review.

## 5. Functional Requirements

- **FR-1.** The iOS login and signup screens MUST display a native Sign in with Apple button using the system `SignInWithAppleButton`, respecting the Human Interface Guidelines (official style, corner radius, localized label, min height).
- **FR-2.** When any other social/third-party login button is visible on a screen, the Sign in with Apple button MUST be at least as prominent (equal or higher placement, equal size).
- **FR-3.** The iOS client MUST generate a cryptographically random raw nonce, set `ASAuthorizationAppleIDRequest.nonce = SHA256(rawNonce)`, request `.fullName` and `.email` scopes, and send the raw nonce to the server for verification.
- **FR-4.** The server MUST expose `POST /api/v1/auth/oidc/apple/native` accepting `{ id_token, raw_nonce, authorization_code?, full_name?, email? }` and MUST verify: Apple JWKS signature, `iss == https://appleid.apple.com`, `aud ∈ {configured native audience(s)}`, `exp`/`iat`, and `nonce == SHA256(raw_nonce)`.
- **FR-5.** On a verified token the server MUST resolve the account via the existing identity path: find identity by `(provider="apple", sub)`; else link to an existing user by verified email; else create a new individual account — then issue Lextures tokens (honoring MFA / `requires_mfa`).
- **FR-6.** The server MUST persist Apple-provided `full_name`/`email` **only on first authorization** (Apple omits them on subsequent sign-ins) and MUST accept Apple private-relay addresses (`@privaterelay.appleid.com`) as valid emails.
- **FR-7.** The Android login and signup screens MUST display a native "Continue with Google" button using Credential Manager (`GetSignInWithGoogleOption`), returning a Google ID token.
- **FR-8.** The server MUST expose `POST /api/v1/auth/oidc/google/native` verifying the Google ID token (Google JWKS, `iss ∈ {accounts.google.com, https://accounts.google.com}`, `aud == configured Google server client ID`, nonce) and resolve the account through the same identity path as FR-5.
- **FR-9.** Native social buttons MUST be available on the default (self-learner) path independent of tenant OIDC status; where a tenant forces SSO, the existing forced-SSO behavior MUST continue to take precedence.
- **FR-10.** User-visible cancellation (`ASAuthorizationError.canceled` / Credential Manager cancel) MUST return the user to the form silently; all other failures MUST surface a localized, non-technical error.
- **FR-11.** The app MUST send the same client metadata (device UA, etc.) it sends on password login so sessions are labeled consistently.
- **FR-12.** The native endpoints SHOULD be rate-limited consistently with `/api/v1/auth/login`.

## 6. Non-Functional Requirements

- **Performance** — Token verification p95 < 300 ms server-side; Apple/Google JWKS fetched once and cached (respecting `Cache-Control`, refresh on unknown `kid`). Sign-in tap → session established p95 < 3 s on LTE.
- **Security** — Identity token signature + `iss`/`aud`/`exp`/`nonce` verified server-side; the raw nonce is single-use and never logged. Audience for the Apple *native* flow is the **bundle ID** (`com.lextures.ios`), distinct from the web **Services ID** (`OIDC_APPLE_CLIENT_ID`); accepting the wrong audience is a hard fail. Access/refresh tokens minted only through the existing `authservice.AuthResponseForUser`.
- **Privacy & Compliance** — Data collection limited to name + email (Guideline 4.8; GDPR data-minimization). Private-relay emails supported end-to-end. No advertising/interaction tracking tied to the login. Update the privacy policy + App Store privacy nutrition labels ("Name", "Email Address", "User ID" linked to identity, used for App Functionality only). COPPA: native self-signup creates individual accounts; under-13 school provisioning stays on the SSO path.
- **Accessibility** — Native buttons inherit system VoiceOver/TalkBack labels and Dynamic Type/font scaling; WCAG 2.1 AA contrast met by the system button styles; keyboard/switch-control operable.
- **Scalability** — Stateless verification; only new rows are OIDC identity links (already modeled). No new hot path.
- **Reliability** — JWKS fetch failures degrade gracefully with a retriable error message; idempotent: repeat sign-in with the same `sub` never creates a duplicate account.
- **Observability** — Counters `auth_native_signin_{start,success,cancel,error}` tagged by `provider` and `outcome` (no PII, no token contents); errors traced with a redacted reason. Wire through the existing telemetry layer (`server/internal/telemetry`).
- **Maintainability** — Extract a shared `finishProviderIdentityLogin(...)` in `oidcauth` so web `CompleteLogin` and the native handlers share the find-or-create/link logic.
- **Internationalization** — All new client copy externalized (`Localizable.xcstrings` for iOS, `strings.xml`/mobile locale sync for Android). Apple/Google button labels use the OS-localized system strings.
- **Backward compatibility** — Additive: new endpoints + new config keys with safe defaults; existing web OIDC and password flows unchanged. Feature gated behind a flag for staged rollout.

## 7. Acceptance Criteria

- **AC-1.** *Given* a fresh iPhone with no Lextures account, *When* the user taps Sign in with Apple on the signup screen and authorizes with name + email, *Then* a new individual account is created, name/email are persisted, and the app lands authenticated on the home screen.
- **AC-2.** *Given* an existing account already linked to an Apple `sub`, *When* the user taps Sign in with Apple again, *Then* they are signed into the same account (no duplicate) and Apple returns no name/email the second time without error.
- **AC-3.** *Given* an existing password account with email `x@y.com`, *When* the user signs in with Apple returning verified email `x@y.com`, *Then* the Apple identity links to that account (per link rules) rather than creating a new one.
- **AC-4.** *Given* a tampered or expired `id_token`, or a `nonce`/`aud` mismatch, *When* it reaches the native endpoint, *Then* the server rejects with 401/400 and no session is issued.
- **AC-5.** *Given* the user taps Sign in with Apple and cancels the system sheet, *Then* the app returns to the form with no error banner.
- **AC-6.** *Given* an account with MFA enabled resolves via Apple, *When* the endpoint returns `requires_mfa`, *Then* the app routes to the MFA challenge (same as password login).
- **AC-7.** *Given* an Android device, *When* the user taps Continue with Google and selects an account, *Then* the Google ID token is verified server-side and a session is issued to the same account model as web Google OIDC.
- **AC-8.** *Given* Apple returns a private-relay email, *Then* sign-in succeeds and the relay address is stored as the account email.
- **AC-9.** *Given* the App Review default (self-learner) path, *Then* Sign in with Apple is visible and at least as prominent as any other social button on both login and signup — verified in an automated snapshot/UI test.

## 8. Data Model

- **No new tables.** Reuse the existing OIDC identity link table accessed by `oidcrepo.FindIdentityByProviderAndSub` / `TryInsertIdentity` (`provider`, `sub`, `user_id`, `email`). Apple native uses `provider = "apple"`, Google native uses `provider = "google"` — same rows as the web OIDC flow, so a user who linked via web is recognized natively and vice-versa.
- Persist Apple first-authorization `full_name` onto the user profile display name **only if empty** (do not overwrite an existing name).
- If a migration is needed to widen the `email` column for relay addresses or add a `linked_via` audit column, follow the repo convention `server/migrations/NNN_*.sql`; expected: none required (relay emails fit existing constraints).
- **Backfill** — none; additive links created lazily on first native sign-in.

## 9. API Surface

New routes (registered in `server/internal/httpserver/oidc_routes.go`, wired in `registerAuthRoutes`):

- `POST /api/v1/auth/oidc/apple/native` — auth scope: public (pre-auth).
  - Request: `{ "id_token": string, "raw_nonce": string, "authorization_code": string?, "full_name": string?, "email": string? }`
  - Response: `AuthTokenResponse` (same shape as `/api/v1/auth/login`: `access_token`, `refresh_token`, `expires_in`, `requires_mfa`, `mfa_pending_token`, `user`).
- `POST /api/v1/auth/oidc/google/native` — auth scope: public.
  - Request: `{ "id_token": string, "raw_nonce": string? }`
  - Response: `AuthTokenResponse`.
- (Optional consolidation) implement both as `POST /api/v1/auth/oidc/{provider}/native` if it reads cleaner alongside the existing `{provider}` web routes.
- `GET /api/v1/auth/oidc/status` — extend so the client knows native providers are available on the default path (e.g. add `appleNative`/`googleNative` booleans); the native buttons themselves do not require a tenant to enable OIDC.
- Rate-limit parity with `/api/v1/auth/login`. Document all new routes in the OpenAPI spec (`server/internal/openapi/openapi.go`).

**New server config** (env, `server/internal/config/config.go`):
- `OIDC_APPLE_NATIVE_AUDIENCE` — comma-separated allowed audiences for the native token (default `com.lextures.ios`; may include `com.lextures.ios.tests` for CI).
- `OIDC_GOOGLE_NATIVE_AUDIENCE` (or reuse the existing Google web client ID as the Android `serverClientId`).

## 10. UI / UX

**iOS** (`Features/Auth/LoginView.swift`, `SignupView.swift`):
- Add a "Continue with" social section at the top of the `AuthCard` (above the password form on login, above the fields on signup) containing the native `SignInWithAppleButton` and, where shown, existing provider buttons — Apple first / most prominent.
- New controller `AppleSignInController` (sibling of `SSOAuthController.swift`) wrapping `ASAuthorizationController`, nonce generation (raw + SHA256), credential handling, and the POST to the native endpoint; on success calls `session.applyTokenResponse` exactly like `startSSO`.
- **States** — loading (button disabled + spinner), error (localized banner reusing `errorMessage`), cancel (silent return), MFA (route to `onMfaRequired`). Offline → existing "server unreachable" message.
- **Copy / i18n** — button uses the system-localized Apple label; supporting strings (`auth.social.dividerOr`, error strings) added to `Resources/Localizable.xcstrings` (already staged/dirty — extend it).
- **Entitlement** — add the "Sign in with Apple" capability: `com.apple.developer.applesignin = ["Default"]` in `clients/ios/Lextures/Resources/Lextures.entitlements`, and enable the capability on the App ID.

**Android** (`features/auth/LoginScreen.kt`, `SignupScreen.kt`, new `GoogleSignIn.kt` beside `SsoAuth.kt`):
- Add a native "Continue with Google" button (Credential Manager `GetSignInWithGoogleOption`) in the same social section; keep the existing web-redirect Apple button available where a tenant enables it.
- Same loading/error/cancel/MFA states; strings via mobile locale sync (`scripts/sync-mobile-locales.py`).

**Accessibility annotations** — buttons expose native accessibility labels; focus order: social section → divider → password form → footer links; contrast handled by system button styles.

## 11. AI / ML Considerations

Not AI-touching. Skip.

## 12. Integration Points

- **Apple** — `AuthenticationServices` framework (native); token verification against `https://appleid.apple.com/auth/keys` (JWKS) and issuer `https://appleid.apple.com`. Existing `OIDC_APPLE_*` key material (`server/internal/service/oidcauth/apple_secret.go`) reused for optional `authorization_code` exchange.
- **Google** — Android Credential Manager + Google Identity Services; verification against Google JWKS (`https://www.googleapis.com/oauth2/v3/certs`).
- **Internal modules** — `server/internal/service/oidcauth/oidcauth.go` (extract shared finish helper), `server/internal/repos/oidc/identity_ext.go` (find/insert identity), `server/internal/service/authservice` (`AuthResponseForUser`, MFA), `server/internal/httpserver/{auth.go,oidc_routes.go}`, `server/internal/config`, `server/internal/openapi`. iOS `Core/Auth/{AuthAPI.swift,AuthSession.swift}`; Android `core/auth/{AuthApi.kt,AuthSession.kt}`.
- **Emissions** — telemetry counters only; no webhooks.

## 13. Dependencies & Sequencing

- **Must ship after:** existing OIDC service + Apple key material config (already shipped); confirm the Apple Developer account has an App ID with Sign in with Apple enabled and (for web parity) a Services ID.
- **Must ship before:** next iOS App Store submission that includes any social login (to avoid 4.8 rejection).
- **Shared infra:** Apple/Google developer console configuration (App ID capability, Services ID, Google OAuth client IDs); server config secrets; feature flag plumbing.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Wrong audience check (Services ID vs bundle ID) rejects all native tokens | M | H | Explicit `OIDC_APPLE_NATIVE_AUDIENCE` config + unit tests asserting bundle-ID audience; integration test with a real device token in staging |
| Apple omits email/name on non-first sign-in and code treats it as an error | M | H | Only require email on first link; on repeat, resolve by stored `sub`; test AC-2 |
| Duplicate account when email differs from an existing password account | M | M | Deterministic link rules (by `sub`, then verified email) reused from `CompleteLogin`; document precedence |
| Private-relay email later relay-broken / rotated | L | M | Store `sub` as the stable key (not email); email is secondary |
| App Review still flags prominence/placement | L | H | Follow HIG button style, place Apple first, add snapshot test (AC-9); pre-submission checklist |
| JWKS fetch/caching bug causes intermittent verification failures | L | M | Cache with kid-miss refresh + retry; alert on elevated `auth_native_signin_error` |
| Google native adds Play Services complexity | M | L | Isolate behind Credential Manager; graceful fallback to existing web-redirect Google button |

## 15. Rollout Plan

- **No feature flag** — native buttons show whenever `/oidc/status` reports `appleNative` / `googleNative` (Apple by default audience; Google when server client ID is configured).
- **Sequencing** — (1) land server endpoints + config + tests; (2) configure Apple App ID capability + Google client IDs in each environment; (3) ship iOS/Android clients; (4) dogfood on TestFlight / internal track; (5) GA.
- **Dogfood / pilot** — internal team + a small self-learner cohort; verify AC-1…AC-9 on real devices incl. a private-relay Apple ID.
- **GA criteria** — all ACs pass on physical devices; App Review passes on a TestFlight external build; error rate < 1%.
- **Rollback** — clear `OIDC_GOOGLE_*` / override audiences only if needed; endpoints are additive.

## 16. Test Plan

- **Unit** — nonce hashing (raw → SHA256), audience allow-list parsing, JWKS `kid` selection, claim validation (`iss`/`aud`/`exp`/`nonce`), first-auth vs repeat name/email handling, private-relay email acceptance. Extract-and-test the shared `finishProviderIdentityLogin`.
- **Integration (DB/API)** — new account creation, link-by-`sub`, link-by-email, MFA-required path, duplicate-prevention, invalid/expired/tampered token rejection. Follow existing `server/internal/httpserver/auth_db_test.go` patterns.
- **End-to-end** — API-level smoke for both native endpoints (mirror `e2e/tests/mobile-*`), asserting `AuthTokenResponse` and session issuance.
- **Security** — token-forgery, wrong-audience, replayed-nonce, `iss` spoofing, rate-limit; confirm raw nonce/tokens never logged.
- **Accessibility** — VoiceOver/TalkBack label presence, Dynamic Type scaling, contrast; automated axe-equivalent not applicable to native buttons — manual screen-reader script.
- **UI** — iOS snapshot test asserting Apple button presence + prominence on login and signup (AC-9); cancel-returns-silently test.
- **Manual exploratory** — real-device matrix: fresh Apple ID (name+email), returning Apple ID (no email), private-relay ID, Android Google account picker, offline, MFA account.

## 17. Documentation & Training

- **End-user** — help-center article "Sign in with Apple / Google on mobile", incl. private relay explanation and "forget this Apple ID" guidance.
- **Admin** — note in identity docs that native social sign-in creates individual accounts and does not bypass tenant forced-SSO.
- **API reference** — add the two native endpoints + new config keys to OpenAPI and the auth section of internal API docs.
- **Runbook** — update the auth runbook: JWKS caching, rotating Apple key material, new config env vars, and the App Review 4.8 pre-submission checklist. Update App Store privacy nutrition labels + privacy policy.

## 18. Open Questions

1. Consolidate to `POST /api/v1/auth/oidc/{provider}/native`, or keep separate per-provider routes? (Leaning: `{provider}` for consistency with web routes.)
2. Should native social sign-in be offered on the school-tenant path at all, or restricted to self-learner? Apple 4.8 applies app-wide, so the safe answer is "always available on the default path," but confirm with K-12 provisioning owners re: shadow accounts.
3. Do we exchange the Apple `authorization_code` for refresh tokens server-side (enabling server-initiated revocation checks), or is verifying the `id_token` sufficient for v1?
4. Google native audience: reuse the existing web Google client ID as Android `serverClientId`, or mint a dedicated Android OAuth client?
5. Should first-authorization Apple name populate the profile display name automatically, or leave it for the user to set?
6. Account deletion / "Sign in with Apple" unlink: does Apple's requirement to support account deletion in-app affect scope here (likely a separate plan)?

## 19. References

- Existing files: `clients/ios/Lextures/Features/Auth/{LoginView.swift,SignupView.swift,SSOAuthController.swift}`, `clients/ios/Lextures/Core/Auth/{AuthAPI.swift,AuthConstants.swift,AuthCallbackParser.swift}`, `clients/ios/Lextures/Resources/Lextures.entitlements`, `clients/android/app/.../features/auth/{LoginScreen.kt,SignupScreen.kt,SsoAuth.kt}`, `server/internal/service/oidcauth/{oidcauth.go,apple_secret.go}`, `server/internal/repos/oidc/identity_ext.go`, `server/internal/httpserver/{auth.go,oidc_routes.go}`, `server/internal/config/config.go`.
- External standards: Apple App Store Review Guideline 4.8 (Login Services); Human Interface Guidelines — Sign in with Apple; Apple "Sign in with Apple" REST API & token verification; Google Identity — Sign in with Google on Android (Credential Manager) & ID-token verification; OIDC Core 1.0; RFC 7519 (JWT); RFC 7517 (JWK).
- Related plans: [`../../completed/mobile/M1.1-sso-mfa-magic-link.md`](../../completed/mobile/M1.1-sso-mfa-magic-link.md), [`../../completed/mobile/M1.2-biometric-sessions.md`](../../completed/mobile/M1.2-biometric-sessions.md).
