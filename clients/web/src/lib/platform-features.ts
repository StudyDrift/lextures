/** Snapshot of GET /api/v1/platform/features; updated by PlatformFeaturesProvider. */

export type PlatformFeaturesSnapshot = {
  studentProgressEnabled: boolean
  atRiskAlertsEnabled: boolean
  h5pEnabled: boolean
  scormIngestionEnabled: boolean
  oerLibraryEnabled: boolean
  itemAnalysisEnabled: boolean
  engagementTrackingEnabled: boolean
  selfReflectionEnabled: boolean
  learnerProfileEnabled?: boolean
  introCourseEnabled?: boolean
  outcomesReportEnabled: boolean
  xapiEmissionEnabled: boolean
  equationEditorEnabled: boolean
  readingLevelEnabled: boolean
  graderAgentEnabled?: boolean
  graderAgentReviewInboxEnabled?: boolean
  graderAgentSuggestModeEnabled?: boolean
  graderAgentTextEntryGradingEnabled?: boolean
  graderAgentVisionGradingEnabled?: boolean
  graderAgentRunFiltersEnabled?: boolean
  graderAgentCostEstimateEnabled?: boolean
  graderAgentCancelRunEnabled?: boolean
  codeExecutionEnabled?: boolean
  altTextEnforcementEnabled: boolean
  ffAltTextEnforcement: boolean
  speechToTextEnabled: boolean
  accommodationsEngineEnabled: boolean
  ffAccommodationsEngine: boolean
  readAloudEnabled: boolean
  ffReadAloud: boolean
  translationMemoryEnabled: boolean
  storageQuotasEnabled: boolean
  avScanningEnabled: boolean
  virtualClassroomEnabled: boolean
  sessionManagementUiEnabled: boolean
  instructorInsightsEnabled: boolean
  rtlEnabled: boolean
  videoCaptionsEnabled?: boolean
  autoCaptioningEnabled?: boolean
  ffReadingPreferences?: boolean
  ffHighContrastReducedMotion?: boolean
  ffMotionNavigation?: boolean
  ffMotionReveal?: boolean
  ffMotionLists?: boolean
  ffMobileCreateCourse?: boolean
  ffMobileCourseCreateV2?: boolean
  ffMobileCanvasImport?: boolean
  ffMobileAdminConsole?: boolean
  ffLibrary?: boolean
  ffBroadcasts?: boolean
  ffClassroomSignals?: boolean
  ffConferenceScheduling?: boolean
  ffParentPortal?: boolean
  ffParentPortalV2?: boolean
  ffReportCards?: boolean
  ffDemographics?: boolean
  ffContentFilterIntegration?: boolean
  ffSisIntegration?: boolean
  ffWebhooks?: boolean
  adminConsoleEnabled?: boolean
  adminAuditLogEnabled?: boolean
  impersonationEnabled?: boolean
  bulkCsvImportEnabled?: boolean
  adminSearchEnabled?: boolean
  emailTemplateEditorEnabled?: boolean
  maintenanceBannerEnabled?: boolean
  seatManagementEnabled?: boolean
  ffZapierConnector?: boolean
  ffCatalogIntegration?: boolean
  ffEnrollmentStateMachine?: boolean
  ffIncompleteGradeWorkflow?: boolean
  ffUiMode?: boolean
  ffGradeSubmission?: boolean
  ffWhatifGrades?: boolean
  ffGradeCurving?: boolean
  ffAcademicCalendar?: boolean
  ffPlagiarismChecks?: boolean
  ffCourseEvaluations?: boolean
  ffProctoringIntegration?: boolean
  ffCoCurricularTranscript?: boolean
  ffLibraryIntegration?: boolean
  ffEportfolio?: boolean
  ffBookstoreIntegration?: boolean
  ffTranscripts?: boolean
  ffTranscriptInbound?: boolean
  ffDiplomas?: boolean
  ffAdvisingIntegration?: boolean
  ffResearchConsent?: boolean
  ffAccessibilityIntake?: boolean
  ffCeuTracking?: boolean
  ffConsortiumSharing?: boolean
  ffStripeBilling?: boolean
  ffPaymentsEnabled?: boolean
  ffRevenueShare?: boolean
  ffTaxCollection?: boolean
  ffCourseMarketplace?: boolean
  ffLearningPaths?: boolean
  ffConditionalRelease?: boolean
  ffPeerReview?: boolean
  ffCompletionCredentials?: boolean
  ffCourseReviews?: boolean
  ffGamification?: boolean
  ffCompetencyBadges?: boolean
  ffOnboardingFlow?: boolean
  ffStudyReminders?: boolean
  ffAiStudyBuddy?: boolean
  ffLessonGenerator?: boolean
  ffPersistentTutor?: boolean
  ffCalendarFeeds?: boolean
  aiStudyBuddyEnabled?: boolean
  gdprModuleEnabled?: boolean
  aiDisclosureEnabled?: boolean
  /** @deprecated AP.9 — use aiConfigured */
  openRouterConfigured?: boolean
  aiConfigured?: boolean
  aiProvidersConfigured?: string[]
  aiProviderAbstractionEnabled?: boolean
  ragNotebookEnabled?: boolean
  ffFeedback?: boolean
  ffVisualBoards?: boolean
  ffInteractiveQuizzes?: boolean
  ffIqLiveHosting?: boolean
  ffIqTeamMode?: boolean
  ffIqStudentPaced?: boolean
  ffIqHomework?: boolean
  ffIqGradebookPush?: boolean
  ffIqPublicKitCatalog?: boolean
  ffIqGuestJoin?: boolean
  ffIqAiGeneration?: boolean
  ffBoardsRealtime?: boolean
  ffBoardsExternalSharing?: boolean
  ffEmailSes?: boolean
}

