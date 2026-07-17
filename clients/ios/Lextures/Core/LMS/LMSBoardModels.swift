import Foundation

// Decoder locals and canvas position fields (x/y/w/h) match API JSON keys.
// swiftlint:disable identifier_name

/// Visual collaboration board (VC.M1–VC.M7). Subset of web `Board`; unknown fields ignored.
struct Board: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var courseId: String
    var title: String
    var description: String
    var slug: String
    var archived: Bool
    var layout: String
    var layoutLocked: Bool
    var reactionMode: String
    var assignmentId: String?
    var visibility: String
    var visibilityTarget: String?
    var attribution: String
    var canPost: Bool?
    var canInteract: Bool?
    var canArrange: Bool?
    var moderationMode: String
    var filterAction: String
    var locked: Bool
    var frozenUntil: String?
    var capabilities: BoardCapabilities?
    var externalSharingAllowed: Bool?
    var minorModerationFloor: Bool?
    var createdBy: String?
    var createdAt: String
    var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, courseId, title, description, slug, archived, layout, layoutLocked
        case reactionMode, assignmentId, visibility, visibilityTarget, attribution
        case canPost, canInteract, canArrange
        case moderationMode, filterAction, locked, frozenUntil
        case capabilities, externalSharingAllowed, minorModerationFloor
        case createdBy, createdAt, updatedAt
    }

    init(
        id: String,
        courseId: String,
        title: String,
        description: String = "",
        slug: String = "",
        archived: Bool = false,
        layout: String = "wall",
        layoutLocked: Bool = false,
        reactionMode: String = "none",
        assignmentId: String? = nil,
        visibility: String = "course",
        visibilityTarget: String? = nil,
        attribution: String = "named",
        canPost: Bool? = nil,
        canInteract: Bool? = nil,
        canArrange: Bool? = nil,
        moderationMode: String = "open",
        filterAction: String = "flag",
        locked: Bool = false,
        frozenUntil: String? = nil,
        capabilities: BoardCapabilities? = nil,
        externalSharingAllowed: Bool? = nil,
        minorModerationFloor: Bool? = nil,
        createdBy: String? = nil,
        createdAt: String = "",
        updatedAt: String = ""
    ) {
        self.id = id
        self.courseId = courseId
        self.title = title
        self.description = description
        self.slug = slug
        self.archived = archived
        self.layout = layout
        self.layoutLocked = layoutLocked
        self.reactionMode = reactionMode
        self.assignmentId = assignmentId
        self.visibility = visibility
        self.visibilityTarget = visibilityTarget
        self.attribution = attribution
        self.canPost = canPost
        self.canInteract = canInteract
        self.canArrange = canArrange
        self.moderationMode = moderationMode
        self.filterAction = filterAction
        self.locked = locked
        self.frozenUntil = frozenUntil
        self.capabilities = capabilities
        self.externalSharingAllowed = externalSharingAllowed
        self.minorModerationFloor = minorModerationFloor
        self.createdBy = createdBy
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        courseId = try c.decode(String.self, forKey: .courseId)
        title = try c.decode(String.self, forKey: .title)
        description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        slug = try c.decodeIfPresent(String.self, forKey: .slug) ?? ""
        archived = try c.decodeIfPresent(Bool.self, forKey: .archived) ?? false
        layout = try c.decodeIfPresent(String.self, forKey: .layout) ?? "wall"
        layoutLocked = try c.decodeIfPresent(Bool.self, forKey: .layoutLocked) ?? false
        reactionMode = try c.decodeIfPresent(String.self, forKey: .reactionMode) ?? "none"
        assignmentId = try c.decodeIfPresent(String.self, forKey: .assignmentId)
        visibility = try c.decodeIfPresent(String.self, forKey: .visibility) ?? "course"
        visibilityTarget = try c.decodeIfPresent(String.self, forKey: .visibilityTarget)
        attribution = try c.decodeIfPresent(String.self, forKey: .attribution) ?? "named"
        canPost = try c.decodeIfPresent(Bool.self, forKey: .canPost)
        canInteract = try c.decodeIfPresent(Bool.self, forKey: .canInteract)
        canArrange = try c.decodeIfPresent(Bool.self, forKey: .canArrange)
        moderationMode = try c.decodeIfPresent(String.self, forKey: .moderationMode) ?? "open"
        filterAction = try c.decodeIfPresent(String.self, forKey: .filterAction) ?? "flag"
        locked = try c.decodeIfPresent(Bool.self, forKey: .locked) ?? false
        frozenUntil = try c.decodeIfPresent(String.self, forKey: .frozenUntil)
        capabilities = try c.decodeIfPresent(BoardCapabilities.self, forKey: .capabilities)
        externalSharingAllowed = try c.decodeIfPresent(Bool.self, forKey: .externalSharingAllowed)
        minorModerationFloor = try c.decodeIfPresent(Bool.self, forKey: .minorModerationFloor)
        createdBy = try c.decodeIfPresent(String.self, forKey: .createdBy)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

enum BoardVisibility: String, CaseIterable, Codable, Hashable {
    case course, section, group, invite, link, `public`

    /// In-course scopes (always offered). `link`/`public` added when external sharing is allowed (VC.M7).
    static let inCourse: [BoardVisibility] = [.course, .section, .group, .invite]
}

enum BoardModerationMode: String, CaseIterable, Codable, Hashable {
    case open, approval
}

enum BoardFilterAction: String, CaseIterable, Codable, Hashable {
    case flag, block
}

enum BoardPostStatus: String, CaseIterable, Codable, Hashable {
    case approved, pending, rejected
}

enum BoardReportKind: String, CaseIterable, Codable, Hashable {
    case user
    case filter
    case avBlocked = "av_blocked"
}

enum BoardReportStatus: String, CaseIterable, Codable, Hashable {
    case open, resolved, dismissed
}

enum BoardPostSafetyState: Equatable {
    case normal
    case pendingApproval
    case removed
    case fileScanning
    case fileBlocked
}

enum BoardAttribution: String, CaseIterable, Codable, Hashable {
    case named
    case anonToPeers = "anon_to_peers"
    case anonymous
}

enum BoardMemberRole: String, CaseIterable, Codable, Hashable {
    case owner, editor, contributor, viewer
}

enum BoardShareCapability: String, CaseIterable, Codable, Hashable {
    case view, contribute
}

enum BoardReactionMode: String, CaseIterable, Codable, Hashable {
    case none, like, vote, star, grade

    static func fromAPI(_ raw: String?) -> BoardReactionMode {
        guard let raw, let mode = BoardReactionMode(rawValue: raw.lowercased()) else { return .none }
        return mode
    }
}

struct BoardMyReaction: Codable, Hashable, Equatable {
    var kind: String
    var value: Double?

    enum CodingKeys: String, CodingKey { case kind, value }

    init(kind: String, value: Double? = nil) {
        self.kind = kind
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        if let d = try c.decodeIfPresent(Double.self, forKey: .value) {
            value = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .value) {
            value = Double(i)
        } else {
            value = nil
        }
    }
}

struct BoardComment: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var postId: String
    var parentId: String?
    var authorId: String?
    var body: BoardPostBody?
    var hidden: Bool
    var createdAt: String
    var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, postId, parentId, authorId, body, hidden, createdAt, updatedAt
    }

    init(
        id: String,
        postId: String,
        parentId: String? = nil,
        authorId: String? = nil,
        body: BoardPostBody? = nil,
        hidden: Bool = false,
        createdAt: String = "",
        updatedAt: String = ""
    ) {
        self.id = id
        self.postId = postId
        self.parentId = parentId
        self.authorId = authorId
        self.body = body
        self.hidden = hidden
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        postId = try c.decode(String.self, forKey: .postId)
        parentId = try c.decodeIfPresent(String.self, forKey: .parentId)
        authorId = try c.decodeIfPresent(String.self, forKey: .authorId)
        body = try c.decodeIfPresent(BoardPostBody.self, forKey: .body)
        hidden = try c.decodeIfPresent(Bool.self, forKey: .hidden) ?? false
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct BoardCommentsListResponse: Decodable {
    var comments: [BoardComment]?
}

struct PutBoardReactionRequest: Encodable {
    var kind: String?
    var value: Double?
}

struct CreateBoardCommentRequest: Encodable {
    var body: BoardPostBody
    var parentId: String?
}

struct PatchBoardCommentRequest: Encodable {
    var body: BoardPostBody?
    var hidden: Bool?
}

struct BoardReactionResult: Codable, Hashable, Equatable {
    var active: Bool
    var removed: Bool?
    var reactionCount: Int?
    var myReaction: BoardMyReaction?
    var avgStars: Double?
    var commentCount: Int?
    var grade: Double?

    enum CodingKeys: String, CodingKey {
        case active, removed, reactionCount, myReaction, avgStars, commentCount, grade
    }

    init(
        active: Bool = false,
        removed: Bool? = nil,
        reactionCount: Int? = nil,
        myReaction: BoardMyReaction? = nil,
        avgStars: Double? = nil,
        commentCount: Int? = nil,
        grade: Double? = nil
    ) {
        self.active = active
        self.removed = removed
        self.reactionCount = reactionCount
        self.myReaction = myReaction
        self.avgStars = avgStars
        self.commentCount = commentCount
        self.grade = grade
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        active = try c.decodeIfPresent(Bool.self, forKey: .active) ?? false
        removed = try c.decodeIfPresent(Bool.self, forKey: .removed)
        reactionCount = try c.decodeIfPresent(Int.self, forKey: .reactionCount)
        myReaction = try c.decodeIfPresent(BoardMyReaction.self, forKey: .myReaction)
        if let d = try c.decodeIfPresent(Double.self, forKey: .avgStars) {
            avgStars = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .avgStars) {
            avgStars = Double(i)
        } else {
            avgStars = nil
        }
        commentCount = try c.decodeIfPresent(Int.self, forKey: .commentCount)
        if let d = try c.decodeIfPresent(Double.self, forKey: .grade) {
            grade = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .grade) {
            grade = Double(i)
        } else {
            grade = nil
        }
    }
}

struct BoardGradeSyncResult: Codable, Hashable, Equatable {
    var synced: Bool
    var pointsEarned: Double

    enum CodingKeys: String, CodingKey { case synced, pointsEarned }

    init(synced: Bool = false, pointsEarned: Double = 0) {
        self.synced = synced
        self.pointsEarned = pointsEarned
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        synced = try c.decodeIfPresent(Bool.self, forKey: .synced) ?? false
        if let d = try c.decodeIfPresent(Double.self, forKey: .pointsEarned) {
            pointsEarned = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .pointsEarned) {
            pointsEarned = Double(i)
        } else {
            pointsEarned = 0
        }
    }
}

enum BoardLayout: String, CaseIterable, Codable, Hashable {
    case wall, stream, grid, columns, canvas, timeline, map

    static let allAPIValues: [String] = allCases.map(\.rawValue)
}

enum BoardSortMode: String, CaseIterable, Hashable {
    case newest, oldest, author, mostReacted
}

struct BoardSection: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var boardId: String
    var title: String
    var sortIndex: Double
    var createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, boardId, title, sortIndex, createdAt
    }

    init(
        id: String,
        boardId: String,
        title: String,
        sortIndex: Double = 0,
        createdAt: String = ""
    ) {
        self.id = id
        self.boardId = boardId
        self.title = title
        self.sortIndex = sortIndex
        self.createdAt = createdAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        boardId = try c.decode(String.self, forKey: .boardId)
        title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        if let d = try c.decodeIfPresent(Double.self, forKey: .sortIndex) {
            sortIndex = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .sortIndex) {
            sortIndex = Double(i)
        } else {
            sortIndex = 0
        }
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct BoardSectionsListResponse: Decodable {
    var sections: [BoardSection]?
}

struct CreateBoardSectionRequest: Encodable {
    var title: String
    var sortIndex: Double?
}

struct PatchBoardSectionRequest: Encodable {
    var title: String?
    var sortIndex: Double?
}

struct ArrangeBoardPostInput: Encodable, Hashable, Equatable {
    var sectionId: String?
    var sortIndex: Double?
    var position: BoardPostPosition?
    var eventDate: String?
    var lat: Double?
    var lng: Double?
    var clearGeo: Bool?

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encodeIfPresent(sectionId, forKey: .sectionId)
        try c.encodeIfPresent(sortIndex, forKey: .sortIndex)
        try c.encodeIfPresent(position, forKey: .position)
        try c.encodeIfPresent(eventDate, forKey: .eventDate)
        try c.encodeIfPresent(lat, forKey: .lat)
        try c.encodeIfPresent(lng, forKey: .lng)
        if clearGeo == true {
            try c.encode(true, forKey: .clearGeo)
        }
    }

    enum CodingKeys: String, CodingKey {
        case sectionId, sortIndex, position, eventDate, lat, lng, clearGeo
    }
}

