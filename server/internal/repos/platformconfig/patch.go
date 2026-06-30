package platformconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// patch applies only non-nil fields in w to the existing singleton row.
// Partial platform settings updates must not use INSERT ... ON CONFLICT with NULLs for
// NOT NULL columns (PostgreSQL validates the INSERT row before conflict resolution).
func patch(ctx context.Context, pool *pgxpool.Pool, w *Write) error {
	var sets []string
	var args []any

	addBool := func(col string, v *bool) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addString := func(col string, v *string) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addInt32 := func(col string, v *int32) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addFloat64 := func(col string, v *float64) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addBytes := func(col string, v *[]byte) {
		if v == nil {
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if w.OpenRouterAPIKey != nil {
		addString("openrouter_api_key", w.OpenRouterAPIKey)
	}
	addBool("saml_sso_enabled", w.SAMLSSOEnabled)
	addString("saml_public_base_url", w.SAMLPublicBaseURL)
	addString("saml_sp_entity_id", w.SAMLSPEntityID)
	addString("saml_sp_x509_pem", w.SAMLSPX509PEM)
	addString("saml_sp_private_key_pem", w.SAMLSPPrivateKeyPEM)
	addBool("annotation_enabled", w.AnnotationEnabled)
	addBool("feedback_media_enabled", w.FeedbackMediaEnabled)
	addBool("blind_grading_enabled", w.BlindGradingEnabled)
	addBool("moderated_grading_enabled", w.ModeratedGradingEnabled)
	addBool("originality_detection_enabled", w.OriginalityDetectionEnabled)
	addBool("originality_stub_external", w.OriginalityStubExternal)
	addBool("grade_posting_policies_enabled", w.GradePostingPoliciesEnabled)
	addBool("gradebook_csv_enabled", w.GradebookCSVEnabled)
	addBool("resubmission_workflow_enabled", w.ResubmissionWorkflowEnabled)
	addBool("lti_enabled", w.LTIEnabled)
	addBool("oneroster_enabled", w.OneRosterEnabled)
	addBool("scim_enabled", w.ScimEnabled)
	addBool("oidc_sso_enabled", w.OIDCSSOEnabled)
	addBool("clever_sso_enabled", w.CleverSSOEnabled)
	addBool("classlink_sso_enabled", w.ClassLinkSSOEnabled)
	addBool("magic_link_enabled", w.MagicLinkEnabled)
	addBool("magic_link_enrolled_only", w.MagicLinkEnrolledOnly)
	addBool("session_management_ui_enabled", w.SessionManagementUIEnabled)
	addBool("email_notifications_enabled", w.EmailNotificationsEnabled)
	addBool("push_notifications_enabled", w.PushNotificationsEnabled)
	addBool("virtual_classroom_enabled", w.VirtualClassroomEnabled)
	addBool("drm_enabled", w.DRMEnabled)
	addBool("video_transcoding_enabled", w.VideoTranscodingEnabled)
	addBool("auto_captioning_enabled", w.AutoCaptioningEnabled)
	addBool("video_captions_enabled", w.VideoCaptionsEnabled)
	addBool("storage_quotas_enabled", w.StorageQuotasEnabled)
	addBool("at_risk_alerts_enabled", w.AtRiskAlertsEnabled)
	addBool("av_scanning_enabled", w.AvScanningEnabled)
	addBool("clamav_stub", w.ClamAVStub)
	addBool("h5p_enabled", w.H5PEnabled)
	addBool("scorm_ingestion_enabled", w.ScormIngestionEnabled)
	addBool("oer_library_enabled", w.OERLibraryEnabled)
	addBool("oer_stub", w.OERStub)
	addBool("item_analysis_enabled", w.ItemAnalysisEnabled)
	addBool("student_progress_enabled", w.StudentProgressEnabled)
	addBool("engagement_tracking_enabled", w.EngagementTrackingEnabled)
	addBool("self_reflection_enabled", w.SelfReflectionEnabled)
	addBool("outcomes_report_enabled", w.OutcomesReportEnabled)
	addBool("equation_editor_enabled", w.EquationEditorEnabled)
	addBool("reading_level_enabled", w.ReadingLevelEnabled)
	addBool("grader_agent_enabled", w.GraderAgentEnabled)
	addBool("grader_agent_review_inbox_enabled", w.GraderAgentReviewInboxEnabled)
	addBool("grader_agent_suggest_mode_enabled", w.GraderAgentSuggestModeEnabled)
	addBool("grader_agent_text_entry_grading_enabled", w.GraderAgentTextEntryGradingEnabled)
	addBool("grader_agent_vision_grading_enabled", w.GraderAgentVisionGradingEnabled)
	addBool("grader_agent_run_filters_enabled", w.GraderAgentRunFiltersEnabled)
	addBool("grader_agent_cost_estimate_enabled", w.GraderAgentCostEstimateEnabled)
	addBool("grader_agent_cancel_run_enabled", w.GraderAgentCancelRunEnabled)
	addBool("code_execution_enabled", w.CodeExecutionEnabled)
	addBool("alt_text_enforcement_enabled", w.AltTextEnforcementEnabled)
	addBool("ff_alt_text_enforcement", w.FFAltTextEnforcement)
	addBool("speech_to_text_enabled", w.SpeechToTextEnabled)
	addBool("accommodations_engine_enabled", w.AccommodationsEngineEnabled)
	addBool("ff_accommodations_engine", w.FFAccommodationsEngine)
	addBool("read_aloud_enabled", w.ReadAloudEnabled)
	addBool("ff_read_aloud", w.FFReadAloud)
	addBool("translation_memory_enabled", w.TranslationMemoryEnabled)
	addBool("report_export_enabled", w.ReportExportEnabled)
	addBool("xapi_emission_enabled", w.XAPIEmissionEnabled)
	addBool("instructor_insights_enabled", w.InstructorInsightsEnabled)
	addBool("coppa_workflow_enabled", w.CoppaWorkflowEnabled)
	addBool("gdpr_module_enabled", w.GDPRModuleEnabled)
	addBool("ccpa_module_enabled", w.CCPAModuleEnabled)
	addBool("state_privacy_enabled", w.StatePrivacyEnabled)
	addBool("iso_isms_enabled", w.IsoIsmsEnabled)
	addBool("admin_audit_log_enabled", w.AdminAuditLogEnabled)
	addBool("admin_console_enabled", w.AdminConsoleEnabled)
	addBool("impersonation_enabled", w.ImpersonationEnabled)
	addBool("bulk_csv_import_enabled", w.BulkCsvImportEnabled)
	addBool("admin_search_enabled", w.AdminSearchEnabled)
	addBool("email_template_editor_enabled", w.EmailTemplateEditorEnabled)
	addBool("data_residency_enabled", w.DataResidencyEnabled)
	addBool("ai_disclosure_enabled", w.AiDisclosureEnabled)
	addBool("rtl_enabled", w.RTLEnabled)
	addBool("security_disclosure_module_enabled", w.SecurityDisclosureModuleEnabled)
	addBool("backup_module_enabled", w.BackupModuleEnabled)
	addBool("ff_high_contrast_reduced_motion", w.FFHighContrastReducedMotion)
	addBool("ff_parent_portal", w.FFParentPortal)
	addBool("ff_report_cards", w.FFReportCards)
	addBool("ff_sbg_report_cards", w.FFSBGReportCards)
	addBool("ff_sis_integration", w.FFSISIntegration)
	addBool("ff_catalog_integration", w.FFCatalogIntegration)
	addBool("ff_enrollment_state_machine", w.FFEnrollmentStateMachine)
	addBool("ff_incomplete_grade_workflow", w.FFIncompleteGradeWorkflow)
	addBool("ff_library", w.FFLibrary)
	addBool("ff_broadcasts", w.FFBroadcasts)
	addBool("ff_conference_scheduling", w.FFConferenceScheduling)
	addBool("ff_demographics", w.FFDemographics)
	addBool("ff_content_filter_integration", w.FFContentFilterIntegration)
	addBool("ff_grade_submission", w.FFGradeSubmission)
	addBool("ff_whatif_grades", w.FFWhatifGrades)
	addBool("ff_grade_curving", w.FFGradeCurving)
	addBool("ff_academic_calendar", w.FFAcademicCalendar)
	addBool("ff_plagiarism_checks", w.FFPlagiarismChecks)
	addBool("ff_course_evaluations", w.FFCourseEvaluations)
	addBool("ff_proctoring_integration", w.FFProctoringIntegration)
	addBool("ff_co_curricular_transcript", w.FFCoCurricularTranscript)
	addBool("ff_eportfolio", w.FFEportfolio)
	addBool("ff_bookstore_integration", w.FFBookstoreIntegration)
	addBool("ff_transcripts", w.FFTranscripts)
	addBool("ff_webhooks", w.FFWebhooks)
	addBool("ff_zapier_connector", w.FFZapierConnector)
	addBool("ff_marketplace_enabled", w.FFMarketplace)
	addBool("ff_advising_integration", w.FFAdvisingIntegration)
	addBool("ff_research_consent", w.FFResearchConsent)
	addBool("ff_accessibility_intake", w.FFAccessibilityIntake)
	addBool("ff_ceu_tracking", w.FFCEUTracking)
	addBool("ff_consortium_sharing", w.FFConsortiumSharing)
	addBool("ff_self_paced_mode", w.FFSelfPacedMode)
	addBool("ff_public_catalog", w.FFPublicCatalog)
	addBool("ff_public_api", w.FFPublicAPI)
	addBool("ff_stripe_billing", w.FFStripeBilling)
	addBool("ff_payments_enabled", w.FFPaymentsEnabled)
	addBool("ff_revenue_share", w.FFRevenueShare)
	addBool("ff_tax_collection", w.FFTaxCollection)
	addBool("ff_learning_paths", w.FFLearningPaths)
	addBool("ff_conditional_release", w.FFConditionalRelease)
	addBool("ff_peer_review", w.FFPeerReview)
	addBool("ff_completion_credentials", w.FFCompletionCredentials)
	addBool("ff_course_reviews", w.FFCourseReviews)
	addBool("ff_gamification", w.FFGamification)
	addBool("ff_onboarding_flow", w.FFOnboardingFlow)
	addBool("ff_study_reminders", w.FFStudyReminders)
	addBool("ff_ai_study_buddy", w.FFAIStudyBuddy)
	addBool("ff_api_tokens", w.FFAPITokens)
	addBool("ff_bot_slack", w.FFBotSlack)
	addBool("ff_bot_teams", w.FFBotTeams)
	addBool("ff_bot_discord", w.FFBotDiscord)
	addBool("ff_calendar_feeds", w.FFCalendarFeeds)
	addBool("ff_redis_cache", w.FFRedisCache)
	addBool("lrs_anonymize_actors", w.LRSAnonymizeActors)
	addBool("ferpa_workflow_enabled", w.FERPAWorkflowEnabled)
	addBool("dpa_portal_enabled", w.DPAPortalEnabled)
	addBool("soc2_module_enabled", w.SOC2ModuleEnabled)
	addBool("ff_reading_preferences", w.FFReadingPreferences)
	addBool("ff_classroom_signals", w.FFClassroomSignals)
	addBool("ff_library_integration", w.FFLibraryIntegration)
	addBool("diagnostic_assessments_enabled", w.DiagnosticAssessmentsEnabled)
	addBool("srs_practice_enabled", w.SRSPracticeEnabled)
	addBool("irt_cat_mode_enabled", w.IRTCatModeEnabled)
	addBool("adaptive_learner_model_enabled", w.AdaptiveLearnerModelEnabled)
	addFloat64("learner_model_ema_alpha", w.LearnerModelEMAAlpha)
	addBool("ff_ui_mode", w.FFUiMode)
	addBool("mfa_enabled", w.MFAEnabled)
	addString("mfa_enforcement", w.MFAEnforcement)
	addString("smtp_host", w.SMTPHost)
	addInt32("smtp_port", w.SMTPPort)
	addString("smtp_from", w.SMTPFrom)
	addString("smtp_user", w.SMTPUser)
	addBytes("smtp_password_ciphertext", w.SMTPPasswordCiphertext)

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	q := fmt.Sprintf(
		"UPDATE settings.platform_app_settings SET %s WHERE id = 1",
		strings.Join(sets, ", "),
	)
	_, err := pool.Exec(ctx, q, args...)
	return err
}
