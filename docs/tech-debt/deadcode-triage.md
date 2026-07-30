# Deadcode triage (TD.4)

> Working document for [TD.4](../completed/tech_debt/TD.4-delete-confirmed-dead-code.md).
> Generated 2026-07-28. Source: `deadcode ./...` on `server/` after first deletion batch.

## Summary

| Classification | Count | Notes |
|---|---:|---|
| **DELETE** (this batch) | 37 | Definition-only, no callers, no test dependency; removed |
| **WIRE** | 41 | In-flight AP/AC plan scaffolding; decision date **2026-10-28** |
| **KEEP** | 104 | Security/obs/false-positive or not proven abandoned |
| **TEST-ONLY** | 52 | Reachable from tests or mock/interface surface |
| **Remaining live** | 197 | New baseline after deletions |

Sign-off for DELETE batch: **platform-td4** (backend platform, 2026-07-28).
Security KEEP packages reviewed against TD.4 §6 (no dormant control removed).

## Classification legend

- **DELETE** — abandoned; removed in this story.
- **WIRE** — intended API for an active plan; keep until decision date, then re-triage as DELETE if still unwired.
- **KEEP** — false positive, security/obs retain, or insufficient evidence of abandonment.
- **TEST-ONLY** — production-unreachable but serves unit/integration tests or interface mocks.

## DELETE batch (removed)

| Symbol | Package | Sign-off | Rationale |
|---|---|---|---|
| `RollbackLatestFromPool` | `internal/migrate` | platform-td4 | Definition-only; zero production/test callers |
| `ParseItemStatus` | `internal/models/transcriptorder` | platform-td4 | Definition-only; zero production/test callers |
| `EntryFromUsage` | `internal/repos/aiusage` | platform-td4 | Definition-only; zero production/test callers |
| `EntryFromProviderUsage` | `internal/repos/aiusage` | platform-td4 | Definition-only; zero production/test callers |
| `CourseTitleByID` | `internal/repos/badges` | platform-td4 | Definition-only; zero production/test callers |
| `ListAwardsByDefinition` | `internal/repos/badges` | platform-td4 | Definition-only; zero production/test callers |
| `SetAttachmentScanStatus` | `internal/repos/board` | platform-td4 | Definition-only; zero production/test callers |
| `NeedsRenormalize` | `internal/repos/board` | platform-td4 | Definition-only; zero production/test callers |
| `IsValidLayout` | `internal/repos/board` | platform-td4 | Definition-only; zero production/test callers |
| `CountUpdatesSinceSnapshot` | `internal/repos/board` | platform-td4 | Definition-only; zero production/test callers |
| `ListUserPinnedCourseSummaries` | `internal/repos/course` | platform-td4 | Definition-only; zero production/test callers |
| `GetMarketplaceListing` | `internal/repos/course` | platform-td4 | Definition-only; zero production/test callers |
| `IsMarketplaceListed` | `internal/repos/course` | platform-td4 | Definition-only; zero production/test callers |
| `DeleteByToken` | `internal/repos/devicepushtokens` | platform-td4 | Definition-only; zero production/test callers |
| `HashBytes` | `internal/repos/diplomas` | platform-td4 | Definition-only; zero production/test callers |
| `LookupGradePolicy` | `internal/repos/introcourse` | platform-td4 | Definition-only; zero production/test callers |
| `CountEngagementEvents` | `internal/repos/learnerprofile` | platform-td4 | Definition-only; zero production/test callers |
| `IsOfficialCourseID` | `internal/repos/marketplacecourses` | platform-td4 | Definition-only; zero production/test callers |
| `FindActiveByLinkID` | `internal/repos/parentlinkinvites` | platform-td4 | Definition-only; zero production/test callers |
| `GetForSubmissionGrader` | `internal/repos/provisionalgrades` | platform-td4 | Definition-only; zero production/test callers |
| `LeaderboardRows` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `AddPlayerScore` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `ListResponsesForQuestion` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `EnqueueReportedContent` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `AssignPlayerToTeam` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `GetTeam` | `internal/repos/quizgame` | platform-td4 | Definition-only; zero production/test callers |
| `EquivalentIDs` | `internal/repos/terms` | platform-td4 | Definition-only; zero production/test callers |
| `GetDeliveryAttemptByIdempotency` | `internal/repos/transcripts` | platform-td4 | Definition-only; zero production/test callers |
| `ListNotificationLog` | `internal/repos/transcripts` | platform-td4 | Definition-only; zero production/test callers |
| `Effective.HasOperationalSettings` | `internal/service/accommodations` | platform-td4 | Definition-only; zero production/test callers |
| `Effective.HasDisplayAccommodations` | `internal/service/accommodations` | platform-td4 | Definition-only; zero production/test callers |
| `UserMarketplaceAccess` | `internal/service/billing` | platform-td4 | Definition-only; zero production/test callers |
| `Service.GetCourseValues` | `internal/service/customfields` | platform-td4 | Definition-only; zero production/test callers |
| `RationaleForHelpSeeking` | `internal/service/learnerprofile` | platform-td4 | Definition-only; zero production/test callers |
| `RegenerateComponent` | `internal/service/lessonplanai` | platform-td4 | Definition-only; zero production/test callers |
| `Service.AfterSeatCountChange` | `internal/service/licensesvc` | platform-td4 | Definition-only; zero production/test callers |
| `SpanAttr` | `internal/telemetry` | platform-td4 | Definition-only; zero production/test callers |

