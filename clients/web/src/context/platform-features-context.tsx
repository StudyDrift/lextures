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
  learnerProfileEnabled?: boolean
  introCourseEnabled?: boolean
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
  videoCaptionsEnabled: boolean
  autoCaptioningEnabled: boolean
  ffReadingPreferences: boolean
  ffHighContrastReducedMotion: boolean
  ffMotionNavigation: boolean
  ffMotionReveal: boolean
  ffMotionLists: boolean
  ffMobileCreateCourse: boolean
  ffMobileCourseCreateV2: boolean
  ffMobileCanvasImport: boolean
  ffMobileAdminConsole: boolean
  ffMobileEnrollmentAdd: boolean
  ffLibrary: boolean
  ffBroadcasts: boolean
  ffClassroomSignals: boolean
  ffConferenceScheduling: boolean
  ffParentPortal: boolean
  ffParentPortalV2: boolean
  ffReportCards: boolean
  ffDemographics: boolean
  ffContentFilterIntegration: boolean
  ffSisIntegration: boolean
  ffWebhooks: boolean
  adminConsoleEnabled: boolean
  adminAuditLogEnabled: boolean
  impersonationEnabled: boolean
  bulkCsvImportEnabled: boolean
  adminSearchEnabled: boolean
  emailTemplateEditorEnabled: boolean
  maintenanceBannerEnabled: boolean
  seatManagementEnabled: boolean
  ffZapierConnector: boolean
  ffCatalogIntegration: boolean
  ffEnrollmentStateMachine: boolean
  ffGradeSubmission: boolean
  ffWhatifGrades: boolean
  ffGradeCurving: boolean
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
  ffTranscriptInbound: boolean
  ffDiplomas: boolean
  ffAdvisingIntegration: boolean
  ffResearchConsent: boolean
  ffAccessibilityIntake: boolean
  ffCeuTracking: boolean
  ffConsortiumSharing: boolean
  ffStripeBilling: boolean
  ffPaymentsEnabled: boolean
  ffRevenueShare: boolean
  ffTaxCollection: boolean
  ffCourseMarketplace?: boolean
  ffLearningPaths: boolean
  ffConditionalRelease: boolean
  ffPeerReview: boolean
  ffCompletionCredentials: boolean
  ffCourseReviews: boolean
  ffGamification: boolean
  ffCompetencyBadges: boolean
  ffOnboardingFlow: boolean
  ffStudyReminders: boolean
  ffAiStudyBuddy: boolean
  ffLessonGenerator: boolean
  ffPersistentTutor: boolean
  ffCalendarFeeds: boolean
  aiStudyBuddyEnabled: boolean
  gdprModuleEnabled: boolean
  aiDisclosureEnabled: boolean
  /** @deprecated AP.9 — use aiConfigured */
  openRouterConfigured: boolean
  aiConfigured: boolean
  aiProvidersConfigured: string[]
  aiProviderAbstractionEnabled: boolean
  ragNotebookEnabled: boolean
  ffFeedback: boolean
  ffVisualBoards: boolean
  ffBoardsRealtime: boolean
  ffBoardsExternalSharing: boolean
  ffInteractiveQuizzes: boolean
  ffIqLiveHosting: boolean
  ffIqTeamMode: boolean
  ffIqStudentPaced: boolean
  ffIqHomework: boolean
  ffIqGradebookPush: boolean
  ffIqPublicKitCatalog: boolean
  ffIqGuestJoin: boolean
  ffIqAiGeneration: boolean
  ffEmailSes: boolean
  loading: boolean
  refresh: () => Promise<void>
}

