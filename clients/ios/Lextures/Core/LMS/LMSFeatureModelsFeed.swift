import Foundation

// MARK: - Course feed & channels (M7.6)

struct FeedChannel: Codable, Identifiable, Hashable {
    var id: String
    var name: String
    var sortOrder: Int
    var createdAt: String
}

struct FeedMessage: Codable, Identifiable, Hashable {
    var id: String
    var channelId: String
    var authorUserId: String
    var authorEmail: String
    var authorDisplayName: String?
    var parentMessageId: String?
    var body: String
    var mentionsEveryone: Bool
    var mentionUserIds: [String]
    var pinnedAt: String?
    var createdAt: String
    var editedAt: String?
    var likeCount: Int
    var viewerHasLiked: Bool
    var replies: [FeedMessage]

    var authorLabel: String {
        authorDisplayName?.isEmpty == false ? authorDisplayName! : authorEmail
    }

    init(
        id: String,
        channelId: String,
        authorUserId: String,
        authorEmail: String,
        authorDisplayName: String?,
        parentMessageId: String?,
        body: String,
        mentionsEveryone: Bool,
        mentionUserIds: [String],
        pinnedAt: String?,
        createdAt: String,
        editedAt: String?,
        likeCount: Int,
        viewerHasLiked: Bool,
        replies: [FeedMessage]
    ) {
        self.id = id
        self.channelId = channelId
        self.authorUserId = authorUserId
        self.authorEmail = authorEmail
        self.authorDisplayName = authorDisplayName
        self.parentMessageId = parentMessageId
        self.body = body
        self.mentionsEveryone = mentionsEveryone
        self.mentionUserIds = mentionUserIds
        self.pinnedAt = pinnedAt
        self.createdAt = createdAt
        self.editedAt = editedAt
        self.likeCount = likeCount
        self.viewerHasLiked = viewerHasLiked
        self.replies = replies
    }

    // The server can send `null` (not `[]`) for `mentionUserIds`/`replies` when a message has
    // neither — decode those leniently as empty arrays rather than failing the whole payload.
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        channelId = try container.decode(String.self, forKey: .channelId)
        authorUserId = try container.decode(String.self, forKey: .authorUserId)
        authorEmail = try container.decode(String.self, forKey: .authorEmail)
        authorDisplayName = try container.decodeIfPresent(String.self, forKey: .authorDisplayName)
        parentMessageId = try container.decodeIfPresent(String.self, forKey: .parentMessageId)
        body = try container.decode(String.self, forKey: .body)
        mentionsEveryone = try container.decode(Bool.self, forKey: .mentionsEveryone)
        mentionUserIds = try container.decodeIfPresent([String].self, forKey: .mentionUserIds) ?? []
        pinnedAt = try container.decodeIfPresent(String.self, forKey: .pinnedAt)
        createdAt = try container.decode(String.self, forKey: .createdAt)
        editedAt = try container.decodeIfPresent(String.self, forKey: .editedAt)
        likeCount = try container.decode(Int.self, forKey: .likeCount)
        viewerHasLiked = try container.decode(Bool.self, forKey: .viewerHasLiked)
        replies = try container.decodeIfPresent([FeedMessage].self, forKey: .replies) ?? []
    }
}

struct FeedRosterPerson: Codable, Identifiable, Hashable {
    var userId: String
    var email: String
    var displayName: String?
    var avatarUrl: String?

    var id: String { userId }

    var label: String {
        displayName?.isEmpty == false ? displayName! : email
    }
}

struct FeedImageUpload: Decodable {
    var id: String
    var contentPath: String
    var mimeType: String
    var byteSize: Int64

    enum CodingKeys: String, CodingKey {
        case id
        case contentPath = "content_path"
        case mimeType = "mime_type"
        case byteSize = "byte_size"
    }
}

struct FeedChannelsResponse: Decodable {
    var channels: [FeedChannel]
}

struct FeedMessagesResponse: Decodable {
    var messages: [FeedMessage]
}

struct FeedRosterResponse: Decodable {
    var people: [FeedRosterPerson]
}

struct CreateFeedChannelRequest: Encodable {
    var name: String
}

struct PostFeedMessageRequest: Encodable {
    var body: String
    var parentMessageId: String?
    var mentionUserIds: [String]
    var mentionsEveryone: Bool
}

struct PatchFeedMessageRequest: Encodable {
    var body: String
}

struct PinFeedMessageRequest: Encodable {
    var pinned: Bool
}

struct PostFeedMessageResponse: Decodable {
    var id: String
}
