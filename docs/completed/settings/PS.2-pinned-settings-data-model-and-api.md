# PS.2 — Pinned Settings: Per-User Data Model & API

> Implementation plan. Source: authoring-UX gap — instructors need their frequently used assignment/quiz settings promoted to the top, persisted per user. Folder overview: [README](README.md). Active backlog: [docs/plan/settings](../../plan/settings/).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | PS.2 |
| **Section** | Pinned Editor Settings |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | PS.1 |
| **Unblocks** | PS.3, PS.4 |

---

## 1. Problem Statement

Pinned settings are a **per-user** preference: the department chair who lives in grade-posting policy
and the K-12 teacher who only ever touches due dates want different things at the top of the same
editor. Lextures has no storage for editor-layout preferences today — `settings.user_reading_preferences`
(migration 211) and `settings.user_notification_preferences` are the only per-user preference tables,
and neither generalises. Without server-side persistence, pins would live in `localStorage` and vanish
when the instructor moves from their office desktop to a classroom machine, which is exactly the
context where the "hidden setting" problem bites hardest.

## 2. Goals

- Persist an **ordered list of pinned setting IDs per user, per surface** (`assignment`, `quiz`).
- Ship a small, idempotent `/api/v1/me/pinned-settings` API that follows the existing `me` route and
  preference-repo conventions.
- Keep the server **decoupled from the client registry**: validate shape, count, and length — not
  membership — so adding a setting in the web client never requires a server deploy.
- Gate the whole capability behind a platform feature flag that defaults off.
- Guarantee a user with no pins and a user with corrupt/stale pins both get a working editor.

## 3. Non-Goals

- No pin UI — PS.3 owns every pixel.
- No org-level, role-level, or course-level default pins; PS.4 evaluates *suggested* pins, and even
  those are client-side defaults, not stored policy.
- No surfaces beyond `assignment` and `quiz` (the enum is extensible, but adding one is a migration).
- No sync protocol beyond plain HTTP request/response — no WebSocket push of pin changes.
- No admin UI to inspect or reset another user's pins.

## 4. Personas & User Stories

- **As an instructor**, I want my pinned settings to follow me between my laptop and the classroom
  desktop, so that my editor layout is stable wherever I teach.
- **As an instructor who teaches both quizzes and assignments**, I want separate pin sets per editor,
  so that quiz-only settings do not clutter the assignment panel.
- **As a platform admin**, I want the feature behind a flag I control, so that I can enable it after
  our own QA rather than on Lextures' schedule.
- **As a privacy officer**, I want pinned settings to be deleted with the user account and exportable
  in a DSAR, so that the new table does not create an untracked personal-data island.
- **As an on-call engineer**, I want a malformed pin list to be rejected at the API boundary, so that
  no client can write junk that breaks the editor for that user.

## 5. Functional Requirements

- **FR-1.** The system MUST create `settings.user_pinned_settings` storing, per `(user_id, surface)`,
  an **ordered** `TEXT[]` of setting keys.
- **FR-2.** The system MUST expose `GET /api/v1/me/pinned-settings`, returning the caller's pins for
  every surface, with an empty array for surfaces the caller has never pinned.
- **FR-3.** The system MUST expose `PUT /api/v1/me/pinned-settings/{surface}` performing a **full,
  idempotent replace** of that surface's ordered list, returning the stored result.
- **FR-4.** Both routes MUST require an authenticated session and MUST operate only on the caller's own
  `user_id`; there MUST be no route accepting a target user ID.
- **FR-5.** The server MUST validate each key: matches `^[a-z0-9]+(?:[.-][a-z0-9]+)*$`, length ≤ 96,
  and MUST reject the request with `400` + `apierr.CodeValidation` otherwise.
- **FR-6.** The server MUST reject a list longer than **12** keys with `400`, and MUST reject duplicate
  keys after normalisation (trim + lowercase) with `400`.
- **FR-7.** The server MUST NOT validate keys against a registry of known settings; unknown-but-valid
  keys MUST be stored (forward compatibility with a newer web client).
- **FR-8.** An empty array MUST be accepted and MUST clear the surface's pins (the row is upserted with
  `'{}'`, not deleted, so `updated_at` records the clear).
