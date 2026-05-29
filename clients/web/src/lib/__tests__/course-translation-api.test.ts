import { afterEach, describe, expect, it, vi } from 'vitest'
import { resetPlatformFeaturesSnapshot, setPlatformFeaturesSnapshot } from '../platform-features'

describe('course-translation-api feature gate', () => {
  afterEach(() => {
    resetPlatformFeaturesSnapshot()
    vi.resetModules()
  })

  it('isTranslationMemoryEnabled reflects platform snapshot', async () => {
    setPlatformFeaturesSnapshot({
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
      translationMemoryEnabled: true,
      storageQuotasEnabled: false,
      avScanningEnabled: false,
      virtualClassroomEnabled: true,
      sessionManagementUiEnabled: false,
      instructorInsightsEnabled: false,
      rtlEnabled: false,
    })
    const { isTranslationMemoryEnabled } = await import('../course-translation-api')
    expect(isTranslationMemoryEnabled()).toBe(true)
  })
})
