# MB.1 — In-App Browser: Keep Mobile Links Inside Lextures

> Implementation plan. Source: product directive (2026-08-01) — "mobile links should stay within the
> Lextures app." Folder overview: [README](README.md). Template: [_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MB.1 |
| **Section** | Mobile Experience (MB) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | SHIPPED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad (iOS + Android), with trust & safety and privacy review |
| **Depends on** | Shipped `DeepLinkRouter` / `ContentLinkRouter` on both platforms |
| **Unblocks** | Consistent link handling for Content Tools (CT.M*), Boards (VC.M*), Marketplace, Credentials, Library/Textbook resources |

---

## 1. Problem Statement

Today almost every link a student or instructor taps on mobile ejects them from Lextures. iOS has 49
`openURL(…)` call sites across 43 files plus 5 raw `UIApplication.shared.open` calls; Android fires
`Intent(Intent.ACTION_VIEW …)` from 36 files. Each one hands the user to Safari or Chrome, loses the
navigation stack, and forces an app-switch to get back — the single most common way we lose a session
mid-lesson. The one in-app browser we do have, `WebItemView` / `WebItemScreen`, is scoped to course web
items, carries no browser chrome beyond an "Open externally" button, and injects a bearer token using a
string prefix check that a look-alike host can satisfy. MB.1 replaces all of it with one in-app browser
presented as a full-screen popover — the pattern users already know from X, Instagram and Facebook — and
one link-routing policy that decides, in exactly one place, whether a URL becomes native navigation, an
in-app page, or a deliberate handoff to the OS.

## 2. Goals

- Every `http(s)` link that is not a native destination opens in a **full-screen in-app browser** that
  preserves the screen underneath and returns the user to it on dismiss.
- **One** routing policy, expressed as pure testable logic mirrored on both platforms, replaces all ~90
  ad-hoc link call sites.
- Copying the URL and closing the browser are each a **single, obvious gesture** — no menu-diving.
- Links that must not be captured (SSO, checkout, native app deep links, `mailto:`/`tel:`, store links)
  keep working exactly as they do now, by explicit carve-out rather than by accident.
- The browser is safe for a K12 audience and for app review: no address bar, no script injection into
  third-party pages, no third-party URL in telemetry, and an admin policy that can turn it off.

## 3. Non-Goals

- A general-purpose browser: no address bar, no tabs UI, no bookmarks, no history browser, no
  user-entered URLs. The in-app browser only ever shows a page the app decided to open.
- Any form of interaction with third-party page content — no JS injection, no autofill bridging, no DOM
  reading, no form-field observation. (Explicitly out of scope; see FR-24.)
- Replacing the CT.M4 sandboxed WebView tool host — a different component with a different threat model.
- Desktop and web clients: they already open links in browser tabs and are unchanged.
- Offline caching of third-party pages.
- Ad blocking / content filtering beyond what the platform provides.

## 4. Personas & User Stories

- **As a student**, I want to tap a link in a content page, read it, and swipe back to the lesson without
  leaving the app or losing my place.
- **As a student**, I want to copy the link I am reading in one tap so I can paste it into my notes.
- **As a student on a slow connection**, I want to see that the page is loading and get a clear retry if
  it fails, rather than a blank screen.
- **As an instructor**, I want the external reading I attached to open the same way for everyone, on both
  platforms.
- **As a K12 administrator**, I want the ability to force links to the system browser (or block them)
  so our district's filtering and safe-browsing profile still applies.
- **As a parent**, I want to know my child is not being handed an unfiltered browser inside a school app.
- **As a security reviewer**, I want a guarantee that our bearer token never travels to a non-Lextures
  host.
- **As an accessibility user**, I want the browser to announce itself as a modal page, expose a labelled
  close control, and not depend on a swipe gesture I cannot perform.

## 5. Functional Requirements

**Routing policy (the single decision point)**

- **FR-1.** A single `LinkOpener` MUST be the only entry point for opening a URL from app UI; direct
  `openURL` / `ACTION_VIEW` calls MUST be removed from feature code and blocked by a lint rule (FR-26).
