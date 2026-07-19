# Platform Feature Flag Audit

**Date:** 2026-07-18
**Status:** DONE (implemented 2026-07-18)
**Scope:** The DB-managed platform boolean flags applied in
`server/internal/repos/platformconfig/features.go` (surfaced to Global Admins in
**Settings → Global platform**), plus a handful of env-only operational gates in
`server/internal/config/config.go`.
**Ask:** An honest, no-punches-pulled assessment of whether each flag needs to
exist, and whether it should be on by default.

## Implementation notes (2026-07-18)

Prioritized actions from this audit were applied:

1. **DELETE** — `OriginalityStubExternal`, `ClamAVStub`, and `OERStub` are env-only
   (`ORIGINALITY_STUB_EXTERNAL`, `CLAMAV_STUB`, `OER_STUB`); removed from Global platform
   UI. `FFVisualBoards` / `FFInteractiveQuizzes` remain hard-wired on (API keys kept for
   compatibility; UI toggles removed / exempt).
2. **COLLAPSE** — Grader Agent milestone children follow `GraderAgentEnabled` (Vision
   stays independent). IQ hosting/modes/gradebook are always on at platform (course flag
   is the gate). LpAdapt* fan out from any child. Motion uses `FFMotionNavigation` as
   master. Read-aloud, accommodations audit, alt-text, mobile create, and parent portal
   V2 collapsed as recommended.
3. **DEFAULT-ON** — Security/baseline flags flipped in `applyPlatformBools` (session UI,
   MFA availability, email notifications, admin console/search, gradebook CSV,
   resubmission, annotations, peer review, conditional release, etc.).
4. **DERIVE + packs** — Global platform UI groups toggles into capability packs and marks
   credential-gated flags with a Config-gated badge + “Requires: …” hint.
5. **Temporary gates** — `RTLEnabled` (i18n, target 2026-Q4) and `FFReadingPreferences`
   (a11y, target 2026-Q3) documented with owners/removal targets in `features.go`.

Admin-facing definitions: **117** (was ~134+). DB columns for collapsed/deleted flags are
kept for one release (Merge ignores or derives); forward DROP is deferred.

---

## The honest headline

There are **~180 DB-managed platform boolean flags**, two of which
(`FFVisualBoards`, `FFInteractiveQuizzes`) are already dead constants hard-wired
to `true`, plus a float tuning knob (`LearnerModelEMAAlpha`) that isn't a flag at
all. This is **too many by roughly half**, and the reason is structural, not
accidental:

> **Every plan (`N.M`) and nearly every development milestone (`GA-M1…M7`,
> `IQ.6`, `LP09`) shipped with its own permanent platform toggle, defaulted OFF,
> and nobody ever went back to retire the scaffolding.** The flag list is a
> changelog of how the product was built, not a set of switches an operator
> actually needs.

That produces three concrete problems:

1. **Rollout scaffolding that never got cleaned up.** The 8 `GraderAgent*`
   flags, the 8 `FFIq*` flags, the 4 `LpAdapt*` flags, and the mobile/parent
   `V2` pairs are milestone gates for features that have since shipped. They add
   toggles that only make sense to the engineer who wrote that milestone.
2. **Toggles that duplicate configuration.** ~20 flags gate integrations that
   physically cannot run without credentials the operator must also supply
   (SAML, OIDC, Clever, ClassLink, Stripe, PayPal, the three chat bots, SES…).
   The boolean is a *second* switch whose only failure mode is "I turned it on
   and nothing happened."
3. **Test doubles exposed as product settings.** `OriginalityStubExternal`,
   `ClamAVStub`, and `OERStub` are dev/e2e seams that appear in the admin
   platform settings surface. An operator can toggle the app into "pretend"
   mode.

Set against that, a large **legitimate** core remains: compliance modules,
vertical (K-12 / HE) packs, AI-cost/liability gates, and infra-dependent
features genuinely should exist and genuinely should default OFF.

