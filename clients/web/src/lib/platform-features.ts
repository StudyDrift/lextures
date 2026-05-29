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

export function readingPreferencesFeatureEnabled(): boolean {
  return loaded && snapshot.ffReadingPreferences === true
}
