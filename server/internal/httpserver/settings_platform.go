package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
)

const placeholderSecretResponse = "••••••••••••"

func maskSecret(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return placeholderSecretResponse
}

func maskPEMIfSet(pem string) string {
	if strings.TrimSpace(pem) == "" {
		return ""
	}
	return placeholderSecretResponse
}

func smtpPasswordMasked(dbRow *platformconfig.Row, mergedPassword string) string {
	if strings.TrimSpace(mergedPassword) != "" {
		return maskSecret("x")
	}
	if dbRow != nil && len(dbRow.SMTPPasswordCiphertext) > 0 {
		return maskSecret("x")
	}
	return ""
}

type platformSettingsJSON struct {
	OpenRouterAPIKey string `json:"openRouterApiKey"`

	SAMLSSOEnabled      bool   `json:"samlSsoEnabled"`
	SAMLPublicBaseURL   string `json:"samlPublicBaseUrl"`
	SAMLSPEntityID      string `json:"samlSpEntityId"`
	SAMLSPX509PEM       string `json:"samlSpX509Pem"`
	SAMLSPPrivateKeyPEM string `json:"samlSpPrivateKeyPem"`

	AnnotationEnabled           bool `json:"annotationEnabled"`
	FeedbackMediaEnabled        bool `json:"feedbackMediaEnabled"`
	BlindGradingEnabled         bool `json:"blindGradingEnabled"`
	ModeratedGradingEnabled     bool `json:"moderatedGradingEnabled"`
	OriginalityDetectionEnabled bool `json:"originalityDetectionEnabled"`
	OriginalityStubExternal     bool `json:"originalityStubExternal"`
	GradePostingPoliciesEnabled bool `json:"gradePostingPoliciesEnabled"`
	GradebookCSVEnabled         bool `json:"gradebookCsvEnabled"`
	ResubmissionWorkflowEnabled bool `json:"resubmissionWorkflowEnabled"`
	LTIEnabled                  bool `json:"ltiEnabled"`
	OneRosterEnabled            bool `json:"oneRosterEnabled"`
	ScimEnabled                 bool `json:"scimEnabled"`

	OIDCSSOEnabled             bool `json:"oidcSsoEnabled"`
	CleverSSOEnabled           bool `json:"cleverSsoEnabled"`
	ClassLinkSSOEnabled        bool `json:"classlinkSsoEnabled"`
	MagicLinkEnabled           bool `json:"magicLinkEnabled"`
	MagicLinkEnrolledOnly      bool `json:"magicLinkEnrolledOnly"`
	SessionManagementUIEnabled bool `json:"sessionManagementUiEnabled"`
	EmailNotificationsEnabled  bool `json:"emailNotificationsEnabled"`
	PushNotificationsEnabled   bool `json:"pushNotificationsEnabled"`
	VirtualClassroomEnabled    bool `json:"virtualClassroomEnabled"`
	DRMEnabled                 bool `json:"drmEnabled"`
	VideoTranscodingEnabled    bool `json:"videoTranscodingEnabled"`
	AutoCaptioningEnabled      bool `json:"autoCaptioningEnabled"`
	VideoCaptionsEnabled       bool `json:"videoCaptionsEnabled"`
	StorageQuotasEnabled       bool `json:"storageQuotasEnabled"`
	AtRiskAlertsEnabled        bool `json:"atRiskAlertsEnabled"`
	AvScanningEnabled          bool `json:"avScanningEnabled"`
	ClamAVStub                 bool `json:"clamavStub"`
	H5PEnabled                 bool `json:"h5pEnabled"`
	OERLibraryEnabled          bool `json:"oerLibraryEnabled"`
	OERStub                    bool `json:"oerStub"`
	ItemAnalysisEnabled        bool `json:"itemAnalysisEnabled"`
	StudentProgressEnabled     bool `json:"studentProgressEnabled"`
	EngagementTrackingEnabled  bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled      bool `json:"selfReflectionEnabled"`
	OutcomesReportEnabled      bool `json:"outcomesReportEnabled"`
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
	ReportExportEnabled        bool `json:"reportExportEnabled"`
	CoppaWorkflowEnabled       bool `json:"coppaWorkflowEnabled"`
	IsoIsmsEnabled                  bool `json:"isoIsmsEnabled"`
	AdminAuditLogEnabled            bool `json:"adminAuditLogEnabled"`
	DataResidencyEnabled            bool `json:"dataResidencyEnabled"`
	RTLEnabled                      bool `json:"rtlEnabled"`
	SecurityDisclosureModuleEnabled bool `json:"securityDisclosureModuleEnabled"`
	FFParentPortal                  bool `json:"ffParentPortal"`
	FFReportCards                   bool `json:"ffReportCards"`
	FFLibrary                       bool `json:"ffLibrary"`
	FFBroadcasts                    bool `json:"ffBroadcasts"`
	FFConferenceScheduling          bool `json:"ffConferenceScheduling"`
	FFDemographics                  bool `json:"ffDemographics"`
	FFContentFilterIntegration      bool `json:"ffContentFilterIntegration"`
	FFSISIntegration                bool `json:"ffSisIntegration"`
	FFCatalogIntegration            bool `json:"ffCatalogIntegration"`
	FFEnrollmentStateMachine        bool `json:"ffEnrollmentStateMachine"`
	FFGradeSubmission               bool `json:"ffGradeSubmission"`

	MFAEnabled     bool   `json:"mfaEnabled"`
	MFAEnforcement string `json:"mfaEnforcement"`

	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPFrom     string `json:"smtpFrom"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPassword string `json:"smtpPassword"`

	Sources platformSourcesJSON `json:"sources"`
}

type platformSourcesJSON struct {
	OpenRouterAPIKey string `json:"openRouterApiKey"`

	SAMLSSOEnabled      string `json:"samlSsoEnabled"`
	SAMLPublicBaseURL   string `json:"samlPublicBaseUrl"`
	SAMLSPEntityID      string `json:"samlSpEntityId"`
	SAMLSPX509PEM       string `json:"samlSpX509Pem"`
	SAMLSPPrivateKeyPEM string `json:"samlSpPrivateKeyPem"`

	AnnotationEnabled           string `json:"annotationEnabled"`
	FeedbackMediaEnabled        string `json:"feedbackMediaEnabled"`
	BlindGradingEnabled         string `json:"blindGradingEnabled"`
	ModeratedGradingEnabled     string `json:"moderatedGradingEnabled"`
	OriginalityDetectionEnabled string `json:"originalityDetectionEnabled"`
	OriginalityStubExternal     string `json:"originalityStubExternal"`
	GradePostingPoliciesEnabled string `json:"gradePostingPoliciesEnabled"`
	GradebookCSVEnabled         string `json:"gradebookCsvEnabled"`
	ResubmissionWorkflowEnabled string `json:"resubmissionWorkflowEnabled"`
	LTIEnabled                  string `json:"ltiEnabled"`
	OneRosterEnabled            string `json:"oneRosterEnabled"`
	ScimEnabled                 string `json:"scimEnabled"`
	MFAEnabled                  string `json:"mfaEnabled"`
	MFAEnforcement              string `json:"mfaEnforcement"`

	SMTPHost     string `json:"smtpHost"`
	SMTPPort     string `json:"smtpPort"`
	SMTPFrom     string `json:"smtpFrom"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPassword string `json:"smtpPassword"`
}

