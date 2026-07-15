# B1 — Competency Micro-Badges (signed, shareable, verifiable)

> Implementation plan. Source: new feature request (competency/outcome micro-credentialing). Builds directly on the shipped Open Badges 3.0 credential stack (`docs/completed/15-self-learner-specific/15.5-certificates-open-badges.md`, `15.6-linkedin-share-open-badges-export.md`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | B1 |
| **Section** | Badges & Micro-Credentials |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | SHIPPED |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Credentials & Learner-Record squad |
| **Depends on** | 15.5 (Open Badges issuance), 15.6 (LinkedIn share / badge export), 14.13 (CLR signing key + `did:web`), 3.7 (standards-based grading / outcome mastery), 100 (competency-based courses) |
| **Unblocks** | Public learner "badge backpack", employer-verifiable skill signals, teacher-driven micro-credentialing |

---

## 1. Problem Statement

Today Lextures issues signed Open Badges 3.0 credentials **only** for whole-course and learning-path completion (`credentials.issued_credentials`, source types `course` / `path` / `ceu`), and they are viewable only on the authenticated `/me/credentials` page plus a per-credential `/verify/:token` page. There is no way for an **instructor to award a badge for mastering a single learning outcome/competency**, no public per-learner "badge backpack" a student can put on LinkedIn or X, and no learner-owned vanity URL. Competency-based education (already supported via `course_type = 'competency_based'` and standards-based grading) produces granular mastery signals that currently evaporate instead of becoming portable, verifiable credentials. This gap weakens our CBE and self-learner value propositions and cedes the "digital badge" surface to Credly/Badgr.

## 2. Goals

- Let an instructor **award a signed micro-badge** to a student when they achieve competency in a specific learning outcome (manually, or auto-suggested when SBG mastery is reached).
- Give every learner a **public, shareable badge page** at `https://self.lextures.com/badges/<user-code>` (list) and `…/badges/<user-code>/<badge-name>` (single badge), safe to post on LinkedIn / X with rich preview cards.
- Make each badge **cryptographically signed** (reuse Ed25519 `did:web` key), **Open Badges 3.0 compliant**, and **independently verifiable** by anyone without a Lextures account.
- Let students **choose their own `<user-code>`** (vanity handle like `willden`, or an opaque 32-char code) in user settings, with uniqueness, safety, and privacy controls.
- Ship with **privacy defaults appropriate for minors** (public pages opt-in, guardian-gated for K-12) so we do not create a FERPA/COPPA exposure.

## 3. Non-Goals

- **No new signing/crypto primitives.** We reuse `internal/service/vc_signing` (Ed25519 / `did:web`) — no blockchain, no new PKI.
- **No replacement of `gamification.user_badges`** (streak/XP milestone badges, plan 15.9). Those stay separate; this plan is about *competency* micro-credentials. Copy/UX must keep the two visually distinct.
- **Not a full Badgr/Open Badges backpack import**; we issue and host our own badges. Import of externally-issued badges is a future phase.
- **No badge marketplace, stacking into new credentials, or pathways-of-badges** in v1 (future: badge → stackable certificate).
- **No peer- or self-awarded badges** in v1; only staff with grading authority in the course award them.
- **No re-architecture of course outcomes**; we consume the existing outcome/sub-outcome/SBG model as-is.

## 4. Personas & User Stories

- **As an instructor**, I want to define a badge for an outcome in my course and award it to students who demonstrate mastery, so recognition is granular and motivating.
- **As an instructor**, I want the system to *suggest* awarding a badge when a student crosses the SBG mastery threshold for an aligned outcome, so I don't have to track it manually.
- **As a student (HE / self-learner)**, I want a public page of my badges at a URL I control, so I can add it to my LinkedIn profile, résumé, or X bio.
- **As a student**, I want to choose a memorable handle (`willden`) or keep an opaque code, and change it later, so my URL reflects my identity without leaking that I'm at a particular institution if I don't want it to.
- **As an employer / verifier (no account)**, I want to open a badge link and confirm it is authentic, un-revoked, and issued by Lextures for a named skill, so I can trust the claim.
- **As a K-12 parent/guardian**, I want my child's badge page to be private by default and only public with my consent, so my child is not publicly indexable.
- **As a platform admin**, I want to enable/disable the feature per tenant, set default visibility, and revoke badges, so we stay compliant and can correct mistakes.
- **As a compliance officer**, I want public badge pages to expose the minimum PII necessary and honor deletion/withdrawal, so we meet FERPA/COPPA/GDPR obligations.

## 5. Functional Requirements

Written in MUST / SHOULD / MAY (RFC 2119).

**Badge definitions (Achievement / BadgeClass)**
- **FR-1.** An instructor with grading authority in a course MUST be able to create a **badge definition** tied to a `course.course_learning_outcomes` row (or a `course_outcome_sub_outcomes` row) with: name, description, criteria narrative, image, tag list, and alignment (skills/standards).
- **FR-2.** A badge definition MUST have a URL-safe `slug` unique **per issuing course** (used as `<badge-name>`); the system MUST auto-generate it from the name and allow the instructor to edit it.
- **FR-3.** Each badge definition MUST render an Open Badges 3.0 `Achievement` object at a stable, publicly resolvable URL (the `achievement.id`).
- **FR-4.** A badge definition MAY be marked `auto_award = true`, meaning the system awards it automatically when the linked outcome's SBG proficiency for a student reaches the course mastery threshold.

**Awarding (Assertion)**
- **FR-5.** An instructor MUST be able to award a badge to one or many enrolled students; the award MUST be **idempotent** per `(recipient, badge_definition)`.
- **FR-6.** When `auto_award` is set, the system MUST award the badge automatically upon recorded mastery, exactly once, via the same idempotent path, and record the triggering evidence (proficiency row / structure item).
- **FR-7.** Each award MUST be signed as an Open Badges 3.0 / W3C Verifiable Credential using the institution `did:web` key (reuse `vc_signing.SignAchievementCredential`), storing the signed VC as the proof.
- **FR-8.** Each award MUST capture: recipient user, badge definition, `awarded_by` (staff user or `system`), evidence reference, `issued_at`, and a per-award public `share_slug`.
- **FR-9.** Staff who awarded a badge (or a course/tenant admin) MUST be able to **revoke** it with a reason; revoked badges MUST disappear from public pages and verify as `revoked`.
- **FR-10.** The learner MUST be notified (in-app + email, respecting notification prefs) when a badge is awarded.

**Public learner page & handle**
- **FR-11.** A learner MUST be able to set a `badge_handle` (their `<user-code>`) in settings: 3–32 chars, `[a-z0-9-]`, not a reserved word, unique platform-wide, case-insensitive.
- **FR-12.** Until a learner sets a handle, the system MUST mint an unguessable default code (≥22 chars, like `portfolio.public_slug`) so `/badges/<code>` resolves immediately.
- **FR-13.** Changing the handle MUST be rate-limited (e.g. ≤5 changes / 30 days) and the **previous handle MUST 301-redirect** to the current one for a grace window to avoid dead social links.
- **FR-14.** `GET /badges/<user-code>` MUST render a public list of that learner's **public, non-revoked** badges; `GET /badges/<user-code>/<badge-name>` MUST render a single badge with issuer, recipient display name, criteria, issue date, and verify controls.
- **FR-15.** The learner MUST control page visibility: `badge_page_public` (whole page) and per-badge `is_public`. Default visibility is governed by tenant policy and the learner's minor status (see FR-19).
- **FR-16.** Public badge pages MUST expose only the **minimum PII**: the learner's chosen display name (or handle if they opt to hide their name), badge, issuer, dates — never email, student ID, org email domain, or course roster.
- **FR-17.** Each public badge page MUST offer: "Verify", "Download signed JSON", "Download baked PNG", "Add to LinkedIn", and "Share to X" actions (reuse 15.6 helpers).

**Verification**
- **FR-18.** `GET /api/v1/badges/verify/<share-slug>` MUST return the signed VC, revocation status, issuer DID, and a boolean `verified` computed by checking the Ed25519 proof against the resolved public key — usable by third parties with no auth.

**Privacy / governance**
- **FR-19.** For accounts flagged as minors (COPPA/age gating), `badge_page_public` MUST default to **false** and MUST require guardian consent (reuse existing parent-student link + consent flows) before any page or badge can be made public.
- **FR-20.** A platform admin MUST be able to enable/disable the feature per tenant via a feature flag and set the tenant default visibility.
- **FR-21.** Deleting a badge, a learner account, or exercising a data-deletion request MUST remove/tombstone the public page and cause verify to return `revoked`/`not found` (no PII left in caches/OG images).

## 6. Non-Functional Requirements

- **Performance** — Public list page p95 < 300 ms server time; single-badge page p95 < 250 ms; OG image generation p95 < 500 ms (cached thereafter). Public reads served from cache (`objectcache`) keyed by handle+badge+`updated_at`.
- **Security** — Signing key never leaves the server; private seed from `CCR_SIGNING_SEED_B64` (already used). Public endpoints are unauthenticated **read-only** and rate-limited (`ratelimit`) to resist scraping/enumeration. Handles validated against an SSRF/route-injection safe charset. No IDOR: award/revoke/definition mutations require course grading authority (`authz`).
- **Privacy & Compliance** — FERPA: badge is a directory-information-adjacent disclosure requiring student opt-in (and, for minors, guardian consent) — hence FR-15/FR-19. COPPA: minors default private. GDPR/CCPA: public page and OG image are personal data subject to erasure (FR-21); provide "make private" as a one-click withdrawal. Public pages carry `noindex` until the learner explicitly opts into search-engine indexing.
- **Accessibility** — WCAG 2.1 AA on all new UI; public pages fully keyboard-navigable, badge images have descriptive `alt`, verify status announced via `aria-live`, color-independent verified/revoked states.
- **Scalability** — Public traffic is read-heavy and cacheable; badge counts per learner small (10s–100s). OG images generated on demand then cached in object storage.
- **Reliability** — Awarding is idempotent and transactional; signing failures do not lose the award (ret, or store unsigned+re-sign job). Verify degrades gracefully if the DID doc is temporarily unavailable (returns `unverified`, not error).
- **Observability** — Metrics: badges_defined, badges_awarded (manual vs auto), badge_revoked, public_page_views (PII-free, like `portfolio.portfolio_views`), verify_calls, share_clicks by channel; traces on award + sign; alert on sign-failure rate. Reuse `telemetry` (see memory: observability lives in `server/internal/telemetry`).
- **Maintainability** — New Go package `internal/service/badges`; new repo `internal/repos/badges`; reuse `vc_signing`, `credentials` LinkedIn/export helpers, `ccr` signing-key resolution. React under `clients/web/src/pages/badges` + `components/badges`.
- **Internationalization** — All UI strings via i18n; badge name/description authored per course (instructor content, not translated by us); dates/times localized to viewer.
- **Backward compatibility** — Additive migrations only; `credentials.issued_credentials.source_type` CHECK extended to include `outcome` (or badges live in their own schema — see §8 decision). Existing course/path credentials untouched.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor with grading authority in course C, *when* they create a badge definition for outcome O, *then* a definition row exists with a unique per-course slug and a resolvable `Achievement` JSON URL.
- **AC-2.** *Given* a badge definition with `auto_award=true` linked to outcome O, *when* student S's SBG proficiency for O reaches the mastery threshold, *then* exactly one signed award is created for S and S is notified.
- **AC-3.** *Given* an awarded badge, *when* anyone fetches its signed JSON and verifies the Ed25519 proof against the `did:web` document, *then* verification succeeds and `verified=true`.
- **AC-4.** *Given* a revoked badge, *when* its public URL or verify endpoint is hit, *then* the page returns "revoked/not available" and verify returns `revoked=true`.
- **AC-5.** *Given* a learner with handle `willden` and a public badge `algebra-linear-equations`, *when* an unauthenticated user opens `/badges/willden/algebra-linear-equations`, *then* they see the badge, issuer, recipient display name, criteria, issue date, and verify controls — and no other PII.
- **AC-6.** *Given* a learner changes their handle from `oldcode` to `willden`, *when* someone opens the old `/badges/oldcode/...` link within the grace window, *then* they are 301-redirected to `/badges/willden/...`.
- **AC-7.** *Given* a badge page URL is pasted into LinkedIn/X/Slack, *when* the crawler fetches it, *then* it receives server-rendered `og:title`, `og:description`, `og:image` (badge card) and `twitter:card=summary_large_image`.
- **AC-8.** *Given* a minor account with no guardian consent, *when* the learner attempts to make their badge page public, *then* the action is blocked and guardian consent is requested; the page stays private and `noindex`.
- **AC-9.** *Given* two students awarded the same badge definition, *when* the instructor re-runs the award, *then* no duplicate award is created (idempotent).
- **AC-10.** *Given* a learner exercises data deletion, *when* the request completes, *then* their public page 404s, cached OG images are purged, and verify returns not-found.
- **AC-11.** *Given* a handle attempt that is reserved (e.g. `admin`, `api`, `verify`, `settings`), too short, or already taken, *when* the learner submits it, *then* the API rejects it with a specific validation error.

## 8. Data Model

New schema `badges` (keeps concerns separate from `gamification.user_badges` and reuses `credentials` signing).

**Decision:** store awards in `badges` (not by overloading `credentials.issued_credentials`) because badges need a per-award public `share_slug`, per-outcome definitions, and distinct visibility rules. Reuse the *signing* path and Open Badges 3.0 *format* from `vc_signing`. (Alternative considered: extend `issued_credentials.source_type` to `outcome` — rejected to avoid coupling revocation/visibility semantics; may still surface badges in `/me/credentials` via a read-time union.)

```
badges.badge_definitions
  id UUID PK
  course_id UUID NOT NULL REFERENCES course.courses(id) ON DELETE CASCADE
  outcome_id UUID REFERENCES course.course_learning_outcomes(id) ON DELETE SET NULL
  sub_outcome_id UUID REFERENCES course.course_outcome_sub_outcomes(id) ON DELETE SET NULL
  slug TEXT NOT NULL                       -- <badge-name>, unique per course
  name TEXT NOT NULL
  description TEXT NOT NULL DEFAULT ''
  criteria_narrative TEXT NOT NULL DEFAULT ''
  image_key TEXT                            -- object storage key for badge art
  tags TEXT[] NOT NULL DEFAULT '{}'
  alignment_json JSONB                      -- skills/standards alignment (OB 3.0 alignment[])
  auto_award BOOLEAN NOT NULL DEFAULT FALSE
  created_by UUID NOT NULL REFERENCES "user".users(id)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  UNIQUE (course_id, slug)

badges.awarded_badges
  id UUID PK
  definition_id UUID NOT NULL REFERENCES badges.badge_definitions(id) ON DELETE CASCADE
  recipient_id UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE
  awarded_by UUID REFERENCES "user".users(id) ON DELETE SET NULL   -- NULL = system/auto
  award_source TEXT NOT NULL DEFAULT 'manual' CHECK (award_source IN ('manual','auto'))
  evidence_json JSONB                        -- proficiency row id / structure_item_id / narrative
  credential_json JSONB NOT NULL             -- OB 3.0 AchievementSubject
  proof JSONB NOT NULL                       -- signed W3C VC (Ed25519 proof)
  share_slug TEXT NOT NULL UNIQUE            -- opaque token for /verify + public single-badge id fallback
  is_public BOOLEAN NOT NULL DEFAULT FALSE
  revoked BOOLEAN NOT NULL DEFAULT FALSE
  revoked_reason TEXT
  revoked_at TIMESTAMPTZ
  issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  UNIQUE (definition_id, recipient_id)       -- idempotency (FR-5)

badges.badge_page_views                      -- PII-free counter, mirrors portfolio.portfolio_views
  id UUID PK
  handle_owner_id UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE
  awarded_badge_id UUID REFERENCES badges.awarded_badges(id) ON DELETE CASCADE
  viewed_on DATE NOT NULL
  view_count INT NOT NULL DEFAULT 0
  UNIQUE (handle_owner_id, awarded_badge_id, viewed_on)
```

**User handle / visibility** — new table (not columns on `"user".users`, to keep public-profile concerns isolated and support redirect history):

```
"user".user_badge_profiles
  user_id UUID PK REFERENCES "user".users(id) ON DELETE CASCADE
  handle TEXT UNIQUE                          -- current <user-code>; NULL until minted
  handle_lower TEXT UNIQUE GENERATED ALWAYS AS (lower(handle)) STORED
  page_public BOOLEAN NOT NULL DEFAULT FALSE
  search_indexable BOOLEAN NOT NULL DEFAULT FALSE
  display_name_override TEXT                  -- optional; falls back to users.display_name
  hide_real_name BOOLEAN NOT NULL DEFAULT FALSE
  handle_changed_at TIMESTAMPTZ
  handle_change_count_30d INT NOT NULL DEFAULT 0
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

"user".user_badge_handle_history               -- old handle -> redirect (FR-13)
  old_handle_lower TEXT PRIMARY KEY
  user_id UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE
  released_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

**Constraints & indexes**
- `CHECK (handle ~ '^[a-z0-9](?:[a-z0-9-]{1,30})[a-z0-9]$')` (3–32, no leading/trailing/`--`).
- Reserved-handle enforcement in app + a `badges.reserved_handles` seed table (`admin, api, verify, settings, me, badges, www, self, support`, etc.).
- `idx_awarded_badges_recipient (recipient_id, issued_at DESC)`, `idx_awarded_badges_definition (definition_id)`, partial index `WHERE is_public AND NOT revoked` for public list.

**Feature flag** — `settings.platform_app_settings.ff_competency_badges BOOLEAN` (mirrors `ff_completion_credentials` / `ff_gamification`), plus `badges_default_public BOOLEAN` tenant default.

**Migration naming** — next sequential `server/migrations/NNN_competency_badges.sql` (+ `.down.sql`), NNN after the current max (375+). Extend the `user_audit` event-kind CHECK to add `badge_share_linkedin`, `badge_share_x`, `badge_page_view` (following the 283 pattern).

**Backfill** — none required (new tables). Optionally a one-time job to mint default handles lazily on first `/me/badges` load rather than backfilling all users.

## 9. API Surface

Authenticated (`/api/v1`, session/JWT):

- `POST   /api/v1/courses/{courseId}/badge-definitions` — create definition (grading authority). Body: `{outcomeId?, subOutcomeId?, name, slug?, description, criteriaNarrative, tags[], alignment[], autoAward, imageUploadRef?}`.
- `GET    /api/v1/courses/{courseId}/badge-definitions` — list for course.
- `PATCH  /api/v1/badge-definitions/{id}` / `DELETE …` — edit/remove (grading authority).
- `POST   /api/v1/badge-definitions/{id}/award` — award to `{recipientIds: []}` (idempotent). Returns awarded + skipped.
- `GET    /api/v1/badge-definitions/{id}/candidates` — enrolled students + mastery state, to drive the award UI.
- `POST   /api/v1/badges/{awardedId}/revoke` — `{reason}` (awarder/admin).
- `GET    /api/v1/me/badges` — learner's own awards (public + private).
- `PATCH  /api/v1/me/badges/{awardedId}` — `{isPublic}` toggle (learner).
- `GET    /api/v1/me/badge-profile` / `PATCH /api/v1/me/badge-profile` — get/set `{handle, pagePublic, searchIndexable, displayNameOverride, hideRealName}`; validates + rate-limits handle (FR-11/13).
- `GET    /api/v1/badge-handle-available?handle=` — availability/validity check for the settings UI.
- `GET    /api/v1/badges/{awardedId}/linkedin-params` — reuse `credsvc.BuildLinkedInParams`.
- `GET    /api/v1/badges/{awardedId}/badge-export[/download]` — signed JSON / baked PNG (reuse HMAC token pattern from `credsvc.BadgeExportToken`).

Public (unauthenticated, rate-limited, cacheable):

- `GET /badges/{handle}` and `GET /badges/{handle}/{badgeSlug}` — **HTML** routes. Served by the SPA for humans; for crawler user-agents (and always in `<head>`) the server injects OG/Twitter meta via a lightweight prerender shell (see §10). Old handles 301 → current.
- `GET /api/v1/public/badges/{handle}` — JSON list of public, non-revoked badges (name, image, issuer, issuedAt, slug, verifyUrl).
- `GET /api/v1/public/badges/{handle}/{badgeSlug}` — single public badge JSON.
- `GET /api/v1/badges/verify/{shareSlug}` — `{verified, revoked, issuerDid, credential, checkedAt}` (FR-18). No auth.
- `GET /achievements/badge/{definitionId}` — OB 3.0 `Achievement` JSON (the `achievement.id` target).
- `GET /badges/{handle}/{badgeSlug}/og.png` — dynamic OG image (recipient name + badge art + issuer), cached.
- Reuse existing `GET /.well-known/did.json` (issuer key) — no change.

**Rate-limit / quota** — public reads and verify behind `ratelimit` per-IP; handle-change endpoint per-user (FR-13); OG image generation cached to object storage.

**OpenAPI** — all `/api/v1` additions documented in `internal/openapi`; public HTML routes noted in docs.

## 10. UI / UX

**Instructor (course context)**
- New "Badges" tab in course outcomes/competency view: list badge definitions, "New badge" (name, outcome picker, image upload, criteria, tags, auto-award toggle).
- Per-definition "Award" drawer: roster with mastery status (from `candidates`), multi-select, "Award selected". Auto-award badges show a "granted automatically" indicator.
- Revoke action with reason on any awarded badge.

**Learner**
- `/me/badges` page (add route near `/me/credentials`): grid of earned badges (public/private toggle each), empty state ("Earn badges by mastering course outcomes").
- **Settings → Public badge page** panel (new panel in `account-settings-view.tsx`): handle editor with live availability check, current public URL with copy button, `pagePublic` + `searchIndexable` toggles, display-name/hide-name options. For minors: toggles disabled with a "guardian consent required" affordance.

**Public pages** (standalone, add `/badges` to `standalone-public-routes.ts`)
- `/badges/:handle` — header (display name/handle + avatar initial), responsive badge grid, "Verified by Lextures" trust mark, share bar. States: loading skeleton, empty ("No public badges yet"), private ("This badge page is private"), not-found.
- `/badges/:handle/:badgeSlug` — large badge art, name, issuer, recipient, criteria, issued date, `Verify` (calls verify endpoint, shows ✓ Verified / ⚠ Revoked / ✗ Unverifiable via `aria-live`), Download JSON, Download baked PNG, Add to LinkedIn, Share to X. States: loading / revoked / not-public / not-found.

**Social preview / OG** (the current SPA gap)
- The public `/badges/*` routes are served through a server handler that returns the SPA shell **with per-badge `<meta>` injected**: `og:title` = "{Name} earned {Badge} — verified by Lextures", `og:description` = criteria/issuer, `og:image` = `…/og.png`, `og:url`, `twitter:card=summary_large_image`, plus `<link rel="canonical">` and `<meta name="robots" content="noindex">` unless `searchIndexable`. This can be a small chi route that reads the badge, renders `index.html` with substituted tags (bots and humans both get correct tags; humans then hydrate the SPA).

**Mobile / responsive** — grid collapses to single column; share sheet uses native share where available.
**Accessibility** — focus order badge→verify→share; badge images `alt="{Badge name} badge"`; verified state not color-only (icon + text); all controls keyboard reachable.
**Copy & i18n** — new keys under `badges.*`; verify states, empty/private/revoked messages, guardian-consent notice.

## 11. AI / ML Considerations

Not AI-touching in v1. (Future MAY: auto-suggest badge name/criteria from outcome text, or auto-generate badge art — out of scope; if added, route via existing AI gateway with PII redaction and cost budget.)

## 12. Integration Points

- **Signing** — `internal/service/vc_signing` (`SignAchievementCredential`, `VerifyCredential`, `DIDDocument`); key resolved via `ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)`.
- **Existing credentials helpers** — `internal/service/credentials`: `BuildLinkedInParams`, `BadgeExportToken`/`VerifyBadgeExportToken`, PDF builder (reused for a badge certificate variant).
- **Outcomes / mastery** — `course.course_learning_outcomes`, `course_outcome_sub_outcomes`, `course.student_standard_proficiencies` (SBG) and the recompute path that updates proficiency → the auto-award hook subscribes here.
- **Enrollment / roster / authz** — `courseroles` / `authz` for grading-authority checks and the award roster.
- **Notifications** — `notifevents` / `mail` for award notifications (reuse `notifyCertificateIssued` pattern in `credentials_http.go`).
- **Object storage** — badge art + baked PNG + cached OG images (same store as credential PDFs / portfolio assets).
- **Caching** — `objectcache` for public reads; `ratelimit` for public endpoints.
- **Web** — router `clients/web/src/app.tsx`, `lib/standalone-public-routes.ts`, `lazy-pages.ts`; new pages/components under `pages/badges`, `components/badges`.
- **Webhook/events** — emit `badge.awarded` / `badge.revoked` on the existing webhook bus for downstream (e.g., SIS/HR) consumers.

## 13. Dependencies & Sequencing

- **Must ship after:** 15.5 (issuance), 15.6 (LinkedIn/export), 14.13 (signing key + `did:web`), 3.7 (SBG mastery source of truth). All are shipped/completed.
- **Must ship before:** any future "stackable credential / badge pathway" work.
- **Shared infra needed:** object storage (badge art, OG images), the Ed25519 signing seed configured (`CCR_SIGNING_SEED_B64`), `PUBLIC_WEB_ORIGIN`, background worker for auto-award + OG pre-render, email for notifications.
- **Sequencing within this plan:** (a) migrations + repo + signing reuse → (b) award/definition APIs → (c) instructor UI → (d) learner `/me/badges` + settings handle → (e) public pages + verify → (f) OG/prerender + baked PNG → (g) auto-award hook → (h) minor/guardian gating.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Public pages expose minors' PII / FERPA violation | M | H | Default private; minors require guardian consent (FR-19); minimal PII (FR-16); `noindex` default; deletion purges (FR-21). |
| Handle squatting / impersonation (e.g. someone grabs `willden`) | M | M | Reserved words, uniqueness, first-come; report/abuse path; admin re-assign; no institution names as free handles. |
| Social crawlers get no preview (SPA) | H | M | Server-injected OG tags on `/badges/*` (§10, AC-7). |
| Badge devaluation (over-awarding, auto-award noise) | M | M | Auto-award gated on real SBG mastery threshold; instructor can disable; badges scoped to outcomes not participation. |
| Signing key compromise invalidates all badges | L | H | Key stays server-side; rotation plan via `did:web` (publish new `verificationMethod`, keep old for historical verify); alerting on sign failures. |
| Verify fails when DID doc unreachable | L | M | Cache DID doc; verify degrades to `unverified` (not error); self-contained proof includes issuer. |
| Confusion with gamification streak "badges" | M | L | Distinct schema, naming ("micro-credential"), and UI; keep surfaces separate (§3). |
| Enumeration/scraping of public pages | M | M | Rate-limit, opaque default handles, `noindex`, no directory listing endpoint. |
| Stale social links after handle change | M | L | 301 redirect from old handle for a grace window (FR-13). |

## 15. Rollout Plan

- **Feature flag:** `ff_competency_badges` (default **off**), tenant default visibility `badges_default_public` (default off). Gate all routes/UI.
- **Migration sequencing:** schema (badges + user_badge_profiles + reserved seed) → deploy code behind flag → enable for internal/dogfood tenant → pilot with 1–2 CBE courses (HE + a self-learner cohort) → GA.
- **Dogfood / pilot cohort:** Lextures-internal course; a competency-based HE pilot; a self-learner cohort issuing outcome badges.
- **GA criteria:** AC-1…AC-11 pass; verify works from an external verifier; LinkedIn/X unfurl confirmed; a11y (axe + SR) clean; load test on public read path green; privacy review signed off (FERPA/COPPA).
- **Comms:** instructor help-center article, learner "share your badges" announcement, verifier explainer page.
- **Rollback:** flip flag off (hides UI + returns 404 on public routes); data retained; no destructive migration.

## 16. Test Plan

- **Unit** — handle validation (charset, reserved, length, uniqueness, rate-limit); slug generation/uniqueness per course; award idempotency; VC signing/verifying round-trip (reuse `vc_signing` tests); revocation state transitions; OB 3.0 JSON shape.
- **Integration (DB/API)** — create definition → award → fetch public JSON → verify; auto-award fires exactly once on mastery; revoke hides + verifies revoked; handle change writes history + redirect; minor cannot make public without consent; authz matrix (non-grader cannot define/award/revoke).
- **End-to-end (Playwright)** — instructor defines + awards; learner sets handle in settings and toggles public; unauthenticated visitor views list + single badge and clicks Verify (✓); revoked badge shows revoked; LinkedIn param + baked PNG download; old-handle redirect.
- **Security** — IDOR on award/revoke/definition endpoints; enumeration/rate-limit on public + verify; SSRF/route-injection via crafted handles; signature-tamper → `verified=false`; export-token forgery/expiry (reuse `VerifyBadgeExportToken` cases); ensure no PII beyond FR-16 in JSON/OG.
- **Accessibility** — axe on all new pages; screen-reader script for verify status (`aria-live`), badge grid, settings panel; keyboard-only award flow.
- **Performance / load** — k6/vegeta on public list + single + verify + OG image at target p95s; cache-hit ratio check.
- **Manual exploratory** — social unfurl on LinkedIn, X, Slack, iMessage; handle edge cases; guardian-consent flow; deletion purges public page + OG cache.

## 17. Documentation & Training

- **End-user (learner)** help-center: "Earn, share, and verify your badges"; choosing a handle; privacy controls; adding a badge to LinkedIn.
- **Instructor** docs: defining outcome badges, awarding manually vs auto, revoking.
- **Verifier** public explainer page: how Lextures badges are signed and how to verify independently (link to `did:web` + JSON).
- **Admin/runbook:** enabling the flag, tenant default visibility, handling abuse/impersonation reports, key rotation, purge-on-deletion runbook.
- **API reference:** OpenAPI updates for all new `/api/v1` endpoints; note public routes.

## 18. Open Questions

1. **Storage decision confirm** — separate `badges` schema (this plan) vs. extending `credentials.issued_credentials` with `source_type='outcome'`? Recommendation: separate schema, optional read-time union into `/me/credentials`.
2. **Handle namespace scope** — platform-wide unique vs. per-tenant? (Public URL is host-global `self.lextures.com`, so recommend platform-wide.) Multi-tenant custom domains would change this.
3. **Baked badge format** — bake assertion into PNG (Open Badges "baking") and/or SVG? PNG at minimum for portability.
4. **OG/prerender mechanism** — chi HTML-shell injection (proposed) vs. a dedicated prerender/edge worker? Confirm with infra.
5. **Auto-award threshold** — reuse course SBG mastery level exactly, or a separate per-definition threshold?
6. **Revocation transparency** — publish a public revocation list / status endpoint, or only per-badge verify? (OB 3.0 supports `credentialStatus`.)
7. **Guardian consent reuse** — can we reuse the existing parent-student link + consent record as the gate, or is a new consent artifact required for "public web page"?
8. **Institution vs. individual issuer identity** — issuer is the tenant/institution `did:web`; do self-learner (no-institution) badges issue under a generic "Lextures" issuer? (Config `CCRInstitutionName` already handles the fallback.)

## 19. References

- Existing code this work builds on / touches:
  - `server/internal/service/vc_signing/` (`sign.go`, `achievement.go`) — Ed25519 / `did:web` / OB 3.0 proof.
  - `server/internal/service/credentials/service.go` — issuance, LinkedIn params, badge export token, PDF.
  - `server/internal/httpserver/credentials_http.go`, `ccr_http.go` (`/.well-known/did.json`, `handleInstitutionDID`).
  - `server/migrations/283_credentials_linkedin_share.sql` — `credentials` schema + `user_audit` share events pattern.
  - `server/migrations/287_gamification.sql` — the *other* `user_badges` (keep separate).
  - `server/migrations/072_course_learning_outcomes.sql`, `100_competency_courses.sql`, `107_standards_based_grading.sql` — outcome/mastery source.
  - `server/migrations/257_eportfolio_capstone.sql` — public-slug + PII-free `portfolio_views` precedent.
  - `clients/web/src/app.tsx`, `src/lib/standalone-public-routes.ts`, `src/pages/verify/CcrVerify.tsx`, `src/pages/lms/MyCredentials.tsx`, `src/components/settings/account-settings-view.tsx`.
  - `server/internal/config/config.go` — `PublicWebOrigin`, `CCRSigningSeedB64`, `CCRInstitutionName`, `FFCompletionCredentials`.
- External standards:
  - IMS Global / 1EdTech **Open Badges 3.0** and **Comprehensive Learner Record (CLR) 2.0**.
  - **W3C Verifiable Credentials Data Model** + **`did:web`** method.
  - Ed25519Signature2020 proof suite.
  - FERPA (directory-information disclosure), COPPA (minors), GDPR/CCPA (erasure of public personal data), WCAG 2.1 AA.
- Related plans: `../../completed/15-self-learner-specific/15.5-certificates-open-badges.md`, `../../completed/15-self-learner-specific/15.6-linkedin-share-open-badges-export.md`; `../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md` (CLR signing key / `did:web`); `../../completed/03-submissions-grading-integrity/3.7-standards-based-grading.md` (SBG mastery source).
