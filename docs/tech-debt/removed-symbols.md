# Removed symbols ledger (TD.4)

Permanent record of symbols deleted as confirmed dead code. Search this file when looking for a missing helper.

| Symbol | Package | Path | Removed | Rationale | PR / change |
|---|---|---|---|---|---|
| `RollbackLatestFromPool` | `internal/migrate` | `internal/migrate/rollback.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `ParseItemStatus` | `internal/models/transcriptorder` | `internal/models/transcriptorder/state.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `EntryFromUsage` | `internal/repos/aiusage` | `internal/repos/aiusage/aiusage.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `EntryFromProviderUsage` | `internal/repos/aiusage` | `internal/repos/aiusage/aiusage.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `CourseTitleByID` | `internal/repos/badges` | `internal/repos/badges/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `ListAwardsByDefinition` | `internal/repos/badges` | `internal/repos/badges/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `SetAttachmentScanStatus` | `internal/repos/board` | `internal/repos/board/attachments.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `NeedsRenormalize` | `internal/repos/board` | `internal/repos/board/fractional.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `IsValidLayout` | `internal/repos/board` | `internal/repos/board/layout.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `CountUpdatesSinceSnapshot` | `internal/repos/board` | `internal/repos/board/updates.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `ListUserPinnedCourseSummaries` | `internal/repos/course` | `internal/repos/course/catalog_user_prefs.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `GetMarketplaceListing` | `internal/repos/course` | `internal/repos/course/marketplace.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `IsMarketplaceListed` | `internal/repos/course` | `internal/repos/course/marketplace.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `DeleteByToken` | `internal/repos/devicepushtokens` | `internal/repos/devicepushtokens/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `HashBytes` | `internal/repos/diplomas` | `internal/repos/diplomas/hash.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `LookupGradePolicy` | `internal/repos/introcourse` | `internal/repos/introcourse/content_items.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `CountEngagementEvents` | `internal/repos/learnerprofile` | `internal/repos/learnerprofile/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `IsOfficialCourseID` | `internal/repos/marketplacecourses` | `internal/repos/marketplacecourses/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `FindActiveByLinkID` | `internal/repos/parentlinkinvites` | `internal/repos/parentlinkinvites/parentlinkinvites.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `GetForSubmissionGrader` | `internal/repos/provisionalgrades` | `internal/repos/provisionalgrades/repo.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `LeaderboardRows` | `internal/repos/quizgame` | `internal/repos/quizgame/leaderboard.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `AddPlayerScore` | `internal/repos/quizgame` | `internal/repos/quizgame/players.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `ListResponsesForQuestion` | `internal/repos/quizgame` | `internal/repos/quizgame/responses.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `EnqueueReportedContent` | `internal/repos/quizgame` | `internal/repos/quizgame/review.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `AssignPlayerToTeam` | `internal/repos/quizgame` | `internal/repos/quizgame/teams.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `GetTeam` | `internal/repos/quizgame` | `internal/repos/quizgame/teams.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `EquivalentIDs` | `internal/repos/terms` | `internal/repos/terms/terms.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `GetDeliveryAttemptByIdempotency` | `internal/repos/transcripts` | `internal/repos/transcripts/delivery.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `ListNotificationLog` | `internal/repos/transcripts` | `internal/repos/transcripts/notifications.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `Effective.HasOperationalSettings` | `internal/service/accommodations` | `internal/service/accommodations/effective.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `Effective.HasDisplayAccommodations` | `internal/service/accommodations` | `internal/service/accommodations/effective.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `UserMarketplaceAccess` | `internal/service/billing` | `internal/service/billing/entitlements.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `Service.GetCourseValues` | `internal/service/customfields` | `internal/service/customfields/service.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `RationaleForHelpSeeking` | `internal/service/learnerprofile` | `internal/service/learnerprofile/adaptive_rationale.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `RegenerateComponent` | `internal/service/lessonplanai` | `internal/service/lessonplanai/service.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `Service.AfterSeatCountChange` | `internal/service/licensesvc` | `internal/service/licensesvc/service.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |
| `SpanAttr` | `internal/telemetry` | `internal/telemetry/otel.go` | 2026-07-28 | Definition-only unreachable helper; no callers, no tests | TD.4 batch |

See also: [deadcode-triage.md](deadcode-triage.md), baseline `scripts/allowlists/deadcode-baseline.txt`.

