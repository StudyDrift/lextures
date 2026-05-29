/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { authorizedFetch } from '../lib/api'
import {
  setPlatformFeaturesSnapshot,
  type PlatformFeaturesSnapshot,
} from '../lib/platform-features'

export type PlatformFeatures = {
  studentProgressEnabled: boolean
  atRiskAlertsEnabled: boolean
  h5pEnabled: boolean
  oerLibraryEnabled: boolean
  itemAnalysisEnabled: boolean
  outcomesReportEnabled: boolean
  engagementTrackingEnabled: boolean
  selfReflectionEnabled: boolean
  xapiEmissionEnabled: boolean
  equationEditorEnabled: boolean
  readingLevelEnabled: boolean
  altTextEnforcementEnabled: boolean
  ffAltTextEnforcement: boolean
  speechToTextEnabled: boolean
  readAloudEnabled: boolean
  ffReadAloud: boolean
  translationMemoryEnabled: boolean
  storageQuotasEnabled: boolean
  avScanningEnabled: boolean
  virtualClassroomEnabled: boolean
  sessionManagementUiEnabled: boolean
  instructorInsightsEnabled: boolean
  rtlEnabled: boolean
  videoCaptionsEnabled: boolean
  autoCaptioningEnabled: boolean
  ffReadingPreferences: boolean
  loading: boolean
  refresh: () => Promise<void>
}

const defaultFeatures: PlatformFeatures = {
  studentProgressEnabled: false,
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
  loading: true,
  refresh: async () => {},
}

const PlatformFeaturesContext = createContext<PlatformFeatures>(defaultFeatures)

type FeaturesPayload = PlatformFeaturesSnapshot

export function PlatformFeaturesProvider({ children }: { children: ReactNode }) {
  const [features, setFeatures] = useState<Omit<PlatformFeatures, 'loading' | 'refresh'>>({
    studentProgressEnabled: false,
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
  })
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/platform/features')
      const raw: unknown = await res.json().catch(() => ({}))
      if (res.ok) {
        const data = raw as FeaturesPayload
        const next: PlatformFeaturesSnapshot = {
          studentProgressEnabled: data.studentProgressEnabled === true,
          atRiskAlertsEnabled: data.atRiskAlertsEnabled === true,
          h5pEnabled: data.h5pEnabled === true,
          oerLibraryEnabled: data.oerLibraryEnabled === true,
          itemAnalysisEnabled: data.itemAnalysisEnabled === true,
          outcomesReportEnabled: data.outcomesReportEnabled === true,
          engagementTrackingEnabled: data.engagementTrackingEnabled === true,
          selfReflectionEnabled: data.selfReflectionEnabled === true,
          xapiEmissionEnabled: data.xapiEmissionEnabled === true,
          equationEditorEnabled: data.equationEditorEnabled === true,
          readingLevelEnabled: data.readingLevelEnabled === true,
          altTextEnforcementEnabled: data.altTextEnforcementEnabled === true,
          ffAltTextEnforcement: data.ffAltTextEnforcement === true,
          speechToTextEnabled: data.speechToTextEnabled === true,
          readAloudEnabled: data.readAloudEnabled === true,
          ffReadAloud: data.ffReadAloud === true,
          translationMemoryEnabled: data.translationMemoryEnabled === true,
          storageQuotasEnabled: data.storageQuotasEnabled === true,
          avScanningEnabled: data.avScanningEnabled === true,
          virtualClassroomEnabled: data.virtualClassroomEnabled !== false,
          sessionManagementUiEnabled: data.sessionManagementUiEnabled === true,
          instructorInsightsEnabled: data.instructorInsightsEnabled === true,
          rtlEnabled: data.rtlEnabled === true,
          videoCaptionsEnabled: data.videoCaptionsEnabled === true,
          autoCaptioningEnabled: data.autoCaptioningEnabled === true,
          ffReadingPreferences: data.ffReadingPreferences === true,
        }
        setFeatures({
          ...next,
          videoCaptionsEnabled: next.videoCaptionsEnabled === true,
          autoCaptioningEnabled: next.autoCaptioningEnabled === true,
          ffReadingPreferences: next.ffReadingPreferences === true,
        })
        setPlatformFeaturesSnapshot(next)
      }
    } catch {
      /* keep defaults */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const value = useMemo(
    () => ({
      ...features,
      loading,
      refresh,
    }),
    [features, loading, refresh],
  )

  return (
    <PlatformFeaturesContext.Provider value={value}>{children}</PlatformFeaturesContext.Provider>
  )
}

export function usePlatformFeatures(): PlatformFeatures {
  return useContext(PlatformFeaturesContext)
}
