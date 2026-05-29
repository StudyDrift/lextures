import { afterEach, describe, expect, it } from 'vitest'
import {
  resetPlatformFeaturesSnapshot,
  setPlatformFeaturesSnapshot,
  studentProgressFeatureEnabled,
} from '../platform-features'

describe('studentProgressFeatureEnabled', () => {
  afterEach(() => {
    resetPlatformFeaturesSnapshot()
  })

  it('is false before platform features load', () => {
    expect(studentProgressFeatureEnabled()).toBe(false)
  })

  it('is true when platform snapshot enables it', () => {
    setPlatformFeaturesSnapshot({
      studentProgressEnabled: true,
      atRiskAlertsEnabled: false,
      h5pEnabled: false,
      oerLibraryEnabled: false,
      itemAnalysisEnabled: false,
      outcomesReportEnabled: false,
      engagementTrackingEnabled: false,
      selfReflectionEnabled: false,
      xapiEmissionEnabled: false,
      equationEditorEnabled: false,
      readingLevelEnabled: false,
      altTextEnforcementEnabled: false,
      ffAltTextEnforcement: false,
      speechToTextEnabled: false,
      translationMemoryEnabled: false,
      storageQuotasEnabled: false,
      avScanningEnabled: false,
      virtualClassroomEnabled: true,
      sessionManagementUiEnabled: false,
      instructorInsightsEnabled: false,
      rtlEnabled: false,
    })
    expect(studentProgressFeatureEnabled()).toBe(true)
  })
})