### By the numbers

| Verdict | Count (approx) | What it means |
|---|---|---|
| 🔵 **KEEP-OFF** — legit, off is correct | ~95 | Compliance, vertical packs, AI-cost, integrations, infra-dependent |
| 🟢 **DEFAULT-ON** — flip the default (or delete the flag) | ~18 | Shipped, low-risk, broadly wanted; off is user-hostile |
| ⚪ **DERIVE** — auto-enable from credential/dependency presence | ~20 | The real gate is config, not the boolean |
| 🟡 **COLLAPSE** — merge into a parent flag | ~22 | Milestone/sub-flags of one feature |
| 🔴 **DELETE** — should not be a platform flag at all | ~5 | Test stubs + dead constants |
| ⚙️ **NOT-A-FLAG** — it's a setting/param, reclassify | ~4 | Default-value or tuning knobs mislabeled as features |

Realistically, the admin-facing flag surface could drop from ~180 to **~110–120
meaningful toggles**, and if the vertical/compliance flags were grouped into
**capability packs** (K-12, Higher-Ed, Marketplace, Compliance), the operator
would see **fewer than 40 top-level switches**.

---

## How each flag was judged

A flag earns its place only if a real operator would plausibly want it in one
state on Monday and the other on Tuesday **and** flipping it is the only way to
get there. Concretely, I asked five questions:

1. **Does it gate a dependency the operator must configure anyway?** If the
   feature no-ops without a secret/credential/binary, the credential *is* the
   gate → **DERIVE**.
2. **Is it a milestone/sub-flag of a feature that already has a master switch?**
   → **COLLAPSE**.
3. **Is it shipped, low-risk, has no external dependency, and wanted by almost
   everyone?** → **DEFAULT-ON**.
4. **Does turning it on impose legal obligations, spend real money, widen the
   attack surface, or only matter to one market?** → **KEEP-OFF** (correct).
5. **Is it a test seam or a dead constant?** → **DELETE**. **Is it a value/param,
   not a capability?** → **NOT-A-FLAG**.

---

## 🔴 DELETE — should not be a platform flag at all

These are the clearest wins. Nothing about them belongs in an operator-facing
settings screen.

| Flag | Default | Why it must go |
|---|---|---|
| `OriginalityStubExternal` | off | Dev/e2e stub for the originality provider. It puts the app into "fake results" mode. Should be an env/build-time seam, never a platform setting. |
| `ClamAVStub` | off | "In-process EICAR detection (tests/dev without clamd)." Same problem — a test double promoted to a product toggle. |
| `OERStub` | off | "Embedded catalog data instead of live OER APIs (dev/e2e)." Same. |
| `FFVisualBoards` | **hard-coded `true`** | Already documented as deprecated/always-on ("platform master switch removed"). It's a zombie field kept only for payload compatibility. Remove the toggle; keep the JSON key only as long as old clients POST it. |
| `FFInteractiveQuizzes` | **hard-coded `true`** | Same story — "always treated as on; kept for API compatibility." Zombie. |

**Verdict:** move the three stubs to environment/build config; formally retire
the two zombie constants. Net: **−5 admin toggles**, and the app can no longer be
put into pretend-mode from a settings page.

---

## ⚙️ NOT-A-FLAG — reclassify as settings/params

These aren't capability gates; they're values that happen to live in the same
struct. Keeping them in the "features" list makes the list look bigger and scarier
than it is.

| Field | What it actually is | Where it belongs |
|---|---|---|
| `LearnerModelEMAAlpha` (float, def 0.3) | A smoothing coefficient in `(0,1]`. | A numeric setting under the adaptive-learning section, not the feature toggle grid. |
| `BadgesDefaultPublic` | The default value of `is_public` on a new badge award. | A default under the badges feature, gated by `FFCompetencyBadges`. Not its own platform capability. |
| `MagicLinkEnrolledOnly` | A *modifier* of magic-link behavior (who gets a link). | A sub-option of `MagicLinkEnabled`, only meaningful when that's on. |
| `LRSAnonymizeActors` | A data-handling option (hash mbox emails). | A privacy option under `XAPIEmissionEnabled`, only meaningful when that's on. |

