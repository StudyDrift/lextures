package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
	aidisclosurerepo "github.com/lextures/lextures/server/internal/repos/aidisclosure"
	"github.com/lextures/lextures/server/internal/repos/organization"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
)

type platformFeaturesJSON struct {
	StudentProgressEnabled             bool `json:"studentProgressEnabled"`
	AtRiskAlertsEnabled                bool `json:"atRiskAlertsEnabled"`
	H5PEnabled                         bool `json:"h5pEnabled"`
	ScormIngestionEnabled              bool `json:"scormIngestionEnabled"`
	OERLibraryEnabled                  bool `json:"oerLibraryEnabled"`
	ItemAnalysisEnabled                bool `json:"itemAnalysisEnabled"`
	OutcomesReportEnabled              bool `json:"outcomesReportEnabled"`
	EngagementTrackingEnabled          bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled              bool `json:"selfReflectionEnabled"`
	LearnerProfileEnabled              bool `json:"learnerProfileEnabled"`
	IntroCourseEnabled                 bool `json:"introCourseEnabled"`
	InstructorInsightsEnabled          bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled                bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled              bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled                bool `json:"readingLevelEnabled"`
	GraderAgentEnabled                 bool `json:"graderAgentEnabled"`
	GraderAgentReviewInboxEnabled      bool `json:"graderAgentReviewInboxEnabled"`
	GraderAgentSuggestModeEnabled      bool `json:"graderAgentSuggestModeEnabled"`
	GraderAgentTextEntryGradingEnabled bool `json:"graderAgentTextEntryGradingEnabled"`
	GraderAgentVisionGradingEnabled    bool `json:"graderAgentVisionGradingEnabled"`
	GraderAgentRunFiltersEnabled       bool `json:"graderAgentRunFiltersEnabled"`
	GraderAgentCostEstimateEnabled     bool `json:"graderAgentCostEstimateEnabled"`
	GraderAgentCancelRunEnabled        bool `json:"graderAgentCancelRunEnabled"`
	CodeExecutionEnabled               bool `json:"codeExecutionEnabled"`
	AltTextEnforcementEnabled          bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement               bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled                bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled        bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine             bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled                   bool `json:"readAloudEnabled"`
	FFReadAloud                        bool `json:"ffReadAloud"`
	TranslationMemoryEnabled           bool `json:"translationMemoryEnabled"`
	StorageQuotasEnabled               bool `json:"storageQuotasEnabled"`
	AvScanningEnabled                  bool `json:"avScanningEnabled"`
	VirtualClassroomEnabled            bool `json:"virtualClassroomEnabled"`
	SessionManagementUIEnabled         bool `json:"sessionManagementUiEnabled"`
	RTLEnabled                         bool `json:"rtlEnabled"`
	VideoCaptionsEnabled               bool `json:"videoCaptionsEnabled"`
	AutoCaptioningEnabled              bool `json:"autoCaptioningEnabled"`
	FFReadingPreferences               bool `json:"ffReadingPreferences"`
	FFHighContrastReducedMotion        bool `json:"ffHighContrastReducedMotion"`
	FFMotionNavigation                 bool `json:"ffMotionNavigation"`
	FFMotionReveal                     bool `json:"ffMotionReveal"`
	FFMotionLists                      bool `json:"ffMotionLists"`
	FFMotionOverlays                   bool `json:"ffMotionOverlays"`
	FFMotionControls                   bool `json:"ffMotionControls"`
	FFMotionDelight                    bool `json:"ffMotionDelight"`
	FFMobileCreateCourse               bool `json:"ffMobileCreateCourse"`
	FFMobileCourseCreateV2             bool `json:"ffMobileCourseCreateV2"`
	FFMobileCanvasImport               bool `json:"ffMobileCanvasImport"`
	FFMobileAdminConsole               bool `json:"ffMobileAdminConsole"`
	FFMobileEnrollmentAdd              bool `json:"ffMobileEnrollmentAdd"`
	FFMobileLiveQuiz                   bool `json:"ffMobileLiveQuiz"`
	FFMobileWhiteboardEdit             bool `json:"ffMobileWhiteboardEdit"`
	FFMobileMarketplacePurchase        bool `json:"ffMobileMarketplacePurchase"`
	FFMobileBoardsAdvanced             bool `json:"ffMobileBoardsAdvanced"`
	FFParentPortal                     bool `json:"ffParentPortal"`
	FFParentPortalV2                   bool `json:"ffParentPortalV2"`
	FFReportCards                      bool `json:"ffReportCards"`
	FFLibrary                          bool `json:"ffLibrary"`
	FFBroadcasts                       bool `json:"ffBroadcasts"`
	FFClassroomSignals                 bool `json:"ffClassroomSignals"`
	FFConferenceScheduling             bool `json:"ffConferenceScheduling"`
	FFDemographics                     bool `json:"ffDemographics"`
	FFContentFilterIntegration         bool `json:"ffContentFilterIntegration"`
	FFSISIntegration                   bool `json:"ffSisIntegration"`
	FFCatalogIntegration               bool `json:"ffCatalogIntegration"`
	FFEnrollmentStateMachine           bool `json:"ffEnrollmentStateMachine"`
	FFIncompleteGradeWorkflow          bool `json:"ffIncompleteGradeWorkflow"`
	FFUiMode                           bool `json:"ffUiMode"`
	FFGradeSubmission                  bool `json:"ffGradeSubmission"`
	FFWhatifGrades                     bool `json:"ffWhatifGrades"`
	FFGradeCurving                     bool `json:"ffGradeCurving"`
	FFAcademicCalendar                 bool `json:"ffAcademicCalendar"`
	FFPlagiarismChecks                 bool `json:"ffPlagiarismChecks"`
	FFCourseEvaluations                bool `json:"ffCourseEvaluations"`
	FFProctoringIntegration            bool `json:"ffProctoringIntegration"`
	FFCoCurricularTranscript           bool `json:"ffCoCurricularTranscript"`
	FFLibraryIntegration               bool `json:"ffLibraryIntegration"`
	FFBookstoreIntegration             bool `json:"ffBookstoreIntegration"`
	FFEportfolio                       bool `json:"ffEportfolio"`
	FFTranscripts                      bool `json:"ffTranscripts"`
	FFTranscriptInbound                bool `json:"ffTranscriptInbound"`
	FFDiplomas                         bool `json:"ffDiplomas"`
	FFWebhooks                         bool `json:"ffWebhooks"`
	FFZapierConnector                  bool `json:"ffZapierConnector"`
	FFAdvisingIntegration              bool `json:"ffAdvisingIntegration"`
	FFResearchConsent                  bool `json:"ffResearchConsent"`
	FFAccessibilityIntake              bool `json:"ffAccessibilityIntake"`
	FFCEUTracking                      bool `json:"ffCeuTracking"`
	FFConsortiumSharing                bool `json:"ffConsortiumSharing"`
	FFSelfPacedMode                    bool `json:"ffSelfPacedMode"`
	FFPublicCatalog                    bool `json:"ffPublicCatalog"`
	FFCourseMarketplace                bool `json:"ffCourseMarketplace"`
	FFFeedback                         bool `json:"ffFeedback"`
	FFVisualBoards                     bool `json:"ffVisualBoards"`
	FFBoardsRealtime                   bool `json:"ffBoardsRealtime"`
	FFBoardsExternalSharing            bool `json:"ffBoardsExternalSharing"`
	FFInteractiveQuizzes               bool `json:"ffInteractiveQuizzes"`
	FFIqLiveHosting                    bool `json:"ffIqLiveHosting"`
	FFIqTeamMode                       bool `json:"ffIqTeamMode"`
	FFIqStudentPaced                   bool `json:"ffIqStudentPaced"`
	FFIqHomework                       bool `json:"ffIqHomework"`
	FFIqGradebookPush                  bool `json:"ffIqGradebookPush"`
	FFIqPublicKitCatalog               bool `json:"ffIqPublicKitCatalog"`
	FFIqGuestJoin                      bool `json:"ffIqGuestJoin"`
	FFIqAiGeneration                   bool `json:"ffIqAiGeneration"`
	FFEmailSES                         bool `json:"ffEmailSes"`
	FFPublicAPI                        bool `json:"ffPublicApi"`
	FFStripeBilling                    bool `json:"ffStripeBilling"`
	FFPaymentsEnabled                  bool `json:"ffPaymentsEnabled"`
	FFRevenueShare                     bool `json:"ffRevenueShare"`
	FFTaxCollection                    bool `json:"ffTaxCollection"`
	FFLearningPaths                    bool `json:"ffLearningPaths"`
	FFConditionalRelease               bool `json:"ffConditionalRelease"`
	FFPeerReview                       bool `json:"ffPeerReview"`
	FFCompletionCredentials            bool `json:"ffCompletionCredentials"`
	FFCourseReviews                    bool `json:"ffCourseReviews"`
	FFGamification                     bool `json:"ffGamification"`
	FFCompetencyBadges                 bool `json:"ffCompetencyBadges"`
	BadgesDefaultPublic                bool `json:"badgesDefaultPublic"`
	FFOnboardingFlow                   bool `json:"ffOnboardingFlow"`
	FFStudyReminders                   bool `json:"ffStudyReminders"`
	FFAIStudyBuddy                     bool `json:"ffAiStudyBuddy"`
	FFLessonGenerator                  bool `json:"ffLessonGenerator"`
	FFPersistentTutor                  bool `json:"ffPersistentTutor"`
	FFAPITokens                        bool `json:"ffApiTokens"`
	FFBotSlack                         bool `json:"ffBotSlack"`
	FFBotTeams                         bool `json:"ffBotTeams"`
	FFBotDiscord                       bool `json:"ffBotDiscord"`
	FFCalendarFeeds                    bool `json:"ffCalendarFeeds"`

	AiDisclosureEnabled        bool `json:"aiDisclosureEnabled"`
	AdminConsoleEnabled        bool `json:"adminConsoleEnabled"`
	AdminAuditLogEnabled       bool `json:"adminAuditLogEnabled"`
	ImpersonationEnabled       bool `json:"impersonationEnabled"`
	BulkCsvImportEnabled       bool `json:"bulkCsvImportEnabled"`
	AdminSearchEnabled         bool `json:"adminSearchEnabled"`
	EmailTemplateEditorEnabled bool `json:"emailTemplateEditorEnabled"`
	MaintenanceBannerEnabled   bool `json:"maintenanceBannerEnabled"`
	CustomFieldsEnabled        bool `json:"customFieldsEnabled"`
	SeatManagementEnabled      bool `json:"seatManagementEnabled"`
	// OpenRouterConfigured is deprecated (AP.9); alias of AIConfigured for one minor release. Prefer aiConfigured.
	OpenRouterConfigured         bool     `json:"openRouterConfigured"` // Deprecated: use aiConfigured
	AIConfigured                 bool     `json:"aiConfigured"`
	AiProvidersConfigured        []string `json:"aiProvidersConfigured"`
	AiProviderAbstractionEnabled bool     `json:"aiProviderAbstractionEnabled"`
	RagNotebookEnabled           bool     `json:"ragNotebookEnabled"`
	AiStudyBuddyEnabled          bool     `json:"aiStudyBuddyEnabled"`

	LRSAnonymizeActors           bool    `json:"lrsAnonymizeActors"`
	FERPAWorkflowEnabled         bool    `json:"ferpaWorkflowEnabled"`
	GDPRModuleEnabled            bool    `json:"gdprModuleEnabled"`
	DPAPortalEnabled             bool    `json:"dpaPortalEnabled"`
	SOC2ModuleEnabled            bool    `json:"soc2ModuleEnabled"`
	DiagnosticAssessmentsEnabled bool    `json:"diagnosticAssessmentsEnabled"`
	SRSPracticeEnabled           bool    `json:"srsPracticeEnabled"`
	IRTCatModeEnabled            bool    `json:"irtCatModeEnabled"`
	AdaptiveLearnerModelEnabled  bool    `json:"adaptiveLearnerModelEnabled"`
	LearnerModelEMAAlpha         float64 `json:"learnerModelEmaAlpha"`
}

