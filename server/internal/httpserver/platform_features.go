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
	TranslationMemoryEnabled   bool `json:"translationMemoryEnabled"`
	StorageQuotasEnabled       bool `json:"storageQuotasEnabled"`
	AvScanningEnabled          bool `json:"avScanningEnabled"`
	VirtualClassroomEnabled    bool `json:"virtualClassroomEnabled"`
	SessionManagementUIEnabled bool `json:"sessionManagementUiEnabled"`
	RTLEnabled                 bool `json:"rtlEnabled"`
	VideoCaptionsEnabled       bool `json:"videoCaptionsEnabled"`
	AutoCaptioningEnabled      bool `json:"autoCaptioningEnabled"`
	FFReadingPreferences       bool `json:"ffReadingPreferences"`
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
		TranslationMemoryEnabled:   cfg.TranslationMemoryEnabled,
		StorageQuotasEnabled:       cfg.StorageQuotasEnabled,
		AvScanningEnabled:          cfg.AvScanningEnabled,
		VirtualClassroomEnabled:    cfg.VirtualClassroomEnabled,
		SessionManagementUIEnabled: cfg.SessionManagementUIEnabled,
		RTLEnabled:                 cfg.RTLEnabled,
		VideoCaptionsEnabled:       cfg.VideoCaptionsEnabled,
		AutoCaptioningEnabled:      cfg.AutoCaptioningEnabled,
		FFReadingPreferences:       cfg.FFReadingPreferences,
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
