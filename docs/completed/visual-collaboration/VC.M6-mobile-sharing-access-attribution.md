# VC.M6 — Mobile Boards: Sharing, Access Control & Attribution

> Implementation plan. Source: mobile parity for board access/sharing. Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Mirrors web [VC.6](VC.6-sharing-access-contributors.md); consumes the single server-side **access resolver** and the public board-link endpoints, and adds QR/deep-link handling for phones.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M6 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | VC.M1 (gates VC.M2/M3/M5 capabilities) |
| **Unblocks** | correct capability gating across VC.M2/M3/M5/M7 |

---

## 1. Problem Statement

Who can view, post, arrange, and interact — and whether a card is named or anonymous — are the settings that
make a board usable **and safe**. On mobile these settings must be *honored* (a read-only viewer must get no
composer, an anonymous board must never show authorship) and, ideally, *managed* (an instructor sharing a
board from their phone, a parent scanning a QR to see the class wall). VC.M6 brings the board access model,
attribution, contributor policy, and share-link handling to the native apps, wiring every mobile surface to
the server's single access resolver.

## 2. Goals

- Consume the resolved board capabilities `{canView, canPost, canInteract, canArrange, canManage}` from the
  server and use them to gate every mobile board surface (composer, drag, reaction/comment, manage actions).
- Honor **attribution** modes (`named | anon_to_peers | anonymous`): the mobile client MUST render authorship
  exactly as the server serializes it — never infer or reveal an author the server withheld.
- Let managers set **visibility**, **attribution**, and **contributor policy** from the phone (share dialog).
- Handle **share links** on mobile: open a `view` or `contribute` link (universal/app link or QR scan),
  including the password and display-name prompts, in a minimal public board view.
- Manage **members** for invite-only boards (add/remove, per-member role) for managers.

## 3. Non-Goals

- Moderation/approval and abuse handling (VC.M7).
- Reaction/comment mechanics (VC.M5); VC.M6 only decides *whether* a viewer may interact.
- Org-wide/cross-course boards.
- Full account creation for public link visitors beyond an optional display-name prompt.
- Generating QR images for a link (web VC.9 owns QR generation; mobile **scans/opens** links; generating a
  shareable QR on mobile is optional, Open Question 2).

## 4. Personas & User Stories

- **As a student**, I want to only see and act on boards I'm entitled to, with controls that match my role.
- **As an instructor**, I want anonymous posting so students share honestly, while I still see authorship.
- **As an instructor**, I want to create a read-only share link from my phone to show parents the wall.
- **As a parent**, I want to scan a QR / tap a link and see the class board without an account.
- **As a guest speaker**, I want to use a contribute link (with a password) to add a card from my phone.
- **As an instructor**, I want to make a board section-only so sections don't see each other's work.

## 5. Functional Requirements

- **FR-1.** Every mobile board fetch MUST surface the server-resolved capabilities for the viewer; the app MUST
  gate the composer on `canPost`, drag/arrange on `canArrange`, reactions/comments on `canInteract`, and manage
  actions on `canManage` — never inferring capability from role locally.
- **FR-2.** The client MUST render authorship strictly from the server payload: on `anon_to_peers`/`anonymous`
  boards the author field is absent for non-managers and MUST NOT be reconstructed, cached, or shown; presence
  (VC.M4) and comments (VC.M5) MUST follow the same rule.
- **FR-3.** Managers MUST be able to set `visibility ∈ {course, section, group, invite, link, public}` (with a
  target for section/group), `attribution`, and the contributor policy (`canPost`/`canInteract`/`canArrange`
  defaults) via `PATCH …/boards/{id}` from a mobile share dialog.
- **FR-4.** Managers MUST be able to create/revoke **share links** (`GET/POST …/boards/{id}/shares`,
  `DELETE …/shares/{sid}`) with capability (view/contribute), optional expiry, optional password; the created
  URL is copyable/shareable via the native share sheet.
- **FR-5.** The app MUST resolve an incoming board link via `GET /api/v1/board-links/{token}` (universal/app
  link or QR scan), prompting for password when required, and render a **public board view** that is read-only
  for `view` links and contribute-capable for `contribute` links (posting via
  `POST /api/v1/board-links/{token}/posts`), prompting an unauthenticated visitor for a display name.