These four are fine to *keep* — they just shouldn't be counted or presented as
peer "features."

---

## ⚪ DERIVE — the real gate is configuration, not the boolean

Each of these gates a feature that **cannot function without operator-supplied
credentials or a binary/service**. The boolean is redundant with, and strictly
weaker than, "are the credentials present?" The classic failure mode is an admin
flipping the switch, seeing nothing happen, and filing a bug. Recommended pattern:
**auto-enable when the dependency is configured; keep an *optional* explicit
override only where an operator might want the credentials present but the feature
off.**

| Flag | Hard dependency (the true gate) |
|---|---|
| `SAMLSSOEnabled` | SP entity ID + X509 + private key |
| `OIDCSSOEnabled` | Google/Microsoft/Apple client IDs + secrets |
| `CleverSSOEnabled` | Clever client id/secret |
| `ClassLinkSSOEnabled` | ClassLink issuer + client id/secret |
| `OneRosterEnabled` | OneRoster bearer token |
| `ScimEnabled` | SCIM bearer token |
| `LTIEnabled` | LTI RSA private key + key id |
| `FFStripeBilling` / `FFPaymentsEnabled` | Stripe secret + webhook secret |
| `FFRevenueShare` | Stripe Connect config |
| `FFTaxCollection` | Stripe Tax config |
| `FFBotSlack` / `FFBotTeams` / `FFBotDiscord` | Each bot's client id/secret/public key |
| `FFEmailSES` | SES region + verified From (also `EmailProvider=ses`) |
| `PushNotificationsEnabled` | VAPID key pair + subject |
| `SmsNotificationsEnabled` (env) | Twilio credentials |
| `DRMEnabled` | DRM HMAC secret (falls back to insecure derived value) |
| `XAPIEmissionEnabled` | LRS endpoint config |
| `FFRedisCache` | `REDIS_URL` |
| `AvScanningEnabled` | reachable clamd (or the stub) |
| `VideoTranscodingEnabled` / `AutoCaptioningEnabled` | ffmpeg / Whisper backend |

**Verdict:** These don't all need deleting — an explicit "SSO is live" master
switch has some operational value (staged cutover). But **~20 booleans that are
100% redundant with credential presence** is a lot of surface for little gain.
At minimum, the UI should show them as **auto-derived / disabled-until-configured**
rather than free-standing toggles that silently no-op.

---

## 🟡 COLLAPSE — milestone and sub-flags of a single feature

This is where the count really bloated. Each cluster below is **one feature**
wearing many switches, because each development milestone got its own flag and
none were retired.

### Grader Agent — 8 flags for one feature

`GraderAgentEnabled` (parent, off) · `GraderAgentReviewInboxEnabled` (GA-M1) ·
`GraderAgentSuggestModeEnabled` (GA-M3) · `GraderAgentTextEntryGradingEnabled`
(GA-M2, **on**) · `GraderAgentVisionGradingEnabled` (GA-M2) ·
`GraderAgentRunFiltersEnabled` (GA-M5) · `GraderAgentCostEstimateEnabled` (GA-M7)
· `GraderAgentCancelRunEnabled` (GA-M6).

These are literally the milestone ladder (M1…M7). Text-entry grading already
defaults on; cancel-run, run-filters, and the review inbox are not features an
operator would sanely disable once the agent is on. **Keep two:**
`GraderAgentEnabled` (master, off — it spends AI money) and
`GraderAgentVisionGradingEnabled` (genuinely optional: higher cost, image
handling). **Fold the other six into the parent.** Net: **−6**.

### Interactive Quizzes — 8 sub-flags

