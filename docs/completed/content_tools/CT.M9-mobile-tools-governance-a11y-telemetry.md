# CT.M9 — Mobile Content Tools: Governance, Safety, Accessibility & Telemetry

> Implementation plan. Source: mobile half of [CT.8](CT.8-governance-safety-privacy-accessibility.md) and the client half of [CT.7](CT.7-analytics-insights-and-gradebook.md). Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M9 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Mobile squad + trust & safety (consult) |
| **Depends on** | CT.M3 |
| **Unblocks** | The AI tools in CT.M6; third-party tools in CT.M4 |

---

## 1. Problem Statement

CT.8 built the governance layer — per-course tool allowlists, AI disclosure modes, consent, free-text
filtering, crisis escalation, reporting, moderation, data sheets, accessibility conformance records and
an ops kill switch — and the web host honours all of it. A mobile host that renders tools without that
layer is not a smaller feature; it is a compliance and safety hole: a K12 student's free text sent to a
model with no disclosure, no consent check and no way to report what came back. CT.M9 makes the mobile
host honour every governance control the server already enforces, surfaces the safety affordances where
a student can reach them, and closes the accessibility and telemetry loop so we can actually see how
mobile tools behave.

## 2. Goals

- Honour every server-side governance control on mobile: course tool allowlist, per-tool denial,
  capability denial, AI disclosure mode, AI consent, free-text filter action, breaker, tombstone and
  the ops kill switch.
- Ship AI disclosure and consent as native frame chrome that a hostile or buggy tool cannot suppress.
- Put report — and, for staff, moderate — within reach on every tool that can carry user content.
- Discharge the accessibility obligation with an auditable conformance pass across the mobile tool
  surface, not a per-story claim.
- Close the telemetry loop: mobile-specific counters that tell us which tools are used, which fall back
  to placeholders, and where the runtime is failing.

## 3. Non-Goals

- Building governance *policy* — CT.8 owns it server-side; mobile enforces and displays.
- Instructor governance administration on mobile (setting allowlists, disclosure modes, retention) —
  web-only.
- Instructor analytics dashboards and gradebook passback UI (CT.7, web-only).
- The org-level marketplace consent and revocation flow (CT.9, web-only); mobile only honours the
  resulting `tombstone`.
- New server-side safety capability of any kind.

## 4. Personas & User Stories

- **As a student**, I want to know before I type when my words will be sent to an AI model.
- **As a student**, I want to report something harmful in an inline discussion or an AI response, from
  my phone, in a couple of taps.
- **As a parent of a K12 student**, I want AI features to stay off until consent has actually been
  given.
- **As a TA**, I want to moderate a reported post from my phone.
- **As an admin**, I want the kill switch to stop a misbehaving tool on phones without waiting for an
  app-store release.
- **As an accessibility lead**, I want a documented conformance record for the mobile tool surface.
- **As an operator**, I want to see mobile tool failures separately from web ones.

## 5. Functional Requirements

**Policy enforcement**

- **FR-1.** The host MUST fetch the course's content-tools settings and governance policy
  (`GET …/content-tools/settings`) and MUST NOT mount a tool whose `toolId` is outside `allowedToolIds`
  or inside `deniedToolIds`; it renders the CT.M3 placeholder with a "not available in this course"
  reason instead.
- **FR-2.** The host MUST honour `deniedCapabilities`: a tool declaring a denied capability MUST NOT
  mount, and MUST NOT be able to obtain the corresponding OS permission through the CT.M4 sandbox.
- **FR-3.** The host MUST honour `breakerOpen`, `deprecated`/`sunsetAt`, `tombstone` and the ops kill
  switch (`CONTENT_TOOLS_KILL_SWITCH`, and per-tool kills from `POST /api/v1/admin/content-tools/kill`)
  as read-only or no-mount states, each with its own reason string — with **no app release required**.
- **FR-4.** Policy MUST be re-evaluated on foreground and on course change, not only at first load, so
  a kill takes effect within one screen refresh.
- **FR-5.** `studentResetAllowed` MUST gate the self-reset control (CT.M3 FR-18).

**AI disclosure & consent**

- **FR-6.** For any instance whose `capabilities` include `ai`, the frame MUST render the AI disclosure
  required by the course's `aiDisclosureMode` — **natively, above the tool content**, so a sandboxed
  tool cannot cover or fake it.