- **FR-2.** The classification MUST be a pure function — `MobileLinkPolicy` in
  `Core/LMS/…Logic.swift` (iOS) and `core/lms/…Logic.kt` (Android) — taking a URL plus policy state and
  returning one of: `native(destination)`, `inAppBrowser`, `systemBrowser`, `externalApp`,
  `authSession`, `blocked`.
- **FR-3.** URLs resolving to a native destination (`lextures://…`, `lextures.com` / `*.lextures.com`
  paths that `DeepLinkRouter` can resolve, and relative `/…` paths) MUST navigate natively and MUST NOT
  open a browser — preserving today's `ContentLinkRouter` behaviour.
- **FR-4.** Non-web schemes (`mailto:`, `tel:`, `sms:`, `geo:`/`maps:`, `itms-apps:`, `market://`) MUST
  hand off to the OS.
- **FR-5.** SSO/OIDC/SAML sign-in URLs MUST continue to use `ASWebAuthenticationSession` (iOS) and
  Custom Tabs (Android) — never the in-app browser — so the shared cookie jar and platform credential
  autofill keep working (`SsoAuth.kt`, iOS login flow).
- **FR-6.** Payment/checkout and billing-portal URLs MUST continue to open in the system browser
  surface (`BillingCheckout.kt`, `PurchaseFlow.swift`) to stay within Apple/Google payment rules.
- **FR-7.** Known native-app link targets — video-conference join links (Zoom/Meet/Teams), app-store
  links, and any URL for which the OS reports a registered universal/app link handler — MUST be offered
  to the installed app first and MUST NOT be captured by the in-app browser.
- **FR-8.** Any URL scheme not on the allowlist MUST be `blocked` and MUST NOT be passed to the OS.
- **FR-9.** The policy MUST be identical on iOS and Android, enforced by a shared fixture table
  (FR-25).

**The in-app browser**

- **FR-10.** The browser MUST present as a **full-screen cover over the current screen** (iOS
  `fullScreenCover` with an interactive sheet-style transition; Android full-screen route with the same
  slide-up motion), leaving the underlying screen mounted so dismissal restores it exactly.
- **FR-11.** The header MUST show, at all times: a **close control** in the leading position, the page
  **origin host** (registrable domain, with a lock indicator for `https` and an explicit "Not secure"
  treatment for `http`), the page **title** when known, and an **overflow** control trailing.
- **FR-12.** The header MUST show a determinate **progress indicator** tied to load progress and hide it
  on completion.
- **FR-13.** The user MUST be able to dismiss by (a) the close control, (b) a downward drag on the
  header/page, and (c) the platform back gesture / system back button when there is no in-page history.
  Drag-dismiss MUST be interactive and cancellable.
- **FR-14.** **Copy link MUST be reachable in one tap** from a persistent affordance in the header
  (tapping the host chip copies the current URL), MUST confirm with a toast plus haptic, and MUST also
  appear in the overflow menu for discoverability. The copied value MUST be the current page URL, not
  the URL the browser was opened with.
- **FR-15.** The overflow menu MUST contain: Copy link, Share…, Open in {Safari|Chrome}, Reload, and —
  where the source surface supports it — Report.
- **FR-16.** In-page navigation MUST work: back/forward gestures, and a back affordance that appears
  once in-page history exists. `target="_blank"` / `window.open` MUST open in the same in-app browser
  rather than being dropped.
- **FR-17.** Non-renderable responses (downloads, `Content-Disposition: attachment`, unsupported MIME
  types) MUST be handed to the existing file-preview/share path rather than showing a blank page.
- **FR-18.** Media MUST play inline with fullscreen and picture-in-picture support.
- **FR-19.** The browser MUST render explicit **loading**, **error** (with retry), **offline**, and
  **blocked-by-policy** states; a failed load MUST NOT leave a blank white screen.

**Security & privacy**

- **FR-20.** The `Authorization: Bearer` header MUST be attached **only** when the destination host
  exactly equals, or is a subdomain of, the configured API host — a parsed-host comparison, never a
  string prefix. This fixes the current `hasPrefix(apiBaseURL)` / `startsWith(apiBaseUrl)` checks in
  `WebItemView.swift:83` and `WebItemScreen.kt:60`, which a look-alike host such as
  `https://api.lextures.com.example.net/` satisfies today.
