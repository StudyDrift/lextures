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
  videoCaptionsEnabled: boolean
  autoCaptioningEnabled: boolean
  ffReadingPreferences: boolean
  ffHighContrastReducedMotion: boolean
  ffLibrary: boolean
  ffBroadcasts: boolean
  ffClassroomSignals: boolean
  ffConferenceScheduling: boolean
  ffDemographics: boolean
  ffContentFilterIntegration: boolean
  ffSisIntegration: boolean
  ffCatalogIntegration: boolean
  ffEnrollmentStateMachine: boolean
  ffGradeSubmission: boolean
  ffPlagiarismChecks: boolean
  ffIncompleteGradeWorkflow: boolean
  ffAcademicCalendar: boolean
  ffCourseEvaluations: boolean
  ffProctoringIntegration: boolean
  ffCoCurricularTranscript: boolean
  ffLibraryIntegration: boolean
  ffEportfolio: boolean
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
  ffGradeSubmission: false,
  ffPlagiarismChecks: false,
  ffIncompleteGradeWorkflow: false,
  ffAcademicCalendar: false,
  ffCourseEvaluations: false,
  ffProctoringIntegration: false,
  ffCoCurricularTranscript: false,
  ffLibraryIntegration: false,
  ffEportfolio: false,
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
    ffGradeSubmission: false,
    ffPlagiarismChecks: false,
    ffIncompleteGradeWorkflow: false,
    ffAcademicCalendar: false,
    ffCourseEvaluations: false,
    ffProctoringIntegration: false,
    ffCoCurricularTranscript: false,
    ffLibraryIntegration: false,
    ffEportfolio: false,
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
          accommodationsEngineEnabled: data.accommodationsEngineEnabled === true,
          ffAccommodationsEngine: data.ffAccommodationsEngine === true,
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
          ffHighContrastReducedMotion: data.ffHighContrastReducedMotion === true,
          ffLibrary: data.ffLibrary === true,
          ffBroadcasts: data.ffBroadcasts === true,
          ffClassroomSignals: data.ffClassroomSignals === true,
          ffConferenceScheduling: data.ffConferenceScheduling === true,
          ffDemographics: data.ffDemographics === true,
          ffContentFilterIntegration: data.ffContentFilterIntegration === true,
          ffSisIntegration: data.ffSisIntegration === true,
          ffCatalogIntegration: data.ffCatalogIntegration === true,
          ffEnrollmentStateMachine: data.ffEnrollmentStateMachine === true,
          ffGradeSubmission: data.ffGradeSubmission === true,
          ffPlagiarismChecks: data.ffPlagiarismChecks === true,
          ffIncompleteGradeWorkflow: data.ffIncompleteGradeWorkflow === true,
          ffAcademicCalendar: data.ffAcademicCalendar === true,
          ffCourseEvaluations: data.ffCourseEvaluations === true,
          ffProctoringIntegration: data.ffProctoringIntegration === true,
          ffCoCurricularTranscript: data.ffCoCurricularTranscript === true,
          ffLibraryIntegration: data.ffLibraryIntegration === true,
          ffEportfolio: data.ffEportfolio === true,
        }
        setFeatures({
          ...next,
          videoCaptionsEnabled: next.videoCaptionsEnabled === true,
          autoCaptioningEnabled: next.autoCaptioningEnabled === true,
          ffReadingPreferences: next.ffReadingPreferences === true,
          ffHighContrastReducedMotion: next.ffHighContrastReducedMotion === true,
          ffLibrary: next.ffLibrary === true,
          ffBroadcasts: next.ffBroadcasts === true,
          ffClassroomSignals: next.ffClassroomSignals === true,
          ffConferenceScheduling: next.ffConferenceScheduling === true,
          ffDemographics: next.ffDemographics === true,
          ffContentFilterIntegration: next.ffContentFilterIntegration === true,
          ffSisIntegration: next.ffSisIntegration === true,
          ffCatalogIntegration: next.ffCatalogIntegration === true,
          ffEnrollmentStateMachine: next.ffEnrollmentStateMachine === true,
          ffGradeSubmission: next.ffGradeSubmission === true,
          ffPlagiarismChecks: next.ffPlagiarismChecks === true,
          ffIncompleteGradeWorkflow: next.ffIncompleteGradeWorkflow === true,
          ffAcademicCalendar: next.ffAcademicCalendar === true,
          ffCourseEvaluations: next.ffCourseEvaluations === true,
          ffProctoringIntegration: next.ffProctoringIntegration === true,
          ffCoCurricularTranscript: next.ffCoCurricularTranscript === true,
          ffLibraryIntegration: next.ffLibraryIntegration === true,
          ffEportfolio: next.ffEportfolio === true,
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