struct BoardCapabilities: Codable, Hashable, Equatable {
    var canView: Bool?
    var canPost: Bool?
    var canInteract: Bool?
    var canArrange: Bool?
    var canManage: Bool?
}

struct BoardsListResponse: Decodable {
    var boards: [Board]?
}

struct CreateBoardRequest: Encodable {
    var title: String
    var description: String
}

struct PatchBoardRequest: Encodable {
    var title: String?
    var description: String?
    var archived: Bool?
    var layout: String?
    var layoutLocked: Bool?
    var visibility: String?
    var visibilityTarget: String?
    var attribution: String?
    var canPost: Bool?
    var canInteract: Bool?
    var canArrange: Bool?
    var moderationMode: String?
    var filterAction: String?
    var locked: Bool?
    var frozenUntil: String?
    var freezeMinutes: Int?
}

struct BoardReport: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var boardId: String
    var postId: String?
    var commentId: String?
    var reporterId: String?
    var reason: String
    var kind: String
    var status: String
    var createdAt: String
    var resolvedAt: String?
    var resolvedBy: String?

    enum CodingKeys: String, CodingKey {
        case id, boardId, postId, commentId, reporterId, reason, kind, status
        case createdAt, resolvedAt, resolvedBy
    }

    init(
        id: String,
        boardId: String,
        postId: String? = nil,
        commentId: String? = nil,
        reporterId: String? = nil,
        reason: String = "",
        kind: String = "user",
        status: String = "open",
        createdAt: String = "",
        resolvedAt: String? = nil,
        resolvedBy: String? = nil
    ) {
        self.id = id
        self.boardId = boardId
        self.postId = postId
        self.commentId = commentId
        self.reporterId = reporterId
        self.reason = reason
        self.kind = kind
        self.status = status
        self.createdAt = createdAt
        self.resolvedAt = resolvedAt
        self.resolvedBy = resolvedBy
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        boardId = try c.decodeIfPresent(String.self, forKey: .boardId) ?? ""
        postId = try c.decodeIfPresent(String.self, forKey: .postId)
        commentId = try c.decodeIfPresent(String.self, forKey: .commentId)
        reporterId = try c.decodeIfPresent(String.self, forKey: .reporterId)
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? "user"
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "open"
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        resolvedAt = try c.decodeIfPresent(String.self, forKey: .resolvedAt)
        resolvedBy = try c.decodeIfPresent(String.self, forKey: .resolvedBy)
    }
}