func platformFeaturesFromConfig(cfg config.Config) platformFeaturesJSON {
	return platformFeaturesJSON{
		StudentProgressEnabled:             cfg.StudentProgressEnabled,
		AtRiskAlertsEnabled:                cfg.AtRiskAlertsEnabled,
		H5PEnabled:                         cfg.H5PEnabled,
		ScormIngestionEnabled:              cfg.ScormIngestionEnabled,
		OERLibraryEnabled:                  cfg.OERLibraryEnabled,
		ItemAnalysisEnabled:                cfg.ItemAnalysisEnabled,
		EngagementTrackingEnabled:          cfg.EngagementTrackingEnabled,
		SelfReflectionEnabled:              cfg.SelfReflectionEnabled,
		LearnerProfileEnabled:              cfg.LearnerProfileEnabled,
		IntroCourseEnabled:                 cfg.IntroCourseEnabled,
		OutcomesReportEnabled:              cfg.OutcomesReportEnabled,
		InstructorInsightsEnabled:          cfg.InstructorInsightsEnabled,
		XAPIEmissionEnabled:                cfg.XAPIEmissionEnabled,
		EquationEditorEnabled:              cfg.EquationEditorEnabled,
		ReadingLevelEnabled:                cfg.ReadingLevelEnabled,
		GraderAgentEnabled:                 cfg.GraderAgentEnabled,
		GraderAgentReviewInboxEnabled:      cfg.GraderAgentReviewInboxEnabled,
		GraderAgentSuggestModeEnabled:      cfg.GraderAgentSuggestModeEnabled,
		GraderAgentTextEntryGradingEnabled: cfg.GraderAgentTextEntryGradingEnabled,
		GraderAgentVisionGradingEnabled:    cfg.GraderAgentVisionGradingEnabled,
		GraderAgentRunFiltersEnabled:       cfg.GraderAgentRunFiltersEnabled,
		GraderAgentCostEstimateEnabled:     cfg.GraderAgentCostEstimateEnabled,
		GraderAgentCancelRunEnabled:        cfg.GraderAgentCancelRunEnabled,
		CodeExecutionEnabled:               cfg.CodeExecutionEnabled,
		AltTextEnforcementEnabled:          cfg.AltTextEnforcementEnabled,
		FFAltTextEnforcement:               cfg.FFAltTextEnforcement,
		SpeechToTextEnabled:                cfg.SpeechToTextEnabled,
		AccommodationsEngineEnabled:        cfg.AccommodationsEngineEnabled,
		FFAccommodationsEngine:             cfg.FFAccommodationsEngine,
		ReadAloudEnabled:                   cfg.ReadAloudEnabled,
		FFReadAloud:                        cfg.FFReadAloud,
		TranslationMemoryEnabled:           cfg.TranslationMemoryEnabled,
		StorageQuotasEnabled:               cfg.StorageQuotasEnabled,
		AvScanningEnabled:                  cfg.AvScanningEnabled,
		VirtualClassroomEnabled:            cfg.VirtualClassroomEnabled,
		SessionManagementUIEnabled:         cfg.SessionManagementUIEnabled,
		RTLEnabled:                         cfg.RTLEnabled,
		VideoCaptionsEnabled:               cfg.VideoCaptionsEnabled,
		AutoCaptioningEnabled:              cfg.AutoCaptioningEnabled,
		FFReadingPreferences:               cfg.FFReadingPreferences,
		FFHighContrastReducedMotion:        cfg.FFHighContrastReducedMotion,
		FFMotionNavigation:                 cfg.FFMotionNavigation,
		FFMotionReveal:                     cfg.FFMotionReveal,
		FFMotionLists:                      cfg.FFMotionLists,
		FFMotionOverlays:                   cfg.FFMotionOverlays,
		FFMotionControls:                   cfg.FFMotionControls,
		FFMotionDelight:                    cfg.FFMotionDelight,
		FFMobileCreateCourse:               cfg.FFMobileCreateCourse,
		FFMobileCourseCreateV2:             cfg.FFMobileCourseCreateV2,
		FFMobileCanvasImport:               cfg.FFMobileCanvasImport,
		FFMobileAdminConsole:               cfg.FFMobileAdminConsole,
		FFMobileEnrollmentAdd:              cfg.FFMobileEnrollmentAdd,
		FFMobileLiveQuiz:                   cfg.FFMobileLiveQuiz,
		FFMobileWhiteboardEdit:             cfg.FFMobileWhiteboardEdit,
		FFMobileMarketplacePurchase:        cfg.FFMobileMarketplacePurchase,
		FFMobileBoardsAdvanced:             cfg.FFMobileBoardsAdvanced,
		FFParentPortal:                     cfg.FFParentPortal,
		FFParentPortalV2:                   cfg.FFParentPortalV2,
		FFReportCards:                      cfg.FFReportCards,
		FFLibrary:                          cfg.FFLibrary,
		FFBroadcasts:                       cfg.FFBroadcasts,
		FFClassroomSignals:                 cfg.FFClassroomSignals,
		FFConferenceScheduling:             cfg.FFConferenceScheduling,
		FFDemographics:                     cfg.FFDemographics,
		FFContentFilterIntegration:         cfg.FFContentFilterIntegration,
		FFSISIntegration:                   cfg.FFSISIntegration,
		FFCatalogIntegration:               cfg.FFCatalogIntegration,
		FFEnrollmentStateMachine:           cfg.FFEnrollmentStateMachine,
		FFIncompleteGradeWorkflow:          cfg.FFIncompleteGradeWorkflow,
		FFUiMode:                           cfg.FFUiMode,
		FFGradeSubmission:                  cfg.FFGradeSubmission,
		FFWhatifGrades:                     cfg.FFWhatifGrades,
		FFGradeCurving:                     cfg.FFGradeCurving,
		FFAcademicCalendar:                 cfg.FFAcademicCalendar,
		FFPlagiarismChecks:                 cfg.FFPlagiarismChecks,
		FFCourseEvaluations:                cfg.FFCourseEvaluations,
		FFProctoringIntegration:            cfg.FFProctoringIntegration,
		FFCoCurricularTranscript:           cfg.FFCoCurricularTranscript,
		FFLibraryIntegration:               cfg.FFLibraryIntegration,
		FFBookstoreIntegration:             cfg.FFBookstoreIntegration,
		FFEportfolio:                       cfg.FFEportfolio,
		FFTranscripts:                      cfg.FFTranscripts,
		FFTranscriptInbound:                cfg.FFTranscriptInbound,
		FFDiplomas:                         cfg.FFDiplomas,
		FFWebhooks:                         cfg.FFWebhooks,
		FFZapierConnector:                  cfg.FFZapierConnector,
		FFAdvisingIntegration:              cfg.FFAdvisingIntegration,
		FFResearchConsent:                  cfg.FFResearchConsent,
		FFAccessibilityIntake:              cfg.FFAccessibilityIntake,
		FFCEUTracking:                      cfg.FFCEUTracking,
		FFConsortiumSharing:                cfg.FFConsortiumSharing,
		FFSelfPacedMode:                    cfg.FFSelfPacedMode,
		FFPublicCatalog:                    cfg.FFPublicCatalog,
		FFCourseMarketplace:                cfg.FFCourseMarketplace,
		FFFeedback:                         cfg.FFFeedback,
		FFVisualBoards:                     cfg.FFVisualBoards,
		FFBoardsRealtime:                   cfg.FFBoardsRealtime,
		FFBoardsExternalSharing:            cfg.FFBoardsExternalSharing,
		FFInteractiveQuizzes:               cfg.FFInteractiveQuizzes,
		FFIqLiveHosting:                    cfg.FFIqLiveHosting,
		FFIqTeamMode:                       cfg.FFIqTeamMode,
		FFIqStudentPaced:                   cfg.FFIqStudentPaced,
		FFIqHomework:                       cfg.FFIqHomework,
		FFIqGradebookPush:                  cfg.FFIqGradebookPush,
		FFIqPublicKitCatalog:               cfg.FFIqPublicKitCatalog,
		FFIqGuestJoin:                      cfg.FFIqGuestJoin,
		FFIqAiGeneration:                   cfg.FFIqAiGeneration,
		FFEmailSES:                         cfg.FFEmailSES,
		FFPublicAPI:                        cfg.FFPublicAPI,
		FFStripeBilling:                    cfg.FFStripeBilling,
		FFPaymentsEnabled:                  cfg.FFPaymentsEnabled,
		FFRevenueShare:                     cfg.FFRevenueShare,
		FFTaxCollection:                    cfg.FFTaxCollection,
		FFLearningPaths:                    cfg.FFLearningPaths,
		FFConditionalRelease:               cfg.FFConditionalRelease,
		FFPeerReview:                       cfg.FFPeerReview,
		FFCompletionCredentials:            cfg.FFCompletionCredentials,
		FFCourseReviews:                    cfg.FFCourseReviews,
		FFGamification:                     cfg.FFGamification,
		FFCompetencyBadges:                 cfg.FFCompetencyBadges,
		BadgesDefaultPublic:                cfg.BadgesDefaultPublic,
		FFOnboardingFlow:                   cfg.FFOnboardingFlow,
		FFStudyReminders:                   cfg.FFStudyReminders,
		FFAIStudyBuddy:                     cfg.FFAIStudyBuddy,
		FFLessonGenerator:                  cfg.FFLessonGenerator,
		FFPersistentTutor:                  cfg.FFPersistentTutor,
		FFAPITokens:                        cfg.FFAPITokens,
		FFBotSlack:                         cfg.FFBotSlack,
		FFBotTeams:                         cfg.FFBotTeams,
		FFBotDiscord:                       cfg.FFBotDiscord,
		FFCalendarFeeds:                    cfg.FFCalendarFeeds,

		LRSAnonymizeActors:           cfg.LRSAnonymizeActors,
		FERPAWorkflowEnabled:         cfg.FERPAWorkflowEnabled,
		GDPRModuleEnabled:            cfg.GDPRModuleEnabled,
		DPAPortalEnabled:             cfg.DPAPortalEnabled,
		SOC2ModuleEnabled:            cfg.SOC2ModuleEnabled,
		DiagnosticAssessmentsEnabled: cfg.DiagnosticAssessmentsEnabled,
		SRSPracticeEnabled:           cfg.SRSPracticeEnabled,
		IRTCatModeEnabled:            cfg.IRTCatModeEnabled,
		AdaptiveLearnerModelEnabled:  cfg.AdaptiveLearnerModelEnabled,
		LearnerModelEMAAlpha:         cfg.LearnerModelEMAAlpha,
		AdminConsoleEnabled:          cfg.AdminConsoleEnabled,
		AdminAuditLogEnabled:         cfg.AdminAuditLogEnabled,
		ImpersonationEnabled:         cfg.ImpersonationEnabled,
		BulkCsvImportEnabled:         cfg.BulkCsvImportEnabled,
		AdminSearchEnabled:           cfg.AdminSearchEnabled,
		EmailTemplateEditorEnabled:   cfg.EmailTemplateEditorEnabled,
		MaintenanceBannerEnabled:     cfg.MaintenanceBannerEnabled,
		CustomFieldsEnabled:          cfg.CustomFieldsEnabled,
		SeatManagementEnabled:        cfg.SeatManagementEnabled,
	}
}