const defaults: PlatformFeaturesSnapshot = {
  studentProgressEnabled: false,
  atRiskAlertsEnabled: false,
  h5pEnabled: false,
  scormIngestionEnabled: false,
  oerLibraryEnabled: false,
  itemAnalysisEnabled: false,
  engagementTrackingEnabled: false,
  selfReflectionEnabled: false,
  learnerProfileEnabled: true,
  introCourseEnabled: true,
  outcomesReportEnabled: false,
  xapiEmissionEnabled: false,
  equationEditorEnabled: false,
  readingLevelEnabled: false,
  graderAgentEnabled: false,
  codeExecutionEnabled: false,
  altTextEnforcementEnabled: false,
  ffAltTextEnforcement: false,
  speechToTextEnabled: false,
  accommodationsEngineEnabled: false,
  ffAccommodationsEngine: false,
  readAloudEnabled: false,
  ffReadAloud: false,
  translationMemoryEnabled: false,
  storageQuotasEnabled: false,
  avScanningEnabled: false,
  virtualClassroomEnabled: true,
  sessionManagementUiEnabled: false,
  instructorInsightsEnabled: false,
  rtlEnabled: false,
  videoCaptionsEnabled: false,
  autoCaptioningEnabled: false,
  ffReadingPreferences: false,
  ffHighContrastReducedMotion: false,
  ffMotionNavigation: true,
  ffMotionReveal: true,
  ffMotionLists: true,
  ffMobileCreateCourse: false,
  ffMobileCourseCreateV2: false,
  ffMobileCanvasImport: false,
  ffMobileAdminConsole: false,
  ffLibrary: false,
  ffBroadcasts: false,
  ffClassroomSignals: false,
  ffConferenceScheduling: false,
  ffParentPortal: false,
  ffParentPortalV2: false,
  ffReportCards: false,
  ffDemographics: false,
  ffContentFilterIntegration: false,
  ffSisIntegration: false,
  ffWebhooks: false,
  adminConsoleEnabled: false,
  adminAuditLogEnabled: true,
  impersonationEnabled: false,
  bulkCsvImportEnabled: false,
  adminSearchEnabled: false,
  emailTemplateEditorEnabled: false,
  maintenanceBannerEnabled: true,
  seatManagementEnabled: false,
  ffZapierConnector: false,
  ffCatalogIntegration: false,
  ffEnrollmentStateMachine: false,
  ffIncompleteGradeWorkflow: false,
  ffUiMode: false,
  ffGradeSubmission: false,
  ffWhatifGrades: false,
  ffGradeCurving: false,
  ffAcademicCalendar: false,
  ffPlagiarismChecks: false,
  ffCourseEvaluations: false,
  ffProctoringIntegration: false,
  ffCoCurricularTranscript: false,
  ffLibraryIntegration: false,
  ffEportfolio: false,
  ffBookstoreIntegration: false,
  ffTranscripts: false,
  ffTranscriptInbound: false,
  ffDiplomas: false,
  ffAdvisingIntegration: false,
  ffResearchConsent: false,
  ffAccessibilityIntake: false,
  ffCeuTracking: false,
  ffConsortiumSharing: false,
  ffStripeBilling: false,
  ffPaymentsEnabled: false,
  ffRevenueShare: false,
  ffTaxCollection: false,
  ffCourseMarketplace: true,
  ffLearningPaths: false,
  ffConditionalRelease: false,
  ffPeerReview: false,
  ffCompletionCredentials: false,
  ffCourseReviews: false,
  ffGamification: false,
  ffCompetencyBadges: false,
  ffOnboardingFlow: false,
  ffStudyReminders: false,
  ffAiStudyBuddy: false,
  ffLessonGenerator: false,
  ffPersistentTutor: false,
  ffCalendarFeeds: true,
  aiStudyBuddyEnabled: false,
  gdprModuleEnabled: false,
  aiDisclosureEnabled: false,
  openRouterConfigured: false,
  aiConfigured: false,
  aiProvidersConfigured: [],
  aiProviderAbstractionEnabled: false,
  ragNotebookEnabled: false,
  ffFeedback: true,
  ffVisualBoards: true,
  ffInteractiveQuizzes: true,
  ffIqLiveHosting: true,
  ffIqTeamMode: false,
  ffIqStudentPaced: false,
  ffIqHomework: false,
  ffIqGradebookPush: false,
  ffIqPublicKitCatalog: false,
  ffIqGuestJoin: false,
  ffIqAiGeneration: false,
  ffBoardsRealtime: true,
  ffBoardsExternalSharing: false,
  ffEmailSes: false,
}

