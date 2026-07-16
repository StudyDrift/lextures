# VC.6 — Sharing, Access Control & Contributor Management

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.6 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.1 |
| **Unblocks** | VC.7, VC.9 |

---

## 1. Problem Statement

Who can see a board, who can post to it, whether posts are named or anonymous, and how it is shared beyond a
single course are the settings that make a board usable for real classroom scenarios — and the settings that
make it **safe**. Getting access control right is a prerequisite for exposing any external share link. VC.6
delivers the board's **visibility model, contributor roles, attribution controls, and share links**.

## 2. Goals

- Define per-board **visibility scopes**: course, section, group, invite-only, link-shared (unlisted),
  and public read-only.
- Define per-board **contributor permissions**: who can post, who can only react/comment, who is read-only.
- Support **share links** with a capability (view / contribute), optional expiry, and optional password.
- Support **attribution** modes: named (default), anonymous-to-peers (instructor still sees author), or
  fully anonymous.
- Manage **members** explicitly for invite-only boards (add/remove, per-member role).

## 3. Non-Goals

- Moderation/approval of content and abuse handling (VC.7) — VC.6 sets *who can act*; VC.7 governs *what
  they post*.
- Reactions/comments mechanics (VC.5) — VC.6 only gates whether a viewer may interact.
- Org-wide/cross-course boards (deferred; VC.1 Open Question).
- SSO/identity for public link users beyond an optional display-name prompt.

## 4. Personas & User Stories

- **As an instructor**, I want a board visible only to one section so sections don't see each other's work.
- **As an instructor**, I want students to post but not edit or move each other's cards.
- **As an instructor**, I want anonymous posting so students share honestly, while I can still see who wrote
  what.
- **As an instructor**, I want a read-only share link to show parents the class wall without giving them
  accounts.
- **As an instructor**, I want a contribute link (with a password) so a guest speaker can add cards.
- **As a student**, I want to only see boards I'm entitled to.

## 5. Functional Requirements

- **FR-1.** Each board MUST have a `visibility ∈ {course, section, group, invite, link, public}` (default
  `course`) and, for `section`/`group`, a target id.
- **FR-2.** Each board MUST have a `contributor_policy` describing default capability for in-scope members:
  `{canPost, canInteract, canArrange}` booleans (e.g., "students can post + interact but not rearrange").
- **FR-3.** For `invite` boards, the system MUST maintain a `board_members` list with per-member `role ∈
  {owner, editor, contributor, viewer}`; only listed members (plus course managers) have access.
- **FR-4.** The system MUST support **share links** (`board_shares`): a random unguessable token, a
  `capability ∈ {view, contribute}`, optional `expires_at`, optional `password_hash`, and revoke.
- **FR-5.** A `public` board or a `view` share link MUST render **read-only** with no PII beyond what the
  board explicitly shows (respect attribution mode); a `contribute` link MUST allow posting under the
  link's capability and MUST prompt for a display name if the visitor is unauthenticated.
- **FR-6.** Each board MUST have an `attribution ∈ {named, anon_to_peers, anonymous}`: `named` shows author
  to everyone; `anon_to_peers` hides author from students but shows it to managers; `anonymous` stores no
  author reference for display (author id retained server-side only for audit/safety and never exposed to
  peers).
- **FR-7.** Every read/write path (VC.2 posts, VC.5 reactions/comments, VC.4 WS) MUST consult a single
  **authorization resolver** that, given (user, board), returns the effective capabilities
  `{canView, canPost, canInteract, canArrange, canManage}`.
- **FR-8.** Course managers (`item:create`) MUST always be able to manage boards in their course regardless
  of visibility/membership.
- **FR-9.** Changing visibility to a narrower scope MUST immediately revoke access for users no longer in
  scope (checked on next request and on the live WS).
- **FR-10.** Public/link exposure MUST be **off by default** and MUST require the platform to permit external
  sharing (a platform setting gated in VC.10); when external sharing is disabled org-wide, only
  authenticated in-course scopes are selectable.
