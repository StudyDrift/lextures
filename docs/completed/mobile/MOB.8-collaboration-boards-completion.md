# MOB.8 — Collaboration Boards Completion (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: `clients/web/src/components/boards/*`, `lib/boards-api.ts`;
> web plans [`VC.8`–`VC.10`](../../completed/visual-collaboration/).
> Shipped mobile base: [`VC.M1`–`VC.M7`](../../completed/visual-collaboration/).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.8 |
| **Section** | Mobile parity |
| **Severity** | MINOR (parity) / MAJOR for governance |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | VC.M1–VC.M7 (shipped) |
| **Unblocks** | MOB.3 boards-governance item |

## Implementation notes (2026-07-19)

Mobile VC.8–VC.10 parity behind `ffMobileBoardsAdvanced` (DB `ff_mobile_boards_advanced`, default OFF). Link-token viewing already shipped in VC.M6 (`BoardPublicView` / `BoardPublicScreen`).

- **Flag**: `ffMobileBoardsAdvanced` gates templates, duplication, export, present, analytics, and Settings → Boards governance.
- **APIs**: `LMSAPIBoardTemplates` / `BoardTemplatesApi`, `LMSAPIBoardExport` / `BoardExportApi`, `LMSAPIBoardAnalytics` / `BoardAnalyticsApi`.
- **Logic**: `BoardsAdvancedLogic` (gating, template filter, export poll backoff, present ordering) + `BoardsGovernanceAdminLogic`.
- **UI**: template picker, save-as-template, export sheet (OS share), present mode, board analytics sheet; admin governance in Settings hub.
- **i18n**: `mobile.boards.templates|export|present|analytics|admin.*`; synced via `scripts/sync-mobile-locales.py`.
- **Observability**: `board_{template_used,saved_as_template,exported,presented}`, `board_admin_{analytics_viewed,lifecycle_action}`.
- **Tests**: iOS/Android unit coverage; e2e API smoke `e2e/tests/mobile-boards-advanced.spec.ts`.

## 1. Problem Statement

Visual **collaboration boards** (Padlet-style: posts, layouts, realtime
presence, reactions/comments/assessment, sharing, moderation) shipped on mobile
as VC.M1–VC.M7 and are in good shape. The web boards feature goes further
(VC.8–VC.10): **templates & duplication**, **embedding / export / present
mode**, and **admin analytics / quotas / lifecycle governance**. Those three
capabilities have **no** mobile equivalent — there are no board-template,
export, present, or board-analytics wrappers in the mobile LMS layer. As a
result, instructors can run a board on mobile but cannot start one from a
template, export/present it, or (as admins) govern boards from a phone.

## 2. Goals

- Add **templates & duplication** on mobile: browse board templates, create a
  board from a template, and save a board as a template (VC.8 parity).
- Add **export & present mode** on mobile: export a board (async job → download)
  and present it full-screen; open shared/embedded boards via link tokens
  (VC.9 parity).
- Add **admin governance** on mobile: board analytics, quotas, and lifecycle
  actions (VC.10 parity), feeding the settings menu ([MOB.3](MOB.3-system-settings-parity.md)).
- Preserve the shipped VC.M1–M7 behaviour and realtime.

## 3. Non-Goals

- Rebuilding posts/layouts/presence/reactions/moderation (shipped VC.M1–M7).
- HTML embed *authoring* on phones where it makes no sense (viewing an embedded
  board via a link token IS in scope; generating embed snippets is optional —
  see §18).
- Server changes (VC.8–VC.10 endpoints exist).
- Whiteboards (different feature — [MOB.6](MOB.6-whiteboards.md)).

## 4. Personas & User Stories

- **As an instructor**, I want to start a board from a "KWL chart" template on my
  phone instead of building it from scratch.
- **As an instructor**, I want to save a great board as a reusable template.
- **As an instructor**, I want to present a board full-screen from my phone/tablet
  during class.
- **As an instructor**, I want to export a board (PDF/image) to share or archive.
- **As an admin**, I want board analytics, quota status, and lifecycle controls
  on mobile.

## 5. Functional Requirements

- **FR-1.** The app MUST list board templates (`GET /api/v1/board-templates`) and
  create a board from one, and save an existing board as a template
  (`POST …/boards/{id}/save-as-template`) — VC.8.
