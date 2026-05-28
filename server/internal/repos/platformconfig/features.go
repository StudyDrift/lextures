package platformconfig

import "github.com/lextures/lextures/server/internal/config"

func applyPlatformBools(out *config.Config, db *Row, def Defaults) {
	if db == nil {
		out.BlindGradingEnabled = def.BlindGradingEnabled
		out.GradePostingPoliciesEnabled = def.GradePostingPoliciesEnabled
		out.MagicLinkEnabled = def.MagicLinkEnabled
		out.VirtualClassroomEnabled = def.VirtualClassroomEnabled
		return
	}
	out.SAMLSSOEnabled = mergeBool(db.SAMLSSOEnabled, false)
	out.AnnotationEnabled = mergeBool(db.AnnotationEnabled, false)
	out.FeedbackMediaEnabled = mergeBool(db.FeedbackMediaEnabled, false)
	out.BlindGradingEnabled = mergeBool(db.BlindGradingEnabled, def.BlindGradingEnabled)
	out.ModeratedGradingEnabled = mergeBool(db.ModeratedGradingEnabled, false)
	out.OriginalityDetectionEnabled = mergeBool(db.OriginalityDetectionEnabled, false)
	out.OriginalityStubExternal = mergeBool(db.OriginalityStubExternal, false)
	out.GradePostingPoliciesEnabled = mergeBool(db.GradePostingPoliciesEnabled, def.GradePostingPoliciesEnabled)
	out.GradebookCSVEnabled = mergeBool(db.GradebookCSVEnabled, false)
	out.ResubmissionWorkflowEnabled = mergeBool(db.ResubmissionWorkflowEnabled, false)
	out.LTIEnabled = mergeBool(db.LTIEnabled, false)
	out.OneRosterEnabled = mergeBool(db.OneRosterEnabled, false)
	out.ScimEnabled = mergeBool(db.ScimEnabled, false)
	out.OIDCSSOEnabled = mergeBool(db.OIDCSSOEnabled, false)
	out.CleverSSOEnabled = mergeBool(db.CleverSSOEnabled, false)
	out.ClassLinkSSOEnabled = mergeBool(db.ClassLinkSSOEnabled, false)
	out.MFAEnabled = mergeBool(db.MFAEnabled, false)
	out.MagicLinkEnabled = mergeBool(db.MagicLinkEnabled, def.MagicLinkEnabled)
	out.MagicLinkEnrolledOnly = mergeBool(db.MagicLinkEnrolledOnly, false)
	out.SessionManagementUIEnabled = mergeBool(db.SessionManagementUIEnabled, false)
	out.EmailNotificationsEnabled = mergeBool(db.EmailNotificationsEnabled, false)
	out.PushNotificationsEnabled = mergeBool(db.PushNotificationsEnabled, false)
	out.VirtualClassroomEnabled = mergeBool(db.VirtualClassroomEnabled, def.VirtualClassroomEnabled)
	out.DRMEnabled = mergeBool(db.DRMEnabled, false)
	out.VideoTranscodingEnabled = mergeBool(db.VideoTranscodingEnabled, false)
	out.AutoCaptioningEnabled = mergeBool(db.AutoCaptioningEnabled, false)
	out.StorageQuotasEnabled = mergeBool(db.StorageQuotasEnabled, false)
	out.AtRiskAlertsEnabled = mergeBool(db.AtRiskAlertsEnabled, false)
	out.AvScanningEnabled = mergeBool(db.AvScanningEnabled, false)
	out.ClamAVStub = mergeBool(db.ClamAVStub, false)
	out.H5PEnabled = mergeBool(db.H5PEnabled, false)
	out.OERLibraryEnabled = mergeBool(db.OERLibraryEnabled, false)
	out.OERStub = mergeBool(db.OERStub, false)
	out.ItemAnalysisEnabled = mergeBool(db.ItemAnalysisEnabled, false)
	out.StudentProgressEnabled = mergeBool(db.StudentProgressEnabled, false)
	out.EngagementTrackingEnabled = mergeBool(db.EngagementTrackingEnabled, false)
	out.SelfReflectionEnabled = mergeBool(db.SelfReflectionEnabled, false)
	out.OutcomesReportEnabled = mergeBool(db.OutcomesReportEnabled, false)
	out.InstructorInsightsEnabled = mergeBool(db.InstructorInsightsEnabled, false)
	out.EquationEditorEnabled = mergeBool(db.EquationEditorEnabled, false)
	out.ReportExportEnabled = mergeBool(db.ReportExportEnabled, false)
	out.XAPIEmissionEnabled = mergeBool(db.XAPIEmissionEnabled, out.XAPIEmissionEnabled)
	out.CoppaWorkflowEnabled = mergeBool(db.CoppaWorkflowEnabled, out.CoppaWorkflowEnabled)
	out.GDPRModuleEnabled = mergeBool(db.GDPRModuleEnabled, out.GDPRModuleEnabled)
	out.CCPAModuleEnabled = mergeBool(db.CCPAModuleEnabled, out.CCPAModuleEnabled)
	out.StatePrivacyEnabled = mergeBool(db.StatePrivacyEnabled, out.StatePrivacyEnabled)
	out.IsoIsmsEnabled = mergeBool(db.IsoIsmsEnabled, out.IsoIsmsEnabled)
	out.AdminAuditLogEnabled = mergeBool(db.AdminAuditLogEnabled, out.AdminAuditLogEnabled)
	out.DataResidencyEnabled = mergeBool(db.DataResidencyEnabled, out.DataResidencyEnabled)
	out.SecurityDisclosureModuleEnabled = mergeBool(db.SecurityDisclosureModuleEnabled, out.SecurityDisclosureModuleEnabled)
}
