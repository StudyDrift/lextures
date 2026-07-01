import Foundation

enum FilePreviewKind: Equatable {
    case image
    case pdf
    case audio
    case video
    case downloadOnly
}

// MARK: - Discussions (M7.1)

struct DiscussionForum: Codable, Identifiable, Hashable {
    var id: String
    var name: String
    var description: String?
    var position: Int
    var createdAt: String
}

struct DiscussionThreadSummary: Codable, Identifiable, Hashable {
    var id: String
    var forumId: String
    var authorId: String
    var title: String
    var isPinned: Bool
    var isLocked: Bool
    var requirePostFirst: Bool
    var assignmentStructureItemId: String?
    var createdAt: String
    var updatedAt: String
    var replyCount: Int
}

struct DiscussionThreadDetail: Codable, Identifiable, Hashable {
    var id: String
    var forumId: String
    var authorId: String
    var title: String
    var isPinned: Bool
    var isLocked: Bool
    var requirePostFirst: Bool
    var assignmentStructureItemId: String?
    var createdAt: String
    var updatedAt: String
    var replyCount: Int
    var bodyJSON: Data

    enum CodingKeys: String, CodingKey {
        case id, forumId, authorId, title, isPinned, isLocked, requirePostFirst
        case assignmentStructureItemId, createdAt, updatedAt, replyCount, body
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        forumId = try container.decode(String.self, forKey: .forumId)
        authorId = try container.decode(String.self, forKey: .authorId)
        title = try container.decode(String.self, forKey: .title)
        isPinned = try container.decodeIfPresent(Bool.self, forKey: .isPinned) ?? false
        isLocked = try container.decodeIfPresent(Bool.self, forKey: .isLocked) ?? false
        requirePostFirst = try container.decodeIfPresent(Bool.self, forKey: .requirePostFirst) ?? false
        assignmentStructureItemId = try container.decodeIfPresent(String.self, forKey: .assignmentStructureItemId)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? createdAt
        replyCount = try container.decodeIfPresent(Int.self, forKey: .replyCount) ?? 0
        if let object = try? container.decode([String: DiscussionJSONValue].self, forKey: .body) {
            bodyJSON = try JSONSerialization.data(withJSONObject: object.mapValues(\.foundationValue))
        } else {
            bodyJSON = DiscussionLogic.emptyBodyJSON()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(forumId, forKey: .forumId)
        try container.encode(authorId, forKey: .authorId)
        try container.encode(title, forKey: .title)
        try container.encode(isPinned, forKey: .isPinned)
        try container.encode(isLocked, forKey: .isLocked)
        try container.encode(requirePostFirst, forKey: .requirePostFirst)
        try container.encodeIfPresent(assignmentStructureItemId, forKey: .assignmentStructureItemId)
        try container.encode(createdAt, forKey: .createdAt)
        try container.encode(updatedAt, forKey: .updatedAt)
        try container.encode(replyCount, forKey: .replyCount)
        if let object = try JSONSerialization.jsonObject(with: bodyJSON) as? [String: Any] {
            try container.encode(object.mapValues { DiscussionJSONValue(foundationValue: $0) }, forKey: .body)
        }
    }

    var bodyPlainText: String { DiscussionLogic.plainText(from: bodyJSON) }
}

struct DiscussionPost: Codable, Identifiable, Hashable {
    var id: String
    var threadId: String
    var parentPostId: String?
    var authorId: String
    var bodyJSON: Data
    var upvoteCount: Int
    var viewerUpvoted: Bool
    var createdAt: String
    var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, threadId, parentPostId, authorId, body
        case upvoteCount, viewerUpvoted, createdAt, updatedAt
    }

    init(
        id: String,
        threadId: String,
        parentPostId: String?,
        authorId: String,
        bodyJSON: Data,
        upvoteCount: Int,
        viewerUpvoted: Bool,
        createdAt: String,
        updatedAt: String
    ) {
        self.id = id
        self.threadId = threadId
        self.parentPostId = parentPostId
        self.authorId = authorId
        self.bodyJSON = bodyJSON
        self.upvoteCount = upvoteCount
        self.viewerUpvoted = viewerUpvoted
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        threadId = try container.decode(String.self, forKey: .threadId)
        parentPostId = try container.decodeIfPresent(String.self, forKey: .parentPostId)
        authorId = try container.decode(String.self, forKey: .authorId)
        upvoteCount = try container.decodeIfPresent(Int.self, forKey: .upvoteCount) ?? 0
        viewerUpvoted = try container.decodeIfPresent(Bool.self, forKey: .viewerUpvoted) ?? false
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? createdAt
        if let object = try? container.decode([String: DiscussionJSONValue].self, forKey: .body) {
            bodyJSON = try JSONSerialization.data(withJSONObject: object.mapValues(\.foundationValue))
        } else {
            bodyJSON = DiscussionLogic.emptyBodyJSON()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(threadId, forKey: .threadId)
        try container.encodeIfPresent(parentPostId, forKey: .parentPostId)
        try container.encode(authorId, forKey: .authorId)
        try container.encode(upvoteCount, forKey: .upvoteCount)
        try container.encode(viewerUpvoted, forKey: .viewerUpvoted)
        try container.encode(createdAt, forKey: .createdAt)
        try container.encode(updatedAt, forKey: .updatedAt)
        if let object = try JSONSerialization.jsonObject(with: bodyJSON) as? [String: Any] {
            try container.encode(object.mapValues { DiscussionJSONValue(foundationValue: $0) }, forKey: .body)
        }
    }

    var bodyPlainText: String { DiscussionLogic.plainText(from: bodyJSON) }
}

private enum DiscussionJSONValue: Hashable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case object([String: DiscussionJSONValue])
    case array([DiscussionJSONValue])
    case null

