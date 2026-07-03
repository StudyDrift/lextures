import Foundation

/// Gamification display and privacy helpers (M9.3).
enum GamificationLogic {
    static func gamificationEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffGamification
    }

    static func cacheKey() -> String { "gamification:profile" }

    static func leaderboardCacheKey(courseCode: String) -> String {
        "gamification:leaderboard:\(courseCode)"
    }

    static func badgeLabel(_ badgeType: String) -> String {
        switch badgeType {
        case "streak_7":
            return L.text("mobile.gamification.badge.streak7")
        case "streak_30":
            return L.text("mobile.gamification.badge.streak30")
        case "xp_100":
            return L.text("mobile.gamification.badge.xp100")
        case "xp_1000":
            return L.text("mobile.gamification.badge.xp1000")
        case "first_course_complete":
            return L.text("mobile.gamification.badge.firstCourse")
        default:
            let spaced = badgeType.replacingOccurrences(of: "_", with: " ")
            return spaced.prefix(1).uppercased() + spaced.dropFirst()
        }
    }

    static func levelProgressLabel(profile: GamificationProfile) -> String {
        L.format(
            "mobile.gamification.levelProgress",
            profile.level,
            profile.xpToNextLevel,
            Int(profile.levelProgressPct.rounded())
        )
    }

    static func shouldShowLeaderboard(profile: GamificationProfile) -> Bool {
        profile.leaderboardVisible
    }

    static func leaderboardOptOutMessage() -> String {
        L.text("mobile.gamification.leaderboardOptOut")
    }

    static func canUseStreakFreeze(profile: GamificationProfile) -> Bool {
        profile.streakFreezes > 0 && profile.currentStreak > 0
    }

    static func streakRiskMessage(profile: GamificationProfile) -> String? {
        guard profile.streakAtRisk else { return nil }
        if let hours = profile.streakHoursLeft, hours > 0 {
            return L.format("mobile.gamification.streakAtRiskHours", Int(hours.rounded(.up)))
        }
        return L.text("mobile.gamification.streakAtRisk")
    }
}