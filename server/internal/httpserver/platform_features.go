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
	StudentProgressEnabled      bool `json:"studentProgressEnabled"`
	AtRiskAlertsEnabled         bool `json:"atRiskAlertsEnabled"`
	H5PEnabled                  bool `json:"h5pEnabled"`
	OERLibraryEnabled           bool `json:"oerLibraryEnabled"`
	ItemAnalysisEnabled         bool `json:"itemAnalysisEnabled"`
	OutcomesReportEnabled       bool `json:"outcomesReportEnabled"`
	EngagementTrackingEnabled   bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled       bool `json:"selfReflectionEnabled"`
	InstructorInsightsEnabled   bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled         bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled       bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled         bool `json:"readingLevelEnabled"`
	GraderAgentEnabled          bool `json:"graderAgentEnabled"`
	AltTextEnforcementEnabled   bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement        bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled         bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine      bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled            bool `json:"readAloudEnabled"`
	FFReadAloud                 bool `json:"ffReadAloud"`
	TranslationMemoryEnabled    bool `json:"translationMemoryEnabled"`
	StorageQuotasEnabled        bool `json:"storageQuotasEnabled"`
	AvScanningEnabled           bool `json:"avScanningEnabled"`
	VirtualClassroomEnabled     bool `json:"virtualClassroomEnabled"`
	SessionManagementUIEnabled  bool `json:"sessionManagementUiEnabled"`
	RTLEnabled                  bool `json:"rtlEnabled"`
	VideoCaptionsEnabled        bool `json:"videoCaptionsEnabled"`
	AutoCaptioningEnabled       bool `json:"autoCaptioningEnabled"`
	FFReadingPreferences        bool `json:"ffReadingPreferences"`
	FFHighContrastReducedMotion bool `json:"ffHighContrastReducedMotion"`
	FFParentPortal              bool `json:"ffParentPortal"`
	FFReportCards               bool `json:"ffReportCards"`
	FFLibrary                   bool `json:"ffLibrary"`
	FFBroadcasts                bool `json:"ffBroadcasts"`
	FFClassroomSignals          bool `json:"ffClassroomSignals"`
	FFConferenceScheduling      bool `json:"ffConferenceScheduling"`
	FFDemographics              bool `json:"ffDemographics"`
	FFContentFilterIntegration  bool `json:"ffContentFilterIntegration"`
	FFSISIntegration            bool `json:"ffSisIntegration"`
	FFCatalogIntegration        bool `json:"ffCatalogIntegration"`
	FFEnrollmentStateMachine    bool `json:"ffEnrollmentStateMachine"`
	FFIncompleteGradeWorkflow   bool `json:"ffIncompleteGradeWorkflow"`
	FFUiMode                    bool `json:"ffUiMode"`
	FFGradeSubmission           bool `json:"ffGradeSubmission"`
	FFAcademicCalendar          bool `json:"ffAcademicCalendar"`
	FFPlagiarismChecks          bool `json:"ffPlagiarismChecks"`
	FFCourseEvaluations         bool `json:"ffCourseEvaluations"`
	FFProctoringIntegration     bool `json:"ffProctoringIntegration"`
	FFCoCurricularTranscript    bool `json:"ffCoCurricularTranscript"`
	FFLibraryIntegration        bool `json:"ffLibraryIntegration"`
	FFBookstoreIntegration      bool `json:"ffBookstoreIntegration"`
	FFEportfolio                bool `json:"ffEportfolio"`
	FFTranscripts               bool `json:"ffTranscripts"`
	FFAdvisingIntegration       bool `json:"ffAdvisingIntegration"`
	FFResearchConsent           bool `json:"ffResearchConsent"`
	FFAccessibilityIntake       bool `json:"ffAccessibilityIntake"`
	FFCEUTracking               bool `json:"ffCeuTracking"`
	FFConsortiumSharing         bool `json:"ffConsortiumSharing"`
	FFSelfPacedMode             bool `json:"ffSelfPacedMode"`
	FFPublicCatalog             bool `json:"ffPublicCatalog"`
	FFPublicAPI                 bool `json:"ffPublicApi"`
	FFStripeBilling             bool `json:"ffStripeBilling"`
	FFRevenueShare              bool `json:"ffRevenueShare"`
	FFLearningPaths             bool `json:"ffLearningPaths"`
	FFCompletionCredentials     bool `json:"ffCompletionCredentials"`
	FFCourseReviews             bool `json:"ffCourseReviews"`
	FFGamification              bool `json:"ffGamification"`
	FFOnboardingFlow            bool `json:"ffOnboardingFlow"`
	FFStudyReminders            bool `json:"ffStudyReminders"`
	FFAIStudyBuddy              bool `json:"ffAiStudyBuddy"`
	FFAPITokens                 bool `json:"ffApiTokens"`

	AiDisclosureEnabled  bool `json:"aiDisclosureEnabled"`
	OpenRouterConfigured bool `json:"openRouterConfigured"`
	RagNotebookEnabled   bool `json:"ragNotebookEnabled"`
	AiStudyBuddyEnabled  bool `json:"aiStudyBuddyEnabled"`

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
		StudentProgressEnabled:      cfg.StudentProgressEnabled,
		AtRiskAlertsEnabled:         cfg.AtRiskAlertsEnabled,
		H5PEnabled:                  cfg.H5PEnabled,
		OERLibraryEnabled:           cfg.OERLibraryEnabled,
		ItemAnalysisEnabled:         cfg.ItemAnalysisEnabled,
		EngagementTrackingEnabled:   cfg.EngagementTrackingEnabled,
		SelfReflectionEnabled:       cfg.SelfReflectionEnabled,
		OutcomesReportEnabled:       cfg.OutcomesReportEnabled,
		InstructorInsightsEnabled:   cfg.InstructorInsightsEnabled,
		XAPIEmissionEnabled:         cfg.XAPIEmissionEnabled,
		EquationEditorEnabled:       cfg.EquationEditorEnabled,
		ReadingLevelEnabled:         cfg.ReadingLevelEnabled,
		GraderAgentEnabled:          cfg.GraderAgentEnabled,
		AltTextEnforcementEnabled:   cfg.AltTextEnforcementEnabled,
		FFAltTextEnforcement:        cfg.FFAltTextEnforcement,
		SpeechToTextEnabled:         cfg.SpeechToTextEnabled,
		AccommodationsEngineEnabled: cfg.AccommodationsEngineEnabled,
		FFAccommodationsEngine:      cfg.FFAccommodationsEngine,
		ReadAloudEnabled:            cfg.ReadAloudEnabled,
		FFReadAloud:                 cfg.FFReadAloud,
		TranslationMemoryEnabled:    cfg.TranslationMemoryEnabled,
		StorageQuotasEnabled:        cfg.StorageQuotasEnabled,
		AvScanningEnabled:           cfg.AvScanningEnabled,
		VirtualClassroomEnabled:     cfg.VirtualClassroomEnabled,
		SessionManagementUIEnabled:  cfg.SessionManagementUIEnabled,
		RTLEnabled:                  cfg.RTLEnabled,
		VideoCaptionsEnabled:        cfg.VideoCaptionsEnabled,
		AutoCaptioningEnabled:       cfg.AutoCaptioningEnabled,
		FFReadingPreferences:        cfg.FFReadingPreferences,
		FFHighContrastReducedMotion: cfg.FFHighContrastReducedMotion,
		FFParentPortal:              cfg.FFParentPortal,
		FFReportCards:               cfg.FFReportCards,
		FFLibrary:                   cfg.FFLibrary,
		FFBroadcasts:                cfg.FFBroadcasts,
		FFClassroomSignals:          cfg.FFClassroomSignals,
		FFConferenceScheduling:      cfg.FFConferenceScheduling,
		FFDemographics:              cfg.FFDemographics,
		FFContentFilterIntegration:  cfg.FFContentFilterIntegration,
		FFSISIntegration:            cfg.FFSISIntegration,
		FFCatalogIntegration:        cfg.FFCatalogIntegration,
		FFEnrollmentStateMachine:    cfg.FFEnrollmentStateMachine,
		FFIncompleteGradeWorkflow:   cfg.FFIncompleteGradeWorkflow,
		FFUiMode:                    cfg.FFUiMode,
		FFGradeSubmission:           cfg.FFGradeSubmission,
		FFAcademicCalendar:          cfg.FFAcademicCalendar,
		FFPlagiarismChecks:          cfg.FFPlagiarismChecks,
		FFCourseEvaluations:         cfg.FFCourseEvaluations,
		FFProctoringIntegration:     cfg.FFProctoringIntegration,
		FFCoCurricularTranscript:    cfg.FFCoCurricularTranscript,
		FFLibraryIntegration:        cfg.FFLibraryIntegration,
		FFBookstoreIntegration:      cfg.FFBookstoreIntegration,
		FFEportfolio:                cfg.FFEportfolio,
		FFTranscripts:               cfg.FFTranscripts,
		FFAdvisingIntegration:       cfg.FFAdvisingIntegration,
		FFResearchConsent:           cfg.FFResearchConsent,
		FFAccessibilityIntake:       cfg.FFAccessibilityIntake,
		FFCEUTracking:               cfg.FFCEUTracking,
		FFConsortiumSharing:         cfg.FFConsortiumSharing,
		FFSelfPacedMode:             cfg.FFSelfPacedMode,
		FFPublicCatalog:             cfg.FFPublicCatalog,
		FFPublicAPI:                 cfg.FFPublicAPI,
		FFStripeBilling:             cfg.FFStripeBilling,
		FFRevenueShare:              cfg.FFRevenueShare,
		FFLearningPaths:             cfg.FFLearningPaths,
		FFCompletionCredentials:     cfg.FFCompletionCredentials,
		FFCourseReviews:             cfg.FFCourseReviews,
		FFGamification:              cfg.FFGamification,
		FFOnboardingFlow:            cfg.FFOnboardingFlow,
		FFStudyReminders:            cfg.FFStudyReminders,
		FFAIStudyBuddy:              cfg.FFAIStudyBuddy,
		FFAPITokens:                 cfg.FFAPITokens,

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
	}
}

func (d Deps) effectiveRagNotebookEnabled(ctx context.Context, userID uuid.UUID) bool {
	cfg := d.effectiveConfig()
	if !cfg.AiDisclosureEnabled || d.openRouterClient() == nil {
		return false
	}
	if d.Pool == nil {
		return true
	}
	orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		return true
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
	if !cfg.FFAIStudyBuddy || !cfg.AiDisclosureEnabled || d.openRouterClient() == nil {
		return false
	}
	if d.Pool == nil {
		return true
	}
	orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		return true
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
		out.OpenRouterConfigured = d.openRouterClient() != nil
		out.RagNotebookEnabled = d.effectiveRagNotebookEnabled(r.Context(), userID)
		out.AiStudyBuddyEnabled = d.effectiveAIStudyBuddyEnabled(r.Context(), userID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
