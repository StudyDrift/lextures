import Foundation

// Decoder locals match API JSON keys for advanced board payloads.
// swiftlint:disable identifier_name

// MARK: - MOB.8 advanced (templates / export / analytics / governance)

enum BoardTemplateScope: String, Codable, CaseIterable, Hashable {
    case builtin
    case course
    case org
}

struct BoardTemplate: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var scope: String
    var courseId: String?
    var orgId: String?
    var title: String
    var description: String
    var tags: [String]
    var createdBy: String?
    var createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, scope, courseId, orgId, title, description, tags, createdBy, createdAt
    }

    init(
        id: String,
        scope: String = "builtin",
        courseId: String? = nil,
        orgId: String? = nil,
        title: String,
        description: String = "",
        tags: [String] = [],
        createdBy: String? = nil,
        createdAt: String = ""
    ) {
        self.id = id
        self.scope = scope
        self.courseId = courseId
        self.orgId = orgId
        self.title = title
        self.description = description
        self.tags = tags
        self.createdBy = createdBy
        self.createdAt = createdAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        scope = (try? c.decode(String.self, forKey: .scope)) ?? "builtin"
        courseId = try? c.decodeIfPresent(String.self, forKey: .courseId)
        orgId = try? c.decodeIfPresent(String.self, forKey: .orgId)
        title = (try? c.decode(String.self, forKey: .title)) ?? ""
        description = (try? c.decode(String.self, forKey: .description)) ?? ""
        tags = (try? c.decode([String].self, forKey: .tags)) ?? []
        createdBy = try? c.decodeIfPresent(String.self, forKey: .createdBy)
        createdAt = (try? c.decode(String.self, forKey: .createdAt)) ?? ""
    }
}

struct BoardTemplatesListResponse: Decodable {
    var templates: [BoardTemplate]?
}

enum BoardCopyMode: String, CaseIterable, Hashable {
    case structure
    case full
}