- **FR-21.** The header MUST NOT be re-attached across a cross-origin redirect; the redirect chain MUST
  be re-evaluated on every navigation.
- **FR-22.** The in-app browser MUST use a **single app-scoped web data store**, separate from the LMS
  API session, so a login inside the browser survives across opens within a session.
- **FR-23.** That data store — cookies, local storage, caches — MUST be purged on sign-out and MUST be
  purgeable from Settings via a **Clear browsing data** control.
- **FR-24.** The app MUST NOT inject scripts into, or read content from, third-party pages. Message
  handlers, user scripts and JS bridges MUST be absent on the in-app browser configuration (they belong
  only to the CT.M4 sandbox host, which is a separate component).
- **FR-25.** Telemetry MUST NOT record third-party URLs, hosts, paths or page titles. Only: the source
  surface, an `internal|external` classification, outcome and error class.
- **FR-26.** An org/platform policy `mobileLinkHandling ∈ {in_app, system, blocked}` MUST be honoured:
  `system` routes every external link to the OS browser, `blocked` shows a policy message instead of
  opening. The setting MUST be re-read on foreground so a district change applies without an app
  release.

**Migration**

- **FR-27.** All existing link call sites MUST be migrated to `LinkOpener` — 43 iOS files and 36 Android
  files — and a CI lint rule MUST fail on new direct `openURL(` / `UIApplication.shared.open` /
  `Intent(Intent.ACTION_VIEW` usage outside the opener.
- **FR-28.** `WebItemView` / `WebItemScreen` MUST be re-implemented on top of the shared browser,
  keeping first-party auth injection (per FR-20) and losing their bespoke chrome.

## 6. Non-Functional Requirements

- **Performance** — Presentation animation starts within 1 frame of the tap; first paint of the web view
  no later than the system browser would achieve for the same URL (WebView instance pre-warmed, reusing
  the `SandboxWebViewPool` pattern). Dismiss releases the web view and its memory.
- **Security** — Threat model: (a) token exfiltration to a look-alike host — mitigated by FR-20/FR-21;
  (b) a page attempting to phish Lextures credentials — mitigated by a persistent, non-spoofable origin
  display in native chrome (FR-11) that page content cannot cover; (c) scheme abuse — mitigated by the
  allowlist (FR-8); (d) us becoming an observer of third-party browsing — mitigated by FR-24/FR-25.
  TLS errors MUST fail the load; no user override.