let loaded = false
let snapshot: PlatformFeaturesSnapshot = { ...defaults }

export function setPlatformFeaturesSnapshot(next: PlatformFeaturesSnapshot): void {
  snapshot = next
  loaded = true
}

export function resetPlatformFeaturesSnapshot(): void {
  snapshot = { ...defaults }
  loaded = false
}

export function getPlatformFeatures(): PlatformFeaturesSnapshot {
  return snapshot
}

export function videoCaptionsFeatureEnabled(): boolean {
  return loaded && (snapshot.videoCaptionsEnabled === true || snapshot.autoCaptioningEnabled === true)
}

export function studentProgressFeatureEnabled(): boolean {
  return loaded && snapshot.studentProgressEnabled
}

export function atRiskFeatureEnabled(): boolean {
  return loaded && snapshot.atRiskAlertsEnabled
}

export function h5pFeatureEnabled(): boolean {
  return loaded && snapshot.h5pEnabled
}

export function scormIngestionFeatureEnabled(): boolean {
  return loaded && snapshot.scormIngestionEnabled
}

export function oerLibraryEnabled(): boolean {
  return loaded && snapshot.oerLibraryEnabled
}

export function equationEditorFeatureEnabled(): boolean {
  return loaded && snapshot.equationEditorEnabled
}

export function readingLevelFeatureEnabled(): boolean {
  return loaded && snapshot.readingLevelEnabled
}

export function altTextEnforcementFeatureEnabled(): boolean {
  return loaded && snapshot.altTextEnforcementEnabled
}

export function altTextHardBlockEnabled(): boolean {
  return loaded && snapshot.ffAltTextEnforcement
}

export function speechToTextFeatureEnabled(): boolean {
  return loaded && snapshot.speechToTextEnabled
}

