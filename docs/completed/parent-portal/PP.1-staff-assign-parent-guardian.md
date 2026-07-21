# PP.1 — Staff Assign Parent / Guardian

> Implementation plan. Extends shipped parent portal ([13.1](../13-k12-specific/13.1-parent-portal.md), [W02](../web/W02-parent-guardian-portal-completeness.md)). Source flow: staff permission → sidenav → student search → assign 1–3 guardians → immediate link or activate-email pairing.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | PP.1 |
| **Section** | Parent Portal |
| **Severity** | MAJOR |
| **Markets** | K12 |
| **Status (today)** | DONE — `org:parent-links:assign:manage`, parent-assign APIs, activate consume, staff Assign parents UI |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | K-12 pod (backend + frontend) |
| **Depends on** | 13.1 (parent portal + `parent_student_links`), 5.10 (parent role / account type), platform people invite + password-reset email plumbing |
| **Unblocks** | Front-office / counselor workflows for family linking without Org Admin; higher parent-portal adoption |

---

## 1. Problem Statement

K-12 schools need office staff, counselors, and registrars to connect parents/guardians to students without giving them full Org Admin. Today linking is API/CLI-only and requires an **existing** parent user id in the same org; if the guardian has no account, the row is skipped (CSV bulk) or the call fails. Families never get an activate email, and there is no permission-gated home-nav surface for this job. That forces districts onto SIS CSV imports or Global Admin, which blocks day-to-day front-office use of the shipped parent portal.

## 2. Goals

- Add a **grantable custom permission** `org:parent-links:assign:manage` that authorizes assigning parent/guardian links (independent of Org Admin).
- Show a **home sidenav** item only when the signed-in user holds that permission (and the parent-portal feature is on).
- Provide a **student search → assign** UI with a modal for **1–3** guardians (name + email each).
- **Pair immediately** when a matching account exists; otherwise **email an activate link** that creates/activates the account and **auto-pairs** on success.
- Audit who linked whom, and keep FERPA scoping (same org, linked children only for the resulting parent portal).

## 3. Non-Goals

- Rebuilding the parent-facing dashboard (already shipped in 13.1 / W02).
- SIS auto-sync of guardians (13.7) — this plan is the manual staff path; SIS can call the same service later.
- Hard global cap of three guardians per student forever (modal allows up to three **per save**; existing additional links remain unless product later decides a hard cap — see Open Questions).
- Parent self-serve “claim my child” without staff initiation.
- Mobile-native assign UI (web first; mobile can follow).

## 4. Personas & User Stories

- **As a school registrar**, I want a permission to assign parents so that I can do my job without Org Admin rights.
- **As a counselor**, I want to search for a student and assign up to three guardians in one modal so that blended families are connected in one visit.
- **As a parent who already has a Lextures account**, I want to be linked as soon as staff saves my email so that I can open the Family dashboard immediately.
- **As a parent without an account**, I want an email with an activate link so that setting my password automatically connects me to my child.
- **As a Global Admin**, I want to grant `org:parent-links:assign:manage` on a custom role so that front-office staff get only this capability.
- **As a student (or privacy officer)**, I want every link creation audited so that we can answer who authorized guardian access.

## 5. Functional Requirements

- **FR-1.** The system MUST seed a permission `org:parent-links:assign:manage` (“Assign parent/guardian links for students in the user’s organization”) and grant it to **Global Admin** by default; other roles receive it only when explicitly granted in Roles & Permissions.
- **FR-2.** All assign / invite / resend / list-for-student endpoints MUST require `org:parent-links:assign:manage` **or** `global:app:rbac:manage`. Org Admin alone MUST NOT be sufficient unless that role is also granted the new permission (breaking change vs today’s `orgRoleAccess` manage path — see Rollout; migrate Org Admin by optionally seeding the permission onto the Org Admin **app** role or documenting grant steps).
- **FR-3.** When `ffParentPortal` is enabled and the user has `org:parent-links:assign:manage`, the home sidenav MUST show a menu item (e.g. “Assign parents”) linking to the assign page. Without the permission, the item MUST NOT appear and deep links MUST 403 / redirect.
- **FR-4.** The assign page MUST let staff search students in their org (name, email, SID) and select a student, then open a modal titled **Assign a parent or guardian**.
- **FR-5.** The modal MUST accept **1–3** guardian rows; each row MUST require a display name and a valid email; relationship MAY default to `parent` with optional `guardian` / `other`.
- **FR-6.** On save, for each email that matches an existing user in the same org, the system MUST create/reactivate an **active** `parent_student_links` row, set `account_type = parent`, assign the Parent RBAC role (or ensure `app:user:account-parent-dashboard`), and return `status: linked`.
- **FR-7.** On save, for each email with **no** existing user, the system MUST: provision a placeholder parent user in the org (same pattern as People invite), create a **pending** link, send an **activate** email (password-set / invite token), and return `status: invited`.
- **FR-8.** Consuming a valid activate token MUST set the user’s password (or complete signup), mark the account parent, flip the pending link(s) for that email to **active**, and land the user on the Family dashboard (or login → dashboard).
- **FR-9.** The system MUST reject linking a parent to themselves, cross-org students, deactivated students, and duplicate active links (idempotent upsert OK).
- **FR-10.** Staff MUST be able to see existing links for the selected student (active + pending) and revoke or resend invite from the same page.
- **FR-11.** Every create / invite / activate / revoke MUST write an admin audit event with actor, student id, parent email/id, and outcome.
- **FR-12.** Existing CSV bulk and CLI paths SHOULD be updated to use the same service layer (invite when missing) in a follow-up commit within this feature, or documented as still requiring existing accounts until a PP.1b — prefer unifying in PP.1 if effort allows.

