import Foundation

// MARK: - At-risk alerts

struct AtRiskAlert: Codable, Identifiable, Hashable {
    var id: String
    var enrollmentId: String
    var userId: String
    var displayName: String
    var score: Float
    var status: String
    var topFactor: String
    var topFactorLabel: String
    var snoozeUntil: String?
    var notes: String?
    var triggeredDate: String
    var missingPct: Float?
    var quizAvg: Float?
    var daysInactive: Int?
}

struct AtRiskListResponse: Codable {
    var alerts: [AtRiskAlert]
    var resolved: [AtRiskAlert]?
}

// MARK: - Instructor insights ("what's working")

struct InstructorSignalItem: Codable, Identifiable, Hashable {
    var itemId: String
    var title: String
    var kind: String
    var completionRate: Double
    var avgScore: Double?
    var engagement: Double
    var difficulty: Double?
    var compositeScore: Double
    var narrative: String

    var id: String { itemId }
}

struct InstructorInsightsResponse: Codable {
    var courseId: String
    var weekOf: String
    var generatedAt: String
    var workingWell: [InstructorSignalItem]
    var needsAttention: [InstructorSignalItem]
    var scatter: [InstructorScatterPoint]?
}

struct InstructorScatterPoint: Codable, Identifiable, Hashable {
    var itemId: String
    var title: String
    var kind: String
    var difficulty: Double
    var engagement: Double
    var flag: String?

    var id: String { itemId }
}

// MARK: - Student progress (instructor read-only)

struct StudentProgressSummary: Codable, Hashable {
    var enrollmentId: String
    var courseId: String
    var studentUserId: String
    var studentDisplayName: String
    var studentAvatarUrl: String?
    var assignmentsSubmittedPct: Double
    var modulesViewedPct: Double
    var avgQuizScore: Double?
    var avgGradePercent: Double?
    var lastActiveAt: String?
    var missingCount: Int
    var dataAsOf: String
    var staleMinutes: Int
    var canManageNotes: Bool
}

struct StudentProgressMissingItem: Codable, Identifiable, Hashable {
    var itemId: String
    var title: String
    var kind: String
    var dueAt: String?
    var daysOverdue: Int
    var gradeStatus: String

    var id: String { itemId }
}

struct StudentProgressAssignmentRow: Codable, Identifiable, Hashable {
    var itemId: String
    var title: String
    var dueAt: String?
    var submittedAt: String?
    var grade: String
    var status: String

    var id: String { itemId }
}

struct StudentProgressQuizRow: Codable, Identifiable, Hashable {
    var attemptId: String
    var itemId: String
    var title: String
    var submittedAt: String
    var scorePercent: Double?

    var id: String { attemptId }
}

struct StudentProgressResponse: Codable {
    var summary: StudentProgressSummary
    var missing: [StudentProgressMissingItem]
    var assignments: [StudentProgressAssignmentRow]
    var quizzes: [StudentProgressQuizRow]
}

struct StudentProgressActivityEvent: Codable, Identifiable, Hashable {
    var occurredAt: String
    var kind: String
    var label: String
    var detail: String?

    var id: String { "\(occurredAt)-\(kind)-\(label)" }
}

struct StudentProgressActivityResponse: Codable {
    var events: [StudentProgressActivityEvent]
    var nextCursor: String?
}

// MARK: - Navigation routes

enum InstructorInsightsRoute: Hashable {
    case atRiskList
    case whatsWorking
    case studentProgress(enrollmentId: String, displayName: String)
}