- **FR-9.** `{surface}` MUST be validated against the allowed set (`assignment`, `quiz`); anything else
  MUST return `400`, not `404`, to distinguish a bad path segment from a disabled feature.
- **FR-10.** When the platform flag `ff_pinned_settings` is off, both routes MUST return `404` with
  `apierr.CodeNotFound` and the message "Pinned settings are not enabled." — matching the
  `requireReadingPreferences` pattern in `server/internal/httpserver/reading_preferences_http.go`.
- **FR-11.** The flag MUST be added to `settings.platform_app_settings` as `ff_pinned_settings BOOLEAN
  NOT NULL DEFAULT FALSE`, wired through `server/internal/repos/platformconfig/`, and surfaced as
  `ffPinnedSettings` on `GET /api/v1/platform/features`.
- **FR-12.** Rows MUST be deleted by `ON DELETE CASCADE` when the user is deleted, and the table MUST be
  included in the user data-export path alongside other `settings.*` per-user tables.
- **FR-13.** Both routes MUST be documented in `server/internal/openapi/openapi.go`.
- **FR-14.** The web client MUST gain typed helpers in `clients/web/src/lib/` —
  `fetchPinnedSettings()` and `savePinnedSettings(surface, settingKeys)` — returning parsed,
  schema-validated responses consistent with the existing `courses-api-schemas.ts` approach.
- **FR-15.** `PUT` MUST be safe to call at up to **1 request/second per user**; beyond a per-user
  budget of 60 writes/minute the server MUST return `429`.

## 6. Non-Functional Requirements

- **Performance** — `GET` p95 < 30 ms (single indexed read of ≤ 2 rows); `PUT` p95 < 50 ms.
  Response payload < 1 KB. The editor MUST render without waiting on this request (PS.3 renders
  unpinned-first, then reflows).
- **Security** — Session-authenticated; `user_id` taken from the session context only. Keys are stored
  verbatim but are shape-constrained (FR-5) and rendered as text, never as HTML or selectors, by the
  client. No IDOR surface because no user-id path parameter exists.
- **Privacy & Compliance** — Pinned setting keys are low-sensitivity UI preferences, but they are
  user-linked personal data: covered by cascade delete (FR-12), DSAR export, and the retention policy
  applied to `settings.*` per-user tables. No FERPA education-record implications; no COPPA concern
  (staff-facing editors).
- **Accessibility** — No UI in this plan; the API MUST NOT constrain PS.3's a11y (e.g. it returns a
  deterministic order so screen-reader position announcements are stable).
- **Scalability** — Upper bound one row per user per surface (≤ 2 rows/user); at 1 M users that is
  ≤ 2 M narrow rows. No partitioning needed.
- **Reliability** — `PUT` is idempotent full-replace, so a retried request cannot corrupt order.
  `GET` failure MUST degrade to "no pins" in the client rather than blocking the editor.
- **Observability** — Counter `pinned_settings_writes_total{surface}`, counter
  `pinned_settings_rejects_total{reason}` (`shape`, `too_many`, `duplicate`, `bad_surface`), and the
  standard HTTP metrics from `server/internal/telemetry/metrics.go`. Log fields on reject:
  `user_id`, `surface`, `reason`, `key_count`.
- **Maintainability** — New repo package `server/internal/repos/pinnedsettings/` mirroring the shape of
  `server/internal/repos/readingprefs/`; handlers in
  `server/internal/httpserver/pinned_settings_http.go`; routes registered in `registerMeRoutes`.
- **Internationalization** — Error messages go through the existing `apierr` message path; no
  locale-dependent storage. Keys are ASCII identifiers, never displayed raw to users.
- **Backward compatibility** — Additive table, additive routes, additive flag column defaulting false.
  A client that never calls these routes is unaffected. Unknown keys are preserved (FR-7), so an older
  web client cannot silently destroy pins created by a newer one — but see §18 Q2.

## 7. Acceptance Criteria

- **AC-1.** *Given* the flag is on and a user with no stored pins, *When* they `GET
  /api/v1/me/pinned-settings`, *Then* the response is `200` with
  `{"surfaces":{"assignment":[],"quiz":[]}}`.
- **AC-2.** *Given* the flag is on, *When* a user `PUT`s
  `{"settingKeys":["quiz.presentation.lockdown-mode","quiz.scheduling.due-date"]}` to
  `/api/v1/me/pinned-settings/quiz`, *Then* the response is `200` with the same order, and a
  subsequent `GET` returns that exact order.
