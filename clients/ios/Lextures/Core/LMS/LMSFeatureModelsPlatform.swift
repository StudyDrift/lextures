import Foundation

// MARK: - Platform features

/// GET `/api/v1/platform/features` (subset used on mobile).
struct PlatformFeatures: Decodable {
    var ffWhatifGrades: Bool?
    var feedbackMediaEnabled: Bool?
    var ffLibrary: Bool?
    var ffCourseEvaluations: Bool?
    var ffMobileCourseEvaluations: Bool?
    var ffMobileIaRedesign: Bool?
    var ffMobileVibeActivities: Bool?
    var ffMobileUniversalSearch: Bool?
    var ffMobileProfileDepth: Bool?
    var ffMobileLibraryEreserves: Bool?
    var ffMobileImmersiveReader: Bool?
    var ffMobileLiveMeetings: Bool?
    var readAloudEnabled: Bool?
    var ffReadAloud: Bool?
    var videoCaptionsEnabled: Bool?
    var autoCaptioningEnabled: Bool?
    var translationMemoryEnabled: Bool?
    var ffReadingPreferences: Bool?
    var oerLibraryEnabled: Bool?
    var customFieldsEnabled: Bool?
    var ffDemographics: Bool?
    var ffResearchConsent: Bool?
    var ffPersistentTutor: Bool?
    var ffAiStudyBuddy: Bool?
    var ragNotebookEnabled: Bool?
    var aiStudyBuddyEnabled: Bool?
    var aiDisclosureEnabled: Bool?
    var ffPeerReview: Bool?
    var ffLearningPaths: Bool?
    var selfReflectionEnabled: Bool?
    var ffPublicCatalog: Bool?
    var ffSelfPacedMode: Bool?
    var ffCourseReviews: Bool?
    var ffCompletionCredentials: Bool?
    var ffGamification: Bool?
    var ffStripeBilling: Bool?
    var ffPaymentsEnabled: Bool?
    var ffTaxCollection: Bool?
    var ffAdvisingIntegration: Bool?
    var ffMobileAdvising: Bool?
    var ffParentPortal: Bool?
    var ffConferenceScheduling: Bool?
}