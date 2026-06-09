package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lextures/lextures/server/internal/config"
)

type platformFeaturesJSON struct {
	StudentProgressEnabled     bool `json:"studentProgressEnabled"`
	AtRiskAlertsEnabled        bool `json:"atRiskAlertsEnabled"`
	H5PEnabled                 bool `json:"h5pEnabled"`
	OERLibraryEnabled          bool `json:"oerLibraryEnabled"`
	ItemAnalysisEnabled        bool `json:"itemAnalysisEnabled"`
	OutcomesReportEnabled      bool `json:"outcomesReportEnabled"`
	EngagementTrackingEnabled  bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled      bool `json:"selfReflectionEnabled"`
	InstructorInsightsEnabled  bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled        bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled      bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled        bool `json:"readingLevelEnabled"`
	AltTextEnforcementEnabled  bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement       bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled         bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine      bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled            bool `json:"readAloudEnabled"`
	FFReadAloud                 bool `json:"ffReadAloud"`
	TranslationMemoryEnabled    bool `json:"translationMemoryEnabled"`
	StorageQuotasEnabled       bool `json:"storageQuotasEnabled"`
	AvScanningEnabled          bool `json:"avScanningEnabled"`
	VirtualClassroomEnabled    bool `json:"virtualClassroomEnabled"`
	SessionManagementUIEnabled bool `json:"sessionManagementUiEnabled"`
	RTLEnabled                 bool `json:"rtlEnabled"`
	VideoCaptionsEnabled       bool `json:"videoCaptionsEnabled"`
	AutoCaptioningEnabled      bool `json:"autoCaptioningEnabled"`
	FFReadingPreferences            bool `json:"ffReadingPreferences"`
	FFHighContrastReducedMotion     bool `json:"ffHighContrastReducedMotion"`
	FFParentPortal                  bool `json:"ffParentPortal"`
	FFReportCards                   bool `json:"ffReportCards"`
	FFLibrary                       bool `json:"ffLibrary"`
	FFBroadcasts                    bool `json:"ffBroadcasts"`
	FFClassroomSignals              bool `json:"ffClassroomSignals"`
	FFConferenceScheduling          bool `json:"ffConferenceScheduling"`
	FFDemographics                  bool `json:"ffDemographics"`
	FFContentFilterIntegration      bool `json:"ffContentFilterIntegration"`
	FFSISIntegration                bool `json:"ffSisIntegration"`
	FFCatalogIntegration            bool `json:"ffCatalogIntegration"`
	FFEnrollmentStateMachine        bool `json:"ffEnrollmentStateMachine"`
	FFIncompleteGradeWorkflow       bool `json:"ffIncompleteGradeWorkflow"`
	FFUiMode                        bool `json:"ffUiMode"`
	FFGradeSubmission               bool `json:"ffGradeSubmission"`
	FFAcademicCalendar              bool `json:"ffAcademicCalendar"`
	FFPlagiarismChecks              bool `json:"ffPlagiarismChecks"`
	FFCourseEvaluations             bool `json:"ffCourseEvaluations"`
	FFProctoringIntegration         bool `json:"ffProctoringIntegration"`
}

func platformFeaturesFromConfig(cfg config.Config) platformFeaturesJSON {
	return platformFeaturesJSON{
		StudentProgressEnabled:     cfg.StudentProgressEnabled,
		AtRiskAlertsEnabled:        cfg.AtRiskAlertsEnabled,
		H5PEnabled:                 cfg.H5PEnabled,
		OERLibraryEnabled:          cfg.OERLibraryEnabled,
		ItemAnalysisEnabled:        cfg.ItemAnalysisEnabled,
		EngagementTrackingEnabled:  cfg.EngagementTrackingEnabled,
		SelfReflectionEnabled:      cfg.SelfReflectionEnabled,
		OutcomesReportEnabled:      cfg.OutcomesReportEnabled,
		InstructorInsightsEnabled:  cfg.InstructorInsightsEnabled,
		XAPIEmissionEnabled:        cfg.XAPIEmissionEnabled,
		EquationEditorEnabled:      cfg.EquationEditorEnabled,
		ReadingLevelEnabled:        cfg.ReadingLevelEnabled,
		AltTextEnforcementEnabled:  cfg.AltTextEnforcementEnabled,
		FFAltTextEnforcement:       cfg.FFAltTextEnforcement,
		SpeechToTextEnabled:         cfg.SpeechToTextEnabled,
		AccommodationsEngineEnabled: cfg.AccommodationsEngineEnabled,
		FFAccommodationsEngine:      cfg.FFAccommodationsEngine,
		ReadAloudEnabled:            cfg.ReadAloudEnabled,
		FFReadAloud:                 cfg.FFReadAloud,
		TranslationMemoryEnabled:    cfg.TranslationMemoryEnabled,
		StorageQuotasEnabled:       cfg.StorageQuotasEnabled,
		AvScanningEnabled:          cfg.AvScanningEnabled,
		VirtualClassroomEnabled:    cfg.VirtualClassroomEnabled,
		SessionManagementUIEnabled: cfg.SessionManagementUIEnabled,
		RTLEnabled:                 cfg.RTLEnabled,
		VideoCaptionsEnabled:       cfg.VideoCaptionsEnabled,
		AutoCaptioningEnabled:      cfg.AutoCaptioningEnabled,
		FFReadingPreferences:        cfg.FFReadingPreferences,
		FFHighContrastReducedMotion: cfg.FFHighContrastReducedMotion,
		FFParentPortal:              cfg.FFParentPortal,
		FFReportCards:               cfg.FFReportCards,
		FFLibrary:                   cfg.FFLibrary,
		FFBroadcasts:                cfg.FFBroadcasts,
		FFClassroomSignals:          cfg.FFClassroomSignals,
		FFConferenceScheduling:        cfg.FFConferenceScheduling,
		FFDemographics:                cfg.FFDemographics,
		FFContentFilterIntegration:    cfg.FFContentFilterIntegration,
		FFSISIntegration:              cfg.FFSISIntegration,
		FFCatalogIntegration:          cfg.FFCatalogIntegration,
		FFEnrollmentStateMachine:      cfg.FFEnrollmentStateMachine,
		FFIncompleteGradeWorkflow:     cfg.FFIncompleteGradeWorkflow,
		FFUiMode:                    cfg.FFUiMode,
		FFGradeSubmission:           cfg.FFGradeSubmission,
		FFAcademicCalendar:          cfg.FFAcademicCalendar,
		FFPlagiarismChecks:          cfg.FFPlagiarismChecks,
		FFCourseEvaluations:         cfg.FFCourseEvaluations,
		FFProctoringIntegration:     cfg.FFProctoringIntegration,
	}
}

// handleGetPlatformFeatures is GET /api/v1/platform/features (authenticated; read-only effective flags).
func (d Deps) handleGetPlatformFeatures() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		out := platformFeaturesFromConfig(d.effectiveConfig())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
