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
  equationEditorEnabled: boolean
  storageQuotasEnabled: boolean
  avScanningEnabled: boolean
  virtualClassroomEnabled: boolean
  sessionManagementUiEnabled: boolean
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
  equationEditorEnabled: false,
  storageQuotasEnabled: false,
  avScanningEnabled: false,
  virtualClassroomEnabled: true,
  sessionManagementUiEnabled: false,
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
    equationEditorEnabled: false,
    storageQuotasEnabled: false,
    avScanningEnabled: false,
    virtualClassroomEnabled: true,
    sessionManagementUiEnabled: false,
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
          equationEditorEnabled: data.equationEditorEnabled === true,
          storageQuotasEnabled: data.storageQuotasEnabled === true,
          avScanningEnabled: data.avScanningEnabled === true,
          virtualClassroomEnabled: data.virtualClassroomEnabled !== false,
          sessionManagementUiEnabled: data.sessionManagementUiEnabled === true,
        }
        setFeatures(next)
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