struct BoardModerationQueue: Decodable {
    var pending: [BoardPost]?
    var reports: [BoardReport]?
    var flagged: [BoardReport]?
    var minorsFloor: Bool?

    var pendingPosts: [BoardPost] { pending ?? [] }
    var userReports: [BoardReport] { reports ?? [] }
    var flaggedReports: [BoardReport] { flagged ?? [] }
}

struct CreateBoardReportRequest: Encodable {
    var postId: String?
    var commentId: String?
    var reason: String?
}

struct ResolveBoardReportRequest: Encodable {
    var action: String
    var reason: String?
}

struct BoardModerationActionRequest: Encodable {
    var reason: String?
}

struct BoardMember: Codable, Identifiable, Hashable, Equatable {
    var boardId: String
    var userId: String
    var role: String
    var createdAt: String

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case boardId, userId, role, createdAt
    }

    init(boardId: String, userId: String, role: String, createdAt: String = "") {
        self.boardId = boardId
        self.userId = userId
        self.role = role
        self.createdAt = createdAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        boardId = try c.decodeIfPresent(String.self, forKey: .boardId) ?? ""
        userId = try c.decode(String.self, forKey: .userId)
        role = try c.decodeIfPresent(String.self, forKey: .role) ?? "contributor"
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct BoardMembersListResponse: Decodable {
    var members: [BoardMember]?
}

struct UpsertBoardMemberRequest: Encodable {
    var userId: String
    var role: String
}

struct BoardShare: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var boardId: String
    var capability: String
    var hasPassword: Bool
    var expiresAt: String?
    var revokedAt: String?
    var createdBy: String?
    var createdAt: String
    var token: String?
    var url: String?

    enum CodingKeys: String, CodingKey {
        case id, boardId, capability, hasPassword, expiresAt, revokedAt
        case createdBy, createdAt, token, url
    }

    init(
        id: String,
        boardId: String,
        capability: String,
        hasPassword: Bool = false,
        expiresAt: String? = nil,
        revokedAt: String? = nil,
        createdBy: String? = nil,
        createdAt: String = "",
        token: String? = nil,
        url: String? = nil
    ) {
        self.id = id
        self.boardId = boardId
        self.capability = capability
        self.hasPassword = hasPassword
        self.expiresAt = expiresAt
        self.revokedAt = revokedAt
        self.createdBy = createdBy
        self.createdAt = createdAt
        self.token = token
        self.url = url
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        boardId = try c.decodeIfPresent(String.self, forKey: .boardId) ?? ""
        capability = try c.decodeIfPresent(String.self, forKey: .capability) ?? "view"
        hasPassword = try c.decodeIfPresent(Bool.self, forKey: .hasPassword) ?? false
        expiresAt = try c.decodeIfPresent(String.self, forKey: .expiresAt)
        revokedAt = try c.decodeIfPresent(String.self, forKey: .revokedAt)
        createdBy = try c.decodeIfPresent(String.self, forKey: .createdBy)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        token = try c.decodeIfPresent(String.self, forKey: .token)
        url = try c.decodeIfPresent(String.self, forKey: .url)
    }
}

