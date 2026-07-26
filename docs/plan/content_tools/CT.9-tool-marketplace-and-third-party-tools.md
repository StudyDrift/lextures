# CT.9 — Content Tools: Tool Marketplace & Third-Party Tools

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.9 |
| **Section** | Content Tools (CT) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Platform / ecosystem team |
| **Depends on** | CT.5 (sandbox + versioning), CT.8 (conformance bar) |
| **Unblocks** | Third-party ecosystem; paid tools; long-tail subject coverage |

---

## 1. Problem Statement

Lextures cannot build every interaction every subject needs — a chemistry equation balancer, a music
interval trainer, a Mandarin stroke-order practice pad, a nursing dosage calculator. Each is
high-value to a narrow audience and a poor use of a small platform team. The Content Tools framework
already makes a tool a *declarative unit* (manifest + renderer + optional action handler); this story
opens that unit to people outside Lextures: a place to publish, a review process that protects
students, an install flow that respects org policy, and versioned distribution that does not break
courses. It is the difference between a shelf of 20 tools and a shelf of hundreds.

## 2. Goals

- Ship the **publish → review → list → install → update → revoke** lifecycle for third-party tools.
- Make installation an **org-level, admin-consented** act with an explicit capability review, then a
  per-course opt-in — never a surprise for a district.
- Guarantee that a third-party tool is sandboxed (CT.5), policy-governed (CT.8) and observable, with
  the same conformance bar first-party tools meet.
- Reuse the shipped marketplace (`MKT1–MKT10`) and OAuth-app (`16.9`) machinery rather than building a
  second storefront and a second developer portal.
- Leave a clean path to paid tools and revenue share without requiring it for v1.

## 3. Non-Goals

- Payments and payouts for paid tools (design for it; ship free-only in v1, reusing the shipped
  billing stack when enabled).
- Third-party code in the Lextures Go process — never; tools are sandboxed browser bundles plus, at
  most, the developer's own backend.
- Automated certification: v1 review is human, assisted by automated checks.
- A separate developer identity system — developers are Lextures accounts with a developer role.

## 4. Personas & User Stories

- **As an EdTech developer**, I want to publish an interactive tool so that instructors on Lextures can
  drop it into their pages without an integration project.
- **As an instructor**, I want to browse tools by subject and grade so that I can find something that
  fits tomorrow's lesson.
- **As an org admin**, I want to review exactly what a tool collects and where it sends data before it
  is available to my teachers so that I stay compliant.
- **As a security officer**, I want to revoke a tool instantly across the org so that an incident is
  containable.
- **As a developer**, I want to ship v2 of my tool without breaking courses on v1 so that my users
  trust updates.
- **As a student**, I want third-party tools to be as accessible and as private as first-party ones so
  that "a cool tool" is not a downgrade in my rights.

## 5. Functional Requirements

- **FR-1.** The system MUST provide a **developer portal** for tool authors: create a tool, upload a
  bundle + manifest, fill the Tool Data Sheet (CT.8 FR-1), submit for review, and view install/usage
  analytics for their own tool only.
- **FR-2.** Submission MUST run automated checks: manifest validity, schema/semver consistency (CT.5),
  bundle budget, axe on the developer-supplied harness stories, keyboard-operability test presence,
  i18n completeness, no disallowed APIs, declared network hosts resolvable and non-private.
- **FR-3.** A human **review queue** MUST gate listing: reviewers see the data sheet, capability list,
  automated results, a sandboxed live preview, and approve / reject with reasons.
- **FR-4.** Listings MUST carry: name, description, screenshots, subject/grade tags, capabilities,
  data sheet, accessibility statement, version history, developer identity and support contact.
- **FR-5.** Installation MUST be **org-scoped** and require an admin consent screen enumerating
  capabilities in plain language (e.g. "sends student writing to an external service").
- **FR-6.** Installation MUST write the tool's `network.allowedHosts` into the CSP allowlist enforced
  by CT.5, and MUST NOT permit runtime expansion of that list.
- **FR-7.** After org install, each course MUST still opt in via the CT.1 allowlist before the tool
  appears in the palette.
- **FR-8.** Updates MUST follow CT.5 semver rules: patch/minor auto-update after a soak window
  (default 7 days); a major requires explicit admin re-consent and never auto-migrates instances.
- **FR-9.** Revocation MUST be immediate and org-wide: existing instances render a read-only tombstone,
  new interactions are blocked, and student state is preserved and exportable.