## 6. Non-Functional Requirements

- **Performance** — Student search p95 &lt; 300 ms for typical org sizes; assign of 3 guardians p95 &lt; 1 s excluding email send (email async OK).
- **Security** — Permission-checked server-side on every route; activate tokens single-use, short-lived (align with password-reset TTL); tokens MUST bind to org + student + parent email so they cannot be retargeted; rate-limit invite/resend per actor and per email.
- **Privacy & Compliance** — FERPA: only school officials with the permission may create links; parents see only linked children (existing `requireParentLink`). Activation emails MUST NOT include grades or other education records — name of student + school + CTA only. COPPA: parent account is adult; linking does not create under-13 accounts.
- **Accessibility** — Modal focus trap, labelled fields, error announcements, keyboard submit; WCAG 2.1 AA.
- **Scalability** — Reuse indexed `parent_student_links` and people search indexes; no N+1 on list-for-student.
- **Reliability** — If email send fails after user+pending link creation, return a clear error and allow **Resend**; DB transaction should commit user+link before send, or roll back if product prefers all-or-nothing (prefer commit + resend).
- **Observability** — Metrics: `parent_link_assign_total{outcome=linked|invited|error}`, `parent_link_activate_total{outcome=success|expired|invalid}`; structured logs with `org_id`, `student_id`, `actor_id` (no full tokens).
- **Maintainability** — Shared service in `server/internal/service/parentassign/` used by HTTP + (optionally) bulk; UI under `clients/web/src/pages/lms/parent-assign/` (or `pages/admin/`).
- **Internationalization** — Staff UI + email template strings externalised (`en` minimum; reuse parent i18n patterns where sensible).
- **Backward compatibility** — Existing active links and parent portal APIs unchanged; new permission is additive. Tighten authz on existing `/parent-links` routes to the new permission (with Global Admin bypass) and seed Org Admin appropriately so current admins are not locked out.

## 7. Acceptance Criteria

- **AC-1.** *Given* a user without `org:parent-links:assign:manage`, *When* they open `/assign-parents` (or chosen route) or call the assign API, *Then* they receive 403 and the sidenav item is hidden.
- **AC-2.** *Given* a user granted `org:parent-links:assign:manage` and `ffParentPortal` on, *When* they load the home shell, *Then* the sidenav shows **Assign parents** and the page loads.
- **AC-3.** *Given* staff selects student S and enters one guardian email that already exists in the org, *When* they save, *Then* an active link exists, the guardian’s account type is parent, and no activate email is required for pairing.
- **AC-4.** *Given* staff enters a guardian email with no account, *When* they save, *Then* a pending link exists and an email is sent containing a one-time activate URL.
- **AC-5.** *Given* the invitee clicks a valid activate link and sets a password, *When* activation completes, *Then* the link status is `active` and the Family dashboard shows student S.
- **AC-6.** *Given* an expired or reused activate token, *When* it is consumed, *Then* the system returns a safe error, does not activate the link, and offers a path for staff to resend.
- **AC-7.** *Given* the modal has three filled rows (mix of existing + new), *When* saved, *Then* each row is processed independently and the response reports per-email `linked` / `invited` / `error`.
- **AC-8.** *Given* axe (or equivalent) on the assign page + modal, *When* run, *Then* no critical/serious violations on the new UI.

## 8. Data Model

Reuse `"user".parent_student_links` (`status IN ('active','pending','revoked')`) from migration `137`. Extend invite materialization without breaking the NOT NULL `parent_user_id` FK:

**Approach (recommended):** On invite, create a placeholder `"user".users` row (org-scoped, email, display name, placeholder password hash — same as People invite), set `account_type = 'parent'`, insert link with `status = 'pending'`, `linked_by = actor`. Activation is password-reset / set-password consume (existing tokens) **plus** a post-activate hook that flips pending links for that `parent_user_id` to `active`.