- **FR-2.** The app MUST duplicate a board where web allows.
- **FR-3.** The app MUST start a board **export** job
  (`POST …/boards/{id}/export`), poll status
  (`GET …/export/{jobId}`), and download the result
  (`…/export/{jobId}/content`) — VC.9.
- **FR-4.** The app MUST provide a **present mode**: a full-screen,
  distraction-free rendering of the board suitable for a projector/large screen.
- **FR-5.** The app MUST open shared/embedded boards via link tokens
  (`GET /api/v1/board-links/{token}`, `…/board-links/{token}/posts`) with the
  correct access scope; `…/boards/{id}/link-preview` for previews.
- **FR-6.** The app MUST provide **admin governance** (VC.10): board analytics
  (`…/boards/{id}/analytics`), org-level board analytics, quota display, and
  lifecycle actions (archive/lock/delete) consistent with `boards-governance`.
- **FR-7.** All new actions MUST respect the `ff_visual_boards` per-course flag
  and board permissions/roles already enforced in VC.M\*.

## 6. Non-Functional Requirements

- **Performance** — template list p95 < 1 s; present mode renders large boards at
  60 fps with element culling; export polling backs off.
- **Security** — export/analytics/governance permission-gated server-side; link
  tokens are capability URLs (scope-limited, revocable); no elevation on client.
- **Privacy & Compliance** — exports may contain student posts (FERPA); respect
  board visibility/moderation state in exports and present mode; honour
  quota/retention lifecycle.
- **Accessibility** — WCAG 2.1 AA; present mode navigable and readable
  (font scaling, focus); export controls labelled; reduced-motion in present
  transitions (AN.\*).
- **Scalability** — export handled by server job queue; analytics paginated.
- **Reliability** — export job reconnect/resume by jobId; idempotent
  save-as-template; present mode tolerates realtime updates.
- **Observability** — `board_{template_used,saved_as_template,exported,presented}`
  and `board_admin_{analytics_viewed,lifecycle_action}`.
- **Maintainability** — new mobile wrappers: `LMSAPIBoardTemplates`,
  `LMSAPIBoardExport`, `LMSAPIBoardAnalytics` (iOS) and Android equivalents,
  alongside existing `LMSAPIBoard*` / `Board*Api.kt`.
- **Internationalization** — `mobile.boards.templates.*`, `.export.*`,
  `.present.*`, `.admin.*`.
- **Backward compatibility** — no API change; boards created via templates
  render on all clients.

## 7. Acceptance Criteria

- **AC-1.** *Given* templates exist, *when* the instructor creates a board from a
  template, *then* the board is created with the template's structure.
- **AC-2.** *Given* a board, *when* the instructor saves it as a template, *then*
  it appears in the template list.
- **AC-3.** *Given* a board, *when* the instructor exports it, *then* a job runs
  and a downloadable file is produced and opens/shares on device.
- **AC-4.** *Given* present mode, *when* activated, *then* the board renders
  full-screen and updates as posts change.
- **AC-5.** *Given* a board link token, *when* opened, *then* the board (or its
  posts) loads with the correct access scope.
- **AC-6.** *Given* an admin, *when* they open board governance, *then* analytics,
  quota, and lifecycle actions are available and match web; non-admins can't.

## 8. Data Model

- **No new tables.** Board templates, export jobs, analytics, quotas, and
  lifecycle state exist server-side (VC.8–VC.10). Client adds template/export/
  analytics view models only.

## 9. API Surface

Existing endpoints (reused):

- Templates: `GET /api/v1/board-templates`,
  `POST …/boards/{id}/save-as-template`, duplicate.
- Export: `POST …/boards/{id}/export`, `GET …/export/{jobId}`,
  `GET …/export/{jobId}/content`.
- Embed/links: `…/boards/{id}/embed`, `…/boards/{id}/link-preview`,
  `GET /api/v1/board-links/{token}`, `…/board-links/{token}/posts`.
- Sharing: `…/boards/{id}/shares` (already used in VC.M6).
- Governance: `…/boards/{id}/analytics`, org board analytics, quotas, lifecycle
  (as in `boards-governance`).

No new server routes.

## 10. UI / UX

- **New/extended screens:** template picker + "save as template"; export sheet
  (format + progress + share); present mode (full-screen); board governance
  (analytics/quotas/lifecycle) linked from settings.