- **FR-6.** For invite boards, managers MUST manage members (`GET/POST/DELETE …/boards/{id}/members`) with
  per-member role.
- **FR-7.** `public`/`link` options MUST be available only when the platform permits external sharing (server
  gate) and MUST be blocked for age-gated (COPPA) minors' courses — the client MUST hide those options when the
  server indicates they're unavailable, and MUST NOT attempt them.
- **FR-8.** A revoked/expired link MUST show a clear denied state; the app MUST NOT cache a resolved link past
  its validity.
- **FR-9.** When visibility narrows and the viewer loses access, the next fetch/WS event MUST drop them to a
  "no longer available" state (the server enforces; the client handles the `403/404` gracefully).

## 6. Non-Functional Requirements

- **Performance** — capability resolution comes with the board/post fetch (no extra round-trip); membership/
  share lists are small paged fetches.
- **Security** — share tokens are opaque and never logged; password entry uses secure fields and is sent over
  TLS; link capability + expiry + password are all enforced server-side (the client is not the gatekeeper);
  no author leakage in anonymous modes across any surface.
- **Privacy & Compliance** — public board view excludes roster PII; anonymous modes never expose authors;
  align with S08 children's-privacy and FERPA directory rules; the display-name prompt stores nothing beyond
  what the server retains for audit.
- **Accessibility** — share dialog, password prompt, and public view are fully AT-operable with proper focus
  order; password field has show/hide.
- **Reliability** — link revoke takes effect immediately (no cached bypass); expiry enforced server-side.
- **Observability** — reuse app error logging; do not log tokens/passwords.
- **Internationalization** — share/prompt/denied copy externalised; RTL-safe.
- **Backward compatibility** — additive; existing boards default `course` visibility, `named` attribution.

## 7. Acceptance Criteria

- **AC-1.** *Given* `canArrange = false`, *when* a student opens a board, *then* drag is disabled but posting
  (if `canPost`) still works.
- **AC-2.** *Given* `anon_to_peers`, *when* a student views a peer's card, *then* no author is shown; *when* the
  instructor views it, *then* the author is shown.
- **AC-3.** *Given* a `view` share link opened on a phone, *then* the public board renders read-only with no
  interaction controls and no roster PII.
- **AC-4.** *Given* a `contribute` link with a password, *when* a visitor enters the wrong password, *then*
  access is denied; with the right password + a display name, they can post.
- **AC-5.** *Given* a revoked link, *when* reopened, *then* access is denied immediately.
- **AC-6.** *Given* external sharing is disabled org-wide (or a COPPA course), *when* a manager opens the share
  dialog, *then* `public`/`link` options are unavailable.
- **AC-7.** *Given* a section-scoped board, *when* an out-of-section student opens it, *then* they get a
  not-available state and it is not listed.

## 8. Data Model

No server schema change — VC.6's `board.board_members`, `board.board_shares`, and the board
visibility/attribution/policy columns already exist. The client extends its `Board` model with
`visibility`, `attribution`, contributor-policy booleans, and a resolved `capabilities` object, and adds
`ShareLink` / `BoardMember` models.

## 9. API Surface

