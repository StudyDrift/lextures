// Package platformconfig stores optional overrides for env-driven app settings (singleton row).
package platformconfig

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

// Row is the optional DB override layer; nil boolean pointers use documented defaults (see Defaults).
type Row struct {
	OpenRouterAPIKey *string

	SAMLSSOEnabled      *bool
	SAMLPublicBaseURL   *string
	SAMLSPEntityID      *string
	SAMLSPX509PEM       *string
	SAMLSPPrivateKeyPEM *string

	AnnotationEnabled           *bool
	FeedbackMediaEnabled        *bool
	BlindGradingEnabled         *bool
	ModeratedGradingEnabled     *bool
	OriginalityDetectionEnabled *bool
	OriginalityStubExternal     *bool
	GradePostingPoliciesEnabled *bool
	GradebookCSVEnabled         *bool
	ResubmissionWorkflowEnabled *bool
	LTIEnabled                  *bool
	OneRosterEnabled            *bool
	ScimEnabled                 *bool

	OIDCSSOEnabled             *bool
	CleverSSOEnabled           *bool
	ClassLinkSSOEnabled        *bool
	MagicLinkEnabled           *bool
	MagicLinkEnrolledOnly      *bool
	SessionManagementUIEnabled *bool
	EmailNotificationsEnabled  *bool
	PushNotificationsEnabled   *bool
	VirtualClassroomEnabled    *bool
	DRMEnabled                 *bool
	VideoTranscodingEnabled    *bool
	AutoCaptioningEnabled      *bool
	StorageQuotasEnabled       *bool
	AtRiskAlertsEnabled        *bool
	AvScanningEnabled          *bool
	ClamAVStub                 *bool
	H5PEnabled                 *bool
	OERLibraryEnabled          *bool
	OERStub                    *bool
	ItemAnalysisEnabled        *bool
	StudentProgressEnabled     *bool
	EngagementTrackingEnabled  *bool
	SelfReflectionEnabled      *bool
	OutcomesReportEnabled      *bool
	EquationEditorEnabled      *bool
	ReportExportEnabled        *bool
	XAPIEmissionEnabled        *bool
	InstructorInsightsEnabled  *bool
	CoppaWorkflowEnabled       *bool
	GDPRModuleEnabled          *bool
	CCPAModuleEnabled          *bool
	StatePrivacyEnabled        *bool
	IsoIsmsEnabled             *bool
	AdminAuditLogEnabled             *bool
	DataResidencyEnabled             *bool
	SecurityDisclosureModuleEnabled  *bool

	MFAEnabled     *bool
	MFAEnforcement *string

	SMTPHost               *string
	SMTPPort               *int32
	SMTPFrom               *string
	SMTPUser               *string
	SMTPPasswordCiphertext []byte

	UpdatedAt time.Time
}

// Write is the upsert payload (nil pointer = leave column unchanged).
type Write struct {
	OpenRouterAPIKey *string

	SAMLSSOEnabled      *bool
	SAMLPublicBaseURL   *string
	SAMLSPEntityID      *string
	SAMLSPX509PEM       *string
	SAMLSPPrivateKeyPEM *string

	AnnotationEnabled           *bool
	FeedbackMediaEnabled        *bool
	BlindGradingEnabled         *bool
	ModeratedGradingEnabled     *bool
	OriginalityDetectionEnabled *bool
	OriginalityStubExternal     *bool
	GradePostingPoliciesEnabled *bool
	GradebookCSVEnabled         *bool
	ResubmissionWorkflowEnabled *bool
	LTIEnabled                  *bool
	OneRosterEnabled            *bool
	ScimEnabled                 *bool

	OIDCSSOEnabled             *bool
	CleverSSOEnabled           *bool
	ClassLinkSSOEnabled        *bool
	MagicLinkEnabled           *bool
	MagicLinkEnrolledOnly      *bool
	SessionManagementUIEnabled *bool
	EmailNotificationsEnabled  *bool
	PushNotificationsEnabled   *bool
	VirtualClassroomEnabled    *bool
	DRMEnabled                 *bool
	VideoTranscodingEnabled    *bool
	AutoCaptioningEnabled      *bool
	StorageQuotasEnabled       *bool
	AtRiskAlertsEnabled        *bool
	AvScanningEnabled          *bool
	ClamAVStub                 *bool
	H5PEnabled                 *bool
	OERLibraryEnabled          *bool
	OERStub                    *bool
	ItemAnalysisEnabled        *bool
	StudentProgressEnabled     *bool
	EngagementTrackingEnabled  *bool
	SelfReflectionEnabled      *bool
	OutcomesReportEnabled      *bool
	EquationEditorEnabled      *bool
	ReportExportEnabled        *bool
	XAPIEmissionEnabled        *bool
	InstructorInsightsEnabled  *bool
	CoppaWorkflowEnabled       *bool
	GDPRModuleEnabled          *bool
	CCPAModuleEnabled          *bool
	StatePrivacyEnabled        *bool
	IsoIsmsEnabled             *bool
	AdminAuditLogEnabled            *bool
	DataResidencyEnabled            *bool
	SecurityDisclosureModuleEnabled *bool

	MFAEnabled     *bool
	MFAEnforcement *string

	SMTPHost               *string
	SMTPPort               *int32
	SMTPFrom               *string
	SMTPUser               *string
	SMTPPasswordCiphertext *[]byte // nil = leave unchanged in Upsert
}