Optional hardening migration (next number after `430`, e.g. `431_parent_link_assign.sql`):

```sql
-- Permission seed
INSERT INTO "user".permissions (permission_string, description)
VALUES (
  'org:parent-links:assign:manage',
  'Search students and assign parent/guardian links (invite if account missing)'
)
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
CROSS JOIN "user".permissions p
WHERE r.name = 'Global Admin'
  AND p.permission_string = 'org:parent-links:assign:manage'
ON CONFLICT DO NOTHING;

-- Optional: grant to a built-in Org Admin / Registrar role if one exists in app_roles;
-- otherwise document manual grant via Roles & Permissions.

-- Invite metadata (optional if password-reset token alone is enough)
CREATE TABLE IF NOT EXISTS "user".parent_link_invites (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    student_user_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    parent_user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    link_id           UUID NOT NULL REFERENCES "user".parent_student_links (id) ON DELETE CASCADE,
    email             CITEXT NOT NULL,
    invited_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    token_hash        TEXT NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    consumed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (link_id)
);
CREATE INDEX idx_parent_link_invites_token ON "user".parent_link_invites (token_hash)
  WHERE consumed_at IS NULL;
```

Prefer **dedicated invite tokens** over reusing generic password-reset alone if we need student context in the URL and safer resend semantics; otherwise reuse `RequestPasswordReset` and document the activate copy in a new email slot `parent_guardian_invite`.

**Backfill:** none required for existing active links.

## 9. API Surface