- **AC-3.** *Given* an existing quiz pin list, *When* the user `PUT`s a reordered list, *Then* the
  stored order is replaced (not merged) and `updated_at` advances.
- **AC-4.** *Given* the flag is on, *When* a user `PUT`s 13 keys, *Then* the response is `400` with
  `apierr.CodeValidation` and the stored list is unchanged.
- **AC-5.** *Given* the flag is on, *When* a user `PUT`s `["Quiz.Bad Key!"]`, *Then* the response is
  `400` and nothing is written.
- **AC-6.** *Given* the flag is on, *When* a user `PUT`s the same key twice, *Then* the response is
  `400` with reason `duplicate`.
- **AC-7.** *Given* the flag is **off**, *When* a user calls either route, *Then* the response is `404`
  with "Pinned settings are not enabled."
- **AC-8.** *Given* an unauthenticated request, *When* either route is called, *Then* the response is
  `401` and no row is read or written.
- **AC-9.** *Given* a user with stored pins, *When* the user record is deleted, *Then* their
  `settings.user_pinned_settings` rows are gone (cascade).
- **AC-10.** *Given* a `PUT` to `/api/v1/me/pinned-settings/discussion`, *Then* the response is `400`
  (unknown surface), not `404`.
- **AC-11.** *Given* an empty array `PUT`, *Then* the response is `200` with `[]` and the row exists
  with an updated timestamp.
- **AC-12.** *Given* the flag is on, *When* the web client calls `fetchPinnedSettings()` and the server
  returns `500`, *Then* the helper resolves to empty pin lists rather than rejecting (client-side
  degradation contract for PS.3).

## 8. Data Model

Migration pair: `server/migrations/443_user_pinned_settings.sql` / `443_user_pinned_settings.down.sql`
(shipped as 443; adaptive-content migrations took 439–442).

```sql
-- Plan PS.2: per-user pinned settings for the assignment and quiz editor panels.

CREATE TABLE IF NOT EXISTS settings.user_pinned_settings (
    user_id      UUID   NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    surface      TEXT   NOT NULL,
    setting_keys TEXT[] NOT NULL DEFAULT '{}',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, surface),
    CONSTRAINT ups_surface_check  CHECK (surface IN ('assignment', 'quiz')),
    CONSTRAINT ups_max_pins_check CHECK (cardinality(setting_keys) <= 12),
    CONSTRAINT ups_key_len_check  CHECK (
        NOT EXISTS (
            SELECT 1 FROM unnest(setting_keys) AS k
            WHERE char_length(k) = 0 OR char_length(k) > 96
        )
    )
);

COMMENT ON TABLE settings.user_pinned_settings IS
    'Plan PS.2: ordered per-user pinned setting keys for the assignment/quiz editor settings panels.';
COMMENT ON COLUMN settings.user_pinned_settings.setting_keys IS
    'Ordered array; index 0 renders first. Keys come from the web settings registry (PS.1) and are shape-validated, not membership-validated, server-side.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_pinned_settings BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.ff_pinned_settings IS
    'Plan PS.2 (ff_pinned_settings): enables per-user pinned settings in the assignment/quiz editors. Default false; flip true after QA sign-off.';
```

- **Array vs. row-per-pin** — the array shape is chosen because every read and write is
  "the whole list for one surface", ordering is intrinsic, and full-replace writes are a single
  statement with no unique-position index to fight. It follows the precedent set by
  `server/migrations/438_course_grade_levels_array.sql`.
- **Indexes** — the primary key `(user_id, surface)` serves every query. No secondary index.
- **Backfill** — none; absence of a row means "no pins", handled by returning defaults.
- **Down migration** — drops the table and the `ff_pinned_settings` column.

## 9. API Surface

All routes are session-authenticated and registered in `registerMeRoutes`
(`server/internal/httpserver/me.go`), alongside `reading-preferences`.

### `GET /api/v1/me/pinned-settings`

Auth: any authenticated user. Response `200`:

```ts
type PinnedSettingsResponse = {
  surfaces: {
    assignment: string[]  // ordered setting keys, [] when unset
    quiz: string[]
  }
}
```

### `PUT /api/v1/me/pinned-settings/{surface}`