- **FR-7.** Where consent is required, the host MUST read consent state
  (`GET …/content-tools/ai-consent`) and MUST block all AI actions until consent is granted; the
  consent action (`POST …/content-tools/ai-consent`) MUST be reachable from the blocked state.
- **FR-8.** The client MUST NOT assume a default; an unknown or unfetched consent state blocks.
- **FR-9.** Disclosure copy MUST be localised and MUST name what is sent (the student's text and the
  activity context), consistent with the web wording.

**Safety: report, filter, moderate, escalate**

- **FR-10.** Every tool instance that can display user- or model-generated content MUST expose a
  **Report** action from the frame overflow, reachable in at most two taps, posting to
  `POST …/instances/{instance_id}/report` with a category and optional note.
- **FR-11.** Staff MUST see moderation controls where entitled, backed by
  `POST …/instances/{instance_id}/moderate` and `GET …/instances/{instance_id}/moderation`; controls
  MUST be hidden for non-entitled viewers and a server `403` handled gracefully.
- **FR-12.** The host MUST honour `freeTextFilterAction`: when the server blocks or flags a submission,
  the client MUST explain the outcome in plain language without echoing blocked content back as an
  error string, and MUST preserve the student's draft.
- **FR-13.** Where `crisisEscalationEnabled` and the server returns a crisis-escalation response, the
  client MUST render the configured support resources prominently and MUST NOT treat it as a generic
  error or retry it.
- **FR-14.** The client MUST render `filter-flags` state (`GET …/instances/{instance_id}/filter-flags`)
  where the frame surfaces it, and MUST never render content the server marked as removed.

**Accessibility conformance**

- **FR-15.** The host MUST surface the conformance signal from `GET /api/v1/content-tools/conformance`
  for tools known to be non-conformant, particularly for sandboxed third-party tools (CT.M4).
- **FR-16.** A documented accessibility audit MUST be completed across the mobile tool surface — the
  frame plus every shipped renderer — using `docs/accessibility/mobile-audit-checklist.md`, with
  findings tracked to closure before GA.

**Telemetry**

- **FR-17.** The apps MUST record mobile-specific counters, labelled `tool_id` and platform: tool
  mounts, unsupported-placeholder shows, policy-blocked shows, render errors, state save outcomes,
  revision conflicts, offline replays, action outcomes by error class, AI-blocked-by-consent events,
  and report submissions.
- **FR-18.** Telemetry MUST contain **no** learner content: no state documents, no free text, no
  prompts, no peer content — ids, counters and outcome enums only.
- **FR-19.** Students MUST be able to see their own tool progress
  (`GET …/content-tools/my-progress`) on mobile where the app already shows progress surfaces.

## 6. Non-Functional Requirements

- **Performance** — Settings/policy fetched once per course per session and cached; the governance
  check adds ≤ 5 ms per mount and never blocks first paint of non-tool content.
- **Security** — All policy is server-enforced; the client's job is to avoid *presenting* what policy
  forbids and to never obtain a capability policy denies. A client that skips a check must still be
  refused by the API — verified in the test plan by calling actions directly.
- **Privacy & Compliance** — Discharges the mobile share of S01 (DSAR — mobile stores no state the
  server does not), S02 (retention — on-device caches purge on sign-out), S06 (DPIA), S08 (children's
  privacy — consent gating), S13 (EU AI Act — disclosure), S20 (accessibility law — FR-15/FR-16).
  Telemetry is content-free (FR-18).
- **Accessibility** — WCAG 2.1 AA across the tool surface; disclosure and consent are readable and
  operable by assistive tech; report and moderate are labelled controls, never icon-only; crisis
  resources are reachable and announced.
- **Scalability** — Settings cached per course; no per-mount policy request.
- **Reliability** — A failed policy fetch MUST fail **closed** for AI and third-party tools (block) and
  MUST fail **open only** for first-party non-AI tools that were previously allowed, using the cached
  policy; a stale cache older than the configured window blocks.
- **Observability** — The counters in FR-17 are the observability deliverable; alerting mirrors the
  web thresholds (render-error rate > 1% per `tool_id` over 15 min).
- **Internationalization** — `mobile.contentTools.governance.*` (`aiDisclosure`, `consentRequired`,
  `consentGrant`, `notAvailableInCourse`, `killed`, `report*`, `moderate*`, `filtered`, `crisis*`,
  `nonConformant`) in all five locale files; RTL verified.
- **Backward compatibility** — Unknown policy fields and unknown governance states MUST fail closed
  (block, with a generic reason) rather than being ignored — the safe default for a control plane the
  client does not fully understand.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course whose allowlist excludes a tool, *When* the page renders, *Then* the tool
  does not mount and a "not available in this course" placeholder appears.