struct BoardSharesListResponse: Decodable {
    var shares: [BoardShare]?
}

struct CreateBoardShareRequest: Encodable {
    var capability: String
    var password: String?
    var expiresAt: String?
}

struct BoardLinkResolve: Decodable {
    var board: Board
    var capability: String
    var requiresPassword: Bool?
    var posts: [BoardPost]?
}

struct CreateBoardLinkPostRequest: Encodable {
    var displayName: String
    var contentType: String
    var title: String?
    var body: BoardPostBody?
    var linkUrl: String?
}

enum BoardLinkAccessState: Equatable {
    case loading
    case needsPassword
    case denied
    case ready
}

struct BoardRoute: Hashable, Identifiable {
    var id: String { boardId }
    var boardId: String
    var title: String
}

// MARK: - Posts (VC.M2)

enum BoardContentType: String, CaseIterable, Codable, Hashable {
    case text
    case image
    case file
    case link
    case video
    case audio
    case drawing

    static let known: Set<String> = Set(allCases.map(\.rawValue))
}

struct BoardAttachment: Codable, Hashable, Equatable, Identifiable {
    var id: String
    var url: String?
    var fileName: String
    var mimeType: String
    var sizeBytes: Int64
    var altText: String
    var scanStatus: String

    enum CodingKeys: String, CodingKey {
        case id, url, fileName, mimeType, sizeBytes, altText, scanStatus
    }

