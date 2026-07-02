import Foundation

// MARK: - Learning paths (M8.2)

struct CatalogPathSummary: Codable, Identifiable, Hashable {
    var id: String
    var title: String
    var description: String
    var slug: String
    var bundlePriceCents: Int?
    var courseCount: Int
    var totalDurationMinutes: Int
    var individualTotalCents: Int
    var skillTags: [String]
}

struct CatalogPathsListResponse: Decodable {
    var paths: [CatalogPathSummary]?
}

struct PathCourseProgress: Codable, Identifiable, Hashable {
    var courseId: String
    var position: Int
    var courseCode: String
    var title: String
    var description: String?
    var listPriceCents: Int?
    var durationMinutes: Int?
    var skillTags: [String]?
    var completed: Bool?
    var recommended: Bool?

    var id: String { courseId }

    var isCompleted: Bool { completed == true }
    var isRecommended: Bool { recommended == true }
}

struct PathProgress: Codable, Identifiable, Hashable {
    var pathId: String
    var pathTitle: String
    var slug: String?
    var totalCourses: Int
    var completedCourses: Int
    var percent: Int
    var progressLabel: String
    var completedAt: String?
    var justCompleted: Bool
    var courses: [PathCourseProgress]

    var id: String { pathId }
}

struct MyPathsListResponse: Decodable {
    var paths: [PathProgress]?
}

struct LearningPathDetailPath: Decodable {
    var id: String
    var title: String
    var description: String
    var slug: String?
    var bundlePriceCents: Int?
    var isPublic: Bool
}

struct LearningPathDetail: Decodable {
    var path: LearningPathDetailPath
    var courses: [PathCourseProgress]
    var totalDurationMinutes: Int
    var individualTotalCents: Int
    var skillTags: [String]
    var slug: String
}

struct PathEnrollResponse: Decodable {
    var enrollmentId: String
    var progress: PathProgress?
}

struct RecommendationEventBody: Encodable {
    var courseId: String
    var itemId: String?
    var surface: String
    var eventType: String
    var rank: Int?
}