- **AC-2.** *Given* an admin engages the kill switch for a tool, *When* the student foregrounds the
  app, *Then* the tool becomes read-only/no-mount within one refresh, with no app update.
- **AC-3.** *Given* a denied capability, *Then* the tool does not mount and no OS permission prompt can
  be triggered by it.
- **AC-4.** *Given* an AI-capable tool, *Then* the disclosure renders natively above the tool content
  and cannot be covered by the tool.
- **AC-5.** *Given* consent is required and not granted, *Then* every AI action is blocked, the consent
  action is reachable, and no model call is made.
- **AC-6.** *Given* the consent state cannot be fetched, *Then* AI actions stay blocked (fail closed).
- **AC-7.** *Given* any content-bearing tool, *Then* Report is reachable in at most two taps and posts
  successfully with a category.
- **AC-8.** *Given* a non-staff viewer, *Then* moderation controls are absent, and a direct moderate
  call returns 403 handled gracefully.
- **AC-9.** *Given* the server filters a free-text submission, *Then* a plain-language explanation
  appears, the blocked content is not echoed as an error, and the draft is preserved.
- **AC-10.** *Given* a crisis-escalation response, *Then* the configured support resources render
  prominently and no automatic retry occurs.
- **AC-11.** *Given* a tool the conformance record marks non-conformant, *Then* the frame surfaces that
  signal.
- **AC-12.** *Given* the accessibility audit, *Then* the frame and every shipped renderer have a
  completed checklist entry and no open blocker findings.
- **AC-13.** *Given* any telemetry event, *Then* it contains no learner content — asserted by an
  automated test over the emitted payload shape.
- **AC-14.** *Given* an unknown governance state from the server, *Then* the tool fails closed with a
  generic reason rather than mounting.
- **AC-15.** *Given* a stale cached policy beyond the window, *Then* AI and third-party tools block.
- **AC-16.** *Given* CI, *Then* iOS build, Android compile and the governance logic suites pass.

## 8. Data Model

**No server schema change, no migration.** Client models mirror
`server/internal/models/contenttools/governance.go` and `types.go`:

```kotlin
@Serializable data class ToolGovernancePolicy(
  val deniedCapabilities: List<String> = emptyList(),
  val deniedToolIds: List<String> = emptyList(),
  val allowedToolIds: List<String> = emptyList(),
  val aiDisclosureMode: String = "",
  val freeTextFilterAction: String = "",
  val crisisEscalationEnabled: Boolean? = null,
  val aiLogRetentionDays: Int = 0,
  val updatedAt: String? = null,
)

@Serializable data class ToolSettings(
  val allowedToolIds: List<String> = emptyList(),
  val studentResetAllowed: Boolean = false,
  val maxInstancesPerItem: Int = 0,
  val gradeLinksAllowed: Boolean = false,
  /* … mirrors models/contenttools/types.go */
)

@Serializable data class AIConsentState(val decision: String, val toolId: String? = null,
                                        val decidedAt: String? = null)
```

Policy and settings are cached per course in the CT.M3 encrypted cache with a bounded freshness window
(NFR reliability) and purged on sign-out.

## 9. API Surface

**No new endpoints.** CT.M9 consumes the shipped governance surface:

| Verb | Path | Purpose |
|---|---|---|
| GET | `/api/v1/courses/{course_code}/content-tools/settings` | Allowlist, reset policy, limits |
| GET | `/api/v1/courses/{course_code}/content-tools/ai-consent` | Consent state |
| POST | `/api/v1/courses/{course_code}/content-tools/ai-consent` | Record consent |
| POST | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/report` | Student report |
| POST | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/moderate` | Staff moderation |
| GET | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/moderation` | Moderation log |
| GET | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/filter-flags` | Filter state |
| GET | `/api/v1/content-tools/conformance` | Accessibility conformance records |
| GET | `/api/v1/content-tools/data-sheets` | Tool data sheets (linked from disclosure) |
| GET | `/api/v1/courses/{course_code}/content-tools/my-progress` | The student's own progress |

Kill state arrives through the instance payload (`breakerOpen`, `tombstone`) and the admin kill routes
are server-side only.

## 10. UI / UX