    init(
        id: String,
        url: String? = nil,
        fileName: String = "",
        mimeType: String = "",
        sizeBytes: Int64 = 0,
        altText: String = "",
        scanStatus: String = "pending"
    ) {
        self.id = id
        self.url = url
        self.fileName = fileName
        self.mimeType = mimeType
        self.sizeBytes = sizeBytes
        self.altText = altText
        self.scanStatus = scanStatus
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        url = try c.decodeIfPresent(String.self, forKey: .url)
        fileName = try c.decodeIfPresent(String.self, forKey: .fileName) ?? ""
        mimeType = try c.decodeIfPresent(String.self, forKey: .mimeType) ?? ""
        if let n = try c.decodeIfPresent(Int64.self, forKey: .sizeBytes) {
            sizeBytes = n
        } else if let d = try c.decodeIfPresent(Double.self, forKey: .sizeBytes) {
            sizeBytes = Int64(d)
        } else {
            sizeBytes = 0
        }
        altText = try c.decodeIfPresent(String.self, forKey: .altText) ?? ""
        scanStatus = try c.decodeIfPresent(String.self, forKey: .scanStatus) ?? "pending"
    }
}

struct BoardLinkPreview: Codable, Hashable, Equatable {
    var title: String?
    var description: String?
    var image: String?
    var siteName: String?
    var fetchedAt: String?
    var url: String?
    var provider: String?
    var embedId: String?
}