`FFIqLiveHosting` (on) · `FFIqTeamMode` · `FFIqStudentPaced` · `FFIqHomework` ·
`FFIqGradebookPush` · `FFIqPublicKitCatalog` · `FFIqGuestJoin` ·
`FFIqAiGeneration`. The master switch `FFInteractiveQuizzes` is already a dead
`true` constant, and hosting is per-course. **Team mode / student-paced /
homework are just *quiz modes*** — they don't need platform kill-switches; expose
them as authoring options. **Keep as real platform gates:** `FFIqGuestJoin`
(unauthenticated join — real security/COPPA surface), `FFIqAiGeneration` (AI
spend), and arguably `FFIqPublicKitCatalog` (public listing/moderation). **Fold
the modes + gradebook push + live-hosting** into the per-course flag. Net: **−4
to −5**.

### Learner Profile adaptation — 4 flags

`LpAdaptRecommendationsEnabled` · `LpAdaptReviewEnabled` ·
`LpAdaptModalityEnabled` · `LpAdaptTutorEnabled` — all from LP09, all off, all
"use the profile to adapt X." This is one capability ("let the learner profile
drive personalization") split four ways. **Collapse to one** `LpAdaptationEnabled`
unless there's a concrete reason an operator wants modality-adaptation but not
review-adaptation. Net: **−3**.

### Motion / animation — 3 kill-switches

`FFMotionNavigation` · `FFMotionReveal` · `FFMotionLists` (all default **on**,
all "AN.x kill-switch"). One `prefers-reduced-motion`-style master kill-switch is
plenty; three separate ones for shipped, default-on animation polish is
over-granular. **Collapse to one** `FFMotion` kill-switch. Net: **−2**. (The
per-user reduced-motion preference already exists via `FFHighContrastReducedMotion`
— see below.)

### Redundant feature pairs

| Pair | Problem | Recommendation |
|---|---|---|
| `ReadAloudEnabled` + `FFReadAloud` | One "gates the feature," the other "exposes controls to learners" — for the same feature. | Collapse to one. |
| `AccommodationsEngineEnabled` + `FFAccommodationsEngine` | Second flag only toggles whether the engine writes **audit log** entries. Audit logging should follow the engine, not be independently switchable. | Collapse; always log when the engine runs. |
| `AltTextEnforcementEnabled` + `FFAltTextEnforcement` | Two levels (prompt/report vs. hard-block-on-save). This one is *defensible* as soft-vs-hard, but should be a single tri-state ("off / prompt / enforce"). | Merge into one enum. |
| `FFMobileCreateCourse` + `FFMobileCourseCreateV2` | Confirmed in `CourseCreateLogic.swift`: entry point is `V1 || V2`, advanced wizard is `V2`. V1 is nearly subsumed. | Retire V1 once V2 is the shipped path; keep one `FFMobileCreateCourse`. |
| `FFParentPortal` + `FFParentPortalV2` | V2 = "expanded sections." Same two-tier pattern. | Collapse to one once V2 ships; it's a rollout seam. |
| `OriginalityDetectionEnabled` + `OriginalityStubExternal` | Stub already slated for deletion above. | See DELETE. |

**Verdict:** Collapsing these clusters removes roughly **−22 toggles** without
removing a single user-visible capability.

---

## 🟢 DEFAULT-ON — shipped, low-risk, off is user-hostile

These gate features that are built, stable, carry **no external dependency and no
cost/liability**, and that nearly every deployment wants. Defaulting them OFF
means a fresh install ships with basic, expected behavior missing until an admin
goes hunting. Several *already* default on (listed for completeness); the rest
should follow.

| Flag | Default | Why it should be ON |
|---|---|---|
| `SessionManagementUIEnabled` | off | Letting users see and revoke their own sessions is a **security baseline**. Off-by-default is actively user-hostile. |
| `MFAEnabled` | off | MFA should be *available* by default (enforcement stays `none`). Shipping with MFA hidden is a bad security default. |
| `EmailNotificationsEnabled` | off | A fresh install sends **no** transactional notifications until flipped. Should be on wherever SMTP/SES is configured (pair with DERIVE). |
| `GradebookCSVEnabled` | off | Exporting the gradebook to CSV is table-stakes; there's no reason to hide it. |
| `ResubmissionWorkflowEnabled` | off | Assignment resubmissions are core LMS behavior, not a premium add-on. |
| `AnnotationEnabled` | off | Inline highlight/notes on content — shipped, low-risk authoring nicety. |
| `FeedbackMediaEnabled` | off | Audio/video feedback on submissions — low-risk, widely wanted. |
| `ItemAnalysisEnabled` | off | Quiz item statistics — pure reporting, no downside on. |
| `EquationEditorEnabled` | off | Visual math editor — low-risk authoring aid; STEM courses expect it. |
| `OutcomesReportEnabled` | off | Outcomes reporting — read-only analytics. |
| `ReportExportEnabled` | off | PDF export of reports — low-risk. |
| `AdminConsoleEnabled` | off | The **org admin console itself** is off by default. Admins have nowhere to go on a fresh install. Should be on. |
| `AdminSearchEnabled` | off | Org-wide admin search off by default is friction with no upside. |
| `FFWhatifGrades` | off | Student what-if grade projection — shipped, no dependency, broadly wanted. |
| `FFGradeCurving` | off | Instructor curving/scaling — shipped, low-risk. |
| `FFConditionalRelease` | off | Rule-based module release is **core LMS pedagogy**, not a niche add-on. At least reconsider off. |
| `FFPeerReview` | off | Peer review was a documented adoption *blocker* that got built — then hidden behind an off flag. |
| `StudentProgressEnabled` | **on** ✓ | Already correct. |
| `LearnerProfileEnabled` | **on** ✓ | Already correct. |
| `IntroCourseEnabled` | **on** ✓ | Already correct. |
| `MagicLinkEnabled` | **on** ✓ | Already correct. |
| `AdminAuditLogEnabled` | **on** ✓ | Correct — compliance surfaces should default on. |
| `MaintenanceBannerEnabled` | **on** ✓ | Correct. |
| `FFCalendarFeeds` / `FFFeedback` / `FFCourseMarketplace` | **on** ✓ | Deliberate exceptions, correct. |

**Caveat, stated honestly:** "default on" for the notification/analytics ones
assumes their dependency (SMTP, tracking consent) is satisfied — combine with the
DERIVE recommendation. And a few (`FFConditionalRelease`, `FFPeerReview`) are
judgment calls: they're safe on, but if the product intent is "instructors opt
in," keeping them off is defensible. The rest are not judgment calls — they
should be on.

---

## 🔵 KEEP-OFF — legitimately need to exist, off is correct

The large remainder. These earn their keep because turning them on **imposes
legal obligations, spends real money, widens the attack surface, or only matters
to one market.** Off-by-default is the right call; the only fair criticism is
that there are *so many* that they should be grouped into capability packs rather
than presented as a flat wall of ~90 switches.

**Compliance / legal modules** (impose workflows & obligations, jurisdiction-specific):
`FERPAWorkflowEnabled`, `CoppaWorkflowEnabled`, `GDPRModuleEnabled`,
`CCPAModuleEnabled`, `DPAPortalEnabled`, `StatePrivacyEnabled`,
`SOC2ModuleEnabled`, `IsoIsmsEnabled`, `DataResidencyEnabled`,
`SecurityDisclosureModuleEnabled`, `BackupModuleEnabled`, `FFResearchConsent`,
`FFAccessibilityIntake`, `AiDisclosureEnabled` (on ✓, correct for AI transparency).

**K-12 vertical pack** (irrelevant/undesirable outside K-12):
`FFParentPortal`, `FFReportCards`, `FFSISIntegration`, `FFBroadcasts`,
`FFClassroomSignals`, `FFConferenceScheduling`, `FFUiMode`, `FFLibrary`,
`FFDemographics`, `FFContentFilterIntegration`.

**Higher-Ed vertical pack:**
`FFCatalogIntegration`, `FFEnrollmentStateMachine`, `FFIncompleteGradeWorkflow`,
`FFGradeSubmission`, `FFCourseEvaluations`, `FFProctoringIntegration`,
`FFCoCurricularTranscript`, `FFLibraryIntegration`, `FFBookstoreIntegration`,
`FFEportfolio`, `FFTranscripts`, `FFTranscriptInbound`, `FFDiplomas`,
`FFAdvisingIntegration`, `FFCEUTracking`, `FFConsortiumSharing`,
`FFAcademicCalendar`, `FFPlagiarismChecks`.

**Self-learner / monetization pack:**
`FFSelfPacedMode`, `FFPublicCatalog`, `FFLearningPaths`, `FFCompletionCredentials`,
`FFCourseReviews`, `FFGamification`, `FFCompetencyBadges`, `FFOnboardingFlow`,
`FFStudyReminders`.

**AI cost / liability gates** (spend money or carry execution/output risk — off is
prudent): `GraderAgentEnabled`, `FFLessonGenerator`, `FFPersistentTutor`,
`FFAIStudyBuddy`, `ReadingLevelEnabled` (AI simplification), `CodeExecutionEnabled`
(sandboxed exec — real security surface).

**Integration / extensibility / attack surface:**
`FFWebhooks`, `FFZapierConnector`, `FFMarketplace`, `FFPublicAPI`, `FFAPITokens`,
`FFBoardsExternalSharing` (public share links).

**Infra / dependency-heavy** (also DERIVE candidates, but off is a safe default):
`StorageQuotasEnabled`, `H5PEnabled`, `ScormIngestionEnabled`, `OERLibraryEnabled`,
`TranslationMemoryEnabled`, `SpeechToTextEnabled`, `VideoCaptionsEnabled`,
`EngagementTrackingEnabled`, `SelfReflectionEnabled`, `InstructorInsightsEnabled`,
`ModeratedGradingEnabled`, `OriginalityDetectionEnabled`.

**Adaptive-learning engine gates** (advanced, off is fine):
`DiagnosticAssessmentsEnabled`, `SRSPracticeEnabled`, `IRTCatModeEnabled`,
`AdaptiveLearnerModelEnabled`.

**Temporary rollout gates** (correctly off *for now*, but should have an owner and
an expiry — flags with no removal date become permanent):
`RTLEnabled` ("until audit complete"), `FFReadingPreferences` ("flip after QA
sign-off"), the mobile `V2`/parent `V2` pairs (until the new path is the only
path).

**Mobile pack** (staged rollout of native parity — reasonable to gate, candidates
to retire as each ships): `FFMobileCanvasImport`, `FFMobileAdminConsole`,
`FFMobileEnrollmentAdd` (+ the create-course pair under COLLAPSE).

---

## Env-only operational gates (not admin flags, noted for completeness)

These live in `config.go` and are set via environment, not the platform settings
DB. Mostly fine as-is; one is a cleanup candidate.

| Gate | Assessment |
|---|---|
| `BackgroundJobsEnabled` | Legit operational gate (defaults on in local dev). Keep. |
| `SchedulerEnabled` | Legit; depends on background jobs. Keep. |
| `EnableAPIDocs` | Dev/docs convenience. Keep. |
| `AiProviderAbstractionEnabled` | **Documented as "Defaults to true (GA)" with an env rollback.** This is a post-GA rollback flag — a prime **retire** candidate once you're confident the legacy OpenRouter-only path can be deleted. |
| `StorageMigrateLocal`, `PayPalSandbox`, `DisablePIIRedaction` | One-shot/mode switches, correctly env-only. Keep. |

---

## What to actually do (prioritized)

1. **Delete the 5 non-flags first (zero product risk):** move `*Stub` seams to
   env/build config; retire the `FFVisualBoards` / `FFInteractiveQuizzes`
   constants. *(−5, and closes the "pretend-mode from settings" hole.)*
2. **Collapse the milestone clusters** (Grader Agent, Interactive Quizzes,
   LpAdapt, Motion) and the redundant pairs. *(≈ −22, no capability lost.)*
3. **Flip the DEFAULT-ON set** — start with the security ones
   (`SessionManagementUIEnabled`, `MFAEnabled` availability) and the "basic
   behavior missing on fresh install" ones (`EmailNotificationsEnabled`,
   `AdminConsoleEnabled`, `GradebookCSVEnabled`, `ResubmissionWorkflowEnabled`).
