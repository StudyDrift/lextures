import Foundation

/// Secret-free subset of `GET /api/v1/settings/platform` used by mobile admin.
/// Unknown fields (including masked secret fields) are intentionally not decoded.
struct PlatformSettingsSnapshot: Decodable, Equatable {
    var ffFeedback: Bool
    var ffPublicCatalog: Bool
    var ffCourseMarketplace: Bool
    var ffLearningPaths: Bool
    var ffPeerReview: Bool
    var ffCompletionCredentials: Bool
    var ffPersistentTutor: Bool
    var ffAiStudyBuddy: Bool
    var ffClassroomSignals: Bool
    var ffBroadcasts: Bool
    var ffCalendarFeeds: Bool
    var learnerProfileEnabled: Bool

    var samlSsoEnabled: Bool
    var samlPublicBaseUrl: String
    var samlSpEntityId: String
    var mfaEnabled: Bool
    var mfaEnforcement: String
    var smtpHost: String
    var smtpPort: Int
    var smtpFrom: String
}

/// Effective states from the public runtime feature endpoint. Config fields are
/// deliberately absent because this payload is used only to override flag state.
struct PlatformFeatureStates: Decodable {
    var ffFeedback: Bool?
    var ffPublicCatalog: Bool?
    var ffCourseMarketplace: Bool?
    var ffLearningPaths: Bool?
    var ffPeerReview: Bool?
    var ffCompletionCredentials: Bool?
    var ffPersistentTutor: Bool?
    var ffAiStudyBuddy: Bool?
    var ffClassroomSignals: Bool?
    var ffBroadcasts: Bool?
    var ffCalendarFeeds: Bool?
    var learnerProfileEnabled: Bool?
}

struct PlatformFeatureDefinition: Identifiable, Equatable {
    let key: String
    let labelKey: String
    let descriptionKey: String

    var id: String { key }
}