struct BoardCopyJob: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var sourceBoardId: String
    var mode: String
    var title: String
    var status: String
    var progress: Double
    var resultBoardId: String?
    var error: String
    var createdAt: String
    var updatedAt: String

    init(
        id: String,
        sourceBoardId: String = "",
        mode: String = "structure",
        title: String = "",
        status: String = "pending",
        progress: Double = 0,
        resultBoardId: String? = nil,
        error: String = "",
        createdAt: String = "",
        updatedAt: String = ""
    ) {
        self.id = id
        self.sourceBoardId = sourceBoardId
        self.mode = mode
        self.title = title
        self.status = status
        self.progress = progress
        self.resultBoardId = resultBoardId
        self.error = error
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

struct BoardCopyJobResponse: Decodable {
    var job: BoardCopyJob
}

enum BoardCreateResult: Equatable {
    case board(Board)
    case job(BoardCopyJob)
}

struct SaveBoardAsTemplateRequest: Encodable {
    var scope: String
    var title: String
    var description: String
    var tags: [String]
    var includePosts: Bool
}

enum BoardExportFormat: String, CaseIterable, Hashable {
    case pdf
    case csv
    case image
}

struct BoardExportJob: Codable, Identifiable, Hashable, Equatable {
    var id: String
    var boardId: String
    var format: String
    var status: String
    var storageKey: String?
    var error: String
    var includeModeration: Bool
    var requestedBy: String?
    var createdAt: String
    var completedAt: String?
    var downloadUrl: String?

    init(
        id: String,
        boardId: String = "",
        format: String = "pdf",
        status: String = "pending",
        storageKey: String? = nil,
        error: String = "",
        includeModeration: Bool = false,
        requestedBy: String? = nil,
        createdAt: String = "",
        completedAt: String? = nil,
        downloadUrl: String? = nil
    ) {
        self.id = id
        self.boardId = boardId
        self.format = format
        self.status = status
        self.storageKey = storageKey
        self.error = error
        self.includeModeration = includeModeration
        self.requestedBy = requestedBy
        self.createdAt = createdAt
        self.completedAt = completedAt
        self.downloadUrl = downloadUrl
    }
}

struct BoardExportJobResponse: Decodable {
    var job: BoardExportJob
}

struct CreateBoardExportRequest: Encodable {
    var format: String
    var includeModeration: Bool
}

struct BoardContributorStat: Codable, Hashable, Equatable {
    var userId: String
    var postCount: Int
    var commentCount: Int
    var reactionCount: Int
    var contributionTotal: Int
}

struct BoardDailyAnalytics: Codable, Hashable, Equatable {
    var boardId: String
    var day: String
    var cardCount: Int
    var contributorCount: Int
    var reactionCount: Int
    var commentCount: Int
}

struct BoardAnalyticsSummary: Codable, Hashable, Equatable {
    var boardId: String
    var cardCount: Int
    var uniqueContributors: Int
    var reactionCount: Int
    var commentCount: Int
    var lastActivityAt: String?
    var contributors: [BoardContributorStat]
    var daily: [BoardDailyAnalytics]

    init(
        boardId: String = "",
        cardCount: Int = 0,
        uniqueContributors: Int = 0,
        reactionCount: Int = 0,
        commentCount: Int = 0,
        lastActivityAt: String? = nil,
        contributors: [BoardContributorStat] = [],
        daily: [BoardDailyAnalytics] = []
    ) {
        self.boardId = boardId
        self.cardCount = cardCount
        self.uniqueContributors = uniqueContributors
        self.reactionCount = reactionCount
        self.commentCount = commentCount
        self.lastActivityAt = lastActivityAt
        self.contributors = contributors
        self.daily = daily
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        boardId = (try? c.decode(String.self, forKey: .boardId)) ?? ""
        cardCount = (try? c.decode(Int.self, forKey: .cardCount)) ?? 0
        uniqueContributors = (try? c.decode(Int.self, forKey: .uniqueContributors)) ?? 0
        reactionCount = (try? c.decode(Int.self, forKey: .reactionCount)) ?? 0
        commentCount = (try? c.decode(Int.self, forKey: .commentCount)) ?? 0
        lastActivityAt = try? c.decodeIfPresent(String.self, forKey: .lastActivityAt)
        contributors = (try? c.decode([BoardContributorStat].self, forKey: .contributors)) ?? []
        daily = (try? c.decode([BoardDailyAnalytics].self, forKey: .daily)) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case boardId, cardCount, uniqueContributors, reactionCount, commentCount
        case lastActivityAt, contributors, daily
    }
}

struct BoardOrgPolicies: Codable, Hashable, Equatable {
    var orgId: String
    var externalSharing: Bool
    var minorModerationFloor: Bool
    var defaultAttribution: String
    var boardCapPerCourse: Int?
    var updatedAt: String?

    init(
        orgId: String = "",
        externalSharing: Bool = true,
        minorModerationFloor: Bool = false,
        defaultAttribution: String = "named",
        boardCapPerCourse: Int? = nil,
        updatedAt: String? = nil
    ) {
        self.orgId = orgId
        self.externalSharing = externalSharing
        self.minorModerationFloor = minorModerationFloor
        self.defaultAttribution = defaultAttribution
        self.boardCapPerCourse = boardCapPerCourse
        self.updatedAt = updatedAt
    }
}

struct BoardContentTypeCount: Codable, Hashable, Equatable {
    var contentType: String
    var count: Int
}

struct BoardAdminOverview: Codable, Hashable, Equatable {
    var boardCount: Int
    var activeBoardCount: Int
    var coursesWithBoards: Int
    var coursesFeatureEnabled: Int
    var storageBytes: Int64
    var topContentTypes: [BoardContentTypeCount]
    var activeWindowDays: Int

    init(
        boardCount: Int = 0,
        activeBoardCount: Int = 0,
        coursesWithBoards: Int = 0,
        coursesFeatureEnabled: Int = 0,
        storageBytes: Int64 = 0,
        topContentTypes: [BoardContentTypeCount] = [],
        activeWindowDays: Int = 30
    ) {
        self.boardCount = boardCount
        self.activeBoardCount = activeBoardCount
        self.coursesWithBoards = coursesWithBoards
        self.coursesFeatureEnabled = coursesFeatureEnabled
        self.storageBytes = storageBytes
        self.topContentTypes = topContentTypes
        self.activeWindowDays = activeWindowDays
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        boardCount = (try? c.decode(Int.self, forKey: .boardCount)) ?? 0
        activeBoardCount = (try? c.decode(Int.self, forKey: .activeBoardCount)) ?? 0
        coursesWithBoards = (try? c.decode(Int.self, forKey: .coursesWithBoards)) ?? 0
        coursesFeatureEnabled = (try? c.decode(Int.self, forKey: .coursesFeatureEnabled)) ?? 0
        if let n = try? c.decode(Int64.self, forKey: .storageBytes) {
            storageBytes = n
        } else if let n = try? c.decode(Int.self, forKey: .storageBytes) {
            storageBytes = Int64(n)
        } else {
            storageBytes = 0
        }
        topContentTypes = (try? c.decode([BoardContentTypeCount].self, forKey: .topContentTypes)) ?? []
        activeWindowDays = (try? c.decode(Int.self, forKey: .activeWindowDays)) ?? 30
    }

    private enum CodingKeys: String, CodingKey {
        case boardCount, activeBoardCount, coursesWithBoards, coursesFeatureEnabled
        case storageBytes, topContentTypes, activeWindowDays
    }
}

struct PatchBoardOrgPoliciesRequest: Encodable {
    var externalSharing: Bool?
    var minorModerationFloor: Bool?
    var defaultAttribution: String?
    var boardCapPerCourse: Int?
    var clearBoardCap: Bool?
}

// swiftlint:enable identifier_name
