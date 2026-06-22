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
  scormIngestionEnabled: boolean
  oerLibraryEnabled: boolean
  itemAnalysisEnabled: boolean
  outcomesReportEnabled: boolean
  engagementTrackingEnabled: boolean
  selfReflectionEnabled: boolean
  xapiEmissionEnabled: boolean
  equationEditorEnabled: boolean
  readingLevelEnabled: boolean
  graderAgentEnabled?: boolean
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
  ffWebhooks: boolean
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
  ffBookstoreIntegration: boolean
  ffTranscripts: boolean
  ffAdvisingIntegration: boolean
  ffResearchConsent: boolean
  ffAccessibilityIntake: boolean
  ffCeuTracking: boolean
  ffConsortiumSharing: boolean
  ffStripeBilling: boolean
  ffRevenueShare: boolean
  ffLearningPaths: boolean
  ffConditionalRelease: boolean
  ffPeerReview: boolean
  ffCompletionCredentials: boolean
  ffCourseReviews: boolean
  ffGamification: boolean
  ffOnboardingFlow: boolean
  ffStudyReminders: boolean
  ffAiStudyBuddy: boolean
  ffCalendarFeeds: boolean
  aiStudyBuddyEnabled: boolean
  gdprModuleEnabled: boolean
  aiDisclosureEnabled: boolean
  openRouterConfigured: boolean
  ragNotebookEnabled: boolean
  loading: boolean
  refresh: () => Promise<void>
}

const defaultFeatures: PlatformFeatures = {
  studentProgressEnabled: false,
  atRiskAlertsEnabled: false,
  h5pEnabled: false,
  scormIngestionEnabled: false,
  oerLibraryEnabled: false,
  itemAnalysisEnabled: false,
  outcomesReportEnabled: false,
  engagementTrackingEnabled: false,
  selfReflectionEnabled: false,
  xapiEmissionEnabled: false,
  equationEditorEnabled: false,
  readingLevelEnabled: false,
  graderAgentEnabled: false,
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
  ffGradeSubmission: false,
  ffPlagiarismChecks: false,
  ffIncompleteGradeWorkflow: false,
  ffAcademicCalendar: false,
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
  scormIngestionEnabled: false,
    oerLibraryEnabled: false,
    itemAnalysisEnabled: false,
    outcomesReportEnabled: false,
    engagementTrackingEnabled: false,
    selfReflectionEnabled: false,
    xapiEmissionEnabled: false,
    equationEditorEnabled: false,
    readingLevelEnabled: false,
  graderAgentEnabled: false,
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
    ffGradeSubmission: false,
    ffPlagiarismChecks: false,
    ffIncompleteGradeWorkflow: false,
    ffAcademicCalendar: false,
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
          scormIngestionEnabled: data.scormIngestionEnabled === true,
          oerLibraryEnabled: data.oerLibraryEnabled === true,
          itemAnalysisEnabled: data.itemAnalysisEnabled === true,
          outcomesReportEnabled: data.outcomesReportEnabled === true,
          engagementTrackingEnabled: data.engagementTrackingEnabled === true,
          selfReflectionEnabled: data.selfReflectionEnabled === true,
          xapiEmissionEnabled: data.xapiEmissionEnabled === true,
          equationEditorEnabled: data.equationEditorEnabled === true,
          readingLevelEnabled: data.readingLevelEnabled === true,
          graderAgentEnabled: data.graderAgentEnabled === true,
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
          ffWebhooks: data.ffWebhooks === true,
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
          ffBookstoreIntegration: data.ffBookstoreIntegration === true,
          ffTranscripts: data.ffTranscripts === true,
          ffAdvisingIntegration: data.ffAdvisingIntegration === true,
          ffResearchConsent: data.ffResearchConsent === true,
          ffAccessibilityIntake: data.ffAccessibilityIntake === true,
          ffCeuTracking: data.ffCeuTracking === true,
          ffConsortiumSharing: data.ffConsortiumSharing === true,
          ffStripeBilling: data.ffStripeBilling === true,
          ffRevenueShare: data.ffRevenueShare === true,
          ffLearningPaths: data.ffLearningPaths === true,
          ffConditionalRelease: data.ffConditionalRelease === true,
          ffPeerReview: data.ffPeerReview === true,
          ffCompletionCredentials: data.ffCompletionCredentials === true,
          ffCourseReviews: data.ffCourseReviews === true,
          ffGamification: data.ffGamification === true,
          ffOnboardingFlow: data.ffOnboardingFlow === true,
          ffStudyReminders: data.ffStudyReminders === true,
          ffAiStudyBuddy: data.ffAiStudyBuddy === true,
          ffCalendarFeeds: data.ffCalendarFeeds === true,
          aiStudyBuddyEnabled: data.aiStudyBuddyEnabled === true,
          gdprModuleEnabled: data.gdprModuleEnabled === true,
          aiDisclosureEnabled: data.aiDisclosureEnabled === true,
          openRouterConfigured: data.openRouterConfigured === true,
          ragNotebookEnabled: data.ragNotebookEnabled === true,
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
          ffWebhooks: next.ffWebhooks === true,
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
          ffBookstoreIntegration: next.ffBookstoreIntegration === true,
          ffTranscripts: next.ffTranscripts === true,
          ffAdvisingIntegration: next.ffAdvisingIntegration === true,
          ffResearchConsent: next.ffResearchConsent === true,
          ffAccessibilityIntake: next.ffAccessibilityIntake === true,
          ffCeuTracking: next.ffCeuTracking === true,
          ffConsortiumSharing: next.ffConsortiumSharing === true,
          ffStripeBilling: next.ffStripeBilling === true,
          ffRevenueShare: next.ffRevenueShare === true,
          ffLearningPaths: next.ffLearningPaths === true,
          ffConditionalRelease: next.ffConditionalRelease === true,
          ffPeerReview: next.ffPeerReview === true,
          ffCompletionCredentials: next.ffCompletionCredentials === true,
          ffCourseReviews: next.ffCourseReviews === true,
          ffGamification: next.ffGamification === true,
          ffOnboardingFlow: next.ffOnboardingFlow === true,
          ffStudyReminders: next.ffStudyReminders === true,
          ffAiStudyBuddy: next.ffAiStudyBuddy === true,
          ffCalendarFeeds: next.ffCalendarFeeds === true,
          aiStudyBuddyEnabled: next.aiStudyBuddyEnabled === true,
          gdprModuleEnabled: next.gdprModuleEnabled === true,
          aiDisclosureEnabled: next.aiDisclosureEnabled === true,
          openRouterConfigured: next.openRouterConfigured === true,
          ragNotebookEnabled: next.ragNotebookEnabled === true,
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