- **Privacy & Compliance** — Browsing inside an education app is student activity data. FERPA/COPPA
  posture: no third-party URL leaves the device in telemetry (FR-25), browsing data is session-scoped
  and purged on sign-out (FR-23), and no page content is read (FR-24). Carries the mobile share of S02
  (retention) and S08 (children's privacy); the absence of an address bar is also what keeps us out of
  App Store "unfiltered web access" age-rating territory.
- **Accessibility** — WCAG 2.1 AA for the chrome. The cover is an accessible modal: focus moves into it
  on present and returns to the invoking control on dismiss; VoiceOver/TalkBack announce the host on
  open; the close control is labelled text-or-icon-with-label, never icon-only without a label; every
  gesture (drag-dismiss, swipe-back) has a control equivalent; chrome honours Dynamic Type to the
  platform maximum and Reduce Motion (cross-fade instead of slide, per `LexturesMotion.resolve`).
- **Scalability** — Not a server feature; one policy read per session per org.
- **Reliability** — A crashed or unresponsive web content process MUST recover to the error state with
  retry, not take down the host screen. Policy fetch failure MUST fall back to the last cached policy,
  and to `in_app` if none — never to a silent block.
- **Observability** — Counters (content-free per FR-25): opens by source surface, internal vs external,
  dismiss method (close / drag / back), copy-link taps, share taps, escape-to-system taps, load errors
  by class, download handoffs, policy-blocked opens. "Escape to system browser" rate is the primary
  quality signal — a high rate means the in-app experience is failing.
- **Maintainability** — Policy logic lives in the pure-logic modules with mirrored Kotlin/Swift tests,
  matching `ContentToolSandboxLogic` / `ContentToolGovernanceLogic`. UI lives in
  `Features/Browser/` (iOS) and `features/browser/` (Android).
- **Internationalization** — `mobile.browser.*` keys in all five locale files
  (`clients/mobile/locales/{en,es,fr,ar,en-XA}.json` + `Localizable.xcstrings` + `values-*`). RTL: the
  chrome mirrors; web content does not and MUST NOT be forced to.
- **Backward compatibility** — No API or schema change. Unknown values of `mobileLinkHandling` MUST fall
  back to `in_app`. Users on older app versions keep today's behaviour; nothing server-side depends on
  the client's choice.

## 7. Acceptance Criteria

- **AC-1.** *Given* a content page with an external `https` link, *When* a student taps it, *Then* a
  full-screen in-app browser slides up over the lesson and dismissing it returns to the same scroll
  position.
- **AC-2.** *Given* a link to a `lextures.com` path with a native destination, *When* tapped, *Then* the
  app navigates natively and no browser appears.
- **AC-3.** *Given* a `mailto:` link, *When* tapped, *Then* the system mail composer opens.
- **AC-4.** *Given* an SSO sign-in, *When* started, *Then* it uses `ASWebAuthenticationSession` /
  Custom Tabs and completes successfully — unchanged from today.
- **AC-5.** *Given* a Stripe checkout URL, *When* opened, *Then* it opens in the system browser surface,
  not the in-app browser.
- **AC-6.** *Given* a Zoom join link and the Zoom app installed, *When* tapped, *Then* Zoom opens.
- **AC-7.** *Given* an unknown scheme (e.g. `javascript:`, `file:`), *When* encountered, *Then* nothing
  opens and the event is counted as blocked.
- **AC-8.** *Given* the in-app browser is open, *When* the user taps the host chip, *Then* the current
  page URL is on the clipboard and a confirmation toast appears — in one tap, with no menu.
- **AC-9.** *Given* the browser is open, *Then* it can be dismissed by the close control, by dragging
  down, and by the system back gesture; all three restore the underlying screen.
- **AC-10.** *Given* a page that redirects, *When* the user copies the link, *Then* the copied URL is
  the final URL, not the original.
- **AC-11.** *Given* a URL whose host is `api.lextures.com.example.net`, *When* opened, *Then* no
  `Authorization` header is sent — asserted by a unit test over the header-attachment function.
- **AC-12.** *Given* a first-party URL that redirects to a third-party host, *Then* the `Authorization`
  header is not present on the redirected request.
- **AC-13.** *Given* the user signs out, *Then* in-app browser cookies, storage and caches are purged;
  reopening a previously logged-in site shows a logged-out state.
- **AC-14.** *Given* a page that calls `window.open`, *Then* the new URL loads in the in-app browser and
  in-page back returns to the opener.
- **AC-15.** *Given* a URL that responds with a PDF or attachment, *Then* the file-preview/share path
  handles it and no blank web view is shown.
- **AC-16.** *Given* airplane mode, *Then* the offline state renders with a retry that works once
  connectivity returns.
- **AC-17.** *Given* `mobileLinkHandling = system`, *When* an external link is tapped, *Then* the system
  browser opens and the in-app browser never appears.
- **AC-18.** *Given* `mobileLinkHandling = blocked`, *When* an external link is tapped, *Then* a policy
  message appears and no navigation occurs.
- **AC-19.** *Given* an admin changes the policy, *When* the user next foregrounds the app, *Then* the
  new policy applies with no app update.
- **AC-20.** *Given* any emitted telemetry event, *Then* it contains no third-party URL, host, path or
  title — asserted by an automated payload-shape test.
- **AC-21.** *Given* VoiceOver/TalkBack, *When* the browser presents, *Then* focus moves into the cover,
  the host is announced, the close control is labelled, and dismissal returns focus to the invoking
  control.
- **AC-22.** *Given* Reduce Motion, *Then* the browser cross-fades rather than sliding.
- **AC-23.** *Given* the maximum system font scale, *Then* header chrome remains legible with no
  clipping or overlap.
- **AC-24.** *Given* CI, *Then* the lint rule fails a build that introduces a direct `openURL` /
  `ACTION_VIEW` call outside `LinkOpener`.
- **AC-25.** *Given* the shared policy fixture table, *Then* the iOS and Android policy suites produce
  identical classifications for every row.

## 8. Data Model

**No server schema change and no migration for the client feature.**

One platform/org setting is required for FR-26. If an existing settings row can carry it, extend that
row rather than adding a table:

```
platform_settings / org_settings
  mobile_link_handling  TEXT NOT NULL DEFAULT 'in_app'
    CHECK (mobile_link_handling IN ('in_app','system','blocked'))
```

Migration follows the repo convention `server/migrations/NNN_mobile_link_handling.sql`. Backfill: the
column default covers existing rows; no data migration needed. Client-side state (cookies, caches) lives
in the platform web data store and in no database.

## 9. API Surface

**No new endpoints.** The client reads `mobileLinkHandling` from the existing platform/org settings
payload the app already fetches at bootstrap:

| Verb | Path | Change |
|---|---|---|
| GET | `/api/v1/settings/platform` | add `mobileLinkHandling` to the response |
| GET | `/api/v1/orgs/{org}/settings` | add `mobileLinkHandling` (org override) |
| PATCH | `/api/v1/orgs/{org}/settings` | accept `mobileLinkHandling`, admin scope |

Request/response addition:

```ts
type MobileLinkHandling = "in_app" | "system" | "blocked";
interface PlatformSettings { /* … */ mobileLinkHandling: MobileLinkHandling }
```

No WebSocket events. No new rate limits. OpenAPI: update the settings schemas; the repo requires
`/api/openapi.json` to stay valid (TD.3).

## 10. UI / UX

**New (iOS)** — `Features/Browser/{InAppBrowserView, BrowserChromeHeader, BrowserOverflowMenu,
BrowserErrorState, BrowserWebView}.swift`; `Core/Routing/LinkOpener.swift`;
`Core/LMS/MobileLinkPolicy.swift` (pure).
**New (Android)** — `features/browser/{InAppBrowserScreen, BrowserChromeHeader, BrowserOverflowMenu,
BrowserErrorState, BrowserWebView}.kt`; `core/routing/LinkOpener.kt`;
`core/lms/MobileLinkPolicy.kt` (pure).
**Modified** — `ContentLinkRouter` (both platforms) delegates to `LinkOpener`; `WebItemView` /
`WebItemScreen` re-implemented on the shared browser; ~79 feature files migrated; Settings gains
**Clear browsing data**.

**Layout** (the X/Instagram shape, in Lextures chrome):

```
┌───────────────────────────────────────────┐
│  ✕      🔒 en.wikipedia.org          ⋯    │  ← close · host chip (tap = copy) · overflow
│         Photosynthesis — Wikipedia        │  ← page title, secondary style
│▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  ← determinate progress, hides when done
├───────────────────────────────────────────┤
│                                           │
│              page content                 │  ← drag down anywhere at scroll-top to dismiss
│                                           │
└───────────────────────────────────────────┘
   ‹ back appears in the header only once in-page history exists
```

**Key flows**

1. **Read an external link** — tap link → cover slides up (`LexturesMotion.bubble`, reduced-motion:
   cross-fade) → progress fills → page renders → drag down or tap ✕ → underlying screen restored.
2. **Copy the URL** — tap the host chip → clipboard set to the current URL → toast "Link copied" +
   success haptic. (Also in the overflow, for discoverability and for assistive-tech users who reach
   the menu first.)
3. **Escape to the system browser** — ⋯ → "Open in Safari/Chrome" → hands off the current URL; the
   in-app browser stays open behind so the user can come back.
4. **Share** — ⋯ → Share… → platform share sheet with the current URL and title.
5. **Follow links within the page** — in-page navigation stays in the cover; a back affordance appears;
   system back consumes in-page history before dismissing.
6. **Hit a download** — non-renderable response → handed to the file-preview/share path; the cover
   dismisses if there is nothing left to show.
7. **Policy = system** — no cover; the OS browser opens directly, as today.

**States** — *Loading*: progress bar + skeleton, no spinner-on-white. *Error*: title, plain-language
cause, **Retry**, and **Open in browser** as the escape hatch. *Offline*: offline illustration + retry,
consistent with the app's existing offline surfaces. *Insecure (`http`)*: "Not secure" replaces the lock
with a warning treatment. *Blocked by policy*: neutral message naming the organisation policy, with no
retry.

**Accessibility annotations** — cover is `accessibilityAddTraits(.isModal)` / a Compose dialog-semantics
route; focus enters the header on present and returns to the invoking control on dismiss; the host chip
is a button labelled "Copy link, {host}"; the close control is labelled "Close"; progress exposes
`accessibilityValue`; the "Not secure" state is text, not colour alone; toast confirmations announce
politely; drag-dismiss is duplicated by the close control (AC-21).

**Copy & i18n** — `mobile.browser.{close, copyLink, linkCopied, share, openInBrowser, reload, report,
loading, notSecure, errorTitle, errorBody, retry, offlineTitle, offlineBody, blockedByPolicy,
clearBrowsingData, clearBrowsingDataConfirm}` across all five locale files.

## 11. AI / ML Considerations

Not AI-touching. Worth stating explicitly for review: page content rendered in the in-app browser MUST
NOT be captured, summarised or fed to any model — that would turn a browser into a content-ingestion
pipeline over third-party pages with no consent basis. Grounded-context link ingestion (CT.6) remains a
separate, server-side, explicitly-authored path.

## 12. Integration Points

- **Internal (iOS)** — `Core/Routing/{ContentLinkRouter,DeepLinkRouter}.swift`,
  `Features/Home/MainTabView.swift` (`AppShellModel` hosts the cover so it can present above any tab
  or drawer state), `Core/Design/{LexturesMotion,LexturesTheme}.swift`,
  `Core/Accessibility/*`, `Core/Config/AppConfiguration.swift`, `Features/Courses/WebItemView.swift`.
- **Internal (Android)** — `core/routing/{ContentLinkRouter,DeepLinkRouter}.kt`,
  `core/navigation/MobileDestinations.kt`, `core/design/{LexturesMotion,LexturesTheme}.kt`,
  `core/accessibility/*`, `core/config/AppConfiguration.kt`,
  `features/courses/WebItemScreen.kt`.
- **Deliberately untouched** — `features/auth/SsoAuth.kt`, `features/billing/BillingCheckout.kt`,
  `Features/Billing/PurchaseFlow.swift`, and the CT.M4 sandbox host
  (`Sandbox/SandboxWebViewHost.*`, `SandboxWebViewPool.*`) — the sandbox keeps its own configuration,
  bridge and pool.
- **Platform APIs** — `WKWebView` + `WKWebsiteDataStore`, `ASWebAuthenticationSession`,
  `UIApplication.canOpenURL`; `android.webkit.WebView` + `androidx.webkit`,
  `androidx.browser` Custom Tabs (already dependencies), `PackageManager` for app-link resolution.
- **Server** — settings handlers for `mobileLinkHandling`; no other backend change.
- **Events** — client counters only; nothing emitted to other services.

## 13. Dependencies & Sequencing

- Must ship after: nothing hard — `DeepLinkRouter` and `ContentLinkRouter` already exist on both
  platforms.
- Must ship before: any new mobile surface that opens links (future CT, VC and marketplace work should
  adopt `LinkOpener` from the start rather than adding call sites to migrate).
- Shared infra: none new. The settings column (§8) is the only backend dependency and can land ahead of
  the clients.
- Internal ordering: policy logic + tests → browser component → carve-outs (auth/billing/app-links) →
  call-site migration → lint rule → org policy → telemetry → accessibility audit.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Capturing a link that must reach a native app (Zoom, Docs) breaks a live class | M | H | Explicit carve-out list (FR-7) + app-link resolution query; device tests per target (AC-6) |
| Capturing an OAuth/SSO URL breaks sign-in | M | H | Auth flows keep `ASWebAuthenticationSession` / Custom Tabs, untouched (FR-5, AC-4) |
| Capturing checkout violates store payment rules | L | H | Billing paths excluded by policy and covered by AC-5; flagged for store-review sign-off |
| Bearer token leaks to a look-alike host | M | H | Parsed-host comparison replaces prefix matching; redirect re-evaluation; unit tests AC-11/AC-12 |
| We are perceived as surveilling student browsing | M | H | No script injection, no content reads, no third-party URLs in telemetry (FR-24/FR-25, AC-20); documented in the privacy notice |
| App review objects to in-app web browsing in a K12 app | L | H | No address bar or arbitrary navigation (§3); admin policy to force system browser (FR-26); age-rating review before submission |
| District web filtering is bypassed because our WebView ignores the managed profile | M | M | `mobileLinkHandling = system` gives districts a supported way to keep filtering (FR-26, AC-17) |
| ~79-file migration causes regressions in rarely-used surfaces | H | M | Mechanical migration + lint rule + an inventory checklist walked per file during QA |
| Memory growth from long browsing sessions | M | M | Single reused web view, released on dismiss; content-process crash recovers to the error state |
| Chrome covers content or looks cramped at max font scale | M | M | Header layout tested at maximum Dynamic Type (AC-23) |

## 15. Rollout Plan

- **Feature flag** — `mobileInAppBrowserEnabled`, client-read, default **off**. Off = today's behaviour
  (`LinkOpener` routes external links to the system browser), so the call-site migration can land and
  bake independently of the new UI.
- **Sequencing** — settings column + API → policy logic and tests (both platforms) → browser component
  → carve-outs → call-site migration behind the flag → lint rule → telemetry → accessibility audit →
  dogfood → staged enable.
- **Dogfood** — internal builds with the flag on for two weeks; a test course containing an external
  article, a PDF link, a `mailto:`, a Zoom link, an app-store link, a redirecting shortener and a
  first-party textbook resource.
- **GA criteria** — all ACs green; escape-to-system-browser rate below 10% of opens in dogfood; zero
  auth/checkout regressions; accessibility audit closed with no blockers; privacy review of §6 signed
  off; store age-rating check complete.
- **Comms** — release note for students/instructors ("links now open inside the app"); an admin note
  describing `mobileLinkHandling` and how to force the system browser.
- **Rollback** — flip `mobileInAppBrowserEnabled` off: links revert to the system browser through the
  same `LinkOpener`, with no code revert and no app release. `mobileLinkHandling = system` is the
  per-org equivalent.

## 16. Test Plan

- **Unit** — the policy fixture table (§FR-25) run identically in Swift and Kotlin: native paths,
  `lextures://`, relative paths, `mailto:`/`tel:`/`sms:`, SSO, checkout, app-links, store links,
  unknown schemes, punycode and look-alike hosts, uppercase schemes, URLs with credentials, IP-literal
  hosts. Header-attachment function: exact host, subdomain, look-alike, cross-origin redirect.
  Telemetry payload shape. Copy-URL-after-redirect logic.
- **Integration** — settings fetch and foreground re-read for `mobileLinkHandling`; sign-out purges the
  web data store; download handoff to the file-preview path; `window.open` handling.
- **End-to-end (device)** — open/read/dismiss round-trip preserving scroll position; one-tap copy; share
  sheet; escape to system browser and return; in-page back then dismiss; airplane-mode error and
  recovery; SSO sign-in; a real checkout in a test account; a Zoom link with and without Zoom installed.
- **Security** — assert no `Authorization` header on any non-first-party request (proxy-captured);
  assert no user scripts or message handlers on the browser's web configuration; TLS-error page cannot
  be bypassed; unknown schemes never reach the OS; verify no page content is read by inspecting the
  configuration surface in a test.
- **Accessibility** — `docs/accessibility/mobile-audit-checklist.md` entry for the browser; VoiceOver and
  TalkBack passes for present/dismiss/copy/overflow; focus-return verification; 200% font scale; Reduce
  Motion; RTL chrome.
- **Performance / load** — time-to-first-paint vs the system browser on the same URL and network;
  memory after 20 sequential opens; dismissal releases the web view (leak check).
- **Manual exploratory** — rotate mid-load; background/foreground mid-load; policy changed while the
  browser is open; a page that immediately redirects to a native app link; a page requesting camera/mic
  permission; a page with an infinite redirect; a very long host name in the chip; a page that tries to
  render a fake browser bar (verify native chrome stays on top).

## 17. Documentation & Training

- **End-user** — help-centre note: links open inside Lextures; how to copy a link, share it, or open it
  in your browser; how to close.
- **Admin** — `mobileLinkHandling` semantics, why a district might choose `system` (managed web
  filtering), and that changes apply on next app foreground.
- **API reference** — settings schema addition in `/api/openapi.json`.
- **Privacy** — append the in-app browser to the mobile privacy notice and the S02/S08 records: what is
  stored on device, for how long, what is never collected.
- **Internal runbook** — the counters in §6, the escape-to-system-browser rate as the health signal, and
  the two kill paths (feature flag, org policy).
- **Engineering** — a short `clients/mobile/README` section: "never call `openURL` / `ACTION_VIEW`
  directly; use `LinkOpener`", with the lint rule referenced.

## 18. Open Questions

1. **Custom WebView vs. platform browser component.** Recommendation: build on `WKWebView` /
   `android.webkit.WebView` — `SFSafariViewController` and Custom Tabs cannot carry our chrome, our
   one-tap copy affordance, or the drag-dismiss feel, which is the whole point of the request. Cost: we
   lose the shared cookie jar, Reader, and content blockers. Decision owner: mobile lead. (Auth and
   billing keep the platform components regardless — FR-5/FR-6.)
2. Is `mobileLinkHandling` a platform setting, an org setting, or both with org override?
   (Recommendation: both, org wins — matches the existing settings pattern.)
3. Should instructors be able to mark a specific link "open externally" when authoring? (Recommendation:
   not in MB.1; revisit if dogfood surfaces demand.)
4. Which video-conference and productivity hosts belong on the FR-7 native-app carve-out list, and do we
   maintain it manually or rely solely on OS app-link resolution? (Both: OS resolution plus a small
   curated list for hosts that resolve inconsistently.)
5. Do we keep the in-app browser's cookies for the whole signed-in session, or clear on every dismiss?
   (Recommendation: whole session, cleared on sign-out — dismissing after a login and losing it is a bad
   experience; needs privacy sign-off.)
6. Does the existing mobile analytics channel already guarantee content-free payloads, or does FR-25
   need enforcement at the emitter? (Verify before implementing; do not add a second channel — same
   question CT.M9 left open.)
7. Should **Report** appear for all in-app browser pages or only for links originating from
   user-generated content (boards, discussions, tool content)? (Recommendation: only user-generated
   sources; reporting a Wikipedia article to a TA is noise.)

## 19. References

- **iOS** — `clients/ios/Lextures/Core/Routing/ContentLinkRouter.swift`,
  `Core/Routing/DeepLinkRouter.swift`, `Features/Courses/WebItemView.swift`,
  `Features/Home/MainTabView.swift`, `Core/Design/LexturesMotion.swift`,
  `Core/Config/AppConfiguration.swift`, `Features/Billing/PurchaseFlow.swift`.
- **Android** — `clients/android/app/src/main/kotlin/com/lextures/android/core/routing/{ContentLinkRouter,DeepLinkRouter}.kt`,
  `features/courses/WebItemScreen.kt`, `features/auth/SsoAuth.kt`,
  `features/billing/BillingCheckout.kt`, `core/config/AppConfiguration.kt`,
  `core/design/LexturesMotion.kt`.
- **Shared** — `clients/mobile/locales/{en,es,fr,ar,en-XA}.json`.
- **Related plans** — [CT.M4 — mobile sandboxed WebView tool host](../../completed/content_tools/CT.M4-mobile-sandboxed-webview-tool-host.md)
  (separate component, different threat model),
  [CT.M9 — mobile governance, a11y & telemetry](../../completed/content_tools/CT.M9-mobile-tools-governance-a11y-telemetry.md)
  (telemetry and audit conventions reused here),
  [standards](../standards/README.md) — S02 (retention), S08 (children's privacy), S20 (accessibility
  law).
- **Instruments** — `docs/accessibility/mobile-audit-checklist.md`.
- **External** — WCAG 2.1 AA; Apple App Review Guidelines §1.3, §4.7, §5.1.4; Google Play Families
  policy; RFC 3986 (URI parsing) and the Public Suffix List (registrable-domain display).
