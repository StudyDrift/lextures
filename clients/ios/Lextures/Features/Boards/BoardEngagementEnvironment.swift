import SwiftUI

/// Shared handlers for card reactions/comments/moderation (VC.M5 / VC.M7).
struct BoardEngagementHandlers {
    var courseCode: String
    var onPostUpdate: (BoardPost) -> Void
    var onAnnounce: (String) -> Void
    var onHidePost: ((BoardPost) -> Void)? = nil
    var onRemovePost: ((BoardPost) -> Void)? = nil
}

private struct BoardEngagementHandlersKey: EnvironmentKey {
    static let defaultValue: BoardEngagementHandlers? = nil
}

extension EnvironmentValues {
    var boardEngagement: BoardEngagementHandlers? {
        get { self[BoardEngagementHandlersKey.self] }
        set { self[BoardEngagementHandlersKey.self] = newValue }
    }
}