`surface` ∈ `assignment | quiz`. Request:

```ts
type PutPinnedSettingsRequest = {
  settingKeys: string[]   // ordered, ≤ 12, unique, /^[a-z0-9]+(?:[.-][a-z0-9]+)*$/, each ≤ 96 chars
}
```

Response `200`: `PinnedSettingsResponse` (all surfaces, so the client can reconcile in one round trip).

Errors:

| Status | Code | When |
|---|---|---|
| 400 | `validation` | bad surface, > 12 keys, duplicate key, malformed key, malformed JSON |
| 401 | `unauthorized` | no session |
| 404 | `not_found` | `ff_pinned_settings` is off |
| 429 | `rate_limited` | > 60 writes/minute for this user |

- **Rate limiting** — per-user token bucket in the handler, mirroring the `ttsRateByUser` pattern in
  `reading_preferences_http.go`; PS.3 additionally debounces writes at 500 ms.
- **WebSocket** — none.
- **OpenAPI** — both routes, the request/response schemas, and the `ffPinnedSettings` feature field
  MUST be added to `server/internal/openapi/openapi.go`.

## 10. UI / UX

No UI in PS.2. The client-side deliverable is the typed API layer only:

- `clients/web/src/lib/pinned-settings-api.ts` — `fetchPinnedSettings()`,
  `savePinnedSettings(surface, keys)`; both parse through a schema in the style of
  `clients/web/src/lib/courses-api-schemas.ts`.
- `clients/web/src/context/platform-features-context.tsx` — add `ffPinnedSettings: boolean` (default
  `false`) to the context type, the initial value, and the response mapping.
- Error contract for PS.3: `fetchPinnedSettings()` resolves to empty lists on any non-2xx or parse
  failure (AC-12); `savePinnedSettings()` rejects so PS.3 can roll back an optimistic update.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- New: `server/migrations/443_user_pinned_settings.sql` (+ `.down.sql`),
  `server/internal/repos/pinnedsettings/pinnedsettings.go`,
  `server/internal/httpserver/pinned_settings_http.go`,
  `clients/web/src/lib/pinned-settings-api.ts`.
- Modified: `server/internal/httpserver/me.go` (route registration),
  `server/internal/openapi/openapi.go`, `server/internal/httpserver/settings_platform.go` (flag in the
  features response), `server/internal/repos/platformconfig/{platformconfig,features,patch}.go`
  (flag plumbing), `clients/web/src/context/platform-features-context.tsx`.
- Data-lifecycle touchpoints: the user-export and account-deletion paths must list the new table.
- No external services, no webhooks, no events emitted.

## 13. Dependencies & Sequencing

- Must ship after: **PS.1** (defines the key namespace this API stores).
- Must ship before: **PS.3** (consumes the API), **PS.4** (reads write telemetry).
- Shared infra needed: Postgres only. No queue, storage, or email.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Server/client registry drift lets stale keys accumulate | M | L | Server stores unknown keys (FR-7); client prunes via `resolveSettingId` on read and rewrites on next save |
| Two tabs write conflicting orders; last write wins silently | M | L | Full-replace idempotent semantics + PS.3 refetch-on-focus; conflict is a UI preference, not data loss |
| Drag-reorder generates a write per frame | M | M | 500 ms client debounce + per-user 60/min server budget (FR-15) returning `429` |
| Flag plumbing missed in one of the three platformconfig files | M | M | Existing `ff_reading_preferences` is the checklist; nodb test asserts the flag appears in `GET /api/v1/platform/features` |
| `CHECK` on array contents blocks a future longer key format | L | M | Limit is 96 chars, ~3× the longest current key; changing it is a one-line migration |
| New per-user table missed by DSAR export | L | M | FR-12 plus an export-coverage test enumerating `settings.*` user-scoped tables |

## 15. Rollout Plan

- **Feature flag** — `ff_pinned_settings`, default **false**.
- **Sequencing** — (1) migration, (2) repo + handlers + OpenAPI behind the off flag, (3) client API
  layer + feature-context field, (4) PS.3 UI, (5) flip the flag per environment.
- **Dogfood** — Enable on the internal instance for the education team; collect one week of feedback.
- **Pilot** — One K-12 and one HE pilot org enable the flag for instructors only.
- **GA criteria** — Zero `pinned_settings_rejects_total{reason="shape"}` from the shipped client,
  write p95 < 50 ms, no increase in editor error rate, PS.3 a11y sign-off.