- **New (iOS)** — `Features/ContentTools/Governance/{AIDisclosureBanner,ConsentGateView,
  ReportSheet,ModerationSheet,CrisisResourcesView,PolicyBlockedPlaceholder}.swift`,
  `Core/LMS/ContentToolGovernanceLogic.swift` (pure: mount decision from policy + instance flags,
  fail-closed rules, staleness window, consent gating — unit-tested).
- **New (Android)** — `features/contenttools/governance/*`, `core/lms/ContentToolGovernanceLogic.kt`.
- **Modified** — CT.M3's `ToolFrame` (disclosure slot above content, overflow gains Report/Moderate,
  conformance note) and `ToolRendererRegistry` (policy gate before resolution).
- **Key flows** — (1) Open an AI tool → disclosure banner → (if required) consent gate → grant →
  composer unlocked. (2) See harmful content → overflow → Report → category → submit → confirmation.
  (3) Staff sees a reported post → Moderate → action → log updated. (4) Admin kills a tool → student
  foregrounds → tool is read-only with a reason.
- **States** — *Policy blocked*: neutral placeholder naming the reason (not available / withdrawn /
  temporarily unavailable). *Consent required*: blocked composer + explanation + grant action.
  *Filtered*: plain-language explanation, draft preserved. *Crisis*: prominent support resources.
  *Non-conformant*: an informational note in the frame. *Stale policy*: AI/third-party blocked with a
  refresh action.
- **Accessibility annotations** — the disclosure banner is part of the frame's accessible description
  so it is encountered before the tool content; consent and report dialogs are modal with trapped and
  restored focus; crisis resources are announced assertively; every control is named.
- **Copy & i18n** — `mobile.contentTools.governance.*` across all five locale files, wording reviewed
  by trust & safety and matched to the web strings.

## 11. AI / ML Considerations

CT.M9 makes no model calls; it governs them. It owns the mobile surface of: disclosure
(`aiDisclosureMode`, EU AI Act transparency), consent (COPPA/S08), free-text filtering and crisis
escalation outcomes, the data-sheet link explaining what each AI tool does with student text, and the
guarantee that AI actions are blocked when policy says so. Budget and rate-limit enforcement stay
server-side; mobile explains their outcomes honestly rather than retrying.

## 12. Integration Points

- **Internal** — CT.M3 host, frame and cache; CT.M4 sandbox (capability denial, conformance badge);
  CT.M6 AI tools (consent gate, filter outcomes, crisis path); the existing mobile accessibility and
  i18n layers; the app's analytics/error-logging client for FR-17.
- **Server (unchanged)** — `server/internal/httpserver/content_tools_governance.go`,
  `…_governance_admin.go`, `…_governance_policy.go`; `server/internal/service/contenttools/
  {policy,breaker,conformance,datasheet}.go`.
- **Docs** — `docs/accessibility/mobile-audit-checklist.md` is the instrument for FR-16.
- **Events** — none emitted client-side beyond content-free counters.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M3**.
- **Must ship before or with CT.M6** — the AI tools cannot GA without disclosure and consent; if CT.M9
  is not ready, CT.M6's AI renderers stay disabled behind their allowlist entries.
- Must ship before enabling **third-party** tools through CT.M4.
- Can ship in parallel with CT.M5, CT.M7, CT.M8.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| AI ships on mobile without disclosure/consent because governance slipped | M | H | Hard sequencing rule (§13): CT.M6 AI renderers stay flag-disabled until CT.M9 lands |
| Client-side policy checks are treated as the enforcement boundary | M | H | Server is authoritative; the test plan calls actions directly with the client bypassed and asserts refusal |
| Kill switch does not reach phones quickly | M | H | Re-evaluate policy on foreground (FR-4) with an explicit AC on refresh latency |
| A sandboxed tool covers or fakes the AI disclosure | M | H | Disclosure rendered natively above the WebView (FR-6), verified in the CT.M4 hostile-tool fixture |
| Fail-open on a policy fetch failure exposes AI to a non-consented student | M | H | Explicit fail-closed rules with a staleness window (NFR reliability, AC-6, AC-15) |
| Telemetry accidentally captures learner content | M | H | Content-free by construction plus an automated payload-shape test (AC-13) |
| Accessibility audit becomes a rubber stamp | M | M | Uses the existing checklist instrument, findings tracked to closure, blockers gate GA (AC-12) |

## 15. Rollout Plan

- **Feature flag** — no new flag: CT.M9 is enforcement inside the existing
  `mobileContentToolsEnabled` path. It is not optional and has no "off" state; the ops kill switch is
  the emergency control and is server-side.