const defaultFeatures: PlatformFeatures = {
  studentProgressEnabled: false,
  atRiskAlertsEnabled: false,
  h5pEnabled: false,
  scormIngestionEnabled: false,
  oerLibraryEnabled: false,
  itemAnalysisEnabled: true,
  outcomesReportEnabled: true,
  engagementTrackingEnabled: false,
  selfReflectionEnabled: false,
  learnerProfileEnabled: true,
  introCourseEnabled: true,
  xapiEmissionEnabled: false,
  equationEditorEnabled: true,
  readingLevelEnabled: false,
  graderAgentEnabled: false,
  graderAgentReviewInboxEnabled: false,
  graderAgentSuggestModeEnabled: false,
  graderAgentTextEntryGradingEnabled: true,
  graderAgentVisionGradingEnabled: false,
  graderAgentRunFiltersEnabled: false,
  graderAgentCostEstimateEnabled: false,
  graderAgentCancelRunEnabled: false,
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
  sessionManagementUiEnabled: true,
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
  ffMobileEnrollmentAdd: false,
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
  adminConsoleEnabled: true,
  adminAuditLogEnabled: true,
  impersonationEnabled: false,
  bulkCsvImportEnabled: false,
  adminSearchEnabled: true,
  emailTemplateEditorEnabled: false,
  maintenanceBannerEnabled: true,
  seatManagementEnabled: false,
  ffZapierConnector: false,
  ffCatalogIntegration: false,
  ffEnrollmentStateMachine: false,
  ffGradeSubmission: false,
  ffWhatifGrades: true,
  ffGradeCurving: true,
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
  ffConditionalRelease: true,
  ffPeerReview: true,
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
  ffBoardsRealtime: true,
  ffBoardsExternalSharing: false,
  ffInteractiveQuizzes: true,
  ffIqLiveHosting: true,
  ffIqTeamMode: true,
  ffIqStudentPaced: true,
  ffIqHomework: true,
  ffIqGradebookPush: true,
  ffIqPublicKitCatalog: false,
  ffIqGuestJoin: false,
  ffIqAiGeneration: false,
  ffEmailSes: false,
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
    itemAnalysisEnabled: true,
    outcomesReportEnabled: true,
    engagementTrackingEnabled: false,
    selfReflectionEnabled: false,
  learnerProfileEnabled: true,
  introCourseEnabled: true,
    xapiEmissionEnabled: false,
    equationEditorEnabled: true,
    readingLevelEnabled: false,
  graderAgentEnabled: false,
  graderAgentReviewInboxEnabled: false,
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
    sessionManagementUiEnabled: true,
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
    ffMobileEnrollmentAdd: false,
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
  adminConsoleEnabled: true,
  adminAuditLogEnabled: true,
  impersonationEnabled: false,
  bulkCsvImportEnabled: false,
  adminSearchEnabled: true,
  emailTemplateEditorEnabled: false,
  maintenanceBannerEnabled: true,
  seatManagementEnabled: false,
  ffZapierConnector: false,
    ffCatalogIntegration: false,
    ffEnrollmentStateMachine: false,
    ffGradeSubmission: false,
  ffWhatifGrades: true,
  ffGradeCurving: true,
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
  ffConditionalRelease: true,
  ffPeerReview: true,
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
  ffBoardsRealtime: true,
  ffBoardsExternalSharing: false,
  ffInteractiveQuizzes: true,
  ffIqLiveHosting: true,
  ffIqTeamMode: true,
  ffIqStudentPaced: true,
  ffIqHomework: true,
  ffIqGradebookPush: true,
  ffIqPublicKitCatalog: false,
  ffIqGuestJoin: false,
  ffIqAiGeneration: false,
  ffEmailSes: false,
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
          learnerProfileEnabled: data.learnerProfileEnabled !== false,
          introCourseEnabled: data.introCourseEnabled !== false,
          xapiEmissionEnabled: data.xapiEmissionEnabled === true,
          equationEditorEnabled: data.equationEditorEnabled === true,
          readingLevelEnabled: data.readingLevelEnabled === true,
          graderAgentEnabled: data.graderAgentEnabled === true,
          graderAgentReviewInboxEnabled: data.graderAgentReviewInboxEnabled === true,
          graderAgentSuggestModeEnabled: data.graderAgentSuggestModeEnabled === true,
          graderAgentTextEntryGradingEnabled: data.graderAgentTextEntryGradingEnabled !== false,
          graderAgentVisionGradingEnabled: data.graderAgentVisionGradingEnabled === true,
          graderAgentRunFiltersEnabled: data.graderAgentRunFiltersEnabled === true,
          graderAgentCostEstimateEnabled: data.graderAgentCostEstimateEnabled === true,
          graderAgentCancelRunEnabled: data.graderAgentCancelRunEnabled === true,
          codeExecutionEnabled: data.codeExecutionEnabled === true,
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
          ffMotionNavigation: data.ffMotionNavigation !== false,
          ffMotionReveal: data.ffMotionReveal !== false,
          ffMotionLists: data.ffMotionLists !== false,
          ffMobileCreateCourse: data.ffMobileCreateCourse === true,
          ffMobileCourseCreateV2: data.ffMobileCourseCreateV2 === true,
          ffMobileCanvasImport: data.ffMobileCanvasImport === true,
          ffMobileAdminConsole: data.ffMobileAdminConsole === true,
          ffMobileEnrollmentAdd: data.ffMobileEnrollmentAdd === true,
          ffLibrary: data.ffLibrary === true,
          ffBroadcasts: data.ffBroadcasts === true,
          ffClassroomSignals: data.ffClassroomSignals === true,
          ffConferenceScheduling: data.ffConferenceScheduling === true,
          ffParentPortal: data.ffParentPortal === true,
          ffParentPortalV2: data.ffParentPortalV2 === true,
          ffReportCards: data.ffReportCards === true,
          ffDemographics: data.ffDemographics === true,
          ffContentFilterIntegration: data.ffContentFilterIntegration === true,
          ffSisIntegration: data.ffSisIntegration === true,
          ffWebhooks: data.ffWebhooks === true,
          adminConsoleEnabled: data.adminConsoleEnabled === true,
          adminAuditLogEnabled: data.adminAuditLogEnabled !== false,
          impersonationEnabled: data.impersonationEnabled === true,
          bulkCsvImportEnabled: data.bulkCsvImportEnabled === true,
          adminSearchEnabled: data.adminSearchEnabled === true,
          emailTemplateEditorEnabled: data.emailTemplateEditorEnabled === true,
          maintenanceBannerEnabled: data.maintenanceBannerEnabled !== false,
          seatManagementEnabled: data.seatManagementEnabled === true,
          ffZapierConnector: data.ffZapierConnector === true,
          ffCatalogIntegration: data.ffCatalogIntegration === true,
          ffEnrollmentStateMachine: data.ffEnrollmentStateMachine === true,
          ffGradeSubmission: data.ffGradeSubmission === true,
          ffWhatifGrades: data.ffWhatifGrades === true,
          ffGradeCurving: data.ffGradeCurving === true,
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
          ffTranscriptInbound: data.ffTranscriptInbound === true,
          ffDiplomas: data.ffDiplomas === true,
          ffAdvisingIntegration: data.ffAdvisingIntegration === true,
          ffResearchConsent: data.ffResearchConsent === true,
          ffAccessibilityIntake: data.ffAccessibilityIntake === true,
          ffCeuTracking: data.ffCeuTracking === true,
          ffConsortiumSharing: data.ffConsortiumSharing === true,
          ffStripeBilling: data.ffStripeBilling === true,
          ffPaymentsEnabled: data.ffPaymentsEnabled === true,
          ffRevenueShare: data.ffRevenueShare === true,
          ffTaxCollection: data.ffTaxCollection === true,
          ffCourseMarketplace: data.ffCourseMarketplace !== false,
          ffLearningPaths: data.ffLearningPaths === true,
          ffConditionalRelease: data.ffConditionalRelease === true,
          ffPeerReview: data.ffPeerReview === true,
          ffCompletionCredentials: data.ffCompletionCredentials === true,
          ffCourseReviews: data.ffCourseReviews === true,
          ffGamification: data.ffGamification === true,
          ffCompetencyBadges: data.ffCompetencyBadges === true,
          ffOnboardingFlow: data.ffOnboardingFlow === true,
          ffStudyReminders: data.ffStudyReminders === true,
          ffAiStudyBuddy: data.ffAiStudyBuddy === true,
          ffLessonGenerator: data.ffLessonGenerator === true,
          ffPersistentTutor: data.ffPersistentTutor === true,
          ffCalendarFeeds: data.ffCalendarFeeds === true,
          aiStudyBuddyEnabled: data.aiStudyBuddyEnabled === true,
          gdprModuleEnabled: data.gdprModuleEnabled === true,
          aiDisclosureEnabled: data.aiDisclosureEnabled === true,
          openRouterConfigured: data.openRouterConfigured === true,
          aiConfigured: data.aiConfigured === true,
          aiProvidersConfigured: Array.isArray(data.aiProvidersConfigured)
            ? data.aiProvidersConfigured.filter((p) => typeof p === 'string')
            : [],
          aiProviderAbstractionEnabled: data.aiProviderAbstractionEnabled === true,
          ragNotebookEnabled: data.ragNotebookEnabled === true,
          ffFeedback: data.ffFeedback !== false,
          ffVisualBoards: true,
          ffBoardsRealtime: data.ffBoardsRealtime === true,
          ffBoardsExternalSharing: data.ffBoardsExternalSharing === true,
          ffInteractiveQuizzes: true,
          ffIqLiveHosting: data.ffIqLiveHosting !== false,
          ffIqTeamMode: data.ffIqTeamMode === true,
          ffIqStudentPaced: data.ffIqStudentPaced === true,
          ffIqHomework: data.ffIqHomework === true,
          ffIqGradebookPush: data.ffIqGradebookPush === true,
          ffIqPublicKitCatalog: data.ffIqPublicKitCatalog === true,
          ffIqGuestJoin: data.ffIqGuestJoin === true,
          ffIqAiGeneration: data.ffIqAiGeneration === true,
          ffEmailSes: data.ffEmailSes === true,
        }
        setFeatures({
          ...next,
          videoCaptionsEnabled: next.videoCaptionsEnabled === true,
          autoCaptioningEnabled: next.autoCaptioningEnabled === true,
          ffReadingPreferences: next.ffReadingPreferences === true,
          ffHighContrastReducedMotion: next.ffHighContrastReducedMotion === true,
          ffMotionNavigation: next.ffMotionNavigation !== false,
          ffMotionReveal: next.ffMotionReveal !== false,
          ffMotionLists: next.ffMotionLists !== false,
          ffMobileCreateCourse: next.ffMobileCreateCourse === true,
          ffMobileCourseCreateV2: next.ffMobileCourseCreateV2 === true,
          ffMobileCanvasImport: next.ffMobileCanvasImport === true,
          ffMobileAdminConsole: next.ffMobileAdminConsole === true,
          ffMobileEnrollmentAdd: next.ffMobileEnrollmentAdd === true,
          ffLibrary: next.ffLibrary === true,
          ffBroadcasts: next.ffBroadcasts === true,
          ffClassroomSignals: next.ffClassroomSignals === true,
          ffConferenceScheduling: next.ffConferenceScheduling === true,
          ffParentPortal: next.ffParentPortal === true,
          ffParentPortalV2: next.ffParentPortalV2 === true,
          ffReportCards: next.ffReportCards === true,
          ffDemographics: next.ffDemographics === true,
          ffContentFilterIntegration: next.ffContentFilterIntegration === true,
          ffSisIntegration: next.ffSisIntegration === true,
          ffWebhooks: next.ffWebhooks === true,
          adminConsoleEnabled: next.adminConsoleEnabled === true,
          adminAuditLogEnabled: next.adminAuditLogEnabled !== false,
          impersonationEnabled: next.impersonationEnabled === true,
          bulkCsvImportEnabled: next.bulkCsvImportEnabled === true,
          adminSearchEnabled: next.adminSearchEnabled === true,
          emailTemplateEditorEnabled: next.emailTemplateEditorEnabled === true,
          maintenanceBannerEnabled: next.maintenanceBannerEnabled !== false,
          seatManagementEnabled: next.seatManagementEnabled === true,
          ffZapierConnector: next.ffZapierConnector === true,
          ffCatalogIntegration: next.ffCatalogIntegration === true,
          ffEnrollmentStateMachine: next.ffEnrollmentStateMachine === true,
          ffGradeSubmission: next.ffGradeSubmission === true,
          ffWhatifGrades: next.ffWhatifGrades === true,
          ffGradeCurving: next.ffGradeCurving === true,
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
          ffTranscriptInbound: next.ffTranscriptInbound === true,
          ffDiplomas: next.ffDiplomas === true,
          ffAdvisingIntegration: next.ffAdvisingIntegration === true,
          ffResearchConsent: next.ffResearchConsent === true,
          ffAccessibilityIntake: next.ffAccessibilityIntake === true,
          ffCeuTracking: next.ffCeuTracking === true,
          ffConsortiumSharing: next.ffConsortiumSharing === true,
          ffStripeBilling: next.ffStripeBilling === true,
          ffPaymentsEnabled: next.ffPaymentsEnabled === true,
          ffRevenueShare: next.ffRevenueShare === true,
          ffTaxCollection: next.ffTaxCollection === true,
          ffCourseMarketplace: next.ffCourseMarketplace !== false,
          ffLearningPaths: next.ffLearningPaths === true,
          ffConditionalRelease: next.ffConditionalRelease === true,
          ffPeerReview: next.ffPeerReview === true,
          ffCompletionCredentials: next.ffCompletionCredentials === true,
          ffCourseReviews: next.ffCourseReviews === true,
          ffGamification: next.ffGamification === true,
          ffCompetencyBadges: next.ffCompetencyBadges === true,
          ffOnboardingFlow: next.ffOnboardingFlow === true,
          ffStudyReminders: next.ffStudyReminders === true,
          ffAiStudyBuddy: next.ffAiStudyBuddy === true,
          ffLessonGenerator: next.ffLessonGenerator === true,
          ffPersistentTutor: next.ffPersistentTutor === true,
          ffCalendarFeeds: next.ffCalendarFeeds === true,
          aiStudyBuddyEnabled: next.aiStudyBuddyEnabled === true,
          gdprModuleEnabled: next.gdprModuleEnabled === true,
          aiDisclosureEnabled: next.aiDisclosureEnabled === true,
          openRouterConfigured: next.openRouterConfigured === true,
          aiConfigured: next.aiConfigured === true,
          aiProvidersConfigured: next.aiProvidersConfigured ?? [],
          aiProviderAbstractionEnabled: next.aiProviderAbstractionEnabled === true,
          ragNotebookEnabled: next.ragNotebookEnabled === true,
          ffFeedback: next.ffFeedback !== false,
          ffVisualBoards: true,
          ffBoardsRealtime: next.ffBoardsRealtime === true,
          ffBoardsExternalSharing: next.ffBoardsExternalSharing === true,
          ffInteractiveQuizzes: true,
          ffIqLiveHosting: next.ffIqLiveHosting !== false,
          ffIqTeamMode: next.ffIqTeamMode === true,
          ffIqStudentPaced: next.ffIqStudentPaced === true,
          ffIqHomework: next.ffIqHomework === true,
          ffIqGradebookPush: next.ffIqGradebookPush === true,
          ffIqPublicKitCatalog: next.ffIqPublicKitCatalog === true,
          ffIqGuestJoin: next.ffIqGuestJoin === true,
          ffIqAiGeneration: next.ffIqAiGeneration === true,
          ffEmailSes: next.ffEmailSes === true,
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