func src(s platformconfig.Source) string {
	return string(s)
}

// handleGetPlatformSettings is GET /api/v1/settings/platform
func (d Deps) handleGetPlatformSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		ctx := r.Context()
		var dbRow *platformconfig.Row
		var err error
		if d.Pool != nil {
			dbRow, err = platformconfig.Get(ctx, d.Pool)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load platform settings.")
				return
			}
		}
		merged := platformconfig.Merge(d.Config, dbRow)
		sources := platformconfig.ResolveSources(d.Config, dbRow)
		out := platformSettingsJSON{
			OpenRouterAPIKey:            maskSecret(merged.OpenRouterAPIKey),
			SAMLSSOEnabled:              merged.SAMLSSOEnabled,
			SAMLPublicBaseURL:           merged.SAMLPublicBaseURL,
			SAMLSPEntityID:              merged.SAMLSPEntityID,
			SAMLSPX509PEM:               merged.SAMLSPX509PEM,
			SAMLSPPrivateKeyPEM:         maskPEMIfSet(merged.SAMLSPPrivateKeyPEM),
			AnnotationEnabled:           merged.AnnotationEnabled,
			FeedbackMediaEnabled:        merged.FeedbackMediaEnabled,
			BlindGradingEnabled:         merged.BlindGradingEnabled,
			ModeratedGradingEnabled:     merged.ModeratedGradingEnabled,
			OriginalityDetectionEnabled: merged.OriginalityDetectionEnabled,
			OriginalityStubExternal:     merged.OriginalityStubExternal,
			GradePostingPoliciesEnabled: merged.GradePostingPoliciesEnabled,
			GradebookCSVEnabled:         merged.GradebookCSVEnabled,
			ResubmissionWorkflowEnabled: merged.ResubmissionWorkflowEnabled,
			LTIEnabled:                  merged.LTIEnabled,
			OneRosterEnabled:            merged.OneRosterEnabled,
			ScimEnabled:                 merged.ScimEnabled,
			OIDCSSOEnabled:              merged.OIDCSSOEnabled,
			CleverSSOEnabled:            merged.CleverSSOEnabled,
			ClassLinkSSOEnabled:         merged.ClassLinkSSOEnabled,
			MagicLinkEnabled:            merged.MagicLinkEnabled,
			MagicLinkEnrolledOnly:       merged.MagicLinkEnrolledOnly,
			SessionManagementUIEnabled:  merged.SessionManagementUIEnabled,
			EmailNotificationsEnabled:   merged.EmailNotificationsEnabled,
			PushNotificationsEnabled:    merged.PushNotificationsEnabled,
			VirtualClassroomEnabled:     merged.VirtualClassroomEnabled,
			DRMEnabled:                  merged.DRMEnabled,
			VideoTranscodingEnabled:     merged.VideoTranscodingEnabled,
			AutoCaptioningEnabled:       merged.AutoCaptioningEnabled,
			VideoCaptionsEnabled:        merged.VideoCaptionsEnabled,
			StorageQuotasEnabled:        merged.StorageQuotasEnabled,
			AtRiskAlertsEnabled:         merged.AtRiskAlertsEnabled,
			AvScanningEnabled:           merged.AvScanningEnabled,
			ClamAVStub:                  merged.ClamAVStub,
			H5PEnabled:                  merged.H5PEnabled,
			OERLibraryEnabled:           merged.OERLibraryEnabled,
			OERStub:                     merged.OERStub,
			ItemAnalysisEnabled:         merged.ItemAnalysisEnabled,
			StudentProgressEnabled:      merged.StudentProgressEnabled,
			EngagementTrackingEnabled:   merged.EngagementTrackingEnabled,
			SelfReflectionEnabled:       merged.SelfReflectionEnabled,
			OutcomesReportEnabled:       merged.OutcomesReportEnabled,
			InstructorInsightsEnabled:   merged.InstructorInsightsEnabled,
			XAPIEmissionEnabled:         merged.XAPIEmissionEnabled,
			EquationEditorEnabled:       merged.EquationEditorEnabled,
			ReadingLevelEnabled:         merged.ReadingLevelEnabled,
			AltTextEnforcementEnabled:   merged.AltTextEnforcementEnabled,
			FFAltTextEnforcement:        merged.FFAltTextEnforcement,
			SpeechToTextEnabled:          merged.SpeechToTextEnabled,
			AccommodationsEngineEnabled:  merged.AccommodationsEngineEnabled,
			FFAccommodationsEngine:       merged.FFAccommodationsEngine,
			ReadAloudEnabled:             merged.ReadAloudEnabled,
			FFReadAloud:                  merged.FFReadAloud,
			TranslationMemoryEnabled:     merged.TranslationMemoryEnabled,
			ReportExportEnabled:         merged.ReportExportEnabled,
			CoppaWorkflowEnabled:        merged.CoppaWorkflowEnabled,
			IsoIsmsEnabled:                  merged.IsoIsmsEnabled,
			AdminAuditLogEnabled:            merged.AdminAuditLogEnabled,
			DataResidencyEnabled:            merged.DataResidencyEnabled,
			RTLEnabled:                      merged.RTLEnabled,
			SecurityDisclosureModuleEnabled: merged.SecurityDisclosureModuleEnabled,
			FFParentPortal:                  merged.FFParentPortal,
				FFReportCards:                   merged.FFReportCards,
				FFLibrary:                       merged.FFLibrary,
				FFBroadcasts:                    merged.FFBroadcasts,
				FFConferenceScheduling:          merged.FFConferenceScheduling,
				FFDemographics:                  merged.FFDemographics,
				FFContentFilterIntegration:      merged.FFContentFilterIntegration,
				FFSISIntegration:                merged.FFSISIntegration,
				FFCatalogIntegration:            merged.FFCatalogIntegration,
				FFEnrollmentStateMachine:        merged.FFEnrollmentStateMachine,
				FFGradeSubmission:               merged.FFGradeSubmission,
			MFAEnabled:                      merged.MFAEnabled,
			MFAEnforcement:                  merged.MFAEnforcement,
			SMTPHost:                        merged.SMTPHost,
			SMTPPort:                        int(merged.SMTPPort),
			SMTPFrom:                        merged.SMTPFrom,
			SMTPUser:                        merged.SMTPUser,
			SMTPPassword:                    smtpPasswordMasked(dbRow, merged.SMTPPassword),
			Sources: platformSourcesJSON{
				OpenRouterAPIKey:            src(sources.OpenRouterAPIKey),
				SAMLSSOEnabled:              src(sources.SAMLSSOEnabled),
				SAMLPublicBaseURL:           src(sources.SAMLPublicBaseURL),
				SAMLSPEntityID:              src(sources.SAMLSPEntityID),
				SAMLSPX509PEM:               src(sources.SAMLSPX509PEM),
				SAMLSPPrivateKeyPEM:         src(sources.SAMLSPPrivateKeyPEM),
				AnnotationEnabled:           src(sources.AnnotationEnabled),
				FeedbackMediaEnabled:        src(sources.FeedbackMediaEnabled),
				BlindGradingEnabled:         src(sources.BlindGradingEnabled),
				ModeratedGradingEnabled:     src(sources.ModeratedGradingEnabled),
				OriginalityDetectionEnabled: src(sources.OriginalityDetectionEnabled),
				OriginalityStubExternal:     src(sources.OriginalityStubExternal),
				GradePostingPoliciesEnabled: src(sources.GradePostingPoliciesEnabled),
				GradebookCSVEnabled:         src(sources.GradebookCSVEnabled),
				ResubmissionWorkflowEnabled: src(sources.ResubmissionWorkflowEnabled),
				LTIEnabled:                  src(sources.LTIEnabled),
				OneRosterEnabled:            src(sources.OneRosterEnabled),
				ScimEnabled:                 src(sources.ScimEnabled),
				MFAEnabled:                  src(sources.MFAEnabled),
				MFAEnforcement:              src(sources.MFAEnforcement),
				SMTPHost:                    src(sources.SMTPHost),
				SMTPPort:                    src(sources.SMTPPort),
				SMTPFrom:                    src(sources.SMTPFrom),
				SMTPUser:                    src(sources.SMTPUser),
				SMTPPassword:                src(sources.SMTPPasswordCiphertext),
			},
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type putPlatformBody struct {
	OpenRouterAPIKey      *string `json:"openRouterApiKey"`
	ClearOpenRouterAPIKey bool    `json:"clearOpenRouterApiKey"`

	SAMLSSOEnabled      *bool   `json:"samlSsoEnabled"`
	SAMLPublicBaseURL   *string `json:"samlPublicBaseUrl"`
	SAMLSPEntityID      *string `json:"samlSpEntityId"`
	SAMLSPX509PEM       *string `json:"samlSpX509Pem"`
	SAMLSPPrivateKeyPEM *string `json:"samlSpPrivateKeyPem"`

	AnnotationEnabled           *bool `json:"annotationEnabled"`
	FeedbackMediaEnabled        *bool `json:"feedbackMediaEnabled"`
	BlindGradingEnabled         *bool `json:"blindGradingEnabled"`
	ModeratedGradingEnabled     *bool `json:"moderatedGradingEnabled"`
	OriginalityDetectionEnabled *bool `json:"originalityDetectionEnabled"`
	OriginalityStubExternal     *bool `json:"originalityStubExternal"`
	GradePostingPoliciesEnabled *bool `json:"gradePostingPoliciesEnabled"`
	GradebookCSVEnabled         *bool `json:"gradebookCsvEnabled"`
	ResubmissionWorkflowEnabled *bool `json:"resubmissionWorkflowEnabled"`
	LTIEnabled                  *bool `json:"ltiEnabled"`
	OneRosterEnabled            *bool `json:"oneRosterEnabled"`
	ScimEnabled                 *bool `json:"scimEnabled"`

	OIDCSSOEnabled             *bool `json:"oidcSsoEnabled"`
	CleverSSOEnabled           *bool `json:"cleverSsoEnabled"`
	ClassLinkSSOEnabled        *bool `json:"classlinkSsoEnabled"`
	MagicLinkEnabled           *bool `json:"magicLinkEnabled"`
	MagicLinkEnrolledOnly      *bool `json:"magicLinkEnrolledOnly"`
	SessionManagementUIEnabled *bool `json:"sessionManagementUiEnabled"`
	EmailNotificationsEnabled  *bool `json:"emailNotificationsEnabled"`
	PushNotificationsEnabled   *bool `json:"pushNotificationsEnabled"`
	VirtualClassroomEnabled    *bool `json:"virtualClassroomEnabled"`
	DRMEnabled                 *bool `json:"drmEnabled"`
	VideoTranscodingEnabled    *bool `json:"videoTranscodingEnabled"`
	AutoCaptioningEnabled      *bool `json:"autoCaptioningEnabled"`
	VideoCaptionsEnabled       *bool `json:"videoCaptionsEnabled"`
	StorageQuotasEnabled       *bool `json:"storageQuotasEnabled"`
	AtRiskAlertsEnabled        *bool `json:"atRiskAlertsEnabled"`
	AvScanningEnabled          *bool `json:"avScanningEnabled"`
	ClamAVStub                 *bool `json:"clamavStub"`
	H5PEnabled                 *bool `json:"h5pEnabled"`
	OERLibraryEnabled          *bool `json:"oerLibraryEnabled"`
	OERStub                    *bool `json:"oerStub"`
	ItemAnalysisEnabled        *bool `json:"itemAnalysisEnabled"`
	StudentProgressEnabled     *bool `json:"studentProgressEnabled"`
	EngagementTrackingEnabled  *bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled      *bool `json:"selfReflectionEnabled"`
	OutcomesReportEnabled      *bool `json:"outcomesReportEnabled"`
	InstructorInsightsEnabled  *bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled        *bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled      *bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled        *bool `json:"readingLevelEnabled"`
	AltTextEnforcementEnabled  *bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement       *bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled         *bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled *bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine      *bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled            *bool `json:"readAloudEnabled"`
	FFReadAloud                 *bool `json:"ffReadAloud"`
	TranslationMemoryEnabled    *bool `json:"translationMemoryEnabled"`
	ReportExportEnabled        *bool `json:"reportExportEnabled"`
	CoppaWorkflowEnabled       *bool `json:"coppaWorkflowEnabled"`
	IsoIsmsEnabled                  *bool `json:"isoIsmsEnabled"`
	AdminAuditLogEnabled            *bool `json:"adminAuditLogEnabled"`
	DataResidencyEnabled            *bool `json:"dataResidencyEnabled"`
	RTLEnabled                      *bool `json:"rtlEnabled"`
	SecurityDisclosureModuleEnabled *bool `json:"securityDisclosureModuleEnabled"`
	FFParentPortal                  *bool `json:"ffParentPortal"`
	FFReportCards                   *bool `json:"ffReportCards"`
	FFLibrary                       *bool `json:"ffLibrary"`
	FFBroadcasts                    *bool `json:"ffBroadcasts"`
	FFConferenceScheduling          *bool `json:"ffConferenceScheduling"`
	FFDemographics                  *bool `json:"ffDemographics"`
	FFContentFilterIntegration      *bool `json:"ffContentFilterIntegration"`
	FFSISIntegration                *bool `json:"ffSisIntegration"`
	FFCatalogIntegration            *bool `json:"ffCatalogIntegration"`
	FFEnrollmentStateMachine        *bool `json:"ffEnrollmentStateMachine"`
	FFGradeSubmission               *bool `json:"ffGradeSubmission"`

	MFAEnabled     *bool   `json:"mfaEnabled"`
	MFAEnforcement *string `json:"mfaEnforcement"`

	SMTPHost           *string `json:"smtpHost"`
	SMTPPort           *int    `json:"smtpPort"`
	SMTPFrom           *string `json:"smtpFrom"`
	SMTPUser           *string `json:"smtpUser"`
	SMTPPassword       *string `json:"smtpPassword"`
	ClearSMTPPassword  bool    `json:"clearSmtpPassword"`

	UpdateMask []string `json:"updateMask"`
}

// handlePutPlatformSettings is PUT /api/v1/settings/platform
func (d Deps) handlePutPlatformSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Database is not configured.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putPlatformBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		mask := map[string]struct{}{}
		for _, k := range body.UpdateMask {
			k = strings.TrimSpace(k)
			if k != "" {
				mask[strings.ToLower(k)] = struct{}{}
			}
		}

		wr := &platformconfig.Write{}
		clearRouter := body.ClearOpenRouterAPIKey
		clearSMTP := body.ClearSMTPPassword
		if len(mask) > 0 {
			clearRouter = false
			if _, ok := mask["clearopenrouterapikey"]; ok {
				clearRouter = true
			}
			clearSMTP = false
			if _, ok := mask["clearsmtpassword"]; ok {
				clearSMTP = true
			}
		}

		set := func(field string, hasInput bool, apply func()) {
			if len(mask) > 0 {
				if _, ok := mask[strings.ToLower(field)]; !ok {
					return
				}
			} else {
				if !hasInput {
					return
				}
			}
			apply()
		}

		set("openrouterapikey", body.OpenRouterAPIKey != nil, func() {
			s := strings.TrimSpace(*body.OpenRouterAPIKey)
			if s != "" && s != placeholderSecretResponse {
				wr.OpenRouterAPIKey = &s
			}
		})

		if clearRouter && wr.OpenRouterAPIKey != nil && strings.TrimSpace(*wr.OpenRouterAPIKey) != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot set openRouterApiKey and clearOpenRouterApiKey together.")
			return
		}
		if clearRouter {
			if err := platformconfig.ClearOpenRouterAPIKey(r.Context(), d.Pool); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear OpenRouter override.")
				return
			}
		}

		smtpPortActive := len(mask) == 0
		if _, ok := mask["smtpport"]; ok {
			smtpPortActive = true
		}
		if body.SMTPPort != nil && smtpPortActive {
			if *body.SMTPPort < 1 || *body.SMTPPort > 65535 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "smtpPort must be between 1 and 65535.")
				return
			}
		}

		var smtpPasswordErr string
		set("smtppassword", body.SMTPPassword != nil, func() {
			s := strings.TrimSpace(*body.SMTPPassword)
			if s == "" || s == placeholderSecretResponse {
				return
			}
			if len(d.Config.PlatformSecretsKey) != 32 {
				smtpPasswordErr = "Set PLATFORM_SECRETS_KEY to a base64-encoded 32-byte key (e.g. openssl rand -base64 32) before storing an SMTP password."
				return
			}
			blob, err := appsecrets.Encrypt([]byte(s), d.Config.PlatformSecretsKey)
			if err != nil {
				smtpPasswordErr = err.Error()
				return
			}
			wr.SMTPPasswordCiphertext = &blob
		})
		if smtpPasswordErr != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, smtpPasswordErr)
			return
		}
		if clearSMTP && wr.SMTPPasswordCiphertext != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot set smtpPassword and clearSmtpPassword together.")
			return
		}
		if clearSMTP {
			if err := platformconfig.ClearSMTPPassword(r.Context(), d.Pool); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear stored SMTP password.")
				return
			}
		}

		set("smtphost", body.SMTPHost != nil, func() {
			s := strings.TrimSpace(*body.SMTPHost)
			wr.SMTPHost = &s
		})
		set("smtpport", body.SMTPPort != nil, func() {
			v := int32(*body.SMTPPort)
			wr.SMTPPort = &v
		})
		set("smtpfrom", body.SMTPFrom != nil, func() {
			s := strings.TrimSpace(*body.SMTPFrom)
			wr.SMTPFrom = &s
		})
		set("smtpuser", body.SMTPUser != nil, func() {
			s := strings.TrimSpace(*body.SMTPUser)
			wr.SMTPUser = &s
		})

		set("samlssoenabled", body.SAMLSSOEnabled != nil, func() {
			v := *body.SAMLSSOEnabled
			wr.SAMLSSOEnabled = &v
		})
		set("samlpublicbaseurl", body.SAMLPublicBaseURL != nil, func() {
			s := strings.TrimSpace(*body.SAMLPublicBaseURL)
			wr.SAMLPublicBaseURL = &s
		})
		set("samlspentityid", body.SAMLSPEntityID != nil, func() {
			s := strings.TrimSpace(*body.SAMLSPEntityID)
			wr.SAMLSPEntityID = &s
		})
		set("samlspx509pem", body.SAMLSPX509PEM != nil, func() {
			s := strings.TrimSpace(*body.SAMLSPX509PEM)
			if s != "" && s != placeholderSecretResponse {
				wr.SAMLSPX509PEM = &s
			}
		})
		set("samlprivatekeypem", body.SAMLSPPrivateKeyPEM != nil, func() {
			s := strings.TrimSpace(*body.SAMLSPPrivateKeyPEM)
			if s != "" && s != placeholderSecretResponse {
				wr.SAMLSPPrivateKeyPEM = &s
			}
		})
		set("annotationenabled", body.AnnotationEnabled != nil, func() {
			v := *body.AnnotationEnabled
			wr.AnnotationEnabled = &v
		})
		set("feedbackmediaenabled", body.FeedbackMediaEnabled != nil, func() {
			v := *body.FeedbackMediaEnabled
			wr.FeedbackMediaEnabled = &v
		})
		set("blindgradingenabled", body.BlindGradingEnabled != nil, func() {
			v := *body.BlindGradingEnabled
			wr.BlindGradingEnabled = &v
		})
		set("moderatedgradingenabled", body.ModeratedGradingEnabled != nil, func() {
			v := *body.ModeratedGradingEnabled
			wr.ModeratedGradingEnabled = &v
		})
		set("originalitydetectionenabled", body.OriginalityDetectionEnabled != nil, func() {
			v := *body.OriginalityDetectionEnabled
			wr.OriginalityDetectionEnabled = &v
		})
		set("originalitystubexternal", body.OriginalityStubExternal != nil, func() {
			v := *body.OriginalityStubExternal
			wr.OriginalityStubExternal = &v
		})
		set("gradepostingpoliciesenabled", body.GradePostingPoliciesEnabled != nil, func() {
			v := *body.GradePostingPoliciesEnabled
			wr.GradePostingPoliciesEnabled = &v
		})
		set("gradebookcsvenabled", body.GradebookCSVEnabled != nil, func() {
			v := *body.GradebookCSVEnabled
			wr.GradebookCSVEnabled = &v
		})
		set("resubmissionworkflowenabled", body.ResubmissionWorkflowEnabled != nil, func() {
			v := *body.ResubmissionWorkflowEnabled
			wr.ResubmissionWorkflowEnabled = &v
		})
		set("ltienabled", body.LTIEnabled != nil, func() {
			v := *body.LTIEnabled
			wr.LTIEnabled = &v
		})
		set("onerosterenabled", body.OneRosterEnabled != nil, func() {
			v := *body.OneRosterEnabled
			wr.OneRosterEnabled = &v
		})
		set("scimenabled", body.ScimEnabled != nil, func() {
			v := *body.ScimEnabled
			wr.ScimEnabled = &v
		})
		setBool := func(field string, ptr *bool, apply func(bool)) {
			set(field, ptr != nil, func() { apply(*ptr) })
		}
		setBool("oidcssoenabled", body.OIDCSSOEnabled, func(v bool) { wr.OIDCSSOEnabled = &v })
		setBool("cleverssoenabled", body.CleverSSOEnabled, func(v bool) { wr.CleverSSOEnabled = &v })
		setBool("classlinkssoenabled", body.ClassLinkSSOEnabled, func(v bool) { wr.ClassLinkSSOEnabled = &v })
		setBool("magiclinkenabled", body.MagicLinkEnabled, func(v bool) { wr.MagicLinkEnabled = &v })
		setBool("magiclinkenrolledonly", body.MagicLinkEnrolledOnly, func(v bool) { wr.MagicLinkEnrolledOnly = &v })
		setBool("sessionmanagementuienabled", body.SessionManagementUIEnabled, func(v bool) { wr.SessionManagementUIEnabled = &v })
		setBool("emailnotificationsenabled", body.EmailNotificationsEnabled, func(v bool) { wr.EmailNotificationsEnabled = &v })
		setBool("pushnotificationsenabled", body.PushNotificationsEnabled, func(v bool) { wr.PushNotificationsEnabled = &v })
		setBool("virtualclassroomenabled", body.VirtualClassroomEnabled, func(v bool) { wr.VirtualClassroomEnabled = &v })
		setBool("drmenabled", body.DRMEnabled, func(v bool) { wr.DRMEnabled = &v })
		setBool("videotranscodingenabled", body.VideoTranscodingEnabled, func(v bool) { wr.VideoTranscodingEnabled = &v })
		setBool("autocaptioningenabled", body.AutoCaptioningEnabled, func(v bool) { wr.AutoCaptioningEnabled = &v })
		setBool("videocaptionsenabled", body.VideoCaptionsEnabled, func(v bool) { wr.VideoCaptionsEnabled = &v })
		setBool("storagequotasenabled", body.StorageQuotasEnabled, func(v bool) { wr.StorageQuotasEnabled = &v })
		setBool("atriskalertsenabled", body.AtRiskAlertsEnabled, func(v bool) { wr.AtRiskAlertsEnabled = &v })
		setBool("avscanningenabled", body.AvScanningEnabled, func(v bool) { wr.AvScanningEnabled = &v })
		setBool("clamavstub", body.ClamAVStub, func(v bool) { wr.ClamAVStub = &v })
		setBool("h5penabled", body.H5PEnabled, func(v bool) { wr.H5PEnabled = &v })
		setBool("oerlibraryenabled", body.OERLibraryEnabled, func(v bool) { wr.OERLibraryEnabled = &v })
		setBool("oerstub", body.OERStub, func(v bool) { wr.OERStub = &v })
		setBool("itemanalysisenabled", body.ItemAnalysisEnabled, func(v bool) { wr.ItemAnalysisEnabled = &v })
		setBool("studentprogressenabled", body.StudentProgressEnabled, func(v bool) { wr.StudentProgressEnabled = &v })
		setBool("engagementtrackingenabled", body.EngagementTrackingEnabled, func(v bool) { wr.EngagementTrackingEnabled = &v })
		setBool("selfreflectionenabled", body.SelfReflectionEnabled, func(v bool) { wr.SelfReflectionEnabled = &v })
		setBool("outcomesreportenabled", body.OutcomesReportEnabled, func(v bool) { wr.OutcomesReportEnabled = &v })
		setBool("instructorinsightsenabled", body.InstructorInsightsEnabled, func(v bool) { wr.InstructorInsightsEnabled = &v })
		setBool("equationeditorenabled", body.EquationEditorEnabled, func(v bool) { wr.EquationEditorEnabled = &v })
		setBool("readinglevelenabled", body.ReadingLevelEnabled, func(v bool) { wr.ReadingLevelEnabled = &v })
		setBool("alttextenforcementenabled", body.AltTextEnforcementEnabled, func(v bool) { wr.AltTextEnforcementEnabled = &v })
		setBool("ffalttextenforcement", body.FFAltTextEnforcement, func(v bool) { wr.FFAltTextEnforcement = &v })
		setBool("speechtotextenabled", body.SpeechToTextEnabled, func(v bool) { wr.SpeechToTextEnabled = &v })
		setBool("accommodationsengineenabled", body.AccommodationsEngineEnabled, func(v bool) { wr.AccommodationsEngineEnabled = &v })
		setBool("ffaccommodationsengine", body.FFAccommodationsEngine, func(v bool) { wr.FFAccommodationsEngine = &v })
		setBool("readaloudenabled", body.ReadAloudEnabled, func(v bool) { wr.ReadAloudEnabled = &v })
		setBool("ffreadaloud", body.FFReadAloud, func(v bool) { wr.FFReadAloud = &v })
		setBool("translationmemoryenabled", body.TranslationMemoryEnabled, func(v bool) { wr.TranslationMemoryEnabled = &v })
		setBool("reportexportenabled", body.ReportExportEnabled, func(v bool) { wr.ReportExportEnabled = &v })
		setBool("xapiemissionenabled", body.XAPIEmissionEnabled, func(v bool) { wr.XAPIEmissionEnabled = &v })
		setBool("coppaworkflowenabled", body.CoppaWorkflowEnabled, func(v bool) { wr.CoppaWorkflowEnabled = &v })
		setBool("isoismsenabled", body.IsoIsmsEnabled, func(v bool) { wr.IsoIsmsEnabled = &v })
		setBool("adminauditlogenabled", body.AdminAuditLogEnabled, func(v bool) { wr.AdminAuditLogEnabled = &v })
		setBool("dataresidencyenabled", body.DataResidencyEnabled, func(v bool) { wr.DataResidencyEnabled = &v })
		setBool("rtlenabled", body.RTLEnabled, func(v bool) { wr.RTLEnabled = &v })
		setBool("securitydisclosuremoduleenabled", body.SecurityDisclosureModuleEnabled, func(v bool) { wr.SecurityDisclosureModuleEnabled = &v })
		setBool("ffparentportal", body.FFParentPortal, func(v bool) { wr.FFParentPortal = &v })
		setBool("ffreportcards", body.FFReportCards, func(v bool) { wr.FFReportCards = &v })
		setBool("fflibrary", body.FFLibrary, func(v bool) { wr.FFLibrary = &v })
		setBool("ffbroadcasts", body.FFBroadcasts, func(v bool) { wr.FFBroadcasts = &v })
		setBool("ffconferencescheduling", body.FFConferenceScheduling, func(v bool) { wr.FFConferenceScheduling = &v })
		setBool("ffdemographics", body.FFDemographics, func(v bool) { wr.FFDemographics = &v })
		setBool("ffcontentfilterintegration", body.FFContentFilterIntegration, func(v bool) { wr.FFContentFilterIntegration = &v })
		setBool("ffsisintegration", body.FFSISIntegration, func(v bool) { wr.FFSISIntegration = &v })
		setBool("ffcatalogintegration", body.FFCatalogIntegration, func(v bool) { wr.FFCatalogIntegration = &v })
		setBool("ffenrollmentstatemachine", body.FFEnrollmentStateMachine, func(v bool) { wr.FFEnrollmentStateMachine = &v })
		setBool("ffgradesubmission", body.FFGradeSubmission, func(v bool) { wr.FFGradeSubmission = &v })
		set("mfaenabled", body.MFAEnabled != nil, func() {
			v := *body.MFAEnabled
			wr.MFAEnabled = &v
		})
		set("mfaenforcement", body.MFAEnforcement != nil, func() {
			s := strings.ToLower(strings.TrimSpace(*body.MFAEnforcement))
			if s != "none" && s != "all" && s != "staff" {
				return
			}
			wr.MFAEnforcement = &s
		})

		dbRow, err := platformconfig.Upsert(r.Context(), d.Pool, wr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save platform settings.")
			return
		}
		merged := platformconfig.Merge(d.Config, dbRow)
		if err := merged.Validate(); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if d.Platform != nil {
			d.Platform.Reload(merged)
		}

		sources := platformconfig.ResolveSources(d.Config, dbRow)
		out := platformSettingsJSON{
			OpenRouterAPIKey:            maskSecret(merged.OpenRouterAPIKey),
			SAMLSSOEnabled:              merged.SAMLSSOEnabled,
			SAMLPublicBaseURL:           merged.SAMLPublicBaseURL,
			SAMLSPEntityID:              merged.SAMLSPEntityID,
			SAMLSPX509PEM:               merged.SAMLSPX509PEM,
			SAMLSPPrivateKeyPEM:         maskPEMIfSet(merged.SAMLSPPrivateKeyPEM),
			AnnotationEnabled:           merged.AnnotationEnabled,
			FeedbackMediaEnabled:        merged.FeedbackMediaEnabled,
			BlindGradingEnabled:         merged.BlindGradingEnabled,
			ModeratedGradingEnabled:     merged.ModeratedGradingEnabled,
			OriginalityDetectionEnabled: merged.OriginalityDetectionEnabled,
			OriginalityStubExternal:     merged.OriginalityStubExternal,
			GradePostingPoliciesEnabled: merged.GradePostingPoliciesEnabled,
			GradebookCSVEnabled:         merged.GradebookCSVEnabled,
			ResubmissionWorkflowEnabled: merged.ResubmissionWorkflowEnabled,
			LTIEnabled:                  merged.LTIEnabled,
			OneRosterEnabled:            merged.OneRosterEnabled,
			ScimEnabled:                 merged.ScimEnabled,
			OIDCSSOEnabled:              merged.OIDCSSOEnabled,
			CleverSSOEnabled:            merged.CleverSSOEnabled,
			ClassLinkSSOEnabled:         merged.ClassLinkSSOEnabled,
			MagicLinkEnabled:            merged.MagicLinkEnabled,
			MagicLinkEnrolledOnly:       merged.MagicLinkEnrolledOnly,
			SessionManagementUIEnabled:  merged.SessionManagementUIEnabled,
			EmailNotificationsEnabled:   merged.EmailNotificationsEnabled,
			PushNotificationsEnabled:    merged.PushNotificationsEnabled,
			VirtualClassroomEnabled:     merged.VirtualClassroomEnabled,
			DRMEnabled:                  merged.DRMEnabled,
			VideoTranscodingEnabled:     merged.VideoTranscodingEnabled,
			AutoCaptioningEnabled:       merged.AutoCaptioningEnabled,
			VideoCaptionsEnabled:        merged.VideoCaptionsEnabled,
			StorageQuotasEnabled:        merged.StorageQuotasEnabled,
			AtRiskAlertsEnabled:         merged.AtRiskAlertsEnabled,
			AvScanningEnabled:           merged.AvScanningEnabled,
			ClamAVStub:                  merged.ClamAVStub,
			H5PEnabled:                  merged.H5PEnabled,
			OERLibraryEnabled:           merged.OERLibraryEnabled,
			OERStub:                     merged.OERStub,
			ItemAnalysisEnabled:         merged.ItemAnalysisEnabled,
			StudentProgressEnabled:      merged.StudentProgressEnabled,
			EngagementTrackingEnabled:   merged.EngagementTrackingEnabled,
			SelfReflectionEnabled:       merged.SelfReflectionEnabled,
			OutcomesReportEnabled:       merged.OutcomesReportEnabled,
			InstructorInsightsEnabled:   merged.InstructorInsightsEnabled,
			XAPIEmissionEnabled:         merged.XAPIEmissionEnabled,
			EquationEditorEnabled:       merged.EquationEditorEnabled,
			ReadingLevelEnabled:         merged.ReadingLevelEnabled,
			AltTextEnforcementEnabled:   merged.AltTextEnforcementEnabled,
			FFAltTextEnforcement:        merged.FFAltTextEnforcement,
			SpeechToTextEnabled:          merged.SpeechToTextEnabled,
			AccommodationsEngineEnabled:  merged.AccommodationsEngineEnabled,
			FFAccommodationsEngine:       merged.FFAccommodationsEngine,
			ReadAloudEnabled:             merged.ReadAloudEnabled,
			FFReadAloud:                  merged.FFReadAloud,
			TranslationMemoryEnabled:     merged.TranslationMemoryEnabled,
			ReportExportEnabled:         merged.ReportExportEnabled,
			CoppaWorkflowEnabled:        merged.CoppaWorkflowEnabled,
			IsoIsmsEnabled:                  merged.IsoIsmsEnabled,
			AdminAuditLogEnabled:            merged.AdminAuditLogEnabled,
			DataResidencyEnabled:            merged.DataResidencyEnabled,
			RTLEnabled:                      merged.RTLEnabled,
			SecurityDisclosureModuleEnabled: merged.SecurityDisclosureModuleEnabled,
			FFParentPortal:                  merged.FFParentPortal,
				FFReportCards:                   merged.FFReportCards,
				FFLibrary:                       merged.FFLibrary,
				FFBroadcasts:                    merged.FFBroadcasts,
				FFConferenceScheduling:          merged.FFConferenceScheduling,
				FFDemographics:                  merged.FFDemographics,
				FFContentFilterIntegration:      merged.FFContentFilterIntegration,
				FFSISIntegration:                merged.FFSISIntegration,
				FFCatalogIntegration:            merged.FFCatalogIntegration,
				FFEnrollmentStateMachine:        merged.FFEnrollmentStateMachine,
				FFGradeSubmission:               merged.FFGradeSubmission,
			MFAEnabled:                      merged.MFAEnabled,
			MFAEnforcement:                  merged.MFAEnforcement,
			SMTPHost:                        merged.SMTPHost,
			SMTPPort:                        int(merged.SMTPPort),
			SMTPFrom:                        merged.SMTPFrom,
			SMTPUser:                        merged.SMTPUser,
			SMTPPassword:                    smtpPasswordMasked(dbRow, merged.SMTPPassword),
			Sources: platformSourcesJSON{
				OpenRouterAPIKey:            src(sources.OpenRouterAPIKey),
				SAMLSSOEnabled:              src(sources.SAMLSSOEnabled),
				SAMLPublicBaseURL:           src(sources.SAMLPublicBaseURL),
				SAMLSPEntityID:              src(sources.SAMLSPEntityID),
				SAMLSPX509PEM:               src(sources.SAMLSPX509PEM),
				SAMLSPPrivateKeyPEM:         src(sources.SAMLSPPrivateKeyPEM),
				AnnotationEnabled:           src(sources.AnnotationEnabled),
				FeedbackMediaEnabled:        src(sources.FeedbackMediaEnabled),
				BlindGradingEnabled:         src(sources.BlindGradingEnabled),
				ModeratedGradingEnabled:     src(sources.ModeratedGradingEnabled),
				OriginalityDetectionEnabled: src(sources.OriginalityDetectionEnabled),
				OriginalityStubExternal:     src(sources.OriginalityStubExternal),
				GradePostingPoliciesEnabled: src(sources.GradePostingPoliciesEnabled),
				GradebookCSVEnabled:         src(sources.GradebookCSVEnabled),
				ResubmissionWorkflowEnabled: src(sources.ResubmissionWorkflowEnabled),
				LTIEnabled:                  src(sources.LTIEnabled),
				OneRosterEnabled:            src(sources.OneRosterEnabled),
				ScimEnabled:                 src(sources.ScimEnabled),
				MFAEnabled:                  src(sources.MFAEnabled),
				MFAEnforcement:              src(sources.MFAEnforcement),
				SMTPHost:                    src(sources.SMTPHost),
				SMTPPort:                    src(sources.SMTPPort),
				SMTPFrom:                    src(sources.SMTPFrom),
				SMTPUser:                    src(sources.SMTPUser),
				SMTPPassword:                src(sources.SMTPPasswordCiphertext),
			},
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
