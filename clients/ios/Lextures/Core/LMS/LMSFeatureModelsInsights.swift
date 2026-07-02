import Foundation

// MARK: - Study insights & self-reflection (M8.3)

struct StudyTimeAllocationRow: Codable, Identifiable, Hashable {
    var moduleId: String
    var moduleTitle: String
    var minutes: Double

    var id: String { moduleId }
}

struct StudyStats: Codable {
    var optedIn: Bool
    var loginStreakDays: Int
    var timeOnTaskSecondsThisWeek: Int
    var weeklyGoalHours: Float?
    var goalProgressHours: Float
    var goalRemainingHours: Float?
    var studyEfficiency: Double?
    var lowStudyEfficiency: Bool
    var timeAllocation: [StudyTimeAllocationRow]
    var weekStart: String
    var weekEnd: String
}

struct StudyGoal: Codable {
    var weeklyHours: Float
    var optedIn: Bool
}

struct PutStudyGoalBody: Encodable {
    var weeklyHours: Float?
    var optedIn: Bool?
}

struct ReflectionJournalEntry: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String?
    var entryText: String
    var createdAt: String
}

struct ReflectionJournalListResponse: Decodable {
    var entries: [ReflectionJournalEntry]?
}

struct PostReflectionJournalBody: Encodable {
    var entryText: String
    var courseId: String?
}

struct PostReflectionJournalResponse: Decodable {
    var id: String
}

struct CoachingTip: Codable, Identifiable, Hashable {
    var id: String
    var tipText: String
    var weekOf: String
    var rating: Int?
    var createdAt: String
}

struct CoachingTipsResponse: Codable {
    var latest: CoachingTip?
    var history: [CoachingTip]?
}

struct RateCoachingTipBody: Encodable {
    var rating: Int
}

struct ReminderConfig: Codable {
    var dailyGoalMinutes: Int
    var reminderTime: String
    var reminderChannels: [String]
    var weeklySummary: Bool
    var enabled: Bool
    var pausedUntil: String?
    var minutesStudiedToday: Int
    var goalMetToday: Bool
    var streakAtRiskBanner: Bool
}

struct CourseProgressSummary: Identifiable, Hashable {
    var courseCode: String
    var title: String
    var percentComplete: Int

    var id: String { courseCode }
}

struct InsightsRoute: Hashable {}