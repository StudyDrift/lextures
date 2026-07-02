import Foundation

// MARK: - Group spaces & collab docs (M7.4)

struct GroupPublic: Codable, Identifiable, Hashable {
    var id: String
    var groupSetId: String
    var name: String
    var sortOrder: Int
    var createdAt: String
    var memberCount: Int
}

struct GroupsListResponse: Decodable {
    var groups: [GroupPublic]?
}

enum CollabDocType: String, Codable, Hashable {
    case richText = "rich_text"
    case whiteboard
}

struct CollabDoc: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var groupId: String?
    var title: String
    var docType: CollabDocType
    var createdBy: String
    var createdAt: String
    var updatedAt: String
}

struct CollabDocsListResponse: Decodable {
    var docs: [CollabDoc]?
}

struct CollabDocSnapshot: Codable, Identifiable, Hashable {
    var id: String
    var docId: String
    var authorId: String
    var takenAt: String
}

struct CollabDocSnapshotsResponse: Decodable {
    var snapshots: [CollabDocSnapshot]?
}

struct GroupFeedContext: Hashable {
    var groupId: String
    var groupName: String
}

struct GroupChannelRoute: Hashable {
    var groupId: String
    var groupName: String
    var channelId: String
    var channelName: String
}

struct GroupSpaceRoute: Hashable {
    var group: GroupPublic
}

struct CollabDocRoute: Hashable {
    var docId: String
    var title: String
}