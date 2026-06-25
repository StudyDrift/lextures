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
  ffLibrary?: boolean
  ffBroadcasts?: boolean
  ffClassroomSignals?: boolean
  ffConferenceScheduling?: boolean
  ffDemographics?: boolean
  ffContentFilterIntegration?: boolean
  ffSisIntegration?: boolean
  ffWebhooks?: boolean
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
  ffAdvisingIntegration?: boolean
  ffResearchConsent?: boolean
  ffAccessibilityIntake?: boolean
  ffCeuTracking?: boolean
  ffConsortiumSharing?: boolean
  ffStripeBilling?: boolean
  ffRevenueShare?: boolean
  ffTaxCollection?: boolean
  ffLearningPaths?: boolean
  ffConditionalRelease?: boolean
  ffPeerReview?: boolean
  ffCompletionCredentials?: boolean
  ffCourseReviews?: boolean
  ffGamification?: boolean
  ffOnboardingFlow?: boolean
  ffStudyReminders?: boolean
  ffAiStudyBuddy?: boolean
  ffCalendarFeeds?: boolean
  aiStudyBuddyEnabled?: boolean
  gdprModuleEnabled?: boolean
  aiDisclosureEnabled?: boolean
  openRouterConfigured?: boolean
  ragNotebookEnabled?: boolean
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
  ffLibrary: false,
  ffBroadcasts: false,
  ffClassroomSignals: false,
  ffConferenceScheduling: false,
  ffDemographics: false,
  ffContentFilterIntegration: false,
  ffSisIntegration: false,
  ffWebhooks: false,
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
  ffAdvisingIntegration: false,
  ffResearchConsent: false,
  ffAccessibilityIntake: false,
  ffCeuTracking: false,
  ffConsortiumSharing: false,
  ffStripeBilling: false,
  ffRevenueShare: false,
  ffTaxCollection: false,
  ffLearningPaths: false,
  ffConditionalRelease: false,
  ffPeerReview: false,
  ffCompletionCredentials: false,
  ffCourseReviews: false,
  ffGamification: false,
  ffOnboardingFlow: false,
  ffStudyReminders: false,
  ffAiStudyBuddy: false,
  ffCalendarFeeds: true,
  aiStudyBuddyEnabled: false,
  gdprModuleEnabled: false,
  aiDisclosureEnabled: false,
  openRouterConfigured: false,
  ragNotebookEnabled: false,
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

export function platformFeaturesLoaded(): boolean {
  return loaded
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

export function ffAccommodationsEngineEnabled(): boolean {
  return loaded && snapshot.ffAccommodationsEngine
}

export function readAloudFeatureEnabled(): boolean {
  return loaded && snapshot.readAloudEnabled && snapshot.ffReadAloud
}

export function translationMemoryFeatureEnabled(): boolean {
  return loaded && snapshot.translationMemoryEnabled
}

export function engagementTrackingFeatureEnabled(): boolean {
  return loaded && snapshot.engagementTrackingEnabled
}

export function selfReflectionFeatureEnabled(): boolean {
  return loaded && snapshot.selfReflectionEnabled
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

export function broadcastsFeatureEnabled(): boolean {
  return loaded && snapshot.ffBroadcasts === true
}

export function readingPreferencesFeatureEnabled(): boolean {
  return loaded && snapshot.ffReadingPreferences === true
}

export function uiModeFeatureEnabled(): boolean {
  return loaded && snapshot.ffUiMode === true
}

export function catalogFeatureEnabled(): boolean {
  return loaded && snapshot.ffCatalogIntegration === true
}

export function enrollmentStateMachineFeatureEnabled(): boolean {
  return loaded && snapshot.ffEnrollmentStateMachine === true
}

export function finalGradeSubmissionFeatureEnabled(): boolean {
  return loaded && snapshot.ffGradeSubmission === true
}

export function academicCalendarFeatureEnabled(): boolean {
  return loaded && snapshot.ffAcademicCalendar === true
}

export function plagiarismChecksFeatureEnabled(): boolean {
  return loaded && snapshot.ffPlagiarismChecks === true
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

export function advisingIntegrationEnabled(): boolean {
  return loaded && snapshot.ffAdvisingIntegration === true
}

export function eportfolioFeatureEnabled(): boolean {
  return loaded && snapshot.ffEportfolio === true
}

export function incompleteGradeWorkflowFeatureEnabled(): boolean {
  return loaded && snapshot.ffIncompleteGradeWorkflow === true
}

/** Notebook RAG + flashcards when platform AI is on, OpenRouter is configured, and tenant policy allows it. */
export function ragNotebookAiEnabled(s?: PlatformFeaturesSnapshot): boolean {
  const snap = s ?? snapshot
  if (!s && !loaded) {
    return false
  }
  return snap.ragNotebookEnabled === true
}

/** True when GET/PATCH /api/v1/me/reading-preferences is available (matches server readingPreferencesEnabled). */
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
