import Foundation

/// Course outcomes settings models (M13.5).
struct CourseOutcomeLinkProgress: Codable, Hashable {
    var avgScorePercent: Double?
    var gradedLearners: Int
    var enrolledLearners: Int
}

struct CourseOutcomeLink: Codable, Identifiable, Hashable {
    var id: String
    var subOutcomeId: String?
    var structureItemId: String
    var targetKind: String
    var quizQuestionId: String
    var measurementLevel: String
    var intensityLevel: String
    var itemTitle: String
    var itemKind: String
    var progress: CourseOutcomeLinkProgress
}

struct CourseOutcome: Codable, Identifiable, Hashable {
    var id: String
    var title: String
    var description: String
    var sortOrder: Int
    var rollupAvgScorePercent: Double?
    var links: [CourseOutcomeLink]
}

struct CourseOutcomesListResponse: Codable, Hashable {
    var enrolledLearners: Int
    var outcomes: [CourseOutcome]
}

struct CreateCourseOutcomeBody: Encodable {
    var title: String
    var description: String
}

struct PatchCourseOutcomeBody: Encodable {
    var title: String?
    var description: String?
}

struct AddCourseOutcomeLinkBody: Encodable {
    var structureItemId: String
    var targetKind: String
    var quizQuestionId: String?
    var measurementLevel: String
    var intensityLevel: String
    var subOutcomeId: String?
}

struct CourseOutcomeSubOutcome: Codable, Identifiable, Hashable {
    var id: String
    var outcomeId: String
    var title: String
    var description: String
    var sortOrder: Int
}

struct CreateCourseOutcomeSubOutcomeBody: Encodable {
    var title: String
    var description: String
}