- **Sequencing** — policy fetch + mount gating → kill/breaker/tombstone handling → AI disclosure →
  consent gate → report → moderation → filter/crisis outcomes → conformance surfacing → telemetry →
  accessibility audit.
- **Dogfood** — a course exercising each control: a denied tool, a killed tool, an AI tool requiring
  consent, a filtered submission, and a reported post moderated by a TA.
- **GA criteria** — all ACs green; trust & safety review of report/moderate/crisis copy and flows;
  privacy review of disclosure and consent; accessibility audit closed with no blockers; telemetry
  payload test passing.
- **Rollback** — governance cannot be rolled back independently; if a control misbehaves the remedy is
  to disable the affected tools via the server-side allowlist or kill switch, which is exactly what
  this story makes work.

## 16. Test Plan

- **Unit** — mount decision from policy × instance flags (allowlist, denial, capability, breaker,
  tombstone, kill, unknown state); fail-closed rules; staleness window; consent gating; two-tap report
  reachability as a structural assertion; telemetry payload shape.
- **Integration** — policy and consent fetch/refresh; report and moderate round-trips including 403;
  filter-blocked and crisis-escalation responses; kill-switch propagation on foreground.
- **End-to-end (device)** — grant consent then use an AI tool; report a post and see a TA moderate it;
  admin kills a tool mid-session and the student's next foreground reflects it.
- **Security** — bypass tests: call AI, report and moderate actions directly with the client's checks
  removed and assert server refusal; verify a denied capability cannot obtain an OS permission through
  the sandbox; verify no policy decision is trusted from the client.
- **Accessibility** — full audit per `docs/accessibility/mobile-audit-checklist.md` over the frame and
  every shipped renderer; screen-reader passes on disclosure, consent, report, moderation and crisis
  surfaces; 200% font scale; RTL.
- **Performance / load** — policy cache hit rate; mount overhead measurement against the ≤ 5 ms target.
- **Manual exploratory** — consent revoked mid-session; policy changed while offline; kill during an
  in-flight action; report submitted with no network; locale switch mid-flow.

## 17. Documentation & Training

- End-user: what AI disclosure means, how consent works, how to report, and where crisis resources come
  from.
- Instructor/TA: moderating from a phone; what students see when a tool is disallowed or killed.
- Admin: the kill switch reaches mobile clients on foreground — no release required.
- Compliance: append the mobile surface to the CT.8 DPIA record, the accessibility conformance report,
  and the S13 disclosure inventory.
- Internal runbook: mobile tool counters, alert thresholds, and the fail-closed behaviour operators
  should expect during a policy-service incident.

## 18. Open Questions

1. Is AI consent recorded per course, per tool, or per student globally? (Mirror the CT.8 decision
   exactly; the client must not invent a scope.)
2. What is the acceptable policy staleness window before AI and third-party tools block — minutes or
   hours? (Owner: trust & safety + backend.)
3. Does mobile need to display the tool **data sheet** inline, or is a link out sufficient?
   (Recommendation: link from the disclosure banner.)
4. Should report categories on mobile match the web set exactly, or be shortened for a phone?
   (Recommendation: match exactly — divergent taxonomies break triage.)
5. Which existing mobile analytics channel carries FR-17's counters, and does it already guarantee
   content-free payloads? (Verify before implementing; do not add a second channel.)

## 19. References

- Web plans: [CT.8](../../completed/content_tools/CT.8-governance-safety-privacy-accessibility.md),
  [CT.7](../../completed/content_tools/CT.7-analytics-insights-and-gradebook.md),
  [CT.9](../../completed/content_tools/CT.9-tool-marketplace-and-third-party-tools.md).
- Server: `server/internal/httpserver/content_tools_governance.go`, `…_governance_admin.go`,
  `…_governance_policy.go`; `server/internal/models/contenttools/governance.go`;
  `server/internal/service/contenttools/{policy,breaker,conformance,datasheet}.go`.
- Related plans: [CT.M3](CT.M3-mobile-content-tool-host-and-state.md),
  [CT.M4](CT.M4-mobile-sandboxed-webview-tool-host.md), [CT.M6](CT.M6-mobile-tools-text-and-ai.md).
- Instruments: `docs/accessibility/mobile-audit-checklist.md`; standards folder
  [`../../plan/standards/`](../../plan/standards/README.md) — S01, S02, S06, S08, S13, S20.
- External: WCAG 2.1 AA; EU AI Act transparency obligations; COPPA; FERPA.