## Remaining findings

| Symbol | Classification | Reason |
|---|---|---|
| `internal/aidisclosure/disclosure.go:InvalidatePublicDisclosureCache` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/auth/hibp/hibp.go:RequestURLForPassword` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/auth/passwordpolicy/policy.go:StrengthDisplayEnglish` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/auth/passwordpolicy/policy.go:StrengthLabel` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/background/config_source.go:StaticConfigSource.Config` | KEEP | implements ConfigSource interface |
| `internal/background/jobqueue.go:Registry.Types` | KEEP | introspection helper for job registry |
| `internal/httpserver/canvas_assignment_submissions_import.go:canvasFirstSubmissionAttachment` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/httpserver/canvas_submission_sync.go:canvasInstructorCommentFromSubmission` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/l10n/locale.go:NormalizeLocale` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/logging/diploma_metrics.go:DiplomaMetrics.Snapshot` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/logging/metrics.go:RedactionMetrics.Reset` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/logging/transcript_notification_metrics.go:TranscriptNotificationMetrics.Snapshot` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/logging/wallet_metrics.go:WalletMetrics.Snapshot` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/migrate/run.go:FromPool` | KEEP | used by integration tests; alternate entry to RunWithFS |
| `internal/models/coursemoduleassignment/validation.go:ValidateAssignmentDeliverySettings` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/models/coursemoduleassignment/validation.go:ValidateAssignmentLateSettings` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/models/coursestructure/types.go:ItemResponseFromRow` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/models/latesubmissionpolicy/policy.go:ValidateLateSubmissionPolicyPair` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/models/transcriptfees/quote.go:ApplyWaiver` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/models/transcriptorder/state.go:ParseOrderStatus` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/publicapi/openapi_serve.go:SpecBytes` | KEEP | TD.3 public embed surface; served via ServeOpenAPI |
| `internal/quizgame/engine/grade.go:StubPoints` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/quizgame/engine/modes.go:GuestsAllowed` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/quizgame/scoring/scoring.go:Recompute` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/repos/adaptivecontent/jobs.go:HasInFlightOrDoneJob` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/outcomes.go:AdaptivePostScoreForEnrollment` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/outcomes.go:GetOutcome` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/servings.go:GetServing` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/variants.go:BumpUnitsForBaseContentItem` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/variants.go:DeleteKeyTerm` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivecontent/variants.go:InsertKeyTerm` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivepath/repo.go:DeletePathOverride` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivepath/repo.go:GetPathOverride` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivepath/repo.go:InsertPathEvent` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/adaptivepath/repo.go:ListRulesForCourse` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/repos/apitokens/usage.go:ResetUsageQueue` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/board/arrange.go:NextSortIndexBetween` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/board/fractional.go:MidpointSortIndex` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/board/fractional.go:PrependSortIndex` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/board/fractional.go:RenormalizeSortIndexes` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/board/reactions.go:EngagementScore` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/classroomsignals/repo.go:CanTransition` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/course/grade_level.go:SetGradeLevel` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/course/marketplace_storefront.go:MarketplaceFilter.ToPublicCatalogFilter` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/course/marketplace.go:IsMarketplaceListable` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/demographics/lunch_codes.go:boolPtr` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/demographics/lunch_codes.go:MapLunchCode` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/marketplacecourses/content_items.go:MaxStoredContentVersion` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/marketplacecourses/repo.go:LookupBySlug` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/organization/organization.go:Create` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/organization/organization.go:GetRegion` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/organization/organization.go:insertOrg` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/outcomesreport/repo.go:GetImprovementNote` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/passwordcreditevents/repo.go:LatestForUser` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/pinnedsettings/pinnedsettings.go:Get` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/pinnedsettings/pinnedsettings.go:Surfaces` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/quizgame/reports.go:floatPtrEq` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/quizgame/reports.go:intPtrEq` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/quizgame/reports.go:ReportsMatch` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/researchconsent/repo.go:LatestDecision` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/screenshare/sessions.go:GetActiveForCourse` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/screenshare/sessions.go:SetPolicy` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/submissionannotations/repo.go:GetByID` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/transcripts/analytics.go:NetRevenueMinor` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/repos/user/locale.go:NormalizeLocalePrimary` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/screenshare/sfu/room.go:Registry.Len` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/screenshare/sfu/room.go:Room.ViewerCount` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/screenshare/turn/creds.go:ValidateExpiry` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/service/academicrecord/hash.go:VerifyHash` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/adaptivecontent/fairness.go:MaxAbsGap` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/governance.go:VariantPassesServeGates` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/holdout.go:HoldoutBucket` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/pipeline.go:EnqueueOnCacheMiss` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/pipeline.go:LookupReadyVariant` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/profile.go:ValidateModalityPref` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/ratelimit.go:GlobalRateLimiter.ResetForTest` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/reports.go:RefreshAllCourseReports` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/review.go:HasHardKeyTermFailure` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivecontent/service.go:SetKillSwitchForTest` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivepath/service.go:AdaptivePathsActiveForCourse` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivepath/service.go:AdaptivePathsGloballyEnabled` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivepath/service.go:New` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivepath/service.go:Service.Health` | WIRE | In-flight AC plan scaffolding; decision date 2026-10-28 |
| `internal/service/adaptivequizai/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/adaptivequizai/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/aiprovider/anthropic.go:NewAnthropicProviderWithBaseURL` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/auth.go:deploymentMapLookup` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/auth.go:ResolveAzureDeployment` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/bedrock.go:NewBedrockProviderWithSDK` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/catalog.go:ClearCatalogCache` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.Complete` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.CompleteStream` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.CompleteVision` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.Embed` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.GenerateImage` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/mock.go:MockProvider.Name` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/openai.go:NewOpenAIProviderWithBaseURL` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/provider.go:Capabilities` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/resolver.go:IsCapabilityGap` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/resolver.go:Resolver.apiKeyForProvider` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/aiprovider/vertex.go:NewVertexProviderWithTokenSource` | WIRE | In-flight AP plan scaffolding; decision date 2026-10-28 |
| `internal/service/assignmentrubricai/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/assignmentrubricai/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/canvascourseimport/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/canvascourseimport/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/captions/service.go:FormatVTT` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/commoncartridge/manifest.go:elLocalName` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/commoncartridge/manifest.go:isQtiResource` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/commoncartridge/manifest.go:QtiXMLPathsFromManifest` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/commoncartridge/manifest.go:stringOrEmpty` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/commoncartridge/manifest.go:walkElements` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/competencygating/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/coppa/service.go:ClassifyMinor` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/service/coppa/service.go:FlagMinorAccount` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/service/courseexportimport/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/courseexportimport/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/courseimageupload/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/courseimageupload/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/csvimport/service.go:NormalizeEmail` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/drm/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/drm/service.go:SubnetOf` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/emailtemplates/metrics.go:CompileFailTotal` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/emailtemplates/metrics.go:CompileOKTotal` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/emailtemplates/metrics.go:FallbackTotal` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/emailtemplates/metrics.go:SavesTotal` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/emailtemplates/metrics.go:TestSendsTotal` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/enrollments/parse.go:ParseEmailList` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/feedbackmedia/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/feedbackmedia/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/gradeexport/service.go:GenerateCSV` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/grading/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/grading/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/gradingagent/workflow.go:ParseWorkflowGraph` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/hintservice/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/hintservice/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/inlinequestionsai/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/inlinequestionsai/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/integrations/lti.go:GenerateLTILaunchParams` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/integrations/lti.go:normalizeURL` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/integrations/lti.go:oauthEscape` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/integrations/lti.go:signLTI` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/introcourse/completion.go:ShouldNudgeIntroCourse` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/introcourse/fixtures.go:ContentFS` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/introcourse/localization.go:InvalidateLocaleIndex` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/irt/irtmath.go:IccCurvePoints` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/irt/irtmath.go:SortUniqueUUIDs` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/irt/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/irt/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/learningpaths/service.go:CalcProgressPercent` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/mailservice/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/mailservice/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/marketplacecourses/fixtures.go:AllDesiredSlugs` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/marketplacecourses/fixtures.go:ContentFS` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/misconception/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/misconception/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/openrouter/openrouter.go:NewClientWithBaseURL` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/originality/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/originality/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/paymentprovider/mock.go:MockProvider.CreateCheckoutSession` | TEST-ONLY | Mock/interface implementor; call-graph misses interface dispatch |
| `internal/service/peerreview/service.go:BlendGrade` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/push/payload.go:BuildNativePayloadJSON` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/qtiparser/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/qtiparser/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/questionbank/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/questionbank/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizattempt/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizattempt/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizautosubmit/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizautosubmit/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizgenerationai/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizgenerationai/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizlockdown/lockdown.go:ParseLockdownModeSetting` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/quizsubmissions/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/quizsubmissions/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/recommendations/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/recommendations/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/research_consent/consent.go:VerifyRecord` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/service/settingsops/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/settingsops/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/srsscheduler/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/srsscheduler/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/standards/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/standards/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/transcriptnotify/notify.go:MappingForTest` | TEST-ONLY | Test harness helper |
| `internal/service/translationmemory/service.go:PrefixWords` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/translationmemory/service.go:SuggestGlossaryTranslation` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/translationmemory/service.go:trigrams` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/translationmemory/service.go:TrigramSimilarity` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/tutorsession/metrics.go:Snapshot` | KEEP | Metrics reader; may be scraped or used by future dashboards (TD.4 §6) |
| `internal/service/tutorsession/service.go:DisclosureMessage` | KEEP | Retained: has test coverage and/or plausible future call site; not definition-only abandoned |
| `internal/service/zipimport/service.go:New` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/service/zipimport/service.go:Service.Health` | TEST-ONLY | Service stub New/Health exercised by package unit tests; not wired to production health |
| `internal/telemetry/default.go:SetDefaultForTest` | KEEP | test-only harness |
| `internal/webhooks/sign.go:VerifySignature` | KEEP | Security/compliance/observability-sensitive; specialist retain (TD.4 §6) |
| `internal/workers/avscan/worker.go:EnqueueForObject` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/workers/transcode/worker.go:BuildMasterPlaylistContent` | KEEP | Retained pending next audit; not proven abandoned |
| `internal/yrelay/room.go:Registry.Stats` | KEEP | Retained pending next audit; not proven abandoned |

## Decision dates (WIRE)

| Cluster | Owner plan | Decision date | Action if still unwired |
|---|---|---|---|
| `internal/service/aiprovider` | AP.1–AP.9 | 2026-10-28 | Convert to DELETE |
| `internal/service/adaptivecontent` + `repos/adaptivecontent` | AC.5–AC.9 | 2026-10-28 | Convert to DELETE |
| `internal/service/adaptivepath` + `repos/adaptivepath` | AC path plans | 2026-10-28 | Convert to DELETE |

## Regeneration

```bash
cd server && deadcode ./... | sort
scripts/check-deadcode-baseline.sh --update   # after intentional cleanup only
```

