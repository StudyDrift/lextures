import Foundation

/// Pure helpers for the course feed (M7.6) kept separate from views for testability.
enum FeedLogic {
    /// Flattens root messages + their replies into a single chronological list, pinned
    /// messages first (root-only; replies can't be pinned per the server contract).
    static func orderedMessages(_ roots: [FeedMessage]) -> [FeedMessage] {
        let pinned = roots.filter { $0.pinnedAt != nil }.sorted { $0.createdAt < $1.createdAt }
        let unpinned = roots.filter { $0.pinnedAt == nil }
        var flattened: [FeedMessage] = []
        for root in unpinned.sorted(by: { $0.createdAt < $1.createdAt }) {
            flattened.append(root)
            flattened.append(contentsOf: root.replies.sorted { $0.createdAt < $1.createdAt })
        }
        return pinned + flattened
    }

    static func canEdit(_ message: FeedMessage, viewerId: String?) -> Bool {
        guard let viewerId else { return false }
        return message.authorUserId == viewerId
    }

    static func canDelete(_ message: FeedMessage, viewerId: String?) -> Bool {
        canEdit(message, viewerId: viewerId)
    }

    static func canPin(viewerIsStaff: Bool, isReply: Bool) -> Bool {
        viewerIsStaff && !isReply
    }

    /// Markdown image syntax the web composer emits when a feed image is attached:
    /// `![alt](path)`. Extracts the path, if any, so the bubble can render it.
    static func extractImagePath(from body: String) -> (text: String, imagePath: String?) {
        guard
            let range = body.range(of: #"!\[[^\]]*\]\(([^)]+)\)"#, options: .regularExpression)
        else {
            return (body, nil)
        }
        let markdown = String(body[range])
        guard
            let parenOpen = markdown.firstIndex(of: "("),
            let parenClose = markdown.lastIndex(of: ")")
        else {
            return (body, nil)
        }
        let path = String(markdown[markdown.index(after: parenOpen) ..< parenClose])
        var remainder = body
        remainder.removeSubrange(range)
        return (remainder.trimmingCharacters(in: .whitespacesAndNewlines), path)
    }
}