No new endpoints. Mobile consumes web VC.6's routes:

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{id}` — visibility, attribution, contributor policy | `item:create` |
| GET/POST/DELETE | `/boards/{id}/members` | `item:create` |
| GET/POST | `/boards/{id}/shares` (create link) | `item:create` |
| DELETE | `/boards/{id}/shares/{shareId}` (revoke) | `item:create` |
| GET | `/api/v1/board-links/{token}` (public resolve; +password) | token |
| POST | `/api/v1/board-links/{token}/posts` (contribute) | token capability=contribute |

The board list/detail/post responses already carry the resolved capabilities + attribution-correct authorship
— mobile just consumes them.

## 10. UI / UX

- **Share dialog** (managers) — sheet with visibility selector, attribution selector, contributor-policy
  toggles, create-share-link (view/contribute, expiry, password), copy/native-share the URL, member management
  for invite boards, and a revoke list. `public`/`link` hidden when disallowed.
- **Public board view** — minimal chrome (no course nav), read-only or contribute-only, password prompt and
  display-name prompt for anonymous contributors; reached via universal/app link or in-app QR scan.
- **Capability gating** — the composer/drag/reaction/manage controls across VC.M2/M3/M5 read the resolved
  capabilities; no control renders that the viewer can't use.
- **States** — password prompt, expired/revoked link, external-sharing-disabled explainer, minors-blocked
  explainer, not-available (lost access).
- **Accessibility** — dialog focus management; labelled controls; password show/hide.
- **Copy & i18n** — `boards.share.*`, `boards.access.*` keys.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse**: native share sheet + universal/app-link routing already configured for other deep links
  ([M0.1](../mobile/M0.1-push-deep-links.md)); camera for QR scan (if in scope); the server access
  resolver (`board.ResolveAccess`, unchanged).
- **New (iOS)**: `Core/LMS/LMSAPIBoardAccess.swift`, `Features/Boards/{BoardShareSheet}.swift`,
  `Features/Boards/Public/BoardPublicView.swift` → regenerate project.
- **New (Android)**: `core/lms/BoardAccessApi.kt`, `features/boards/BoardShareSheet.kt`,
  `features/boards/publicboard/BoardPublicScreen.kt`; register the board-link URL path in the manifest.

## 13. Dependencies & Sequencing

- Must ship after: VC.M1. Its capability gating should land **before or with** VC.M2/M3/M5 so those surfaces
  gate correctly (until then they gate on the coarse course create-permission).
- Must ship before: VC.M7 (moderation assumes the access model). Public/link UI stays hidden until VC.M7 is
  present, matching web VC.6/VC.7 sequencing.
- Shared infra: deep-link routing; native share; (optional) QR camera.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Author leak in anonymous mode on some mobile surface | M | H | Render authorship only from server payload; audit list/card/presence/comment paths; explicit tests |
| Token/password logged or cached | M | H | Never log tokens/passwords; secure fields; no persistent cache of resolved links |
| Public view exposes course nav/PII | L | H | Dedicated minimal public screen; server strips PII |
| Deep-link handling opens wrong screen | M | M | Route board-link tokens to the public resolver; validate before render |
| Minors exposed via public link | L | H | Hide public/link when server signals COPPA/disabled; never attempt |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled`; external sharing additionally gated by the platform
  external-sharing setting (server). Public/link UI kept behind VC.M7 availability.
- **Sequencing**: capability gating + in-course visibility/attribution + member mgmt → share-link create →
  public board view + link open → enable link/public only after VC.M7.
- **Rollback**: disable external sharing server-side → link/public resolve to denied; in-course scopes
  unaffected.

## 16. Test Plan

- **Unit** — capability → control mapping; attribution serialization respected; link-state (expired/revoked)
  handling.
- **Integration** — scope enforcement (section/group/invite); anonymous serialization across list/card/
  comment/presence; external-sharing + minors gates hide options.
- **End-to-end (device)** — section board isolation; anon-to-peers view diff (student vs instructor); view
  link read-only; contribute link + password + display name; revoke takes effect; QR/deep-link open.
- **Security** — authz matrix; no author leakage; token/password never logged; TLS-only.
- **Accessibility** — share dialog + public view + password prompt AT/keyboard.
- **Manual** — link expiry; org sharing toggle; COPPA course; lost-access transition.

## 17. Documentation & Training

- End-user: sharing a board from mobile (in-course vs link vs public); anonymous posting.
- Parent/guest: opening a board via link/QR.
- Admin: external-sharing + minors policy (shared with web).

## 18. Open Questions

1. Manage-from-mobile scope: full share dialog on v1, or view/consume capabilities first and add management in
   a fast-follow? (Recommendation: consume + gate first; management next.)
2. Do we add **QR scanning** in-app, or rely on the OS camera + universal links? (Recommendation: OS camera +
   universal links first; in-app scan optional.)
3. Retention of anonymous-visitor display names (align with S02) — surface any mobile-specific note?

## 19. References

- Web plan: [VC.6](VC.6-sharing-access-contributors.md); server resolver
  `server/internal/repos/board/access.go`, public `board_links_http.go`; web `share-dialog.tsx`,
  `pages/public/board-share-page.tsx`.
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M5](VC.M5-mobile-reactions-comments-assessment.md), [VC.M7](../../plan/visual-collaboration/VC.M7-mobile-moderation-safety.md); deep links [M0.1](../mobile/M0.1-push-deep-links.md).