- **FR-11.** COPPA/age-gated contexts MUST be able to forbid `public`/`link` visibility entirely for minors'
  courses (policy hook to the children's-privacy standard).

## 6. Non-Functional Requirements

- **Performance** — the authorization resolver is a single cached lookup per request; membership checks
  indexed.
- **Security** — share tokens are ≥128-bit random, constant-time compared; passwords hashed (argon2/bcrypt
  per repo standard); link capability strictly enforced server-side; no author leakage in anonymous modes.
- **Privacy & Compliance** — anonymous modes never expose author to peers via any endpoint, export, or WS
  awareness; public boards exclude roster PII; align with [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md) and FERPA directory-info rules.
- **Accessibility** — password/display-name prompts and share dialogs are fully accessible.
- **Scalability** — membership and share tables indexed by board; resolver O(1) per check.
- **Reliability** — revoking a link takes effect immediately (no cached bypass); expiry enforced server-side.
- **Observability** — audit every visibility/permission change and share-link create/revoke; counters for
  link views by capability.
- **Maintainability** — the resolver is the *only* place capabilities are computed; all handlers call it.
- **Internationalization** — share/prompt copy externalised.
- **Backward compatibility** — additive; existing boards default `course` visibility, `named` attribution.

## 7. Acceptance Criteria

- **AC-1.** *Given* a section-scoped board, *when* a student in another section requests it, *then* the API
  returns `404/403` and it is not listed.
- **AC-2.** *Given* `canArrange = false` for students, *when* a student tries to move a card, *then* the move
  is rejected while posting still works.
- **AC-3.** *Given* `anon_to_peers`, *when* a student views a peer's card, *then* no author is shown; *when*
  the instructor views it, *then* the author is shown.
- **AC-4.** *Given* a `view` share link, *when* an anonymous visitor opens it, *then* the board renders
  read-only with no interaction controls and no roster PII.
- **AC-5.** *Given* a `contribute` link with a password, *when* a visitor enters the wrong password, *then*
  access is denied; with the right password they can post under a prompted display name.
- **AC-6.** *Given* a link is revoked, *when* it is next opened, *then* access is denied immediately.
- **AC-7.** *Given* org external-sharing is disabled, *when* an instructor opens board settings, *then*
  `public`/`link` options are unavailable.
- **AC-8.** *Given* a minors' course with the COPPA policy, *when* an instructor opens sharing, *then*
  external link/public options are blocked.

## 8. Data Model

Migration `384_board_access.sql`:

```sql
ALTER TABLE board.boards
  ADD COLUMN visibility        TEXT NOT NULL DEFAULT 'course',   -- course|section|group|invite|link|public
  ADD COLUMN visibility_target UUID,                              -- section_id or group_id when scoped
  ADD COLUMN attribution       TEXT NOT NULL DEFAULT 'named',     -- named|anon_to_peers|anonymous
  ADD COLUMN can_post          BOOLEAN NOT NULL DEFAULT TRUE,     -- default contributor policy
  ADD COLUMN can_interact      BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN can_arrange       BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE board.board_members (
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'contributor',   -- owner|editor|contributor|viewer
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (board_id, user_id)
);

CREATE TABLE board.board_shares (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id     UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,        -- >=128-bit random, url-safe
    capability   TEXT NOT NULL DEFAULT 'view',-- view|contribute
    password_hash TEXT,
    expires_at   TIMESTAMPTZ,
    created_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_board_shares_board ON board.board_shares (board_id);
```

Anonymous attribution: `board.posts.author_id` is retained for audit but the API omits it from peer-facing
responses when `attribution != named` (never `NULL` it — that would break safety/audit).

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{id}` (extend) — `visibility`, `attribution`, contributor policy | `item:create` |
| GET/POST/DELETE | `/boards/{id}/members` | `item:create` |
| GET/POST | `/boards/{id}/shares` (create link) | `item:create` |
| DELETE | `/boards/{id}/shares/{shareId}` (revoke) | `item:create` |
| GET | `/api/v1/board-links/{token}` (public resolve → board view/contribute context) | token (+ password) |
| POST | `/api/v1/board-links/{token}/posts` (contribute link post) | token capability=contribute |

- Public link endpoints live outside the course-auth middleware and enforce token/password/expiry/capability
  directly. Rate-limit link resolution and password attempts (lockout on brute force).
- **OpenAPI**: members, shares, and public link endpoints.

## 10. UI / UX

- **Share dialog** (`components/boards/share-dialog.tsx`): visibility selector, attribution selector,
  contributor-policy toggles, "create share link" (view/contribute, expiry, password), copyable URL + QR
  (QR in VC.9), member management for invite boards, and a revoke list.
- **Public board view** (`pages/public/board-share-page.tsx`): minimal chrome, read-only or contribute-only,
  display-name prompt for anonymous contributors, no course nav.
- **States**: password prompt, expired/revoked link, external-sharing-disabled explainer, minors-blocked
  explainer.
- **Mobile**: share dialog is a sheet; public view is responsive.
- **Accessibility**: dialog focus trap; labelled controls; password field with show/hide.
- **Copy & i18n**: `boards.share.*`, `boards.access.*` keys.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse**: `courseroles.UserHasPermission` for manager checks; `enrollment` for scope membership; section
  (plan 5.4) and group (plan 6.6) ids for scoped visibility; password hashing helper used elsewhere in the
  repo.
- **New**: `server/internal/repos/board/access.go` (the resolver), `board/members.go`, `board/shares.go`,
  `server/internal/httpserver/board_access_http.go`, `board_links_http.go` (public),
  `clients/web/src/components/boards/share-dialog.tsx`, `clients/web/src/pages/public/board-share-page.tsx`.

## 13. Dependencies & Sequencing

- Must ship after: VC.1. Reads section/group ids from plans 5.4 / 6.6 when those scopes are chosen.
- Must ship before: VC.7 (moderation assumes the access model), and before any external share link is
  exposed in the UI. The public/link options MUST remain hidden until VC.7 ships.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Author leak in anonymous mode via some endpoint/export/WS | M | H | Single resolver + serializer that strips author; explicit tests across REST/WS/export |
| Share-link brute force | M | M | High-entropy tokens; per-token password lockout; rate limits |
| Stale access after scope change | M | M | Resolver re-checked each request + WS re-auth on scope change |
| Minors exposed via public link | L | H | COPPA policy hook forbids public/link for age-gated courses |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; external sharing additionally gated by a platform
  `boards_external_sharing` setting (VC.10), default off.
- **Sequencing**: migration `383` → ship in-course scopes + attribution + contributor policy → enable
  invite/members → enable link/public only after VC.7.
- **Rollback**: disable external sharing platform setting → link/public boards resolve to denied; in-course
  scopes unaffected.

## 16. Test Plan

- **Unit** — resolver capability matrix; token entropy; password verify; expiry/revoke logic.
- **Integration** — scope enforcement (section/group/invite); anonymous serialization across every read
  path; external-sharing + minors gates.
- **End-to-end** — Playwright: section board isolation; anon-to-peers view diff (student vs instructor);
  view link read-only; contribute link with password + display name; revoke takes effect.
- **Security** — authz matrix; brute-force lockout; no author leakage; SSRF-free public endpoints; CSRF on
  contribute link posts.
- **Accessibility** — share dialog + public view axe/keyboard.
- **Manual** — link expiry; org sharing toggle; COPPA course.

## 17. Documentation & Training

- End-user: sharing a board (in-course vs link vs public); anonymous posting.
- Admin: enabling/disabling external sharing; minors policy.
- API reference: members, shares, public link endpoints.
- Runbook: revoking a leaked link; audit trail of access changes.

## 18. Open Questions

1. Should `contribute` links require any verification beyond a display name (e.g., email, captcha)?
   (Recommendation: add captcha/rate-limit; optional email for audit.)
2. Do we allow cross-course board sharing (board owned by course A, linked into course B)? (Defer to VC.10.)
3. Retention of anonymous contributors' display names for audit — how long? (Align with S02 retention.)

## 19. References

- Existing files: `server/internal/courseroles/*`, `server/internal/repos/enrollment/*`, sections (plan
  5.4), group spaces (plan 6.6), repo password-hashing helper.
- Related plans: [VC.7](../../plan/visual-collaboration/VC.7-moderation-safety-governance.md), [VC.9](../../plan/visual-collaboration/VC.9-embedding-export-presentation.md),
  [VC.10](../../plan/visual-collaboration/VC.10-admin-analytics-quotas-lifecycle.md),
  [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
