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

	OIDCSSOEnabled                  bool `json:"oidcSsoEnabled"`
	CleverSSOEnabled                bool `json:"cleverSsoEnabled"`
	ClassLinkSSOEnabled             bool `json:"classlinkSsoEnabled"`
	MagicLinkEnabled                bool `json:"magicLinkEnabled"`
	MagicLinkEnrolledOnly           bool `json:"magicLinkEnrolledOnly"`
	SessionManagementUIEnabled      bool `json:"sessionManagementUiEnabled"`
	EmailNotificationsEnabled       bool `json:"emailNotificationsEnabled"`
	PushNotificationsEnabled        bool `json:"pushNotificationsEnabled"`
	VirtualClassroomEnabled         bool `json:"virtualClassroomEnabled"`
	DRMEnabled                      bool `json:"drmEnabled"`
	VideoTranscodingEnabled         bool `json:"videoTranscodingEnabled"`
	AutoCaptioningEnabled           bool `json:"autoCaptioningEnabled"`
	VideoCaptionsEnabled            bool `json:"videoCaptionsEnabled"`
	StorageQuotasEnabled            bool `json:"storageQuotasEnabled"`
	AtRiskAlertsEnabled             bool `json:"atRiskAlertsEnabled"`
	AvScanningEnabled               bool `json:"avScanningEnabled"`
	ClamAVStub                      bool `json:"clamavStub"`
	H5PEnabled                      bool `json:"h5pEnabled"`
	ScormIngestionEnabled           bool `json:"scormIngestionEnabled"`
	OERLibraryEnabled               bool `json:"oerLibraryEnabled"`
	OERStub                         bool `json:"oerStub"`
	ItemAnalysisEnabled             bool `json:"itemAnalysisEnabled"`
	StudentProgressEnabled          bool `json:"studentProgressEnabled"`
	EngagementTrackingEnabled       bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled           bool `json:"selfReflectionEnabled"`
	LearnerProfileEnabled           bool `json:"learnerProfileEnabled"`
	LpAdaptRecommendationsEnabled   bool `json:"lpAdaptRecommendationsEnabled"`
	LpAdaptReviewEnabled            bool `json:"lpAdaptReviewEnabled"`
	LpAdaptModalityEnabled          bool `json:"lpAdaptModalityEnabled"`
	LpAdaptTutorEnabled             bool `json:"lpAdaptTutorEnabled"`
	IntroCourseEnabled              bool `json:"introCourseEnabled"`
	OutcomesReportEnabled           bool `json:"outcomesReportEnabled"`
	InstructorInsightsEnabled       bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled             bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled           bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled             bool `json:"readingLevelEnabled"`
	GraderAgentEnabled              bool `json:"graderAgentEnabled"`
	GraderAgentReviewInboxEnabled   bool `json:"graderAgentReviewInboxEnabled"`
	GraderAgentSuggestModeEnabled      bool `json:"graderAgentSuggestModeEnabled"`
	GraderAgentTextEntryGradingEnabled bool `json:"graderAgentTextEntryGradingEnabled"`
	GraderAgentVisionGradingEnabled    bool `json:"graderAgentVisionGradingEnabled"`
	GraderAgentRunFiltersEnabled       bool `json:"graderAgentRunFiltersEnabled"`
	GraderAgentCostEstimateEnabled     bool `json:"graderAgentCostEstimateEnabled"`
	GraderAgentCancelRunEnabled        bool `json:"graderAgentCancelRunEnabled"`
	CodeExecutionEnabled            bool `json:"codeExecutionEnabled"`
	AltTextEnforcementEnabled       bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement            bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled             bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled     bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine          bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled                bool `json:"readAloudEnabled"`
	FFReadAloud                     bool `json:"ffReadAloud"`
	TranslationMemoryEnabled        bool `json:"translationMemoryEnabled"`
	ReportExportEnabled             bool `json:"reportExportEnabled"`
	CoppaWorkflowEnabled            bool `json:"coppaWorkflowEnabled"`
	IsoIsmsEnabled                  bool `json:"isoIsmsEnabled"`
	AdminAuditLogEnabled            bool `json:"adminAuditLogEnabled"`
	AdminConsoleEnabled             bool `json:"adminConsoleEnabled"`
	ImpersonationEnabled            bool `json:"impersonationEnabled"`
	BulkCsvImportEnabled            bool `json:"bulkCsvImportEnabled"`
	AdminSearchEnabled              bool `json:"adminSearchEnabled"`
	EmailTemplateEditorEnabled      bool `json:"emailTemplateEditorEnabled"`
	MaintenanceBannerEnabled        bool `json:"maintenanceBannerEnabled"`
	CustomFieldsEnabled             bool `json:"customFieldsEnabled"`
	SeatManagementEnabled           bool `json:"seatManagementEnabled"`
	DataResidencyEnabled            bool `json:"dataResidencyEnabled"`
	RTLEnabled                      bool `json:"rtlEnabled"`
	SecurityDisclosureModuleEnabled bool `json:"securityDisclosureModuleEnabled"`
	FFParentPortal                  bool `json:"ffParentPortal"`
	FFParentPortalV2                bool `json:"ffParentPortalV2"`
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
	FFWhatifGrades                  bool `json:"ffWhatifGrades"`
	FFGradeCurving                  bool `json:"ffGradeCurving"`
	FFPlagiarismChecks              bool `json:"ffPlagiarismChecks"`
	FFIncompleteGradeWorkflow       bool `json:"ffIncompleteGradeWorkflow"`
	FFAcademicCalendar              bool `json:"ffAcademicCalendar"`
	FFCourseEvaluations             bool `json:"ffCourseEvaluations"`
	FFProctoringIntegration         bool `json:"ffProctoringIntegration"`
	FFCoCurricularTranscript        bool `json:"ffCoCurricularTranscript"`
	FFEportfolio                    bool `json:"ffEportfolio"`
	FFBookstoreIntegration          bool `json:"ffBookstoreIntegration"`
	FFTranscripts                   bool `json:"ffTranscripts"`
	FFWebhooks                      bool `json:"ffWebhooks"`
	FFZapierConnector               bool `json:"ffZapierConnector"`
	FFAdvisingIntegration           bool `json:"ffAdvisingIntegration"`
	FFResearchConsent               bool `json:"ffResearchConsent"`
	FFAccessibilityIntake           bool `json:"ffAccessibilityIntake"`
	FFCEUTracking                   bool `json:"ffCeuTracking"`
	FFConsortiumSharing             bool `json:"ffConsortiumSharing"`
	FFSelfPacedMode                 bool `json:"ffSelfPacedMode"`
	FFPublicCatalog                 bool `json:"ffPublicCatalog"`
	FFPublicAPI                     bool `json:"ffPublicApi"`
	FFStripeBilling                 bool `json:"ffStripeBilling"`
	FFPaymentsEnabled               bool `json:"ffPaymentsEnabled"`
	FFRevenueShare                  bool `json:"ffRevenueShare"`
	FFTaxCollection                 bool `json:"ffTaxCollection"`
	FFLearningPaths                 bool `json:"ffLearningPaths"`
	FFConditionalRelease            bool `json:"ffConditionalRelease"`
	FFPeerReview                    bool `json:"ffPeerReview"`
	FFCompletionCredentials         bool `json:"ffCompletionCredentials"`
	FFCourseReviews                 bool `json:"ffCourseReviews"`
	FFGamification                  bool `json:"ffGamification"`
	FFOnboardingFlow                bool `json:"ffOnboardingFlow"`
	FFStudyReminders                bool `json:"ffStudyReminders"`
	FFAIStudyBuddy                  bool `json:"ffAiStudyBuddy"`
	FFLessonGenerator               bool `json:"ffLessonGenerator"`
	FFPersistentTutor               bool `json:"ffPersistentTutor"`
	FFAPITokens                     bool `json:"ffApiTokens"`
	FFBotSlack                      bool `json:"ffBotSlack"`
	FFBotTeams                      bool `json:"ffBotTeams"`
	FFBotDiscord                    bool `json:"ffBotDiscord"`
	FFCalendarFeeds                 bool `json:"ffCalendarFeeds"`
	FFRedisCache                    bool `json:"ffRedisCache"`

	LRSAnonymizeActors           bool    `json:"lrsAnonymizeActors"`
	FERPAWorkflowEnabled         bool    `json:"ferpaWorkflowEnabled"`
	DPAPortalEnabled             bool    `json:"dpaPortalEnabled"`
	SOC2ModuleEnabled            bool    `json:"soc2ModuleEnabled"`
	FFReadingPreferences         bool    `json:"ffReadingPreferences"`
	FFClassroomSignals           bool    `json:"ffClassroomSignals"`
	FFLibraryIntegration         bool    `json:"ffLibraryIntegration"`
	DiagnosticAssessmentsEnabled bool    `json:"diagnosticAssessmentsEnabled"`
	SRSPracticeEnabled           bool    `json:"srsPracticeEnabled"`
	IRTCatModeEnabled            bool    `json:"irtCatModeEnabled"`
	AdaptiveLearnerModelEnabled  bool    `json:"adaptiveLearnerModelEnabled"`
	LearnerModelEMAAlpha         float64 `json:"learnerModelEmaAlpha"`

	GDPRModuleEnabled   bool `json:"gdprModuleEnabled"`
	CCPAModuleEnabled   bool `json:"ccpaModuleEnabled"`
	StatePrivacyEnabled bool `json:"statePrivacyEnabled"`
	BackupModuleEnabled bool `json:"backupModuleEnabled"`
	FFUiMode            bool `json:"ffUiMode"`

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
			SAMLSSOEnabled:                  merged.SAMLSSOEnabled,
			SAMLPublicBaseURL:               merged.SAMLPublicBaseURL,
			SAMLSPEntityID:                  merged.SAMLSPEntityID,
			SAMLSPX509PEM:                   merged.SAMLSPX509PEM,
			SAMLSPPrivateKeyPEM:             maskPEMIfSet(merged.SAMLSPPrivateKeyPEM),
			AnnotationEnabled:               merged.AnnotationEnabled,
			FeedbackMediaEnabled:            merged.FeedbackMediaEnabled,
			BlindGradingEnabled:             merged.BlindGradingEnabled,
			ModeratedGradingEnabled:         merged.ModeratedGradingEnabled,
			OriginalityDetectionEnabled:     merged.OriginalityDetectionEnabled,
			OriginalityStubExternal:         merged.OriginalityStubExternal,
			GradePostingPoliciesEnabled:     merged.GradePostingPoliciesEnabled,
			GradebookCSVEnabled:             merged.GradebookCSVEnabled,
			ResubmissionWorkflowEnabled:     merged.ResubmissionWorkflowEnabled,
			LTIEnabled:                      merged.LTIEnabled,
			OneRosterEnabled:                merged.OneRosterEnabled,
			ScimEnabled:                     merged.ScimEnabled,
			OIDCSSOEnabled:                  merged.OIDCSSOEnabled,
			CleverSSOEnabled:                merged.CleverSSOEnabled,
			ClassLinkSSOEnabled:             merged.ClassLinkSSOEnabled,
			MagicLinkEnabled:                merged.MagicLinkEnabled,
			MagicLinkEnrolledOnly:           merged.MagicLinkEnrolledOnly,
			SessionManagementUIEnabled:      merged.SessionManagementUIEnabled,
			EmailNotificationsEnabled:       merged.EmailNotificationsEnabled,
			PushNotificationsEnabled:        merged.PushNotificationsEnabled,
			VirtualClassroomEnabled:         merged.VirtualClassroomEnabled,
			DRMEnabled:                      merged.DRMEnabled,
			VideoTranscodingEnabled:         merged.VideoTranscodingEnabled,
			AutoCaptioningEnabled:           merged.AutoCaptioningEnabled,
			VideoCaptionsEnabled:            merged.VideoCaptionsEnabled,
			StorageQuotasEnabled:            merged.StorageQuotasEnabled,
			AtRiskAlertsEnabled:             merged.AtRiskAlertsEnabled,
			AvScanningEnabled:               merged.AvScanningEnabled,
			ClamAVStub:                      merged.ClamAVStub,
			H5PEnabled:                      merged.H5PEnabled,
			ScormIngestionEnabled:           merged.ScormIngestionEnabled,
			OERLibraryEnabled:               merged.OERLibraryEnabled,
			OERStub:                         merged.OERStub,
			ItemAnalysisEnabled:             merged.ItemAnalysisEnabled,
			StudentProgressEnabled:          merged.StudentProgressEnabled,
			EngagementTrackingEnabled:       merged.EngagementTrackingEnabled,
			SelfReflectionEnabled:           merged.SelfReflectionEnabled,
			LearnerProfileEnabled:           merged.LearnerProfileEnabled,
			LpAdaptRecommendationsEnabled:   merged.LpAdaptRecommendationsEnabled,
			LpAdaptReviewEnabled:            merged.LpAdaptReviewEnabled,
			LpAdaptModalityEnabled:          merged.LpAdaptModalityEnabled,
			LpAdaptTutorEnabled:             merged.LpAdaptTutorEnabled,
			IntroCourseEnabled:              merged.IntroCourseEnabled,
			OutcomesReportEnabled:           merged.OutcomesReportEnabled,
			InstructorInsightsEnabled:       merged.InstructorInsightsEnabled,
			XAPIEmissionEnabled:             merged.XAPIEmissionEnabled,
			EquationEditorEnabled:           merged.EquationEditorEnabled,
			ReadingLevelEnabled:             merged.ReadingLevelEnabled,
			GraderAgentEnabled:              merged.GraderAgentEnabled,
			GraderAgentReviewInboxEnabled:   merged.GraderAgentReviewInboxEnabled,
			GraderAgentSuggestModeEnabled:      merged.GraderAgentSuggestModeEnabled,
			GraderAgentTextEntryGradingEnabled: merged.GraderAgentTextEntryGradingEnabled,
			GraderAgentVisionGradingEnabled:    merged.GraderAgentVisionGradingEnabled,
			GraderAgentRunFiltersEnabled:       merged.GraderAgentRunFiltersEnabled,
			GraderAgentCostEstimateEnabled:     merged.GraderAgentCostEstimateEnabled,
			GraderAgentCancelRunEnabled:        merged.GraderAgentCancelRunEnabled,
			CodeExecutionEnabled:            merged.CodeExecutionEnabled,
			AltTextEnforcementEnabled:       merged.AltTextEnforcementEnabled,
			FFAltTextEnforcement:            merged.FFAltTextEnforcement,
			SpeechToTextEnabled:             merged.SpeechToTextEnabled,
			AccommodationsEngineEnabled:     merged.AccommodationsEngineEnabled,
			FFAccommodationsEngine:          merged.FFAccommodationsEngine,
			ReadAloudEnabled:                merged.ReadAloudEnabled,
			FFReadAloud:                     merged.FFReadAloud,
			TranslationMemoryEnabled:        merged.TranslationMemoryEnabled,
			ReportExportEnabled:             merged.ReportExportEnabled,
			CoppaWorkflowEnabled:            merged.CoppaWorkflowEnabled,
			IsoIsmsEnabled:                  merged.IsoIsmsEnabled,
			AdminAuditLogEnabled:            merged.AdminAuditLogEnabled,
			AdminConsoleEnabled:             merged.AdminConsoleEnabled,
			ImpersonationEnabled:            merged.ImpersonationEnabled,
			BulkCsvImportEnabled:            merged.BulkCsvImportEnabled,
			AdminSearchEnabled:              merged.AdminSearchEnabled,
			EmailTemplateEditorEnabled:      merged.EmailTemplateEditorEnabled,
			MaintenanceBannerEnabled:        merged.MaintenanceBannerEnabled,
			CustomFieldsEnabled:             merged.CustomFieldsEnabled,
			SeatManagementEnabled:           merged.SeatManagementEnabled,
			DataResidencyEnabled:            merged.DataResidencyEnabled,
			RTLEnabled:                      merged.RTLEnabled,
			SecurityDisclosureModuleEnabled: merged.SecurityDisclosureModuleEnabled,
			FFParentPortal:                  merged.FFParentPortal,
			FFParentPortalV2:                merged.FFParentPortalV2,
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
			FFWhatifGrades:                  merged.FFWhatifGrades,
			FFGradeCurving:                  merged.FFGradeCurving,
			FFPlagiarismChecks:              merged.FFPlagiarismChecks,
			FFIncompleteGradeWorkflow:       merged.FFIncompleteGradeWorkflow,
			FFAcademicCalendar:              merged.FFAcademicCalendar,
			FFCourseEvaluations:             merged.FFCourseEvaluations,
			FFProctoringIntegration:         merged.FFProctoringIntegration,
			FFCoCurricularTranscript:        merged.FFCoCurricularTranscript,
			FFEportfolio:                    merged.FFEportfolio,
			FFBookstoreIntegration:          merged.FFBookstoreIntegration,
			FFTranscripts:                   merged.FFTranscripts,
			FFWebhooks:                      merged.FFWebhooks,
			FFZapierConnector:               merged.FFZapierConnector,
			FFAdvisingIntegration:           merged.FFAdvisingIntegration,
			FFResearchConsent:               merged.FFResearchConsent,
			FFAccessibilityIntake:           merged.FFAccessibilityIntake,
			FFCEUTracking:                   merged.FFCEUTracking,
			FFConsortiumSharing:             merged.FFConsortiumSharing,
			FFSelfPacedMode:                 merged.FFSelfPacedMode,
			FFPublicCatalog:                 merged.FFPublicCatalog,
			FFPublicAPI:                     merged.FFPublicAPI,
			FFStripeBilling:                 merged.FFStripeBilling,
			FFPaymentsEnabled:               merged.FFPaymentsEnabled,
			FFRevenueShare:                  merged.FFRevenueShare,
			FFTaxCollection:                 merged.FFTaxCollection,
			FFLearningPaths:                 merged.FFLearningPaths,
			FFConditionalRelease:            merged.FFConditionalRelease,
			FFPeerReview:                    merged.FFPeerReview,
			FFCompletionCredentials:         merged.FFCompletionCredentials,
			FFCourseReviews:                 merged.FFCourseReviews,
			FFGamification:                  merged.FFGamification,
			FFOnboardingFlow:                merged.FFOnboardingFlow,
			FFStudyReminders:                merged.FFStudyReminders,
			FFAIStudyBuddy:                  merged.FFAIStudyBuddy,
			FFLessonGenerator:               merged.FFLessonGenerator,
			FFPersistentTutor:               merged.FFPersistentTutor,
			FFAPITokens:                     merged.FFAPITokens,
			FFBotSlack:                      merged.FFBotSlack,
			FFBotTeams:                      merged.FFBotTeams,
			FFBotDiscord:                    merged.FFBotDiscord,
			FFCalendarFeeds:                 merged.FFCalendarFeeds,
			FFRedisCache:                    merged.FFRedisCache,
			LRSAnonymizeActors:              merged.LRSAnonymizeActors,
			FERPAWorkflowEnabled:            merged.FERPAWorkflowEnabled,
			DPAPortalEnabled:                merged.DPAPortalEnabled,
			SOC2ModuleEnabled:               merged.SOC2ModuleEnabled,
			FFReadingPreferences:            merged.FFReadingPreferences,
			FFClassroomSignals:              merged.FFClassroomSignals,
			FFLibraryIntegration:            merged.FFLibraryIntegration,
			DiagnosticAssessmentsEnabled:    merged.DiagnosticAssessmentsEnabled,
			SRSPracticeEnabled:              merged.SRSPracticeEnabled,
			IRTCatModeEnabled:               merged.IRTCatModeEnabled,
			AdaptiveLearnerModelEnabled:     merged.AdaptiveLearnerModelEnabled,
			LearnerModelEMAAlpha:            merged.LearnerModelEMAAlpha,
			GDPRModuleEnabled:               merged.GDPRModuleEnabled,
			CCPAModuleEnabled:               merged.CCPAModuleEnabled,
			StatePrivacyEnabled:             merged.StatePrivacyEnabled,
			BackupModuleEnabled:             merged.BackupModuleEnabled,
			FFUiMode:                        merged.FFUiMode,
			MFAEnabled:                      merged.MFAEnabled,
			MFAEnforcement:                  merged.MFAEnforcement,
			SMTPHost:                        merged.SMTPHost,
			SMTPPort:                        int(merged.SMTPPort),
			SMTPFrom:                        merged.SMTPFrom,
			SMTPUser:                        merged.SMTPUser,
			SMTPPassword:                    smtpPasswordMasked(dbRow, merged.SMTPPassword),
			Sources: platformSourcesJSON{
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

	OIDCSSOEnabled                  *bool `json:"oidcSsoEnabled"`
	CleverSSOEnabled                *bool `json:"cleverSsoEnabled"`
	ClassLinkSSOEnabled             *bool `json:"classlinkSsoEnabled"`
	MagicLinkEnabled                *bool `json:"magicLinkEnabled"`
	MagicLinkEnrolledOnly           *bool `json:"magicLinkEnrolledOnly"`
	SessionManagementUIEnabled      *bool `json:"sessionManagementUiEnabled"`
	EmailNotificationsEnabled       *bool `json:"emailNotificationsEnabled"`
	PushNotificationsEnabled        *bool `json:"pushNotificationsEnabled"`
	VirtualClassroomEnabled         *bool `json:"virtualClassroomEnabled"`
	DRMEnabled                      *bool `json:"drmEnabled"`
	VideoTranscodingEnabled         *bool `json:"videoTranscodingEnabled"`
	AutoCaptioningEnabled           *bool `json:"autoCaptioningEnabled"`
	VideoCaptionsEnabled            *bool `json:"videoCaptionsEnabled"`
	StorageQuotasEnabled            *bool `json:"storageQuotasEnabled"`
	AtRiskAlertsEnabled             *bool `json:"atRiskAlertsEnabled"`
	AvScanningEnabled               *bool `json:"avScanningEnabled"`
	ClamAVStub                      *bool `json:"clamavStub"`
	H5PEnabled                      *bool `json:"h5pEnabled"`
	ScormIngestionEnabled           *bool `json:"scormIngestionEnabled"`
	OERLibraryEnabled               *bool `json:"oerLibraryEnabled"`
	OERStub                         *bool `json:"oerStub"`
	ItemAnalysisEnabled             *bool `json:"itemAnalysisEnabled"`
	StudentProgressEnabled          *bool `json:"studentProgressEnabled"`
	EngagementTrackingEnabled       *bool `json:"engagementTrackingEnabled"`
	SelfReflectionEnabled           *bool `json:"selfReflectionEnabled"`
	LearnerProfileEnabled           *bool `json:"learnerProfileEnabled"`
	LpAdaptRecommendationsEnabled   *bool `json:"lpAdaptRecommendationsEnabled"`
	LpAdaptReviewEnabled            *bool `json:"lpAdaptReviewEnabled"`
	LpAdaptModalityEnabled          *bool `json:"lpAdaptModalityEnabled"`
	LpAdaptTutorEnabled             *bool `json:"lpAdaptTutorEnabled"`
	IntroCourseEnabled              *bool `json:"introCourseEnabled"`
	OutcomesReportEnabled           *bool `json:"outcomesReportEnabled"`
	InstructorInsightsEnabled       *bool `json:"instructorInsightsEnabled"`
	XAPIEmissionEnabled             *bool `json:"xapiEmissionEnabled"`
	EquationEditorEnabled           *bool `json:"equationEditorEnabled"`
	ReadingLevelEnabled             *bool `json:"readingLevelEnabled"`
	GraderAgentEnabled              *bool `json:"graderAgentEnabled"`
	GraderAgentReviewInboxEnabled   *bool `json:"graderAgentReviewInboxEnabled"`
	GraderAgentSuggestModeEnabled      *bool `json:"graderAgentSuggestModeEnabled"`
	GraderAgentTextEntryGradingEnabled *bool `json:"graderAgentTextEntryGradingEnabled"`
	GraderAgentVisionGradingEnabled    *bool `json:"graderAgentVisionGradingEnabled"`
	GraderAgentRunFiltersEnabled       *bool `json:"graderAgentRunFiltersEnabled"`
	GraderAgentCostEstimateEnabled     *bool `json:"graderAgentCostEstimateEnabled"`
	GraderAgentCancelRunEnabled        *bool `json:"graderAgentCancelRunEnabled"`
	CodeExecutionEnabled            *bool `json:"codeExecutionEnabled"`
	AltTextEnforcementEnabled       *bool `json:"altTextEnforcementEnabled"`
	FFAltTextEnforcement            *bool `json:"ffAltTextEnforcement"`
	SpeechToTextEnabled             *bool `json:"speechToTextEnabled"`
	AccommodationsEngineEnabled     *bool `json:"accommodationsEngineEnabled"`
	FFAccommodationsEngine          *bool `json:"ffAccommodationsEngine"`
	ReadAloudEnabled                *bool `json:"readAloudEnabled"`
	FFReadAloud                     *bool `json:"ffReadAloud"`
	TranslationMemoryEnabled        *bool `json:"translationMemoryEnabled"`
	ReportExportEnabled             *bool `json:"reportExportEnabled"`
	CoppaWorkflowEnabled            *bool `json:"coppaWorkflowEnabled"`
	GDPRModuleEnabled               *bool `json:"gdprModuleEnabled"`
	CCPAModuleEnabled               *bool `json:"ccpaModuleEnabled"`
	StatePrivacyEnabled             *bool `json:"statePrivacyEnabled"`
	IsoIsmsEnabled                  *bool `json:"isoIsmsEnabled"`
	AdminAuditLogEnabled            *bool `json:"adminAuditLogEnabled"`
	AdminConsoleEnabled             *bool `json:"adminConsoleEnabled"`
	ImpersonationEnabled            *bool `json:"impersonationEnabled"`
	BulkCsvImportEnabled            *bool `json:"bulkCsvImportEnabled"`
	AdminSearchEnabled              *bool `json:"adminSearchEnabled"`
	EmailTemplateEditorEnabled      *bool `json:"emailTemplateEditorEnabled"`
	MaintenanceBannerEnabled        *bool `json:"maintenanceBannerEnabled"`
	CustomFieldsEnabled             *bool `json:"customFieldsEnabled"`
	SeatManagementEnabled           *bool `json:"seatManagementEnabled"`
	DataResidencyEnabled            *bool `json:"dataResidencyEnabled"`
	BackupModuleEnabled             *bool `json:"backupModuleEnabled"`
	RTLEnabled                      *bool `json:"rtlEnabled"`
	SecurityDisclosureModuleEnabled *bool `json:"securityDisclosureModuleEnabled"`
	FFUiMode                        *bool `json:"ffUiMode"`
	FFParentPortal                  *bool `json:"ffParentPortal"`
	FFParentPortalV2                *bool `json:"ffParentPortalV2"`
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
	FFWhatifGrades                  *bool `json:"ffWhatifGrades"`
	FFGradeCurving                  *bool `json:"ffGradeCurving"`
	FFPlagiarismChecks              *bool `json:"ffPlagiarismChecks"`
	FFIncompleteGradeWorkflow       *bool `json:"ffIncompleteGradeWorkflow"`
	FFAcademicCalendar              *bool `json:"ffAcademicCalendar"`
	FFCourseEvaluations             *bool `json:"ffCourseEvaluations"`
	FFProctoringIntegration         *bool `json:"ffProctoringIntegration"`
	FFCoCurricularTranscript        *bool `json:"ffCoCurricularTranscript"`
	FFEportfolio                    *bool `json:"ffEportfolio"`
	FFBookstoreIntegration          *bool `json:"ffBookstoreIntegration"`
	FFTranscripts                   *bool `json:"ffTranscripts"`
	FFWebhooks                      *bool `json:"ffWebhooks"`
	FFZapierConnector               *bool `json:"ffZapierConnector"`
	FFAdvisingIntegration           *bool `json:"ffAdvisingIntegration"`
	FFResearchConsent               *bool `json:"ffResearchConsent"`
	FFAccessibilityIntake           *bool `json:"ffAccessibilityIntake"`
	FFCEUTracking                   *bool `json:"ffCeuTracking"`
	FFConsortiumSharing             *bool `json:"ffConsortiumSharing"`
	FFSelfPacedMode                 *bool `json:"ffSelfPacedMode"`
	FFPublicCatalog                 *bool `json:"ffPublicCatalog"`
	FFPublicAPI                     *bool `json:"ffPublicApi"`
	FFStripeBilling                 *bool `json:"ffStripeBilling"`
	FFPaymentsEnabled               *bool `json:"ffPaymentsEnabled"`
	FFRevenueShare                  *bool `json:"ffRevenueShare"`
	FFTaxCollection                 *bool `json:"ffTaxCollection"`
	FFLearningPaths                 *bool `json:"ffLearningPaths"`
	FFConditionalRelease            *bool `json:"ffConditionalRelease"`
	FFPeerReview                    *bool `json:"ffPeerReview"`
	FFCompletionCredentials         *bool `json:"ffCompletionCredentials"`
	FFCourseReviews                 *bool `json:"ffCourseReviews"`
	FFGamification                  *bool `json:"ffGamification"`
	FFOnboardingFlow                *bool `json:"ffOnboardingFlow"`
	FFStudyReminders                *bool `json:"ffStudyReminders"`
	FFAIStudyBuddy                  *bool `json:"ffAiStudyBuddy"`
	FFLessonGenerator               *bool `json:"ffLessonGenerator"`
	FFPersistentTutor               *bool `json:"ffPersistentTutor"`
	FFAPITokens                     *bool `json:"ffApiTokens"`
	FFBotSlack                      *bool `json:"ffBotSlack"`
	FFBotTeams                      *bool `json:"ffBotTeams"`
	FFBotDiscord                    *bool `json:"ffBotDiscord"`
	FFCalendarFeeds                 *bool `json:"ffCalendarFeeds"`
	FFRedisCache                    *bool `json:"ffRedisCache"`

	LRSAnonymizeActors           *bool    `json:"lrsAnonymizeActors"`
	FERPAWorkflowEnabled         *bool    `json:"ferpaWorkflowEnabled"`
	DPAPortalEnabled             *bool    `json:"dpaPortalEnabled"`
	SOC2ModuleEnabled            *bool    `json:"soc2ModuleEnabled"`
	FFReadingPreferences         *bool    `json:"ffReadingPreferences"`
	FFClassroomSignals           *bool    `json:"ffClassroomSignals"`
	FFLibraryIntegration         *bool    `json:"ffLibraryIntegration"`
	DiagnosticAssessmentsEnabled *bool    `json:"diagnosticAssessmentsEnabled"`
	SRSPracticeEnabled           *bool    `json:"srsPracticeEnabled"`
	IRTCatModeEnabled            *bool    `json:"irtCatModeEnabled"`
	AdaptiveLearnerModelEnabled  *bool    `json:"adaptiveLearnerModelEnabled"`
	LearnerModelEMAAlpha         *float64 `json:"learnerModelEmaAlpha"`

	MFAEnabled     *bool   `json:"mfaEnabled"`
	MFAEnforcement *string `json:"mfaEnforcement"`

	SMTPHost          *string `json:"smtpHost"`
	SMTPPort          *int    `json:"smtpPort"`
	SMTPFrom          *string `json:"smtpFrom"`
	SMTPUser          *string `json:"smtpUser"`
	SMTPPassword      *string `json:"smtpPassword"`
	ClearSMTPPassword bool    `json:"clearSmtpPassword"`

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
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Database is not configured.")
			return
		}
		prevIntroCourseEnabled := d.effectiveConfig().IntroCourseEnabled
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
		clearSMTP := body.ClearSMTPPassword
		if len(mask) > 0 {
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
		if body.LearnerModelEMAAlpha != nil {
			if *body.LearnerModelEMAAlpha <= 0 || *body.LearnerModelEMAAlpha > 1 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "learnerModelEmaAlpha must be in the range (0, 1].")
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
		setBool("scormingestionenabled", body.ScormIngestionEnabled, func(v bool) { wr.ScormIngestionEnabled = &v })
		setBool("oerlibraryenabled", body.OERLibraryEnabled, func(v bool) { wr.OERLibraryEnabled = &v })
		setBool("oerstub", body.OERStub, func(v bool) { wr.OERStub = &v })
		setBool("itemanalysisenabled", body.ItemAnalysisEnabled, func(v bool) { wr.ItemAnalysisEnabled = &v })
		setBool("studentprogressenabled", body.StudentProgressEnabled, func(v bool) { wr.StudentProgressEnabled = &v })
		setBool("engagementtrackingenabled", body.EngagementTrackingEnabled, func(v bool) { wr.EngagementTrackingEnabled = &v })
		setBool("selfreflectionenabled", body.SelfReflectionEnabled, func(v bool) { wr.SelfReflectionEnabled = &v })
		setBool("learnerprofileenabled", body.LearnerProfileEnabled, func(v bool) { wr.LearnerProfileEnabled = &v })
		setBool("lpadaptrecommendationsenabled", body.LpAdaptRecommendationsEnabled, func(v bool) { wr.LpAdaptRecommendationsEnabled = &v })
		setBool("lpadaptreviewenabled", body.LpAdaptReviewEnabled, func(v bool) { wr.LpAdaptReviewEnabled = &v })
		setBool("lpadaptmodalityenabled", body.LpAdaptModalityEnabled, func(v bool) { wr.LpAdaptModalityEnabled = &v })
		setBool("lpadapttutorenabled", body.LpAdaptTutorEnabled, func(v bool) { wr.LpAdaptTutorEnabled = &v })
		setBool("introcourseenabled", body.IntroCourseEnabled, func(v bool) { wr.IntroCourseEnabled = &v })
		setBool("outcomesreportenabled", body.OutcomesReportEnabled, func(v bool) { wr.OutcomesReportEnabled = &v })
		setBool("instructorinsightsenabled", body.InstructorInsightsEnabled, func(v bool) { wr.InstructorInsightsEnabled = &v })
		setBool("equationeditorenabled", body.EquationEditorEnabled, func(v bool) { wr.EquationEditorEnabled = &v })
		setBool("readinglevelenabled", body.ReadingLevelEnabled, func(v bool) { wr.ReadingLevelEnabled = &v })
		setBool("graderagentenabled", body.GraderAgentEnabled, func(v bool) { wr.GraderAgentEnabled = &v })
		setBool("graderagentreviewinboxenabled", body.GraderAgentReviewInboxEnabled, func(v bool) { wr.GraderAgentReviewInboxEnabled = &v })
		setBool("graderagentsuggestmodeenabled", body.GraderAgentSuggestModeEnabled, func(v bool) { wr.GraderAgentSuggestModeEnabled = &v })
		setBool("graderagenttextentrygradingenabled", body.GraderAgentTextEntryGradingEnabled, func(v bool) { wr.GraderAgentTextEntryGradingEnabled = &v })
		setBool("graderagentvisiongradingenabled", body.GraderAgentVisionGradingEnabled, func(v bool) { wr.GraderAgentVisionGradingEnabled = &v })
		setBool("graderagentrunfiltersenabled", body.GraderAgentRunFiltersEnabled, func(v bool) { wr.GraderAgentRunFiltersEnabled = &v })
		setBool("graderagentcostestimateenabled", body.GraderAgentCostEstimateEnabled, func(v bool) { wr.GraderAgentCostEstimateEnabled = &v })
		setBool("graderagentcancelrunenabled", body.GraderAgentCancelRunEnabled, func(v bool) { wr.GraderAgentCancelRunEnabled = &v })
		setBool("codeexecutionenabled", body.CodeExecutionEnabled, func(v bool) { wr.CodeExecutionEnabled = &v })
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
		setBool("gdprmoduleenabled", body.GDPRModuleEnabled, func(v bool) { wr.GDPRModuleEnabled = &v })
		setBool("ccpamoduleenabled", body.CCPAModuleEnabled, func(v bool) { wr.CCPAModuleEnabled = &v })
		setBool("stateprivacyenabled", body.StatePrivacyEnabled, func(v bool) { wr.StatePrivacyEnabled = &v })
		setBool("backupmoduleenabled", body.BackupModuleEnabled, func(v bool) { wr.BackupModuleEnabled = &v })
		setBool("originalitydetectionenabled", body.OriginalityDetectionEnabled, func(v bool) { wr.OriginalityDetectionEnabled = &v })
		setBool("originalitystubexternal", body.OriginalityStubExternal, func(v bool) { wr.OriginalityStubExternal = &v })
		setBool("ffuimode", body.FFUiMode, func(v bool) { wr.FFUiMode = &v })
		setBool("isoismsenabled", body.IsoIsmsEnabled, func(v bool) { wr.IsoIsmsEnabled = &v })
		setBool("adminauditlogenabled", body.AdminAuditLogEnabled, func(v bool) { wr.AdminAuditLogEnabled = &v })
		setBool("adminconsoleenabled", body.AdminConsoleEnabled, func(v bool) { wr.AdminConsoleEnabled = &v })
		setBool("impersonationenabled", body.ImpersonationEnabled, func(v bool) { wr.ImpersonationEnabled = &v })
		setBool("bulkcsvimportenabled", body.BulkCsvImportEnabled, func(v bool) { wr.BulkCsvImportEnabled = &v })
		setBool("adminsearchenabled", body.AdminSearchEnabled, func(v bool) { wr.AdminSearchEnabled = &v })
		setBool("emailtemplateeditorenabled", body.EmailTemplateEditorEnabled, func(v bool) { wr.EmailTemplateEditorEnabled = &v })
		setBool("maintenancebannerenabled", body.MaintenanceBannerEnabled, func(v bool) { wr.MaintenanceBannerEnabled = &v })
		setBool("customfieldsenabled", body.CustomFieldsEnabled, func(v bool) { wr.CustomFieldsEnabled = &v })
		setBool("seatmanagementenabled", body.SeatManagementEnabled, func(v bool) { wr.SeatManagementEnabled = &v })
		setBool("dataresidencyenabled", body.DataResidencyEnabled, func(v bool) { wr.DataResidencyEnabled = &v })
		setBool("rtlenabled", body.RTLEnabled, func(v bool) { wr.RTLEnabled = &v })
		setBool("securitydisclosuremoduleenabled", body.SecurityDisclosureModuleEnabled, func(v bool) { wr.SecurityDisclosureModuleEnabled = &v })
		setBool("ffparentportal", body.FFParentPortal, func(v bool) { wr.FFParentPortal = &v })
		setBool("ffparentportalv2", body.FFParentPortalV2, func(v bool) { wr.FFParentPortalV2 = &v })
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
		setBool("ffwhatifgrades", body.FFWhatifGrades, func(v bool) { wr.FFWhatifGrades = &v })
		setBool("ffgradecurving", body.FFGradeCurving, func(v bool) { wr.FFGradeCurving = &v })
		setBool("ffplagiarismchecks", body.FFPlagiarismChecks, func(v bool) { wr.FFPlagiarismChecks = &v })
		setBool("ffincompletegradeworkflow", body.FFIncompleteGradeWorkflow, func(v bool) { wr.FFIncompleteGradeWorkflow = &v })
		setBool("ffacademiccalendar", body.FFAcademicCalendar, func(v bool) { wr.FFAcademicCalendar = &v })
		setBool("ffcourseevaluations", body.FFCourseEvaluations, func(v bool) { wr.FFCourseEvaluations = &v })
		setBool("ffproctoringintegration", body.FFProctoringIntegration, func(v bool) { wr.FFProctoringIntegration = &v })
		setBool("ffcocurriculartranscript", body.FFCoCurricularTranscript, func(v bool) { wr.FFCoCurricularTranscript = &v })
		setBool("ffeportfolio", body.FFEportfolio, func(v bool) { wr.FFEportfolio = &v })
		setBool("ffbookstoreintegration", body.FFBookstoreIntegration, func(v bool) { wr.FFBookstoreIntegration = &v })
		setBool("fftranscripts", body.FFTranscripts, func(v bool) { wr.FFTranscripts = &v })
		setBool("ffwebhooks", body.FFWebhooks, func(v bool) { wr.FFWebhooks = &v })
		setBool("ffzapierconnector", body.FFZapierConnector, func(v bool) { wr.FFZapierConnector = &v })
		setBool("ffadvisingintegration", body.FFAdvisingIntegration, func(v bool) { wr.FFAdvisingIntegration = &v })
		setBool("ffresearchconsent", body.FFResearchConsent, func(v bool) { wr.FFResearchConsent = &v })
		setBool("ffaccessibilityintake", body.FFAccessibilityIntake, func(v bool) { wr.FFAccessibilityIntake = &v })
		setBool("ffceutracking", body.FFCEUTracking, func(v bool) { wr.FFCEUTracking = &v })
		setBool("ffconsortiumsharing", body.FFConsortiumSharing, func(v bool) { wr.FFConsortiumSharing = &v })
		setBool("ffselfpacedmode", body.FFSelfPacedMode, func(v bool) { wr.FFSelfPacedMode = &v })
		setBool("ffpubliccatalog", body.FFPublicCatalog, func(v bool) { wr.FFPublicCatalog = &v })
		setBool("ffpublicapi", body.FFPublicAPI, func(v bool) { wr.FFPublicAPI = &v })
		setBool("ffstripebilling", body.FFStripeBilling, func(v bool) { wr.FFStripeBilling = &v })
		setBool("ffpaymentsenabled", body.FFPaymentsEnabled, func(v bool) { wr.FFPaymentsEnabled = &v })
		setBool("ffrevenueshare", body.FFRevenueShare, func(v bool) { wr.FFRevenueShare = &v })
		setBool("fftaxcollection", body.FFTaxCollection, func(v bool) { wr.FFTaxCollection = &v })
		setBool("fflearningpaths", body.FFLearningPaths, func(v bool) { wr.FFLearningPaths = &v })
		setBool("ffconditionalrelease", body.FFConditionalRelease, func(v bool) { wr.FFConditionalRelease = &v })
		setBool("ffpeerreview", body.FFPeerReview, func(v bool) { wr.FFPeerReview = &v })
		setBool("ffcompletioncredentials", body.FFCompletionCredentials, func(v bool) { wr.FFCompletionCredentials = &v })
		setBool("ffcoursereviews", body.FFCourseReviews, func(v bool) { wr.FFCourseReviews = &v })
		setBool("ffgamification", body.FFGamification, func(v bool) { wr.FFGamification = &v })
		setBool("ffonboardingflow", body.FFOnboardingFlow, func(v bool) { wr.FFOnboardingFlow = &v })
		setBool("ffstudyreminders", body.FFStudyReminders, func(v bool) { wr.FFStudyReminders = &v })
		setBool("ffaistudybuddy", body.FFAIStudyBuddy, func(v bool) { wr.FFAIStudyBuddy = &v })
		setBool("fflessongenerator", body.FFLessonGenerator, func(v bool) { wr.FFLessonGenerator = &v })
		setBool("ffpersistenttutor", body.FFPersistentTutor, func(v bool) { wr.FFPersistentTutor = &v })
		setBool("ffapitokens", body.FFAPITokens, func(v bool) { wr.FFAPITokens = &v })
		setBool("ffbotslack", body.FFBotSlack, func(v bool) { wr.FFBotSlack = &v })
		setBool("ffbotteams", body.FFBotTeams, func(v bool) { wr.FFBotTeams = &v })
		setBool("ffbotdiscord", body.FFBotDiscord, func(v bool) { wr.FFBotDiscord = &v })
		setBool("ffcalendarfeeds", body.FFCalendarFeeds, func(v bool) { wr.FFCalendarFeeds = &v })
		setBool("ffrediscache", body.FFRedisCache, func(v bool) { wr.FFRedisCache = &v })
		setBool("lrsanonymizeactors", body.LRSAnonymizeActors, func(v bool) { wr.LRSAnonymizeActors = &v })
		setBool("ferpaworkflowenabled", body.FERPAWorkflowEnabled, func(v bool) { wr.FERPAWorkflowEnabled = &v })
		setBool("dpaportalenabled", body.DPAPortalEnabled, func(v bool) { wr.DPAPortalEnabled = &v })
		setBool("soc2moduleenabled", body.SOC2ModuleEnabled, func(v bool) { wr.SOC2ModuleEnabled = &v })
		setBool("ffreadingpreferences", body.FFReadingPreferences, func(v bool) { wr.FFReadingPreferences = &v })
		setBool("ffclassroomsignals", body.FFClassroomSignals, func(v bool) { wr.FFClassroomSignals = &v })
		setBool("fflibraryintegration", body.FFLibraryIntegration, func(v bool) { wr.FFLibraryIntegration = &v })
		setBool("diagnosticassessmentsenabled", body.DiagnosticAssessmentsEnabled, func(v bool) { wr.DiagnosticAssessmentsEnabled = &v })
		setBool("srspracticeenabled", body.SRSPracticeEnabled, func(v bool) { wr.SRSPracticeEnabled = &v })
		setBool("irtcatmodeenabled", body.IRTCatModeEnabled, func(v bool) { wr.IRTCatModeEnabled = &v })
		setBool("adaptivelearnermodelenabled", body.AdaptiveLearnerModelEnabled, func(v bool) { wr.AdaptiveLearnerModelEnabled = &v })
		set("learnermodelemaalpha", body.LearnerModelEMAAlpha != nil, func() {
			v := *body.LearnerModelEMAAlpha
			wr.LearnerModelEMAAlpha = &v
		})
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
		if wr.IntroCourseEnabled != nil && merged.IntroCourseEnabled != prevIntroCourseEnabled {
			d.recordIntroCourseAdminAudit(r, actorID, "toggle_flag", map[string]any{
				"action":  "toggle_flag",
				"enabled": merged.IntroCourseEnabled,
			})
		}

		sources := platformconfig.ResolveSources(d.Config, dbRow)
		out := platformSettingsJSON{
			SAMLSSOEnabled:                  merged.SAMLSSOEnabled,
			SAMLPublicBaseURL:               merged.SAMLPublicBaseURL,
			SAMLSPEntityID:                  merged.SAMLSPEntityID,
			SAMLSPX509PEM:                   merged.SAMLSPX509PEM,
			SAMLSPPrivateKeyPEM:             maskPEMIfSet(merged.SAMLSPPrivateKeyPEM),
			AnnotationEnabled:               merged.AnnotationEnabled,
			FeedbackMediaEnabled:            merged.FeedbackMediaEnabled,
			BlindGradingEnabled:             merged.BlindGradingEnabled,
			ModeratedGradingEnabled:         merged.ModeratedGradingEnabled,
			OriginalityDetectionEnabled:     merged.OriginalityDetectionEnabled,
			OriginalityStubExternal:         merged.OriginalityStubExternal,
			GradePostingPoliciesEnabled:     merged.GradePostingPoliciesEnabled,
			GradebookCSVEnabled:             merged.GradebookCSVEnabled,
			ResubmissionWorkflowEnabled:     merged.ResubmissionWorkflowEnabled,
			LTIEnabled:                      merged.LTIEnabled,
			OneRosterEnabled:                merged.OneRosterEnabled,
			ScimEnabled:                     merged.ScimEnabled,
			OIDCSSOEnabled:                  merged.OIDCSSOEnabled,
			CleverSSOEnabled:                merged.CleverSSOEnabled,
			ClassLinkSSOEnabled:             merged.ClassLinkSSOEnabled,
			MagicLinkEnabled:                merged.MagicLinkEnabled,
			MagicLinkEnrolledOnly:           merged.MagicLinkEnrolledOnly,
			SessionManagementUIEnabled:      merged.SessionManagementUIEnabled,
			EmailNotificationsEnabled:       merged.EmailNotificationsEnabled,
			PushNotificationsEnabled:        merged.PushNotificationsEnabled,
			VirtualClassroomEnabled:         merged.VirtualClassroomEnabled,
			DRMEnabled:                      merged.DRMEnabled,
			VideoTranscodingEnabled:         merged.VideoTranscodingEnabled,
			AutoCaptioningEnabled:           merged.AutoCaptioningEnabled,
			VideoCaptionsEnabled:            merged.VideoCaptionsEnabled,
			StorageQuotasEnabled:            merged.StorageQuotasEnabled,
			AtRiskAlertsEnabled:             merged.AtRiskAlertsEnabled,
			AvScanningEnabled:               merged.AvScanningEnabled,
			ClamAVStub:                      merged.ClamAVStub,
			H5PEnabled:                      merged.H5PEnabled,
			ScormIngestionEnabled:           merged.ScormIngestionEnabled,
			OERLibraryEnabled:               merged.OERLibraryEnabled,
			OERStub:                         merged.OERStub,
			ItemAnalysisEnabled:             merged.ItemAnalysisEnabled,
			StudentProgressEnabled:          merged.StudentProgressEnabled,
			EngagementTrackingEnabled:       merged.EngagementTrackingEnabled,
			SelfReflectionEnabled:           merged.SelfReflectionEnabled,
			LearnerProfileEnabled:           merged.LearnerProfileEnabled,
			LpAdaptRecommendationsEnabled:   merged.LpAdaptRecommendationsEnabled,
			LpAdaptReviewEnabled:            merged.LpAdaptReviewEnabled,
			LpAdaptModalityEnabled:          merged.LpAdaptModalityEnabled,
			LpAdaptTutorEnabled:             merged.LpAdaptTutorEnabled,
			IntroCourseEnabled:              merged.IntroCourseEnabled,
			OutcomesReportEnabled:           merged.OutcomesReportEnabled,
			InstructorInsightsEnabled:       merged.InstructorInsightsEnabled,
			XAPIEmissionEnabled:             merged.XAPIEmissionEnabled,
			EquationEditorEnabled:           merged.EquationEditorEnabled,
			ReadingLevelEnabled:             merged.ReadingLevelEnabled,
			GraderAgentEnabled:              merged.GraderAgentEnabled,
			GraderAgentReviewInboxEnabled:   merged.GraderAgentReviewInboxEnabled,
			GraderAgentSuggestModeEnabled:      merged.GraderAgentSuggestModeEnabled,
			GraderAgentTextEntryGradingEnabled: merged.GraderAgentTextEntryGradingEnabled,
			GraderAgentVisionGradingEnabled:    merged.GraderAgentVisionGradingEnabled,
			GraderAgentRunFiltersEnabled:       merged.GraderAgentRunFiltersEnabled,
			GraderAgentCostEstimateEnabled:     merged.GraderAgentCostEstimateEnabled,
			GraderAgentCancelRunEnabled:        merged.GraderAgentCancelRunEnabled,
			CodeExecutionEnabled:            merged.CodeExecutionEnabled,
			AltTextEnforcementEnabled:       merged.AltTextEnforcementEnabled,
			FFAltTextEnforcement:            merged.FFAltTextEnforcement,
			SpeechToTextEnabled:             merged.SpeechToTextEnabled,
			AccommodationsEngineEnabled:     merged.AccommodationsEngineEnabled,
			FFAccommodationsEngine:          merged.FFAccommodationsEngine,
			ReadAloudEnabled:                merged.ReadAloudEnabled,
			FFReadAloud:                     merged.FFReadAloud,
			TranslationMemoryEnabled:        merged.TranslationMemoryEnabled,
			ReportExportEnabled:             merged.ReportExportEnabled,
			CoppaWorkflowEnabled:            merged.CoppaWorkflowEnabled,
			IsoIsmsEnabled:                  merged.IsoIsmsEnabled,
			AdminAuditLogEnabled:            merged.AdminAuditLogEnabled,
			AdminConsoleEnabled:             merged.AdminConsoleEnabled,
			ImpersonationEnabled:            merged.ImpersonationEnabled,
			BulkCsvImportEnabled:            merged.BulkCsvImportEnabled,
			AdminSearchEnabled:              merged.AdminSearchEnabled,
			EmailTemplateEditorEnabled:      merged.EmailTemplateEditorEnabled,
			MaintenanceBannerEnabled:        merged.MaintenanceBannerEnabled,
			CustomFieldsEnabled:             merged.CustomFieldsEnabled,
			SeatManagementEnabled:           merged.SeatManagementEnabled,
			DataResidencyEnabled:            merged.DataResidencyEnabled,
			RTLEnabled:                      merged.RTLEnabled,
			SecurityDisclosureModuleEnabled: merged.SecurityDisclosureModuleEnabled,
			FFParentPortal:                  merged.FFParentPortal,
			FFParentPortalV2:                merged.FFParentPortalV2,
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
			FFWhatifGrades:                  merged.FFWhatifGrades,
			FFGradeCurving:                  merged.FFGradeCurving,
			FFPlagiarismChecks:              merged.FFPlagiarismChecks,
			FFIncompleteGradeWorkflow:       merged.FFIncompleteGradeWorkflow,
			FFAcademicCalendar:              merged.FFAcademicCalendar,
			FFCourseEvaluations:             merged.FFCourseEvaluations,
			FFProctoringIntegration:         merged.FFProctoringIntegration,
			FFCoCurricularTranscript:        merged.FFCoCurricularTranscript,
			FFEportfolio:                    merged.FFEportfolio,
			FFBookstoreIntegration:          merged.FFBookstoreIntegration,
			FFTranscripts:                   merged.FFTranscripts,
			FFWebhooks:                      merged.FFWebhooks,
			FFZapierConnector:               merged.FFZapierConnector,
			FFAdvisingIntegration:           merged.FFAdvisingIntegration,
			FFResearchConsent:               merged.FFResearchConsent,
			FFAccessibilityIntake:           merged.FFAccessibilityIntake,
			FFCEUTracking:                   merged.FFCEUTracking,
			FFConsortiumSharing:             merged.FFConsortiumSharing,
			FFSelfPacedMode:                 merged.FFSelfPacedMode,
			FFPublicCatalog:                 merged.FFPublicCatalog,
			FFPublicAPI:                     merged.FFPublicAPI,
			FFStripeBilling:                 merged.FFStripeBilling,
			FFPaymentsEnabled:               merged.FFPaymentsEnabled,
			FFRevenueShare:                  merged.FFRevenueShare,
			FFTaxCollection:                 merged.FFTaxCollection,
			FFLearningPaths:                 merged.FFLearningPaths,
			FFConditionalRelease:            merged.FFConditionalRelease,
			FFPeerReview:                    merged.FFPeerReview,
			FFCompletionCredentials:         merged.FFCompletionCredentials,
			FFCourseReviews:                 merged.FFCourseReviews,
			FFGamification:                  merged.FFGamification,
			FFOnboardingFlow:                merged.FFOnboardingFlow,
			FFStudyReminders:                merged.FFStudyReminders,
			FFAIStudyBuddy:                  merged.FFAIStudyBuddy,
			FFLessonGenerator:               merged.FFLessonGenerator,
			FFPersistentTutor:               merged.FFPersistentTutor,
			FFAPITokens:                     merged.FFAPITokens,
			FFBotSlack:                      merged.FFBotSlack,
			FFBotTeams:                      merged.FFBotTeams,
			FFBotDiscord:                    merged.FFBotDiscord,
			FFCalendarFeeds:                 merged.FFCalendarFeeds,
			FFRedisCache:                    merged.FFRedisCache,
			LRSAnonymizeActors:              merged.LRSAnonymizeActors,
			FERPAWorkflowEnabled:            merged.FERPAWorkflowEnabled,
			DPAPortalEnabled:                merged.DPAPortalEnabled,
			SOC2ModuleEnabled:               merged.SOC2ModuleEnabled,
			FFReadingPreferences:            merged.FFReadingPreferences,
			FFClassroomSignals:              merged.FFClassroomSignals,
			FFLibraryIntegration:            merged.FFLibraryIntegration,
			DiagnosticAssessmentsEnabled:    merged.DiagnosticAssessmentsEnabled,
			SRSPracticeEnabled:              merged.SRSPracticeEnabled,
			IRTCatModeEnabled:               merged.IRTCatModeEnabled,
			AdaptiveLearnerModelEnabled:     merged.AdaptiveLearnerModelEnabled,
			LearnerModelEMAAlpha:            merged.LearnerModelEMAAlpha,
			GDPRModuleEnabled:               merged.GDPRModuleEnabled,
			CCPAModuleEnabled:               merged.CCPAModuleEnabled,
			StatePrivacyEnabled:             merged.StatePrivacyEnabled,
			BackupModuleEnabled:             merged.BackupModuleEnabled,
			FFUiMode:                        merged.FFUiMode,
			MFAEnabled:                      merged.MFAEnabled,
			MFAEnforcement:                  merged.MFAEnforcement,
			SMTPHost:                        merged.SMTPHost,
			SMTPPort:                        int(merged.SMTPPort),
			SMTPFrom:                        merged.SMTPFrom,
			SMTPUser:                        merged.SMTPUser,
			SMTPPassword:                    smtpPasswordMasked(dbRow, merged.SMTPPassword),
			Sources: platformSourcesJSON{
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
