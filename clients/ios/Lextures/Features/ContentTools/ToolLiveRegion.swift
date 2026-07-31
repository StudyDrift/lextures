import UIKit

/// Shared live-region announcements for Content Tools on a screen (FR-15).
@MainActor
enum ToolLiveRegion {
    static func announce(_ message: String, assertive: Bool = false) {
        // UIAccessibility has a single announcement channel; assertive maps to immediate post.
        // Focus is not stolen — callers must not move first responder.
        _ = assertive
        UIAccessibility.post(notification: .announcement, argument: message)
    }
}