- **FR-10.** Third-party tools MUST run in `sandbox: 'iframe'` mode (CT.5 FR-10) — the manifest field
  is forced, not trusted.
- **FR-11.** Bundles MUST be served from Lextures-controlled storage/CDN with Subresource Integrity,
  so revocation and integrity are platform-controlled.
- **FR-12.** The platform MUST record and expose per-tool operational health (errors, latency, breaker
  state) to both admins and the developer.
- **FR-13.** A developer MUST be able to deprecate and sunset a tool, with notice to installed orgs and
  a documented minimum notice period (default 90 days).
- **FR-14.** The system MUST support a **private/unlisted** distribution mode so a district or
  university can publish an internal tool without a public listing.
- **FR-15.** The system MUST expose install and usage counts to developers in aggregate only, with no
  student-level data ever leaving the platform to a developer.
- **FR-16.** The listing model MUST carry pricing fields (`free`, `paid`, `trial`) even though v1 only
  permits `free`, so enabling commerce later is configuration plus billing wiring, not a redesign.
- **FR-17.** Every lifecycle action (submit, approve, reject, install, update, revoke, sunset) MUST be
  audited and MUST emit a webhook event.

## 6. Non-Functional Requirements

- **Performance** — Marketplace browse p95 ≤ 300 ms (cached); install completes ≤ 2 s; bundle load
  respects the CT.5 budget per tool.
- **Security** — This is the highest-risk story in the folder. Controls: forced sandbox, platform-hosted
  bundles with SRI, CSP allowlists derived from consent, no credentials in the iframe, opaque
  participant ids, review gate, immediate revocation, and a pre-GA external penetration test focused on
  sandbox escape and bridge abuse.
- **Privacy & Compliance** — A third-party tool that receives data is a **sub-processor**: install
  updates the org's sub-processor disclosure (S07), the DPA flow (`service/dpa`) applies, and the data
  sheet is the source of truth for the RoPA (S05). Tools processing children's data face the S08
  requirements; the review checklist encodes them.