func (d Deps) effectiveRagNotebookEnabled(ctx context.Context, userID uuid.UUID) bool {
	cfg := d.effectiveConfig()
	if !cfg.AiDisclosureEnabled {
		return false
	}
	if d.Pool == nil {
		return d.aiConfigured(ctx, nil)
	}
	orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		return d.aiConfigured(ctx, nil)
	}
	if !d.aiConfigured(ctx, &orgID) {
		return false
	}
	tc, err := aidisclosurerepo.GetTenantConfig(ctx, d.Pool, orgID)
	if err != nil || tc == nil {
		return true
	}
	if disabled, ok := tc.FeaturesEnabled[aigateway.FeatureRAGNotebook]; ok && !disabled {
		return false
	}
	return true
}

func (d Deps) effectiveAIStudyBuddyEnabled(ctx context.Context, userID uuid.UUID) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFAIStudyBuddy || !cfg.AiDisclosureEnabled {
		return false
	}
	if d.Pool == nil {
		return d.aiConfigured(ctx, nil)
	}
	orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		return d.aiConfigured(ctx, nil)
	}
	if !d.aiConfigured(ctx, &orgID) {
		return false
	}
	tc, err := aidisclosurerepo.GetTenantConfig(ctx, d.Pool, orgID)
	if err != nil || tc == nil {
		return true
	}
	if disabled, ok := tc.FeaturesEnabled[aigateway.FeatureAIStudyBuddy]; ok && !disabled {
		return false
	}
	return true
}

// handleGetPlatformFeatures is GET /api/v1/platform/features (authenticated; read-only effective flags).
func (d Deps) handleGetPlatformFeatures() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		out := platformFeaturesFromConfig(cfg)
		out.AiDisclosureEnabled = cfg.AiDisclosureEnabled
		orgID := d.orgIDPtrForUser(r.Context(), userID)
		out.AIConfigured = d.aiConfigured(r.Context(), orgID)
		out.AiProvidersConfigured = d.aiProvidersConfigured(r.Context(), orgID)
		// OpenRouterConfigured is a deprecated alias of AIConfigured (AP.4 FR-8 / AP.9 FR-6).
		// Dual-read window: ≥1 minor release after GA; remove after clients use aiConfigured only.
		out.OpenRouterConfigured = out.AIConfigured
		out.AiProviderAbstractionEnabled = cfg.AiProviderAbstractionEnabled
		out.RagNotebookEnabled = d.effectiveRagNotebookEnabled(r.Context(), userID)
		out.AiStudyBuddyEnabled = d.effectiveAIStudyBuddyEnabled(r.Context(), userID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