- **Reused:** `Boards/*` list/detail/composer, `BoardShareSheet`, `Public/`
  link viewers, `BoardSocket` realtime.
- **Flows:** New board → from template; board menu → save as template / export /
  present; settings → boards governance.
- **States:** template loading/empty, export queued/running/ready/failed,
  present mode active, link-scope (view-only), admin empty/loading, offline.
- **Mobile/responsive:** present mode uses full screen + minimal chrome; export
  uses the OS share sheet; analytics as cards/charts.
- **Accessibility:** present mode focus + font scaling; labelled export/lifecycle
  controls; reduced-motion.
- **Copy & i18n:** `mobile.boards.templates|export|present|admin.*`.

## 11. AI / ML Considerations

Not AI-touching. (AI template suggestions are a possible future add; out of
scope.)

## 12. Integration Points

- iOS: extend `Features/Boards/*` (add template/export/present views + governance);
  new `Core/LMS/LMSAPIBoardTemplates.swift`, `LMSAPIBoardExport.swift`,
  `LMSAPIBoardAnalytics.swift`; reuse `BoardsLogic.swift`, `BoardSocket`,
  `BoardShareSheet`, `Boards/Public/*`.
- Android: extend `features/boards/*`; new template/export/analytics APIs
  alongside `core/lms/Board*Api.kt`, `BoardsLogic.kt`.
- Governance surfaces via [MOB.3](MOB.3-system-settings-parity.md).

## 13. Dependencies & Sequencing

- Must ship after: VC.M1–VC.M7 (done).
- **Phase 1:** templates & duplication (VC.8).
- **Phase 2:** export & present mode + link/embed viewing (VC.9).
- **Phase 3:** admin governance/analytics/quotas/lifecycle (VC.10) → feeds MOB.3.
- Shared infra: export job queue (exists).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Present mode performance on large boards | M | M | Element culling; native rendering; test on min-spec |
| Export fidelity vs web (fonts/layout) | M | M | Server-side export (reuse web pipeline); mobile just triggers/downloads |
| Exports leak moderated/hidden posts | L | H | Server honours visibility/moderation in export; verify in test |
| Governance authz gaps on mobile | M | M | Server-authoritative; parity authz tests |

## 15. Rollout Plan

- Flag: reuse `ff_visual_boards`; add `ff_mobile_boards_advanced` client gate.
- Sequence: Phase 1 → 2 → 3 behind the client gate; dogfood per phase.
- GA criteria: AC-1..6 pass; export fidelity accepted; governance authz green.
- Rollback: client gate off leaves shipped VC.M1–M7 intact.

## 16. Test Plan

- **Unit** — template create/save; export job polling; analytics/quota data
  parsing; link-token scope.
- **Integration** — template → board; export → download; governance actions
  against a seeded org.
- **End-to-end** — create-from-template, export+share, present, open-shared-link
  on device.
- **Security** — export/analytics/lifecycle authz; link-token scope; moderated
  content excluded from exports.
- **Accessibility** — present mode + export + governance screen-reader runs;
  reduced-motion.
- **Performance** — present-mode fps on large boards; export polling backoff.
- **Manual** — cross-client parity (template board opens on web); offline.

## 17. Documentation & Training

- "Board templates, export, and present mode on mobile" help article.
- Admin "Govern boards from mobile" note.
- Parity checklist update in the visual-collaboration README.

## 18. Open Questions

1. Do we generate **embed snippets** on mobile, or only *view* embedded/shared
   boards via link tokens (recommended v1)?
2. Which export formats matter most on mobile (PDF vs image vs data), given the
   OS share sheet?
3. Should present mode support external-display/AirPlay/Cast output explicitly?

## 19. References

- Web: `clients/web/src/components/boards/*`, `clients/web/src/lib/boards-api.ts`,
  `pages/admin/boards-governance.tsx`.
- Web plans: `docs/completed/visual-collaboration/VC.8–VC.10`.
- Mobile base: `docs/completed/visual-collaboration/VC.M1–VC.M7`;
  `clients/ios/Lextures/Features/Boards/*`, `Core/LMS/BoardsLogic.swift`;
  `clients/android/.../features/boards/*`, `core/lms/BoardsLogic.kt`.
- Related: [MOB.3](MOB.3-system-settings-parity.md),
  [MOB.6](MOB.6-whiteboards.md).
