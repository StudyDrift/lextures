import Foundation

// MARK: - Gamification (M9.3)

struct GamificationBadge: Codable, Identifiable, Hashable {
    var badgeType: String
    var awardedAt: String

    var id: String { "\(badgeType)-\(awardedAt)" }
}

struct GamificationProfile: Codable {
    var xpTotal: Int
    var level: Int
    var xpToNextLevel: Int
    var levelProgressPct: Double
    var currentStreak: Int
    var longestStreak: Int
    var streakFreezes: Int
    var streakAtRisk: Bool
    var streakHoursLeft: Double?
    var streakEnded: Bool?
    var leaderboardVisible: Bool
    var badges: [GamificationBadge]?
    var recentBadges: [GamificationBadge]?
}

struct LeaderboardEntry: Codable, Identifiable, Hashable {
    var rank: Int
    var userId: String
    var displayName: String
    var xpEarned: Int
    var isCurrentUser: Bool?

    var id: String { "\(rank)-\(userId)" }
}

struct CourseLeaderboardResponse: Codable {
    var topEntries: [LeaderboardEntry]?
    var currentUser: LeaderboardEntry?
}

struct GamificationBadgesListResponse: Codable {
    var badges: [GamificationBadge]?
}