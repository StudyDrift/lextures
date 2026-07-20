import Foundation

enum PlatformSettingsAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    /// Deliberate incident-safe allowlist. Infrastructure, identity, billing, and the
    /// mobile-admin flag itself remain web-only to avoid lockout and high-impact mistakes.
    /// In-app feedback is always on (platform master removed) and is not toggled here.
    static let featureDefinitions: [PlatformFeatureDefinition] = [
        definition("ffPublicCatalog", "publicCatalog"),
        definition("ffCourseMarketplace", "courseMarketplace"),
        definition("ffLearningPaths", "learningPaths"),
        definition("ffPeerReview", "peerReview"),
        definition("ffCompletionCredentials", "completionCredentials"),
        definition("ffPersistentTutor", "persistentTutor"),
        definition("ffAiStudyBuddy", "aiStudyBuddy"),
        definition("ffClassroomSignals", "classroomSignals"),
        definition("ffBroadcasts", "broadcasts"),
        definition("ffCalendarFeeds", "calendarFeeds"),
        definition("learnerProfileEnabled", "learnerProfile"),
    ]

    private static func definition(_ key: String, _ name: String) -> PlatformFeatureDefinition {
        PlatformFeatureDefinition(
            key: key,
            labelKey: "mobile.admin.platform.feature.\(name).label",
            descriptionKey: "mobile.admin.platform.feature.\(name).description"
        )
    }

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions: permissions)
    }

    static func canView(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        adminSettingsEnabled(features) && canManage(permissions: permissions)
    }

    static func value(for key: String, in settings: PlatformSettingsSnapshot) -> Bool {
        switch key {
        case "ffPublicCatalog": settings.ffPublicCatalog
        case "ffCourseMarketplace": settings.ffCourseMarketplace
        case "ffLearningPaths": settings.ffLearningPaths
        case "ffPeerReview": settings.ffPeerReview
        case "ffCompletionCredentials": settings.ffCompletionCredentials
        case "ffPersistentTutor": settings.ffPersistentTutor
        case "ffAiStudyBuddy": settings.ffAiStudyBuddy
        case "ffClassroomSignals": settings.ffClassroomSignals
        case "ffBroadcasts": settings.ffBroadcasts
        case "ffCalendarFeeds": settings.ffCalendarFeeds
        case "learnerProfileEnabled": settings.learnerProfileEnabled
        default: false
        }
    }

    static func applyingEffectiveFeatures(
        _ features: PlatformFeatureStates,
        to settings: PlatformSettingsSnapshot
    ) -> PlatformSettingsSnapshot {
        var result = settings
        result.ffPublicCatalog = features.ffPublicCatalog ?? result.ffPublicCatalog
        result.ffCourseMarketplace = features.ffCourseMarketplace ?? result.ffCourseMarketplace
        result.ffLearningPaths = features.ffLearningPaths ?? result.ffLearningPaths
        result.ffPeerReview = features.ffPeerReview ?? result.ffPeerReview
        result.ffCompletionCredentials = features.ffCompletionCredentials ?? result.ffCompletionCredentials
        result.ffPersistentTutor = features.ffPersistentTutor ?? result.ffPersistentTutor
        result.ffAiStudyBuddy = features.ffAiStudyBuddy ?? result.ffAiStudyBuddy
        result.ffClassroomSignals = features.ffClassroomSignals ?? result.ffClassroomSignals
        result.ffBroadcasts = features.ffBroadcasts ?? result.ffBroadcasts
        result.ffCalendarFeeds = features.ffCalendarFeeds ?? result.ffCalendarFeeds
        result.learnerProfileEnabled = features.learnerProfileEnabled ?? result.learnerProfileEnabled
        return result
    }

    static func webSettingsPath() -> String { "/settings/platform" }
}