All under org scope; auth: bearer + `org:parent-links:assign:manage` (or Global Admin).

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/orgs/{orgId}/parent-assign/students?q=` | Search students (account_type ≠ parent; active) |
| `GET` | `/api/v1/orgs/{orgId}/parent-assign/students/{studentId}/links` | List active/pending links for student |
| `POST` | `/api/v1/orgs/{orgId}/parent-assign/students/{studentId}/guardians` | Assign 1–3 guardians |
| `POST` | `/api/v1/orgs/{orgId}/parent-assign/links/{linkId}/resend` | Resend activate email (pending only) |
| `DELETE` | `/api/v1/orgs/{orgId}/parent-links/{lid}` | Keep revoke; switch authz to new permission |
| `POST` | `/api/v1/auth/parent-invite/consume` | Public: token → set password + activate pending links |

**POST guardians body (pseudo-TypeScript):**

```ts
{
  guardians: Array<{
    name: string
    email: string
    relationship?: 'parent' | 'guardian' | 'other'
  }> // length 1..3
}
```

**Response:**

```ts
{
  results: Array<{
    email: string
    status: 'linked' | 'invited' | 'error'
    linkId?: string
    parentUserId?: string
    message?: string
  }>
}
```

- Rate-limit: assign 30/min/actor; resend 5/hour/email.
- OpenAPI: document under a new `parent-assign` tag.
- Tighten existing `POST/GET/DELETE .../parent-links*` to the same permission (deprecate Org Admin-only gate).

## 10. UI / UX

**New page:** e.g. `/assign-parents` (home shell).

**Sidenav:** `side-nav-main-links.tsx` — under Administration (or a small “School” section): show when `allows(PERM_PARENT_LINKS_MANAGE) && ffParentPortal`. Icon: `UsersRound` or `UserPlus`.

**Flow:**

1. Staff opens **Assign parents**.
2. Searches student; results list name, email, SID.
3. Selects student → summary of current guardians + **Assign a parent or guardian**.
4. Modal: 1–3 slots (add/remove row; min 1 on submit); Name, Email, Relationship.
5. Save → toast summarizing linked vs invited; pending rows show **Resend**.
6. Invitee email → activate page (extend `/reset-password` or dedicated `/activate-parent`) → Family dashboard.

**States:** empty search, no results, loading, permission denied, partial success (some rows error), email failure with resend CTA.

**A11y:** dialog `aria-labelledby`, focus return to trigger, inline field errors.

**Copy / i18n keys (examples):** `parentAssign.title`, `parentAssign.assignCta`, `parentAssign.modalTitle`, `parentAssign.invitedToast`, `email.parent_guardian_invite.*`.

## 11. AI / ML Considerations

(Skip — not AI-touching.)

## 12. Integration Points

- **Internal modules:**
  - `server/internal/repos/parentlinks/` — upsert pending/active, list by student
  - `server/internal/httpserver/parent_http.go` / new `parent_assign_http.go`
  - `server/internal/service/parentassign/` — assign orchestration
  - `server/internal/httpserver/platform_people.go` — reuse placeholder user + invite email patterns
  - Auth password-reset / new consume handler
  - `clients/web/src/lib/rbac-api.ts` — `PERM_PARENT_LINKS_MANAGE`
  - `clients/web/src/components/layout/side-nav-main-links.tsx`
  - `clients/web/src/pages/...` assign page + modal
  - Email templates: new system slot `parent_guardian_invite`
  - Feature flag: `ffParentPortal` (existing); no new flag required unless soft-launch desired (`ffParentAssign` optional — default off only if we need dogfood; prefer ship behind existing portal flag)
- **External:** transactional email provider (existing).
- **Events:** audit log; optional notification event `parent_guardian_invite`.

## 13. Dependencies & Sequencing

- Must ship after: 13.1 / W02 (done), permission-first RBAC (done), People invite / password reset (done).
- Must ship before: none hard; improves SIS/manual parity for 13.7.
- Shared infra: email + job queue if send is async.

**Suggested implementation order:**

1. Migration (permission + optional invites table).
2. Service + HTTP assign / list / resend + activate consume.
3. Authz migration on legacy parent-links routes.
4. Web page + modal + sidenav + i18n + email template.
5. Tests (unit, API, e2e).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Locking out Org Admins who relied on `orgRoleAccess` | M | H | Seed permission onto roles that currently manage links; dual-accept during one release if needed |
| Email deliverability / spam | M | M | Resend; reuse proven invite templates; clear From/subject |
| Staff typos create orphan placeholder accounts | M | M | Confirm email field; allow revoke pending + deactivate unused placeholders; conflict if email later belongs to a student |
| Linking wrong student (FERPA) | L | H | Confirm student identity (SID) in UI; audit trail |
| Token leakage grants parent access | L | H | Hash tokens, short TTL, single-use, HTTPS-only links |
| Partial row failures confuse staff | M | L | Per-email result list in modal + toast |

## 15. Rollout Plan

- **Feature flag:** Gate UI + new routes with existing `ffParentPortal` (default unchanged). Optional child flag `ffParentAssign` only if dogfood needed — prefer not adding a new flag (see flags.md COLLAPSE guidance).
- **Migration:** schema/permission → deploy API → deploy web → grant permission to registrar roles in pilot orgs.
- **Dogfood:** one K-12 pilot school; measure invite→activate conversion.
- **GA:** AC-1–8 green; no elevate in `parent_auth_error_rate`; help-center article published.
- **Rollback:** hide sidenav + disable routes via flag/permission revoke; leave links table intact; pending invites expire naturally.

## 16. Test Plan

- **Unit** — email normalize; 1–3 validation; relationship enum; token hash/consume; upsert pending→active.
- **Integration** — assign existing user; assign missing user + activate; resend; revoke; authz matrix (no perm / with perm / Global Admin); cross-org rejected.
- **End-to-end** — Playwright: grant perm → sidenav → search → modal → invite path (mock mail / capture token in test) → activate → Family dashboard shows child.
- **Security** — cannot assign without perm; cannot consume another email’s token; rate limits.
- **Accessibility** — axe on page + modal; keyboard-only assign.
- **Performance** — search under load with 10k users (spot check).
- **Manual exploratory** — three guardians mixed outcomes; resend after expiry; student who is also a parent account edge case.

## 17. Documentation & Training

- Help center: “Assign parents and guardians”.
- Admin docs: how to grant `org:parent-links:assign:manage` on a custom role.
- API reference / OpenAPI for parent-assign routes.
- Runbook: resend invites, revoke wrong links, clean up unused placeholder users.

## 18. Open Questions (resolved for ship)

1. **Org Admin grant** — Dual-accept `org_admin` grants for one release alongside `org:parent-links:assign:manage` / Global Admin so existing admins are not locked out.
2. **Guardian cap** — Modal allows up to **3 per submit**; no hard cap on active links.
3. **Activate URL** — Dedicated `/activate-parent?token=` + `POST /api/v1/auth/parent-invite/consume`.
4. **Student email** — Staff assign UI **blocks** emails that belong to accounts with the Student app role; CSV bulk still links existing accounts then invites when missing.
5. **Org-unit scope** — Whole org (no unit filter in PP.1).
6. **CSV unify** — Done in PP.1: existing emails link immediately; missing emails invite.

## 19. References

- Shipped portal: `docs/completed/13-k12-specific/13.1-parent-portal.md`, `docs/completed/web/W02-parent-guardian-portal-completeness.md`
- Links repo: `server/internal/repos/parentlinks/parentlinks.go`
- Org link APIs: `server/internal/httpserver/parent_http.go`, `server/internal/httpserver/org_routes.go`
- Permission seed pattern: `server/migrations/416_transcript_analytics.sql`
- Parent dashboard perm: `app:user:account-parent-dashboard` (`server/migrations/143_permission_first_rbac.sql`)
- People invite: `server/internal/httpserver/platform_people.go`
- Sidenav gating: `clients/web/src/components/layout/side-nav-main-links.tsx`, `clients/web/src/lib/rbac-api.ts`
- Schema: `server/migrations/137_parent_student_links.sql`, `216_parent_portal.sql`