struct BoardPostBody: Codable, Hashable, Equatable {
    var html: String?
    var text: String?
}

struct BoardPostPosition: Codable, Hashable, Equatable {
    var x: Double?
    var y: Double?
    var w: Double?
    var h: Double?
}

struct BoardPost: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var boardId: String
    var authorId: String?
    var guestDisplayName: String?
    var contentType: String
    var title: String
    var body: BoardPostBody?
    var linkUrl: String?
    var linkPreview: BoardLinkPreview?
    var drawingData: JSONValue?
    var attachment: BoardAttachment?
    var sectionId: String?
    var sortIndex: Double
    var position: BoardPostPosition?
    var eventDate: String?
    var lat: Double?
    var lng: Double?
    var status: String
    var hidden: Bool
    var reactionCount: Int?
    var myReaction: BoardMyReaction?
    var avgStars: Double?
    var commentCount: Int?
    var grade: Double?
    var createdAt: String
    var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, boardId, authorId, guestDisplayName, contentType, title, body, linkUrl, linkPreview
        case drawingData, attachment, sectionId, sortIndex, position, eventDate, lat, lng
        case status, hidden
        case reactionCount, myReaction, avgStars, commentCount, grade, createdAt, updatedAt
    }

    init(
        id: String,
        boardId: String,
        authorId: String? = nil,
        guestDisplayName: String? = nil,
        contentType: String,
        title: String = "",
        body: BoardPostBody? = nil,
        linkUrl: String? = nil,
        linkPreview: BoardLinkPreview? = nil,
        drawingData: JSONValue? = nil,
        attachment: BoardAttachment? = nil,
        sectionId: String? = nil,
        sortIndex: Double = 0,
        position: BoardPostPosition? = nil,
        eventDate: String? = nil,
        lat: Double? = nil,
        lng: Double? = nil,
        status: String = "approved",
        hidden: Bool = false,
        reactionCount: Int? = nil,
        myReaction: BoardMyReaction? = nil,
        avgStars: Double? = nil,
        commentCount: Int? = nil,
        grade: Double? = nil,
        createdAt: String = "",
        updatedAt: String = ""
    ) {
        self.id = id
        self.boardId = boardId
        self.authorId = authorId
        self.guestDisplayName = guestDisplayName
        self.contentType = contentType
        self.title = title
        self.body = body
        self.linkUrl = linkUrl
        self.linkPreview = linkPreview
        self.drawingData = drawingData
        self.attachment = attachment
        self.sectionId = sectionId
        self.sortIndex = sortIndex
        self.position = position
        self.eventDate = eventDate
        self.lat = lat
        self.lng = lng
        self.status = status
        self.hidden = hidden
        self.reactionCount = reactionCount
        self.myReaction = myReaction
        self.avgStars = avgStars
        self.commentCount = commentCount
        self.grade = grade
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        boardId = try c.decode(String.self, forKey: .boardId)
        authorId = try c.decodeIfPresent(String.self, forKey: .authorId)
        guestDisplayName = try c.decodeIfPresent(String.self, forKey: .guestDisplayName)
        contentType = try c.decode(String.self, forKey: .contentType)
        title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        body = try c.decodeIfPresent(BoardPostBody.self, forKey: .body)
        linkUrl = try c.decodeIfPresent(String.self, forKey: .linkUrl)
        linkPreview = try c.decodeIfPresent(BoardLinkPreview.self, forKey: .linkPreview)
        drawingData = try c.decodeIfPresent(JSONValue.self, forKey: .drawingData)
        attachment = try c.decodeIfPresent(BoardAttachment.self, forKey: .attachment)
        sectionId = try c.decodeIfPresent(String.self, forKey: .sectionId)
        if let d = try c.decodeIfPresent(Double.self, forKey: .sortIndex) {
            sortIndex = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .sortIndex) {
            sortIndex = Double(i)
        } else {
            sortIndex = 0
        }
        position = try c.decodeIfPresent(BoardPostPosition.self, forKey: .position)
        eventDate = try c.decodeIfPresent(String.self, forKey: .eventDate)
        lat = try c.decodeIfPresent(Double.self, forKey: .lat)
        lng = try c.decodeIfPresent(Double.self, forKey: .lng)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "approved"
        hidden = try c.decodeIfPresent(Bool.self, forKey: .hidden) ?? false
        reactionCount = try c.decodeIfPresent(Int.self, forKey: .reactionCount)
        myReaction = try c.decodeIfPresent(BoardMyReaction.self, forKey: .myReaction)
        if let d = try c.decodeIfPresent(Double.self, forKey: .avgStars) {
            avgStars = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .avgStars) {
            avgStars = Double(i)
        } else {
            avgStars = nil
        }
        commentCount = try c.decodeIfPresent(Int.self, forKey: .commentCount)
        if let d = try c.decodeIfPresent(Double.self, forKey: .grade) {
            grade = d
        } else if let i = try c.decodeIfPresent(Int.self, forKey: .grade) {
            grade = Double(i)
        } else {
            grade = nil
        }
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct BoardPostsListResponse: Decodable {
    var posts: [BoardPost]?
}

struct CreateBoardPostRequest: Encodable {
    var contentType: String
    var title: String?
    var body: BoardPostBody?
    var linkUrl: String?
    var attachmentId: String?
}

struct PatchBoardPostRequest: Encodable {
    var title: String?
    var body: BoardPostBody?
    var linkUrl: String?
}

struct BoardLinkPreviewRequest: Encodable {
    var url: String
}

struct BoardVideoEmbed: Hashable, Equatable {
    var provider: String // youtube | vimeo
    var id: String
}

enum BoardComposeValidation: Equatable {
    case ok
    case missingText
    case missingLink
    case missingFile
    case missingAltText
    case missingAudio
}

// swiftlint:enable identifier_name