4. **Reframe the DERIVE set** in the UI as *auto-enabled when configured /
   disabled until you add credentials*, so admins stop flipping switches that
   silently do nothing.
5. **Group the KEEP-OFF remainder into capability packs** (Compliance, K-12,
   Higher-Ed, Marketplace, AI, Integrations). Same flags underneath; a settings
   page that shows ~6 packs + a handful of standalone toggles instead of a wall
   of ~90.
6. **Give every remaining "temporary" gate an owner and a removal date.**
   `RTLEnabled` "until audit complete" and `FFReadingPreferences` "flip after QA"
   are how permanent flags are born.

---

## Appendix — reclassification summary

| Verdict | Flags |
|---|---|
| 🔴 DELETE | `OriginalityStubExternal`, `ClamAVStub`, `OERStub`, `FFVisualBoards`, `FFInteractiveQuizzes` |
| ⚙️ NOT-A-FLAG | `LearnerModelEMAAlpha`, `BadgesDefaultPublic`, `MagicLinkEnrolledOnly`, `LRSAnonymizeActors` |
| 🟢 DEFAULT-ON | `SessionManagementUIEnabled`, `MFAEnabled`, `EmailNotificationsEnabled`, `GradebookCSVEnabled`, `ResubmissionWorkflowEnabled`, `AnnotationEnabled`, `FeedbackMediaEnabled`, `ItemAnalysisEnabled`, `EquationEditorEnabled`, `OutcomesReportEnabled`, `ReportExportEnabled`, `AdminConsoleEnabled`, `AdminSearchEnabled`, `FFWhatifGrades`, `FFGradeCurving`, `FFConditionalRelease`*, `FFPeerReview`* |
| 🟡 COLLAPSE | `GraderAgent{ReviewInbox,SuggestMode,TextEntryGrading,RunFilters,CostEstimate,CancelRun}`, `FFIq{LiveHosting,TeamMode,StudentPaced,Homework,GradebookPush}`, `LpAdapt{Recommendations,Review,Modality,Tutor}`, `FFMotion{Navigation,Reveal,Lists}`, `FFReadAloud`+`ReadAloudEnabled`, `FFAccommodationsEngine`, `FFAltTextEnforcement`+`AltTextEnforcementEnabled`, `FFMobileCreateCourse`, `FFParentPortalV2` |
| ⚪ DERIVE | SSO set (`SAML/OIDC/Clever/ClassLink/OneRoster/Scim/LTI`), payments (`FFStripeBilling/FFPaymentsEnabled/FFRevenueShare/FFTaxCollection`), bots (`FFBot{Slack,Teams,Discord}`), `FFEmailSES`, `PushNotificationsEnabled`, `DRMEnabled`, `XAPIEmissionEnabled`, `FFRedisCache`, `AvScanningEnabled` |
| 🔵 KEEP-OFF | Compliance, K-12, Higher-Ed, Self-learner/monetization, AI-cost, integration, infra, adaptive-engine, and temporary-rollout sets (see section above) |

\* judgment calls — safe to default on, but defensible to keep off if the product
intent is explicit instructor opt-in.