- **Accessibility** — Third-party tools meet the identical CT.8 gate; a listing displays its WCAG level
  and limitations, and a failing tool cannot be listed (S20 exposure is ours, not the developer's).
- **Scalability** — Listings and manifests cached at the edge; install state is small and org-scoped.
- **Reliability** — A misbehaving third-party tool trips the CT.5 breaker and degrades to a
  placeholder; platform availability never depends on a developer's uptime except for their own
  backend calls, which fail soft.
- **Observability** — `lextures_content_tool_marketplace_installs_total{tool_id,action}`,
  `…_marketplace_review_queue_depth`, `…_thirdparty_tool_errors_total{tool_id}`,
  `…_thirdparty_bundle_load_seconds{tool_id}`. Alerts on error spikes and on breaker opens.
- **Maintainability** — One lifecycle service; review criteria are a versioned checklist document, so a
  policy change is a document change plus a checklist version bump.
- **Internationalization** — Listings support localized name/description; the review checklist requires
  at least one fully localized locale; RTL verified in the sandbox preview.
- **Backward compatibility** — First-party tools are unaffected. Orgs that never install anything see
  no change.

## 7. Acceptance Criteria

- **AC-1.** *Given* a submitted tool failing an automated check, *When* submission completes, *Then* it
  is rejected before reaching the human queue with the failing check named.
- **AC-2.** *Given* an approved tool, *When* an org admin installs it, *Then* the consent screen lists
  every capability in plain language and installation is recorded with the consenting admin.
- **AC-3.** *Given* an installed tool, *When* an instructor opens the palette in a course that has not
  opted in, *Then* the tool is absent until the course allowlist includes it.
- **AC-4.** *Given* an installed third-party tool, *When* it attempts a network request to a host not
  in its consented list, *Then* the request is blocked by CSP and reported.
- **AC-5.** *Given* a minor version update, *When* the soak window elapses, *Then* installations move to
  the new version automatically and instances continue to render.
- **AC-6.** *Given* a major version, *When* it is published, *Then* installed orgs stay on the prior
  major until an admin re-consents.
- **AC-7.** *Given* an org revokes a tool, *When* a student loads a page containing it, *Then* a
  read-only tombstone renders, no bundle is fetched, and their state remains exportable.
- **AC-8.** *Given* a developer views their dashboard, *Then* they see aggregate installs and usage and
  **no** student-identifiable data anywhere in the payload.
- **AC-9.** *Given* an unlisted tool, *When* a non-invited org browses the marketplace, *Then* it is not
  discoverable and direct install by id fails with 404.
- **AC-10.** *Given* a hostile fixture tool, *When* the sandbox test suite runs, *Then* every escape
  attempt (cookie, storage, parent DOM, disallowed fetch, message flood, oversized state) fails safely.
- **AC-11.** *Given* a tool announces sunset, *When* the notice is published, *Then* installed org
  admins are notified and the listing shows the end date at least 90 days ahead.
- **AC-12.** *Given* any lifecycle action, *Then* an audit row and a webhook event exist for it.

## 8. Data Model

Migration `server/migrations/457_content_tool_marketplace.sql` (+ `.down.sql`).

```sql
-- 457_content_tool_marketplace.sql

CREATE SCHEMA IF NOT EXISTS toolmarket;

-- A publishable tool owned by a developer (person or org).
CREATE TABLE IF NOT EXISTS toolmarket.tools (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id           TEXT NOT NULL UNIQUE,      -- global namespace, immutable, e.g. 'acme.titration_lab'
    owner_user_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    owner_org_id      UUID REFERENCES tenant.organizations (id) ON DELETE SET NULL,
    display_name      TEXT NOT NULL,
    summary           TEXT NOT NULL,
    description_md    TEXT NOT NULL DEFAULT '',
    subject_tags      TEXT[] NOT NULL DEFAULT '{}',
    grade_tags        TEXT[] NOT NULL DEFAULT '{}',
    support_url       TEXT,
    privacy_url       TEXT,
    visibility        TEXT NOT NULL DEFAULT 'private'
                        CHECK (visibility IN ('private','unlisted','public')),
    pricing_model     TEXT NOT NULL DEFAULT 'free'
                        CHECK (pricing_model IN ('free','paid','trial')),
    status            TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','in_review','approved','rejected','suspended','sunset')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Immutable published versions with their bundle + manifest + data sheet.
CREATE TABLE IF NOT EXISTS toolmarket.tool_releases (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_pk           UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    version           TEXT NOT NULL,
    manifest_json     JSONB NOT NULL,
    data_sheet_json   JSONB NOT NULL,
    bundle_object_id  UUID REFERENCES storage.objects (id) ON DELETE SET NULL,
    bundle_sri        TEXT NOT NULL,
    checks_json       JSONB NOT NULL DEFAULT '{}'::jsonb,   -- automated results
    review_status     TEXT NOT NULL DEFAULT 'pending'
                        CHECK (review_status IN ('pending','approved','rejected')),
    reviewed_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    review_notes      TEXT,
    published_at      TIMESTAMPTZ,
    sunset_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tool_pk, version)
);
CREATE INDEX IF NOT EXISTS idx_tmr_review ON toolmarket.tool_releases (review_status, created_at);

-- Org installations, with the consented capability + host set frozen at consent time.
CREATE TABLE IF NOT EXISTS toolmarket.tool_installations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    tool_pk           UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    pinned_major      INTEGER NOT NULL,
    current_version   TEXT NOT NULL,
    consented_capabilities TEXT[] NOT NULL DEFAULT '{}',
    consented_hosts   TEXT[] NOT NULL DEFAULT '{}',
    auto_update_minor BOOLEAN NOT NULL DEFAULT TRUE,
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','revoked','suspended')),
    installed_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    installed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at        TIMESTAMPTZ,
    UNIQUE (org_id, tool_pk)
);
CREATE INDEX IF NOT EXISTS idx_tmi_org_status ON toolmarket.tool_installations (org_id, status);

-- Invitations for unlisted distribution.
CREATE TABLE IF NOT EXISTS toolmarket.tool_access_grants (
    tool_pk   UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    org_id    UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    granted_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tool_pk, org_id)
);
```

**Namespacing** — third-party `tool_id`s are prefixed with the developer namespace (`acme.`), so they
can never collide with first-party ids and CT.1's immutability rule still holds.

## 9. API Surface

**Developer portal** (`/api/v1/developer/tools/*`) — create, upload release, submit, list own
installs/usage. **Marketplace** (`/api/v1/tool-marketplace/*`) — browse, detail (public read).
**Admin** (`/api/v1/orgs/{org_id}/tool-installations/*`) — install, consent, update, revoke.
**Review** (`/api/v1/admin/tool-reviews/*`) — queue, approve, reject.

| Verb | Path | Auth scope |
|---|---|---|
| `POST` | `/api/v1/developer/tools` | developer |
| `POST` | `/api/v1/developer/tools/{tool_id}/releases` | developer |
| `POST` | `/api/v1/developer/tools/{tool_id}/releases/{version}/submit` | developer |
| `GET` | `/api/v1/developer/tools/{tool_id}/analytics` | developer (aggregate only) |
| `GET` | `/api/v1/tool-marketplace/tools?subject=&grade=&q=` | public |
| `GET` | `/api/v1/tool-marketplace/tools/{tool_id}` | public |
| `POST` | `/api/v1/orgs/{org_id}/tool-installations` | org admin |
| `PATCH` | `/api/v1/orgs/{org_id}/tool-installations/{id}` (update/consent/auto-update) | org admin |
| `DELETE` | `/api/v1/orgs/{org_id}/tool-installations/{id}` (revoke) | org admin |
| `GET` | `/api/v1/admin/tool-reviews?status=` | platform reviewer |
| `POST` | `/api/v1/admin/tool-reviews/{release_id}/decision` | platform reviewer |

- **Rate limits** — release upload 10/day/developer; browse 600/min/IP (cached at edge).
- **Webhooks** — `tool.submitted`, `tool.approved`, `tool.installed`, `tool.updated`, `tool.revoked`,
  `tool.sunset`.
- **OpenAPI** — all routes documented; public marketplace routes included in the public API spec.

## 10. UI / UX

**Developer portal** (new area, reusing the shipped developer-portal shell from 16.9): tool list →
tool detail (releases, checks, review status, analytics) → new release wizard (upload bundle, fill
data sheet, run checks, preview in sandbox, submit).

**Marketplace browse** (reusing the shipped marketplace UI patterns): filter by subject, grade,
capability and price; cards with accessibility and data badges; detail page with screenshots,
capabilities, data sheet, version history and an **Install** action for admins.

**Admin install flow**: capability consent screen (plain language, grouped by risk), host allowlist
review, auto-update preference, confirm → installed → per-course enablement guidance.

**Reviewer queue**: list ordered by wait time; detail shows automated check results, data sheet,
sandboxed live preview and an approve/reject form requiring a reason on rejection.

**Instructor/student** — third-party tools are visually identical to first-party ones except for a
"Provided by {developer}" attribution line and an info affordance opening the data sheet.

**States** — *Pending review*: developer-visible status timeline. *Revoked*: tombstone for students,
explanatory banner for instructors. *Update available*: admin badge with changelog. *Sunset*: countdown
on the listing and in the editor card.

**Accessibility** — the marketplace and portal meet WCAG 2.1 AA; consent screens are keyboard-complete
with no colour-only risk encoding; the sandboxed preview is labelled and focus-managed.

**Copy & i18n** — `contentTools.marketplace.*`, `contentTools.developer.*`; capability descriptions are
reviewed plain-language strings, not developer-supplied text.

## 11. AI / ML Considerations

- Tools that use AI must declare it; if they use **their own** provider, the listing states that
  clearly, the org's AI policy governs *installation*, and the tool must present its own disclosure
  through the SDK's disclosure primitive.
- Tools may **not** call Lextures' model credentials. A third-party tool that wants platform AI must
  request the `platform_ai` capability, which routes through `aigateway` with the org's budget and
  disclosure — reviewed case by case and off by default in v1.
- Review includes an AI-safety pass for tools that generate content shown to students: prompt-injection
  resistance, age-appropriateness, and a documented human-oversight story.

## 12. Integration Points

- **Internal** — `service/marketplace` (storefront patterns), `service/licensesvc`, `service/dpa`,
  `service/webhooks`, `service/adminaudit`, `service/filestorage` (bundles), `service/contenttools`
  (registry federation: DB-backed third-party manifests merged with in-process first-party ones),
  `service/billing` (dormant until paid tools).
- **16.9 OAuth apps** — a tool that also needs API access registers a 16.9 app; the two consents are
  presented together at install to avoid a second, confusing flow.
- **Infra** — tool-bundle storage + CDN, the CT.5 tool origin, CSP management.
- **www** — public marketplace pages for SEO, following the shipped `www/` marketplace patterns.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.5 (sandbox, versioning, budgets), CT.8 (conformance and policy).
- **Must ship before:** any external developer onboarding.
- **Shared infra needed:** object storage + CDN, review staffing, webhook infrastructure, developer
  identity/role.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Malicious tool exfiltrates student data | L | H | Forced sandbox, no credentials, CSP allowlist from consent, platform-hosted bundles, review gate, external pen-test, instant revocation |
| Review becomes a bottleneck or is rubber-stamped | H | M | Automated checks first, versioned checklist, SLA + queue-depth metric, unlisted mode for internal tools to reduce queue pressure |
| Low-quality tools erode trust in the shelf | M | M | Conformance bar, ratings/report affordance, health metrics, suspension path |
| Developer abandons a tool used by courses | M | M | Sunset policy with 90-day notice, export of student state, platform-hosted bundle keeps it running through the notice period |
| Sub-processor obligations missed on install | M | H | Install writes the disclosure automatically; DPA flow triggered; RoPA entry generated from the data sheet |
| Namespace squatting or impersonation | M | M | Reserved-prefix policy, verified developer identity, trademark takedown path |

## 15. Rollout Plan

- **Feature flag** — `ff_content_tool_marketplace` (platform) plus per-org opt-in. Ships **unlisted-only**
  first (districts publishing internal tools), then approved public listings.
- **Sequencing** — migration `457_*` → developer portal + automated checks → review queue → install
  consent + CSP wiring → marketplace browse → webhooks/audit → pen-test → limited public launch with a
  hand-picked cohort of 3–5 partners.
- **Dogfood** — publish two first-party tools *through the marketplace pipeline* to prove the path.
- **GA criteria** — pen-test findings closed; review SLA met for the pilot cohort; zero policy-bypass
  findings; legal sign-off on the developer agreement and sub-processor flow.
- **Rollback** — disable the platform flag (installed tools continue or are suspended by choice);
  per-tool suspension without a deploy.

## 16. Test Plan

- **Unit** — automated check runners; semver/consent interaction; install-state resolution; namespace
  validation; unlisted access grants.
- **Integration** — full lifecycle (submit → approve → install → render → update → revoke); CSP
  allowlist derivation; SRI enforcement; breaker interaction; audit + webhook emission.
- **End-to-end** — Playwright: developer publishes, reviewer approves, admin installs and consents,
  instructor enables and places the tool, student completes it, admin revokes and the tombstone renders.
- **Security** — hostile-tool suite (CT.5) run against a marketplace-distributed bundle; tampered SRI;
  attempts to widen hosts post-consent; cross-org install-by-id on unlisted tools; developer analytics
  payload asserted free of student data; external pen-test.
- **Accessibility** — axe on portal, marketplace and consent screens; screen-reader script for install.
- **Performance / load** — browse under edge cache; install under concurrency; bundle fetch latency.
- **Manual exploratory** — a deliberately mediocre tool taken through review to validate the checklist;
  sunset flow with real notice timing compressed.

## 17. Documentation & Training

- **Developer** — "Publish a Content Tool": manifest, SDK, budgets, data sheet, review checklist,
  versioning and sunset policy, plus the developer agreement.
- **Admin** — evaluating and installing a tool; what consent means; revocation; sub-processor impact.
- **Reviewer** — the versioned review checklist and decision guidance.
- **Public** — marketplace listing pages and the trust centre data sheets.
- **Runbook** — suspending a tool, handling a security report, emergency revocation across orgs.

## 18. Open Questions

1. Do we allow a third-party backend at all in v1, or restrict to fully client-side tools (state via
   our API only)? Proposed: client-side-only for v1 — dramatically smaller privacy surface — with
   backends unlocked in a follow-up once the DPA flow is proven.
2. Revenue share and payout timing when paid tools open — reuse the course marketplace's Stripe path?
   Proposed: yes, but only after tax handling (15.13) lands.
3. Should ratings/reviews be instructor-only or open to students? Proposed: instructor-only, to avoid
   moderation load and gaming.
4. Who staffs review at volume, and what is the SLA? Proposed: 5 business days for v1 with a
   queue-depth alert; revisit before public launch.

## 19. References

- Existing files this work touches: `server/internal/service/marketplace/`,
  `server/internal/service/dpa/`, `server/internal/service/webhooks/`,
  `server/internal/service/filestorage/`, `www/` marketplace pages,
  `server/migrations/457_content_tool_marketplace.sql`.
- Related shipped work: [16.9 marketplace / plugin system](../../completed/16.9-marketplace-plugin-system.md),
  [MKT1–MKT10 course marketplace](../../completed/marketplace/).
- External standards: OAuth 2.1 (for tools that also register an app), CSP Level 3, Subresource
  Integrity, GDPR Art. 28 (processors/sub-processors).
- Related plans: [CT.5](CT.5-tool-sdk-sandboxing-and-versioning.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [S07 transfers & sub-processors](../standards/S07-cross-border-transfer-subprocessor-governance.md).