    var foundationValue: Any {
        switch self {
        case .string(let value): return value
        case .number(let value): return value
        case .bool(let value): return value
        case .object(let value): return value.mapValues(\.foundationValue)
        case .array(let value): return value.map(\.foundationValue)
        case .null: return NSNull()
        }
    }

    init(foundationValue: Any) {
        switch foundationValue {
        case let value as String: self = .string(value)
        case let value as Double: self = .number(value)
        case let value as Int: self = .number(Double(value))
        case let value as Bool: self = .bool(value)
        case let value as [String: Any]:
            self = .object(value.mapValues { DiscussionJSONValue(foundationValue: $0) })
        case let value as [Any]:
            self = .array(value.map { DiscussionJSONValue(foundationValue: $0) })
        default:
            self = .null
        }
    }
}

extension DiscussionJSONValue: Codable {
    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let value = try? container.decode(Bool.self) {
            self = .bool(value)
        } else if let value = try? container.decode(Double.self) {
            self = .number(value)
        } else if let value = try? container.decode(String.self) {
            self = .string(value)
        } else if let value = try? container.decode([String: DiscussionJSONValue].self) {
            self = .object(value)
        } else if let value = try? container.decode([DiscussionJSONValue].self) {
            self = .array(value)
        } else {
            self = .null
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let value): try container.encode(value)
        case .number(let value): try container.encode(value)
        case .bool(let value): try container.encode(value)
        case .object(let value): try container.encode(value)
        case .array(let value): try container.encode(value)
        case .null: try container.encodeNil()
        }
    }
}

struct DiscussionForumsResponse: Codable {
    var forums: [DiscussionForum]?
}

struct DiscussionThreadsResponse: Codable {
    var threads: [DiscussionThreadSummary]?
}

struct DiscussionPostsResponse: Codable {
    var posts: [DiscussionPost]?
    var hiddenUntilFirstPost: Bool?
}

struct CreateDiscussionThreadRequest: Encodable {
    var title: String
    var body: Data
    var assignmentStructureItemId: String?
    var requirePostFirst: Bool?

    enum CodingKeys: String, CodingKey {
        case title, body, assignmentStructureItemId, requirePostFirst
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(title, forKey: .title)
        let object = try JSONSerialization.jsonObject(with: body)
        try container.encode(AnyEncodableJSON(object), forKey: .body)
        try container.encodeIfPresent(assignmentStructureItemId, forKey: .assignmentStructureItemId)
        try container.encodeIfPresent(requirePostFirst, forKey: .requirePostFirst)
    }
}

struct CreateDiscussionPostRequest: Encodable {
    var parentPostId: String?
    var body: Data
    var idempotencyKey: String?

    enum CodingKeys: String, CodingKey {
        case parentPostId, body, idempotencyKey
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encodeIfPresent(parentPostId, forKey: .parentPostId)
        let object = try JSONSerialization.jsonObject(with: body)
        try container.encode(AnyEncodableJSON(object), forKey: .body)
        try container.encodeIfPresent(idempotencyKey, forKey: .idempotencyKey)
    }
}

struct DiscussionUpvoteResponse: Decodable {
    var wasAdded: Bool
    var upvoteCount: Int
}

private struct AnyEncodableJSON: Encodable {
    let value: Any

    init(_ value: Any) { self.value = value }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case let value as String: try container.encode(value)
        case let value as Bool: try container.encode(value)
        case let value as Int: try container.encode(value)
        case let value as Double: try container.encode(value)
        case is NSNull: try container.encodeNil()
        case let value as [Any]:
            try container.encode(value.map { AnyEncodableJSON($0) })
        case let value as [String: Any]:
            try container.encode(value.mapValues { AnyEncodableJSON($0) })
        default:
            try container.encodeNil()
        }
    }
}

struct DiscussionThreadRoute: Hashable {
    var threadId: String
}