// Get returns the singleton row or (nil, nil) if missing.
func Get(ctx context.Context, pool *pgxpool.Pool) (*Row, error) {
	var r Row
	err := pool.QueryRow(ctx, `
SELECT
	openrouter_api_key,
	saml_sso_enabled,
	saml_public_base_url,
	saml_sp_entity_id,
	saml_sp_x509_pem,
	saml_sp_private_key_pem,
	annotation_enabled,
	feedback_media_enabled,
	blind_grading_enabled,
	moderated_grading_enabled,
	originality_detection_enabled,
	originality_stub_external,
	grade_posting_policies_enabled,
	gradebook_csv_enabled,
	resubmission_workflow_enabled,
	lti_enabled,
	oneroster_enabled,
	scim_enabled,
	oidc_sso_enabled,
	clever_sso_enabled,
	classlink_sso_enabled,
	magic_link_enabled,
	magic_link_enrolled_only,
	session_management_ui_enabled,
	email_notifications_enabled,
	push_notifications_enabled,
	virtual_classroom_enabled,
	drm_enabled,
	video_transcoding_enabled,
	auto_captioning_enabled,
	storage_quotas_enabled,
	at_risk_alerts_enabled,
	av_scanning_enabled,
	clamav_stub,
	h5p_enabled,
	oer_library_enabled,
	oer_stub,
	item_analysis_enabled,
	student_progress_enabled,
	engagement_tracking_enabled,
	self_reflection_enabled,
	outcomes_report_enabled,
	equation_editor_enabled,
	report_export_enabled,
	xapi_emission_enabled,
	instructor_insights_enabled,
	coppa_workflow_enabled,
	gdpr_module_enabled,
	ccpa_module_enabled,
	state_privacy_enabled,
	iso_isms_enabled,
	admin_audit_log_enabled,
	data_residency_enabled,
	security_disclosure_module_enabled,
	mfa_enabled,
	mfa_enforcement,
	smtp_host,
	smtp_port,
	smtp_from,
	smtp_user,
	smtp_password_ciphertext,
	updated_at
FROM settings.platform_app_settings
WHERE id = 1
`).Scan(
		&r.OpenRouterAPIKey,
		&r.SAMLSSOEnabled,
		&r.SAMLPublicBaseURL,
		&r.SAMLSPEntityID,
		&r.SAMLSPX509PEM,
		&r.SAMLSPPrivateKeyPEM,
		&r.AnnotationEnabled,
		&r.FeedbackMediaEnabled,
		&r.BlindGradingEnabled,
		&r.ModeratedGradingEnabled,
		&r.OriginalityDetectionEnabled,
		&r.OriginalityStubExternal,
		&r.GradePostingPoliciesEnabled,
		&r.GradebookCSVEnabled,
		&r.ResubmissionWorkflowEnabled,
		&r.LTIEnabled,
		&r.OneRosterEnabled,
		&r.ScimEnabled,
		&r.OIDCSSOEnabled,
		&r.CleverSSOEnabled,
		&r.ClassLinkSSOEnabled,
		&r.MagicLinkEnabled,
		&r.MagicLinkEnrolledOnly,
		&r.SessionManagementUIEnabled,
		&r.EmailNotificationsEnabled,
		&r.PushNotificationsEnabled,
		&r.VirtualClassroomEnabled,
		&r.DRMEnabled,
		&r.VideoTranscodingEnabled,
		&r.AutoCaptioningEnabled,
		&r.StorageQuotasEnabled,
		&r.AtRiskAlertsEnabled,
		&r.AvScanningEnabled,
		&r.ClamAVStub,
		&r.H5PEnabled,
		&r.OERLibraryEnabled,
		&r.OERStub,
		&r.ItemAnalysisEnabled,
		&r.StudentProgressEnabled,
		&r.EngagementTrackingEnabled,
		&r.SelfReflectionEnabled,
		&r.OutcomesReportEnabled,
		&r.EquationEditorEnabled,
		&r.ReportExportEnabled,
		&r.XAPIEmissionEnabled,
		&r.InstructorInsightsEnabled,
		&r.CoppaWorkflowEnabled,
		&r.GDPRModuleEnabled,
		&r.CCPAModuleEnabled,
		&r.StatePrivacyEnabled,
		&r.IsoIsmsEnabled,
		&r.AdminAuditLogEnabled,
		&r.DataResidencyEnabled,
		&r.SecurityDisclosureModuleEnabled,
		&r.MFAEnabled,
		&r.MFAEnforcement,
		&r.SMTPHost,
		&r.SMTPPort,
		&r.SMTPFrom,
		&r.SMTPUser,
		&r.SMTPPasswordCiphertext,
		&r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ClearOpenRouterAPIKey removes the stored OpenRouter override so the environment key is used again.
func ClearOpenRouterAPIKey(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET openrouter_api_key = NULL, updated_at = NOW()
WHERE id = 1
`)
	return err
}

// ClearSMTPPassword removes the stored encrypted SMTP password override.
func ClearSMTPPassword(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET smtp_password_ciphertext = NULL, updated_at = NOW()
WHERE id = 1
`)
	return err
}

// Upsert applies non-nil fields in w to the singleton row (COALESCE keeps existing values).
func Upsert(ctx context.Context, pool *pgxpool.Pool, w *Write) (*Row, error) {
	var smtpPort any
	if w.SMTPPort != nil {
		smtpPort = *w.SMTPPort
	}
	var smtpCipher any
	if w.SMTPPasswordCiphertext != nil {
		smtpCipher = *w.SMTPPasswordCiphertext
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.platform_app_settings (
	id,
	openrouter_api_key,
	saml_sso_enabled,
	saml_public_base_url,
	saml_sp_entity_id,
	saml_sp_x509_pem,
	saml_sp_private_key_pem,
	annotation_enabled,
	feedback_media_enabled,
	blind_grading_enabled,
	moderated_grading_enabled,
	originality_detection_enabled,
	originality_stub_external,
	grade_posting_policies_enabled,
	gradebook_csv_enabled,
	resubmission_workflow_enabled,
	lti_enabled,
	oneroster_enabled,
	scim_enabled,
	oidc_sso_enabled,
	clever_sso_enabled,
	classlink_sso_enabled,
	magic_link_enabled,
	magic_link_enrolled_only,
	session_management_ui_enabled,
	email_notifications_enabled,
	push_notifications_enabled,
	virtual_classroom_enabled,
	drm_enabled,
	video_transcoding_enabled,
	auto_captioning_enabled,
	storage_quotas_enabled,
	at_risk_alerts_enabled,
	av_scanning_enabled,
	clamav_stub,
	h5p_enabled,
	oer_library_enabled,
	oer_stub,
	item_analysis_enabled,
	student_progress_enabled,
	engagement_tracking_enabled,
	self_reflection_enabled,
	outcomes_report_enabled,
	equation_editor_enabled,
	report_export_enabled,
	xapi_emission_enabled,
	instructor_insights_enabled,
	coppa_workflow_enabled,
	gdpr_module_enabled,
	ccpa_module_enabled,
	state_privacy_enabled,
	iso_isms_enabled,
	admin_audit_log_enabled,
	data_residency_enabled,
	security_disclosure_module_enabled,
	mfa_enabled,
	mfa_enforcement,
	smtp_host,
	smtp_port,
	smtp_from,
	smtp_user,
	smtp_password_ciphertext,
	updated_at
) VALUES (
	1,
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
	$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
	$41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59, $60, $61,
	NOW()
)
ON CONFLICT (id) DO UPDATE SET
	openrouter_api_key = COALESCE(EXCLUDED.openrouter_api_key, settings.platform_app_settings.openrouter_api_key),
	saml_sso_enabled = COALESCE(EXCLUDED.saml_sso_enabled, settings.platform_app_settings.saml_sso_enabled),
	saml_public_base_url = COALESCE(EXCLUDED.saml_public_base_url, settings.platform_app_settings.saml_public_base_url),
	saml_sp_entity_id = COALESCE(EXCLUDED.saml_sp_entity_id, settings.platform_app_settings.saml_sp_entity_id),
	saml_sp_x509_pem = COALESCE(EXCLUDED.saml_sp_x509_pem, settings.platform_app_settings.saml_sp_x509_pem),
	saml_sp_private_key_pem = COALESCE(EXCLUDED.saml_sp_private_key_pem, settings.platform_app_settings.saml_sp_private_key_pem),
	annotation_enabled = COALESCE(EXCLUDED.annotation_enabled, settings.platform_app_settings.annotation_enabled),
	feedback_media_enabled = COALESCE(EXCLUDED.feedback_media_enabled, settings.platform_app_settings.feedback_media_enabled),
	blind_grading_enabled = COALESCE(EXCLUDED.blind_grading_enabled, settings.platform_app_settings.blind_grading_enabled),
	moderated_grading_enabled = COALESCE(EXCLUDED.moderated_grading_enabled, settings.platform_app_settings.moderated_grading_enabled),
	originality_detection_enabled = COALESCE(EXCLUDED.originality_detection_enabled, settings.platform_app_settings.originality_detection_enabled),
	originality_stub_external = COALESCE(EXCLUDED.originality_stub_external, settings.platform_app_settings.originality_stub_external),
	grade_posting_policies_enabled = COALESCE(EXCLUDED.grade_posting_policies_enabled, settings.platform_app_settings.grade_posting_policies_enabled),
	gradebook_csv_enabled = COALESCE(EXCLUDED.gradebook_csv_enabled, settings.platform_app_settings.gradebook_csv_enabled),
	resubmission_workflow_enabled = COALESCE(EXCLUDED.resubmission_workflow_enabled, settings.platform_app_settings.resubmission_workflow_enabled),
	lti_enabled = COALESCE(EXCLUDED.lti_enabled, settings.platform_app_settings.lti_enabled),
	oneroster_enabled = COALESCE(EXCLUDED.oneroster_enabled, settings.platform_app_settings.oneroster_enabled),
	scim_enabled = COALESCE(EXCLUDED.scim_enabled, settings.platform_app_settings.scim_enabled),
	oidc_sso_enabled = COALESCE(EXCLUDED.oidc_sso_enabled, settings.platform_app_settings.oidc_sso_enabled),
	clever_sso_enabled = COALESCE(EXCLUDED.clever_sso_enabled, settings.platform_app_settings.clever_sso_enabled),
	classlink_sso_enabled = COALESCE(EXCLUDED.classlink_sso_enabled, settings.platform_app_settings.classlink_sso_enabled),
	magic_link_enabled = COALESCE(EXCLUDED.magic_link_enabled, settings.platform_app_settings.magic_link_enabled),
	magic_link_enrolled_only = COALESCE(EXCLUDED.magic_link_enrolled_only, settings.platform_app_settings.magic_link_enrolled_only),
	session_management_ui_enabled = COALESCE(EXCLUDED.session_management_ui_enabled, settings.platform_app_settings.session_management_ui_enabled),
	email_notifications_enabled = COALESCE(EXCLUDED.email_notifications_enabled, settings.platform_app_settings.email_notifications_enabled),
	push_notifications_enabled = COALESCE(EXCLUDED.push_notifications_enabled, settings.platform_app_settings.push_notifications_enabled),
	virtual_classroom_enabled = COALESCE(EXCLUDED.virtual_classroom_enabled, settings.platform_app_settings.virtual_classroom_enabled),
	drm_enabled = COALESCE(EXCLUDED.drm_enabled, settings.platform_app_settings.drm_enabled),
	video_transcoding_enabled = COALESCE(EXCLUDED.video_transcoding_enabled, settings.platform_app_settings.video_transcoding_enabled),
	auto_captioning_enabled = COALESCE(EXCLUDED.auto_captioning_enabled, settings.platform_app_settings.auto_captioning_enabled),
	storage_quotas_enabled = COALESCE(EXCLUDED.storage_quotas_enabled, settings.platform_app_settings.storage_quotas_enabled),
	at_risk_alerts_enabled = COALESCE(EXCLUDED.at_risk_alerts_enabled, settings.platform_app_settings.at_risk_alerts_enabled),
	av_scanning_enabled = COALESCE(EXCLUDED.av_scanning_enabled, settings.platform_app_settings.av_scanning_enabled),
	clamav_stub = COALESCE(EXCLUDED.clamav_stub, settings.platform_app_settings.clamav_stub),
	h5p_enabled = COALESCE(EXCLUDED.h5p_enabled, settings.platform_app_settings.h5p_enabled),
	oer_library_enabled = COALESCE(EXCLUDED.oer_library_enabled, settings.platform_app_settings.oer_library_enabled),
	oer_stub = COALESCE(EXCLUDED.oer_stub, settings.platform_app_settings.oer_stub),
	item_analysis_enabled = COALESCE(EXCLUDED.item_analysis_enabled, settings.platform_app_settings.item_analysis_enabled),
	student_progress_enabled = COALESCE(EXCLUDED.student_progress_enabled, settings.platform_app_settings.student_progress_enabled),
	engagement_tracking_enabled = COALESCE(EXCLUDED.engagement_tracking_enabled, settings.platform_app_settings.engagement_tracking_enabled),
	self_reflection_enabled = COALESCE(EXCLUDED.self_reflection_enabled, settings.platform_app_settings.self_reflection_enabled),
	outcomes_report_enabled = COALESCE(EXCLUDED.outcomes_report_enabled, settings.platform_app_settings.outcomes_report_enabled),
	equation_editor_enabled = COALESCE(EXCLUDED.equation_editor_enabled, settings.platform_app_settings.equation_editor_enabled),
	report_export_enabled = COALESCE(EXCLUDED.report_export_enabled, settings.platform_app_settings.report_export_enabled),
	xapi_emission_enabled = COALESCE(EXCLUDED.xapi_emission_enabled, settings.platform_app_settings.xapi_emission_enabled),
	instructor_insights_enabled = COALESCE(EXCLUDED.instructor_insights_enabled, settings.platform_app_settings.instructor_insights_enabled),
	coppa_workflow_enabled = COALESCE(EXCLUDED.coppa_workflow_enabled, settings.platform_app_settings.coppa_workflow_enabled),
	gdpr_module_enabled = COALESCE(EXCLUDED.gdpr_module_enabled, settings.platform_app_settings.gdpr_module_enabled),
	ccpa_module_enabled = COALESCE(EXCLUDED.ccpa_module_enabled, settings.platform_app_settings.ccpa_module_enabled),
	state_privacy_enabled = COALESCE(EXCLUDED.state_privacy_enabled, settings.platform_app_settings.state_privacy_enabled),
	iso_isms_enabled = COALESCE(EXCLUDED.iso_isms_enabled, settings.platform_app_settings.iso_isms_enabled),
	admin_audit_log_enabled = COALESCE(EXCLUDED.admin_audit_log_enabled, settings.platform_app_settings.admin_audit_log_enabled),
	data_residency_enabled = COALESCE(EXCLUDED.data_residency_enabled, settings.platform_app_settings.data_residency_enabled),
	security_disclosure_module_enabled = COALESCE(EXCLUDED.security_disclosure_module_enabled, settings.platform_app_settings.security_disclosure_module_enabled),
	mfa_enabled = COALESCE(EXCLUDED.mfa_enabled, settings.platform_app_settings.mfa_enabled),
	mfa_enforcement = COALESCE(EXCLUDED.mfa_enforcement, settings.platform_app_settings.mfa_enforcement),
	smtp_host = COALESCE(EXCLUDED.smtp_host, settings.platform_app_settings.smtp_host),
	smtp_port = COALESCE(EXCLUDED.smtp_port, settings.platform_app_settings.smtp_port),
	smtp_from = COALESCE(EXCLUDED.smtp_from, settings.platform_app_settings.smtp_from),
	smtp_user = COALESCE(EXCLUDED.smtp_user, settings.platform_app_settings.smtp_user),
	smtp_password_ciphertext = COALESCE(EXCLUDED.smtp_password_ciphertext, settings.platform_app_settings.smtp_password_ciphertext),
	updated_at = NOW()
`,
		w.OpenRouterAPIKey,
		w.SAMLSSOEnabled,
		w.SAMLPublicBaseURL,
		w.SAMLSPEntityID,
		w.SAMLSPX509PEM,
		w.SAMLSPPrivateKeyPEM,
		w.AnnotationEnabled,
		w.FeedbackMediaEnabled,
		w.BlindGradingEnabled,
		w.ModeratedGradingEnabled,
		w.OriginalityDetectionEnabled,
		w.OriginalityStubExternal,
		w.GradePostingPoliciesEnabled,
		w.GradebookCSVEnabled,
		w.ResubmissionWorkflowEnabled,
		w.LTIEnabled,
		w.OneRosterEnabled,
		w.ScimEnabled,
		w.OIDCSSOEnabled,
		w.CleverSSOEnabled,
		w.ClassLinkSSOEnabled,
		w.MagicLinkEnabled,
		w.MagicLinkEnrolledOnly,
		w.SessionManagementUIEnabled,
		w.EmailNotificationsEnabled,
		w.PushNotificationsEnabled,
		w.VirtualClassroomEnabled,
		w.DRMEnabled,
		w.VideoTranscodingEnabled,
		w.AutoCaptioningEnabled,
		w.StorageQuotasEnabled,
		w.AtRiskAlertsEnabled,
		w.AvScanningEnabled,
		w.ClamAVStub,
		w.H5PEnabled,
		w.OERLibraryEnabled,
		w.OERStub,
		w.ItemAnalysisEnabled,
		w.StudentProgressEnabled,
		w.EngagementTrackingEnabled,
		w.SelfReflectionEnabled,
		w.OutcomesReportEnabled,
		w.EquationEditorEnabled,
		w.ReportExportEnabled,
		w.XAPIEmissionEnabled,
		w.InstructorInsightsEnabled,
		w.CoppaWorkflowEnabled,
		w.GDPRModuleEnabled,
		w.CCPAModuleEnabled,
		w.StatePrivacyEnabled,
		w.IsoIsmsEnabled,
		w.AdminAuditLogEnabled,
		w.DataResidencyEnabled,
		w.SecurityDisclosureModuleEnabled,
		w.MFAEnabled,
		w.MFAEnforcement,
		w.SMTPHost,
		smtpPort,
		w.SMTPFrom,
		w.SMTPUser,
		smtpCipher,
	)
	if err != nil {
		return nil, err
	}
	return Get(ctx, pool)
}

// Merge applies platform settings from the database (booleans) and optional DB overrides for secrets/URLs.
func Merge(env config.Config, db *Row) config.Config {
	out := env
	applyPlatformBools(&out, db, DefaultDefaults())
	if db == nil {
		return out
	}
	if db.OpenRouterAPIKey != nil {
		if strings.TrimSpace(*db.OpenRouterAPIKey) != "" {
			out.OpenRouterAPIKey = strings.TrimSpace(*db.OpenRouterAPIKey)
		}
	}
	if db.SAMLPublicBaseURL != nil && strings.TrimSpace(*db.SAMLPublicBaseURL) != "" {
		out.SAMLPublicBaseURL = strings.TrimRight(strings.TrimSpace(*db.SAMLPublicBaseURL), "/")
	}
	if db.SAMLSPEntityID != nil && strings.TrimSpace(*db.SAMLSPEntityID) != "" {
		out.SAMLSPEntityID = strings.TrimSpace(*db.SAMLSPEntityID)
	}
	if db.SAMLSPX509PEM != nil && strings.TrimSpace(*db.SAMLSPX509PEM) != "" {
		out.SAMLSPX509PEM = strings.TrimSpace(*db.SAMLSPX509PEM)
	}
	if db.SAMLSPPrivateKeyPEM != nil && strings.TrimSpace(*db.SAMLSPPrivateKeyPEM) != "" {
		out.SAMLSPPrivateKeyPEM = strings.TrimSpace(*db.SAMLSPPrivateKeyPEM)
	}
	if db.MFAEnforcement != nil && strings.TrimSpace(*db.MFAEnforcement) != "" {
		out.MFAEnforcement = strings.ToLower(strings.TrimSpace(*db.MFAEnforcement))
	}
	if db.SMTPHost != nil && strings.TrimSpace(*db.SMTPHost) != "" {
		out.SMTPHost = strings.TrimSpace(*db.SMTPHost)
	}
	if db.SMTPPort != nil && *db.SMTPPort > 0 && *db.SMTPPort <= 65535 {
		out.SMTPPort = uint16(*db.SMTPPort)
	}
	if db.SMTPFrom != nil && strings.TrimSpace(*db.SMTPFrom) != "" {
		out.SMTPFrom = strings.TrimSpace(*db.SMTPFrom)
	}
	if db.SMTPUser != nil {
		out.SMTPUser = strings.TrimSpace(*db.SMTPUser)
	}
	if len(db.SMTPPasswordCiphertext) > 0 {
		if len(env.PlatformSecretsKey) != 32 {
			out.SMTPPassword = ""
		} else {
			plain, err := appsecrets.Decrypt(db.SMTPPasswordCiphertext, env.PlatformSecretsKey)
			if err != nil {
				out.SMTPPassword = ""
			} else {
				out.SMTPPassword = strings.TrimSpace(string(plain))
			}
		}
	}
	return out
}

// Source describes whether the effective value came from the DB row or the environment.
type Source string

const (
	SourceEnvironment Source = "environment"
	SourceDatabase    Source = "database"
	// SourceDefault means the column is unset and the documented default applies.
	SourceDefault Source = "default"
)

// Sources indicates which layer won for mergeable fields (for admin transparency).
type Sources struct {
	OpenRouterAPIKey Source

	SAMLSSOEnabled      Source
	SAMLPublicBaseURL   Source
	SAMLSPEntityID      Source
	SAMLSPX509PEM       Source
	SAMLSPPrivateKeyPEM Source

	AnnotationEnabled           Source
	FeedbackMediaEnabled        Source
	BlindGradingEnabled         Source
	ModeratedGradingEnabled     Source
	OriginalityDetectionEnabled Source
	OriginalityStubExternal     Source
	GradePostingPoliciesEnabled Source
	GradebookCSVEnabled         Source
	ResubmissionWorkflowEnabled Source
	LTIEnabled                  Source
	OneRosterEnabled            Source
	ScimEnabled                 Source
	MFAEnabled                  Source
	MFAEnforcement              Source

	SMTPHost               Source
	SMTPPort               Source
	SMTPFrom               Source
	SMTPUser               Source
	SMTPPasswordCiphertext Source
}

// ResolveSources compares env vs DB row to label each field.
func ResolveSources(env config.Config, db *Row) Sources {
	var s Sources
	if db == nil {
		return sourcesAllEnvironment(env)
	}
	s.OpenRouterAPIKey = sourceString(env.OpenRouterAPIKey, db.OpenRouterAPIKey)
	s.SAMLSSOEnabled = sourceBoolDB(db.SAMLSSOEnabled)
	s.SAMLPublicBaseURL = sourceString(env.SAMLPublicBaseURL, db.SAMLPublicBaseURL)
	s.SAMLSPEntityID = sourceString(env.SAMLSPEntityID, db.SAMLSPEntityID)
	s.SAMLSPX509PEM = sourceString(env.SAMLSPX509PEM, db.SAMLSPX509PEM)
	s.SAMLSPPrivateKeyPEM = sourceString(env.SAMLSPPrivateKeyPEM, db.SAMLSPPrivateKeyPEM)
	s.AnnotationEnabled = sourceBoolDB(db.AnnotationEnabled)
	s.FeedbackMediaEnabled = sourceBoolDB(db.FeedbackMediaEnabled)
	s.BlindGradingEnabled = sourceBoolDB(db.BlindGradingEnabled)
	s.ModeratedGradingEnabled = sourceBoolDB(db.ModeratedGradingEnabled)
	s.OriginalityDetectionEnabled = sourceBoolDB(db.OriginalityDetectionEnabled)
	s.OriginalityStubExternal = sourceBoolDB(db.OriginalityStubExternal)
	s.GradePostingPoliciesEnabled = sourceBoolDB(db.GradePostingPoliciesEnabled)
	s.GradebookCSVEnabled = sourceBoolDB(db.GradebookCSVEnabled)
	s.ResubmissionWorkflowEnabled = sourceBoolDB(db.ResubmissionWorkflowEnabled)
	s.LTIEnabled = sourceBoolDB(db.LTIEnabled)
	s.OneRosterEnabled = sourceBoolDB(db.OneRosterEnabled)
	s.ScimEnabled = sourceBoolDB(db.ScimEnabled)
	s.MFAEnabled = sourceBoolDB(db.MFAEnabled)
	s.MFAEnforcement = sourceString(env.MFAEnforcement, db.MFAEnforcement)
	s.SMTPHost = sourceString(env.SMTPHost, db.SMTPHost)
	s.SMTPPort = sourceSMTPPort(env.SMTPPort, db.SMTPPort)
	s.SMTPFrom = sourceString(env.SMTPFrom, db.SMTPFrom)
	s.SMTPUser = sourceOptionalStringPtr(db.SMTPUser)
	s.SMTPPasswordCiphertext = sourceSMTPPasswordCiphertext(db.SMTPPasswordCiphertext)
	return s
}

func sourcesAllEnvironment(env config.Config) Sources {
	_ = env
	return Sources{
		OpenRouterAPIKey:            SourceEnvironment,
		SAMLSSOEnabled:              SourceDefault,
		SAMLPublicBaseURL:           SourceEnvironment,
		SAMLSPEntityID:              SourceEnvironment,
		SAMLSPX509PEM:               SourceEnvironment,
		SAMLSPPrivateKeyPEM:         SourceEnvironment,
		AnnotationEnabled:           SourceDefault,
		FeedbackMediaEnabled:        SourceDefault,
		BlindGradingEnabled:         SourceDefault,
		ModeratedGradingEnabled:     SourceDefault,
		OriginalityDetectionEnabled: SourceDefault,
		OriginalityStubExternal:     SourceDefault,
		GradePostingPoliciesEnabled: SourceDefault,
		GradebookCSVEnabled:         SourceDefault,
		ResubmissionWorkflowEnabled: SourceDefault,
		LTIEnabled:                  SourceDefault,
		OneRosterEnabled:            SourceDefault,
		ScimEnabled:                 SourceDefault,
		MFAEnabled:                  SourceDefault,
		MFAEnforcement:              SourceEnvironment,
		SMTPHost:                    SourceEnvironment,
		SMTPPort:                    SourceEnvironment,
		SMTPFrom:                    SourceEnvironment,
		SMTPUser:                    SourceEnvironment,
		SMTPPasswordCiphertext:      SourceEnvironment,
	}
}

func sourceSMTPPort(envPort uint16, dbPtr *int32) Source {
	if dbPtr != nil {
		return SourceDatabase
	}
	return SourceEnvironment
}

func sourceOptionalStringPtr(dbPtr *string) Source {
	if dbPtr != nil {
		return SourceDatabase
	}
	return SourceEnvironment
}

func sourceSMTPPasswordCiphertext(ciphertext []byte) Source {
	if len(ciphertext) > 0 {
		return SourceDatabase
	}
	return SourceEnvironment
}

func sourceString(envVal string, dbPtr *string) Source {
	if dbPtr != nil {
		if strings.TrimSpace(*dbPtr) != "" {
			return SourceDatabase
		}
	}
	return SourceEnvironment
}

func sourceBoolDB(dbPtr *bool) Source {
	if dbPtr != nil {
		return SourceDatabase
	}
	return SourceDefault
}
