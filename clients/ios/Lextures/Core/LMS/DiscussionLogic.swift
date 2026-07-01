import Foundation

// MARK: - Course discussions (M7.1)

struct DiscussionNestedPost: Identifiable, Equatable {
    let post: DiscussionPost
    let depth: Int
    var id: String { post.id }
}

enum DiscussionLogic {
    static func nestPosts(_ posts: [DiscussionPost]) -> [DiscussionNestedPost] {
        var byParent: [String?: [DiscussionPost]] = [:]
        for post in posts {
            byParent[post.parentPostId, default: []].append(post)
        }
        for key in byParent.keys {
            byParent[key]?.sort { $0.createdAt < $1.createdAt }
        }

        var output: [DiscussionNestedPost] = []
        func walk(parentId: String?, depth: Int) {
            for child in byParent[parentId] ?? [] {
                output.append(DiscussionNestedPost(post: child, depth: depth))
                walk(parentId: child.id, depth: depth + 1)
            }
        }
        walk(parentId: nil, depth: 0)
        return output
    }

    static func sortThreads(_ threads: [DiscussionThreadSummary]) -> [DiscussionThreadSummary] {
        threads.sorted { lhs, rhs in
            if lhs.isPinned != rhs.isPinned { return lhs.isPinned && !rhs.isPinned }
            return lhs.updatedAt > rhs.updatedAt
        }
    }

    static func authorLabel(authorId: String, viewerId: String?) -> String {
        if authorId == viewerId {
            return L.text("mobile.discussions.authorYou")
        }
        let trimmed = authorId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count > 8 else { return trimmed }
        return String(trimmed.prefix(8))
    }

    static func canReply(thread: DiscussionThreadDetail, viewerIsStaff: Bool) -> Bool {
        !thread.isLocked || viewerIsStaff
    }

    static func canDeletePost(post: DiscussionPost, viewerId: String?) -> Bool {
        guard let viewerId else { return false }
        return post.authorId == viewerId
    }

    static func isBodyEmpty(_ text: String) -> Bool {
        text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    static func plainText(from bodyJSON: Data) -> String {
        guard !bodyJSON.isEmpty,
              let root = try? JSONSerialization.jsonObject(with: bodyJSON) else { return "" }
        return extractText(from: root).trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func encodeBody(text: String) throws -> Data {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let lines = trimmed.isEmpty ? [""] : trimmed.components(separatedBy: "\n")
        let paragraphs: [[String: Any]] = lines.map { line in
            [
                "type": "paragraph",
                "content": [["type": "text", "text": line]],
            ]
        }
        let doc: [String: Any] = ["type": "doc", "content": paragraphs]
        return try JSONSerialization.data(withJSONObject: doc)
    }

    static func emptyBodyJSON() -> Data {
        (try? encodeBody(text: "")) ?? Data()
    }

    private static func extractText(from value: Any) -> String {
        if let string = value as? String { return string }
        if let number = value as? NSNumber { return number.stringValue }
        if let array = value as? [Any] {
            return array.map { extractText(from: $0) }.joined()
        }
        if let dict = value as? [String: Any] {
            if let text = dict["text"] as? String { return text }
            if let content = dict["content"] {
                let separator = (dict["type"] as? String) == "doc" ? "\n" : ""
                if let array = content as? [Any] {
                    return array.map { extractText(from: $0) }.joined(separator: separator)
                }
                return extractText(from: content)
            }
        }
        return ""
    }
}
