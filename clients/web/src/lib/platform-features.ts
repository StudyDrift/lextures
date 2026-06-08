/** Snapshot of GET /api/v1/platform/features; updated by PlatformFeaturesProvider. */

export type PlatformFeaturesSnapshot = {
  studentProgressEnabled: boolean
  atRiskAlertsEnabled: boolean
  h5pEnabled: boolean
  oerLibraryEnabled: boolean
  itemAnalysisEnabled: boolean
  engagementTrackingEnabled: boolean
  selfReflectionEnabled: boolean
  outcomesReportEnabled: boolean
  xapiEmissionEnabled: boolean
  equationEditorEnabled: boolean
  readingLevelEnabled: boolean
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
  ffCatalogIntegration?: boolean
  ffEnrollmentStateMachine?: boolean
  ffIncompleteGradeWorkflow?: boolean
  ffUiMode?: boolean
  ffGradeSubmission?: boolean
  ffAcademicCalendar?: boolean
}

const defaults: PlatformFeaturesSnapshot = {
  studentProgressEnabled: false,
  atRiskAlertsEnabled: false,
  h5pEnabled: false,
  oerLibraryEnabled: false,
  itemAnalysisEnabled: false,
  engagementTrackingEnabled: false,
  selfReflectionEnabled: false,
  outcomesReportEnabled: false,
  xapiEmissionEnabled: false,
  equationEditorEnabled: false,
  readingLevelEnabled: false,
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
  ffCatalogIntegration: false,
  ffEnrollmentStateMachine: false,
  ffIncompleteGradeWorkflow: false,
  ffUiMode: false,
  ffGradeSubmission: false,
  ffAcademicCalendar: false,
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

export function incompleteGradeWorkflowFeatureEnabled(): boolean {
  return loaded && snapshot.ffIncompleteGradeWorkflow === true
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