- **Rollback path** — Flip the flag off: routes return `404`, the client falls back to the unpinned
  panel, rows are retained so re-enabling restores every user's pins. Full revert also drops the table
  via the down migration.

## 16. Test Plan

- **Unit** — Key validation (valid, uppercase, spaces, punctuation, empty, 97 chars); duplicate
  detection after normalisation; cap of 12; surface validation; repo `Get` returns defaults for a
  missing row; repo `Upsert` replaces rather than appends.
- **Integration (DB)** — Round-trip write/read preserves order exactly; empty-array clear; cascade
  delete on user removal; `CHECK` constraints reject an over-long key and a 13-element array when
  bypassing the handler.
- **HTTP (nodb + db)** — Follow the `reading_preferences_nodb_test.go` pattern: `404` when flagged off,
  `401` unauthenticated, `400` matrix per AC-4/5/6/10, `200` happy paths, `429` past the write budget,
  and that a `GET` never returns another user's row.
- **End-to-end** — Deferred to PS.3, which exercises the API through the UI.
- **Security** — Authz matrix (anonymous / student / instructor / admin — all only ever touch their own
  row); attempt to reach another user's pins by any parameter shape; oversized body rejection;
  JSON with `settingKeys: null` and with a nested object.
- **Accessibility** — Not applicable (no UI).
- **Performance / load** — `k6` or equivalent: 200 rps mixed read/write for 5 minutes, p95 within
  targets, no connection-pool saturation.
- **Manual exploratory** — Flip the flag off with pins stored; confirm the editor still loads and the
  pins reappear when the flag returns.

## 17. Documentation & Training

- API reference: OpenAPI entries for both routes plus the `ffPinnedSettings` feature field.
- Admin docs: one paragraph in the platform-settings feature-flag list describing
  `ff_pinned_settings` and its default.
- Internal runbook: `docs/runbooks/` entry covering "how to disable pinned settings" (flip flag,
  expected user-visible effect, data retained) — modelled on
  `docs/runbooks/adaptive-content-kill-switch.md`.
- End-user docs: deferred to PS.3.

## 18. Open Questions

1. Should the cap be 12 pins, or lower (6–8) to keep the pinned group from becoming a second panel?
   **Proposed:** 12 in the schema, with PS.3 enforcing a softer UI ceiling that can be tuned without a
   migration.
2. Should a `PUT` from an older client that drops keys it does not recognise be prevented (e.g. by
   sending an `unknownKeys` echo)? **Proposed:** no — PS.3 preserves unresolved keys in the list it
   sends back, so pruning is explicit rather than incidental. Confirm during PS.3 implementation.
3. Do we want `GET /api/v1/me/pinned-settings?surface=quiz` for a narrower read? Not needed while the
   payload is ≤ 2 arrays.
4. Should pins be included in the "copy my settings to a new device" onboarding flow, if one is ever
   built? Out of scope; noted for the preferences roadmap.
5. Does the DSAR export pipeline need a display-name mapping for setting keys, or is the raw key
   acceptable in an export? **Proposed:** raw key, documented in the export schema.

## 19. References

- Existing files: `server/migrations/211_user_reading_preferences.sql`,
  `server/internal/repos/readingprefs/readingprefs.go`,
  `server/internal/httpserver/reading_preferences_http.go`,
  `server/internal/httpserver/me.go`,
  `server/internal/repos/platformconfig/{platformconfig,features,patch}.go`,
  `server/internal/openapi/openapi.go`, `server/internal/apierr/apierr.go`,
  `server/internal/telemetry/metrics.go`,
  `clients/web/src/context/platform-features-context.tsx`,
  `clients/web/src/lib/courses-api-schemas.ts`.
- Related plans: [PS.1](PS.1-settings-registry-and-addressable-controls.md),
  [PS.3](PS.3-pin-and-reorder-ux-in-editor-panels.md),
  [PS.4](../../plan/settings/PS.4-suggested-pins-telemetry-and-rollout.md).
- External standards: RFC 2119; RFC 9110 §9.3.4 (PUT idempotency); FERPA (directory of user-linked
  records — pins are non-education-record preferences).
