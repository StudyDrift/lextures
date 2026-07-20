package platformconfig

import "github.com/lextures/lextures/server/internal/config"

func applyPlatformBools(out *config.Config, db *Row, def Defaults) {
	// Feature flags are DB-managed (no env seed); a missing settings row means "all columns
	// unset", so every flag falls back to its documented default below.
	//
	// Exception: OriginalityStubExternal, ClamAVStub, and OERStub are env-only test/dev
	// seams (ORIGINALITY_STUB_EXTERNAL / CLAMAV_STUB / OER_STUB). They are loaded in
	// config.Load and must not be overwritten from the platform settings row.
	if db == nil {
		db = &Row{}
	}
	out.SAMLSSOEnabled = mergeBool(db.SAMLSSOEnabled, false)
	// DEFAULT-ON: shipped, low-risk authoring/security baselines (docs/plan/flags.md).
	out.AnnotationEnabled = mergeBool(db.AnnotationEnabled, true)
	out.FeedbackMediaEnabled = mergeBool(db.FeedbackMediaEnabled, true)
	out.BlindGradingEnabled = mergeBool(db.BlindGradingEnabled, def.BlindGradingEnabled)
	out.ModeratedGradingEnabled = mergeBool(db.ModeratedGradingEnabled, false)
	out.OriginalityDetectionEnabled = mergeBool(db.OriginalityDetectionEnabled, false)
	// OriginalityStubExternal: env-only — leave out.* from config.Load.
	out.GradePostingPoliciesEnabled = mergeBool(db.GradePostingPoliciesEnabled, def.GradePostingPoliciesEnabled)
	out.GradebookCSVEnabled = mergeBool(db.GradebookCSVEnabled, true)
	out.ResubmissionWorkflowEnabled = mergeBool(db.ResubmissionWorkflowEnabled, true)
	out.LTIEnabled = mergeBool(db.LTIEnabled, false)
	out.OneRosterEnabled = mergeBool(db.OneRosterEnabled, false)
	out.ScimEnabled = mergeBool(db.ScimEnabled, false)
	out.OIDCSSOEnabled = mergeBool(db.OIDCSSOEnabled, false)
	out.CleverSSOEnabled = mergeBool(db.CleverSSOEnabled, false)
	out.ClassLinkSSOEnabled = mergeBool(db.ClassLinkSSOEnabled, false)
	out.MFAEnabled = mergeBool(db.MFAEnabled, true)
	out.MagicLinkEnabled = mergeBool(db.MagicLinkEnabled, def.MagicLinkEnabled)
	out.MagicLinkEnrolledOnly = mergeBool(db.MagicLinkEnrolledOnly, false)
	out.SessionManagementUIEnabled = mergeBool(db.SessionManagementUIEnabled, true)
	out.EmailNotificationsEnabled = mergeBool(db.EmailNotificationsEnabled, true)
	// SES email provider is opt-in (default off).
	out.FFEmailSES = mergeBool(db.FFEmailSES, false)
	out.PushNotificationsEnabled = mergeBool(db.PushNotificationsEnabled, false)
	out.VirtualClassroomEnabled = mergeBool(db.VirtualClassroomEnabled, def.VirtualClassroomEnabled)
	out.DRMEnabled = mergeBool(db.DRMEnabled, false)
	out.VideoTranscodingEnabled = mergeBool(db.VideoTranscodingEnabled, false)
	out.AutoCaptioningEnabled = mergeBool(db.AutoCaptioningEnabled, false)
	out.VideoCaptionsEnabled = mergeBool(db.VideoCaptionsEnabled, false)
	out.StorageQuotasEnabled = mergeBool(db.StorageQuotasEnabled, false)
	out.AtRiskAlertsEnabled = mergeBool(db.AtRiskAlertsEnabled, false)
	out.AvScanningEnabled = mergeBool(db.AvScanningEnabled, false)
	// ClamAVStub / OERStub: env-only — leave out.* from config.Load.
	out.H5PEnabled = mergeBool(db.H5PEnabled, false)
	out.ScormIngestionEnabled = mergeBool(db.ScormIngestionEnabled, false)
	out.OERLibraryEnabled = mergeBool(db.OERLibraryEnabled, false)
	out.ItemAnalysisEnabled = mergeBool(db.ItemAnalysisEnabled, true)
	out.StudentProgressEnabled = mergeBool(db.StudentProgressEnabled, true)
	out.EngagementTrackingEnabled = mergeBool(db.EngagementTrackingEnabled, false)
	out.SelfReflectionEnabled = mergeBool(db.SelfReflectionEnabled, false)
	out.LearnerProfileEnabled = mergeBool(db.LearnerProfileEnabled, true)

	// COLLAPSE LpAdapt*: one capability ("profile drives personalization").
	// Any legacy child column that was ON enables the pack; all children then follow.
	lpAdapt := mergeBool(db.LpAdaptRecommendationsEnabled, false) ||
		mergeBool(db.LpAdaptReviewEnabled, false) ||
		mergeBool(db.LpAdaptModalityEnabled, false) ||
		mergeBool(db.LpAdaptTutorEnabled, false)
	out.LpAdaptRecommendationsEnabled = lpAdapt
	out.LpAdaptReviewEnabled = lpAdapt
	out.LpAdaptModalityEnabled = lpAdapt
	out.LpAdaptTutorEnabled = lpAdapt

	out.IntroCourseEnabled = mergeBool(db.IntroCourseEnabled, true)
	out.OutcomesReportEnabled = mergeBool(db.OutcomesReportEnabled, true)
	out.InstructorInsightsEnabled = mergeBool(db.InstructorInsightsEnabled, false)
	out.EquationEditorEnabled = mergeBool(db.EquationEditorEnabled, true)
	out.ReadingLevelEnabled = mergeBool(db.ReadingLevelEnabled, false)

	// COLLAPSE Grader Agent milestone flags into the parent (keep Vision as optional cost gate).
	graderOn := mergeBool(db.GraderAgentEnabled, false)
	out.GraderAgentEnabled = graderOn
	out.GraderAgentReviewInboxEnabled = graderOn
	out.GraderAgentSuggestModeEnabled = graderOn
	out.GraderAgentTextEntryGradingEnabled = graderOn
	out.GraderAgentVisionGradingEnabled = mergeBool(db.GraderAgentVisionGradingEnabled, false)
	out.GraderAgentRunFiltersEnabled = graderOn
	out.GraderAgentCostEstimateEnabled = graderOn
	out.GraderAgentCancelRunEnabled = graderOn

	out.CodeExecutionEnabled = mergeBool(db.CodeExecutionEnabled, false)

	// COLLAPSE alt-text soft/hard into one gate (enforce when enabled).
	altText := mergeBool(db.AltTextEnforcementEnabled, false) || mergeBool(db.FFAltTextEnforcement, false)
	out.AltTextEnforcementEnabled = altText
	out.FFAltTextEnforcement = altText

	out.FFHighContrastReducedMotion = mergeBool(db.FFHighContrastReducedMotion, false)

	// COLLAPSE motion kill-switches into one (Navigation column is the master store).
	motion := mergeBool(db.FFMotionNavigation, true)
	out.FFMotionNavigation = motion
	out.FFMotionReveal = motion
	out.FFMotionLists = motion
	out.FFMotionOverlays = motion
	out.FFMotionControls = motion
	out.FFMotionDelight = motion

	// COLLAPSE mobile create V1/V2: either column enables create; V2 wizard is the only path.
	mobileCreate := mergeBool(db.FFMobileCreateCourse, false) || mergeBool(db.FFMobileCourseCreateV2, false)
	out.FFMobileCreateCourse = mobileCreate
	out.FFMobileCourseCreateV2 = mobileCreate

	out.FFMobileCanvasImport = mergeBool(db.FFMobileCanvasImport, false)
	out.FFMobileAdminConsole = mergeBool(db.FFMobileAdminConsole, false)
	out.FFMobileEnrollmentAdd = mergeBool(db.FFMobileEnrollmentAdd, false)
	out.FFMobileLiveQuiz = mergeBool(db.FFMobileLiveQuiz, false)
	out.FFMobileWhiteboardEdit = mergeBool(db.FFMobileWhiteboardEdit, false)
	out.FFMobileMarketplacePurchase = mergeBool(db.FFMobileMarketplacePurchase, false)
	out.FFMobileBoardsAdvanced = mergeBool(db.FFMobileBoardsAdvanced, false)

	// COLLAPSE parent portal V2 into the parent (expanded sections always on with the portal).
	parentPortal := mergeBool(db.FFParentPortal, false)
	out.FFParentPortal = parentPortal
	out.FFParentPortalV2 = parentPortal

	out.FFReportCards = mergeBool(db.FFReportCards, false)
	out.FFSISIntegration = mergeBool(db.FFSISIntegration, false)
	out.FFCatalogIntegration = mergeBool(db.FFCatalogIntegration, false)
	out.FFEnrollmentStateMachine = mergeBool(db.FFEnrollmentStateMachine, false)
	out.FFIncompleteGradeWorkflow = mergeBool(db.FFIncompleteGradeWorkflow, false)
	out.FFLibrary = mergeBool(db.FFLibrary, false)
	out.FFBroadcasts = mergeBool(db.FFBroadcasts, false)
	out.FFConferenceScheduling = mergeBool(db.FFConferenceScheduling, false)
	out.FFDemographics = mergeBool(db.FFDemographics, false)
	out.FFContentFilterIntegration = mergeBool(db.FFContentFilterIntegration, false)
	out.FFUiMode = mergeBool(db.FFUiMode, false)
	out.FFGradeSubmission = mergeBool(db.FFGradeSubmission, false)
	out.FFWhatifGrades = mergeBool(db.FFWhatifGrades, true)
	out.FFGradeCurving = mergeBool(db.FFGradeCurving, true)
	out.FFAcademicCalendar = mergeBool(db.FFAcademicCalendar, false)
	out.FFPlagiarismChecks = mergeBool(db.FFPlagiarismChecks, false)
	out.FFCourseEvaluations = mergeBool(db.FFCourseEvaluations, false)
	out.FFProctoringIntegration = mergeBool(db.FFProctoringIntegration, false)
	out.FFCoCurricularTranscript = mergeBool(db.FFCoCurricularTranscript, false)
	out.FFEportfolio = mergeBool(db.FFEportfolio, false)
	out.FFBookstoreIntegration = mergeBool(db.FFBookstoreIntegration, false)
	out.FFTranscripts = mergeBool(db.FFTranscripts, false)
	out.FFTranscriptInbound = mergeBool(db.FFTranscriptInbound, false)
	out.FFDiplomas = mergeBool(db.FFDiplomas, false)
	out.FFWebhooks = mergeBool(db.FFWebhooks, false)
	out.FFZapierConnector = mergeBool(db.FFZapierConnector, false)
	out.FFMarketplace = mergeBool(db.FFMarketplace, false)
	out.FFAdvisingIntegration = mergeBool(db.FFAdvisingIntegration, false)
	out.FFResearchConsent = mergeBool(db.FFResearchConsent, false)
	out.FFAccessibilityIntake = mergeBool(db.FFAccessibilityIntake, false)
	out.FFCEUTracking = mergeBool(db.FFCEUTracking, false)
	out.FFConsortiumSharing = mergeBool(db.FFConsortiumSharing, false)
	out.FFSelfPacedMode = mergeBool(db.FFSelfPacedMode, false)
	out.FFPublicCatalog = mergeBool(db.FFPublicCatalog, false)
	// Course marketplace defaults ON (exception to the usual default-off convention; plan MKT1).
	out.FFCourseMarketplace = mergeBool(db.FFCourseMarketplace, true)
	out.FFFeedback = mergeBool(db.FFFeedback, true)
	// Collaboration boards are course-scoped only; platform master switch removed.
	out.FFVisualBoards = true
	// Boards realtime defaults ON so multi-user boards sync live without a refresh.
	out.FFBoardsRealtime = mergeBool(db.FFBoardsRealtime, true)
	out.FFBoardsExternalSharing = mergeBool(db.FFBoardsExternalSharing, false)
	// Live Quizzes are course-scoped only; platform master switch removed.
	out.FFInteractiveQuizzes = true
	// COLLAPSE IQ hosting/modes/gradebook into the per-course Live Quizzes flag (always on at platform).
	out.FFIqLiveHosting = true
	out.FFIqTeamMode = true
	out.FFIqStudentPaced = true
	out.FFIqHomework = true
	out.FFIqGradebookPush = true
	// Real platform gates for Live Quizzes (security / AI spend / public listing).
	out.FFIqPublicKitCatalog = mergeBool(db.FFIqPublicKitCatalog, false)
	out.FFIqGuestJoin = mergeBool(db.FFIqGuestJoin, false)
	out.FFIqAiGeneration = mergeBool(db.FFIqAiGeneration, false)
	out.FFPublicAPI = mergeBool(db.FFPublicAPI, false)
	out.FFStripeBilling = mergeBool(db.FFStripeBilling, false)
	out.FFPaymentsEnabled = mergeBool(db.FFPaymentsEnabled, false)
	out.FFRevenueShare = mergeBool(db.FFRevenueShare, false)
	out.FFTaxCollection = mergeBool(db.FFTaxCollection, false)
	out.FFLearningPaths = mergeBool(db.FFLearningPaths, false)
	out.FFConditionalRelease = mergeBool(db.FFConditionalRelease, true)
	out.FFPeerReview = mergeBool(db.FFPeerReview, true)
	out.FFCompletionCredentials = mergeBool(db.FFCompletionCredentials, false)
	out.FFCourseReviews = mergeBool(db.FFCourseReviews, false)
	out.FFGamification = mergeBool(db.FFGamification, false)
	out.FFCompetencyBadges = mergeBool(db.FFCompetencyBadges, false)
	out.BadgesDefaultPublic = mergeBool(db.BadgesDefaultPublic, false)
	out.FFOnboardingFlow = mergeBool(db.FFOnboardingFlow, false)
	out.FFStudyReminders = mergeBool(db.FFStudyReminders, false)
	out.FFAIStudyBuddy = mergeBool(db.FFAIStudyBuddy, false)
	out.FFLessonGenerator = mergeBool(db.FFLessonGenerator, false)
	out.FFPersistentTutor = mergeBool(db.FFPersistentTutor, false)
	out.FFAPITokens = mergeBool(db.FFAPITokens, false)
	out.FFBotSlack = mergeBool(db.FFBotSlack, false)
	out.FFBotTeams = mergeBool(db.FFBotTeams, false)
	out.FFBotDiscord = mergeBool(db.FFBotDiscord, false)
	out.FFCalendarFeeds = mergeBool(db.FFCalendarFeeds, true)
	out.FFRedisCache = mergeBool(db.FFRedisCache, false)
	out.SpeechToTextEnabled = mergeBool(db.SpeechToTextEnabled, false)

	// COLLAPSE accommodations audit into the engine master (always audit when engine runs).
	accommodations := mergeBool(db.AccommodationsEngineEnabled, false)
	out.AccommodationsEngineEnabled = accommodations
	out.FFAccommodationsEngine = accommodations

	// COLLAPSE ReadAloud pair into one gate.
	readAloud := mergeBool(db.ReadAloudEnabled, false) || mergeBool(db.FFReadAloud, false)
	out.ReadAloudEnabled = readAloud
	out.FFReadAloud = readAloud

	out.TranslationMemoryEnabled = mergeBool(db.TranslationMemoryEnabled, false)
	out.ReportExportEnabled = mergeBool(db.ReportExportEnabled, true)
	out.XAPIEmissionEnabled = mergeBool(db.XAPIEmissionEnabled, false)
	out.LRSAnonymizeActors = mergeBool(db.LRSAnonymizeActors, false)
	out.FERPAWorkflowEnabled = mergeBool(db.FERPAWorkflowEnabled, false)
	out.CoppaWorkflowEnabled = mergeBool(db.CoppaWorkflowEnabled, false)
	out.GDPRModuleEnabled = mergeBool(db.GDPRModuleEnabled, false)
	out.CCPAModuleEnabled = mergeBool(db.CCPAModuleEnabled, false)
	out.DPAPortalEnabled = mergeBool(db.DPAPortalEnabled, false)
	out.StatePrivacyEnabled = mergeBool(db.StatePrivacyEnabled, false)
	out.SOC2ModuleEnabled = mergeBool(db.SOC2ModuleEnabled, false)
	out.IsoIsmsEnabled = mergeBool(db.IsoIsmsEnabled, false)
	out.AdminAuditLogEnabled = mergeBool(db.AdminAuditLogEnabled, def.AdminAuditLogEnabled)
	out.AdminConsoleEnabled = mergeBool(db.AdminConsoleEnabled, true)
	out.ImpersonationEnabled = mergeBool(db.ImpersonationEnabled, false)
	out.BulkCsvImportEnabled = mergeBool(db.BulkCsvImportEnabled, false)
	out.AdminSearchEnabled = mergeBool(db.AdminSearchEnabled, true)
	out.EmailTemplateEditorEnabled = mergeBool(db.EmailTemplateEditorEnabled, false)
	out.MaintenanceBannerEnabled = mergeBool(db.MaintenanceBannerEnabled, true)
	out.CustomFieldsEnabled = mergeBool(db.CustomFieldsEnabled, false)
	out.SeatManagementEnabled = mergeBool(db.SeatManagementEnabled, false)
	out.DataResidencyEnabled = mergeBool(db.DataResidencyEnabled, false)
	out.AiDisclosureEnabled = mergeBool(db.AiDisclosureEnabled, def.AiDisclosureEnabled)
	// Temporary rollout gates — owner + removal target (docs/completed/flags.md):
	// RTLEnabled: i18n — remove after RTL audit; target 2026-Q4.
	// FFReadingPreferences: a11y — flip default on after QA sign-off; target 2026-Q3.
	out.RTLEnabled = mergeBool(db.RTLEnabled, false)
	out.SecurityDisclosureModuleEnabled = mergeBool(db.SecurityDisclosureModuleEnabled, false)
	out.BackupModuleEnabled = mergeBool(db.BackupModuleEnabled, false)
	out.FFReadingPreferences = mergeBool(db.FFReadingPreferences, false)
	out.FFClassroomSignals = mergeBool(db.FFClassroomSignals, false)
	out.FFLibraryIntegration = mergeBool(db.FFLibraryIntegration, false)

	// Adaptive-learning platform gates (previously env-only service flags).
	out.DiagnosticAssessmentsEnabled = mergeBool(db.DiagnosticAssessmentsEnabled, false)
	out.SRSPracticeEnabled = mergeBool(db.SRSPracticeEnabled, false)
	out.IRTCatModeEnabled = mergeBool(db.IRTCatModeEnabled, false)
	out.AdaptiveLearnerModelEnabled = mergeBool(db.AdaptiveLearnerModelEnabled, false)
	out.LearnerModelEMAAlpha = mergeFloat64(db.LearnerModelEMAAlpha, def.LearnerModelEMAAlpha)
}