export function accommodationsEngineFeatureEnabled(): boolean {
  return loaded && snapshot.accommodationsEngineEnabled
}

export function readAloudFeatureEnabled(): boolean {
  return loaded && snapshot.readAloudEnabled && snapshot.ffReadAloud
}

export function translationMemoryFeatureEnabled(): boolean {
  return loaded && snapshot.translationMemoryEnabled
}

export function outcomesReportFeatureEnabled(): boolean {
  return loaded && snapshot.outcomesReportEnabled
}

export function xapiEmissionFeatureEnabled(): boolean {
  return loaded && snapshot.xapiEmissionEnabled
}

export function instructorInsightsFeatureEnabled(): boolean {
  return loaded && snapshot.instructorInsightsEnabled
}

export function libraryFeatureEnabled(): boolean {
  return loaded && snapshot.ffLibrary === true
}

export function enrollmentStateMachineFeatureEnabled(): boolean {
  return loaded && snapshot.ffEnrollmentStateMachine === true
}

export function finalGradeSubmissionFeatureEnabled(): boolean {
  return loaded && snapshot.ffGradeSubmission === true
}

export function heLibraryIntegrationEnabled(): boolean {
  return loaded && snapshot.ffLibraryIntegration === true
}

export function bookstoreIntegrationEnabled(): boolean {
  return loaded && snapshot.ffBookstoreIntegration === true
}

export function transcriptsFeatureEnabled(): boolean {
  return loaded && snapshot.ffTranscripts === true
}

export function transcriptInboundFeatureEnabled(): boolean {
  return loaded && snapshot.ffTranscripts === true && snapshot.ffTranscriptInbound === true
}

/** T09 wallet is available when any credential source flag is on. */
export function credentialWalletFeatureEnabled(): boolean {
  return (
    loaded &&
    (snapshot.ffTranscripts === true ||
      snapshot.ffCoCurricularTranscript === true ||
      snapshot.ffCompetencyBadges === true ||
      snapshot.ffCompletionCredentials === true ||
      snapshot.ffCeuTracking === true ||
      snapshot.ffDiplomas === true)
  )
}

export function diplomasFeatureEnabled(): boolean {
  return loaded && snapshot.ffDiplomas === true
}

export function eportfolioFeatureEnabled(): boolean {
  return loaded && snapshot.ffEportfolio === true
}

export function incompleteGradeWorkflowFeatureEnabled(): boolean {
  return loaded && snapshot.ffIncompleteGradeWorkflow === true
}

/** True when GET/PATCH /api/v1/me/reading-preferences is available (matches server readingPreferencesEnabled). */
export function learnerProfileFeatureEnabled(): boolean {
  return loaded && snapshot.learnerProfileEnabled !== false
}

export function readingPreferencesApiEnabled(s?: PlatformFeaturesSnapshot): boolean {
  const snap = s ?? snapshot
  if (!s && !loaded) {
    return false
  }
  return (
    snap.speechToTextEnabled ||
    snap.accommodationsEngineEnabled ||
    (snap.readAloudEnabled && snap.ffReadAloud) ||
    snap.ffReadingPreferences === true ||
    snap.ffHighContrastReducedMotion === true
  )
}

/** AN.2: splash/route/section transitions (default on; kill-switch via Settings). */
export function motionNavigationEnabled(s?: PlatformFeaturesSnapshot): boolean {
  const snap = s ?? snapshot
  if (!s && !loaded) {
    return true
  }
  return snap.ffMotionNavigation !== false
}

/** AN.3: skeleton→content load choreography (default on; kill-switch via Settings). */
export function motionRevealEnabled(s?: PlatformFeaturesSnapshot): boolean {
  const snap = s ?? snapshot
  if (!s && !loaded) {
    return true
  }
  return snap.ffMotionReveal !== false
}

/** AN.4: list insert/remove/reorder motion (default on; kill-switch via Settings). */
export function motionListsEnabled(s?: PlatformFeaturesSnapshot): boolean {
  const snap = s ?? snapshot
  if (!s && !loaded) {
    return true
  }
  return snap.ffMotionLists !== false
}
