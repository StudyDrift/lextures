import XCTest
@testable import Lextures

final class GamificationLogicTests: XCTestCase {
    func testGamificationEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(GamificationLogic.gamificationEnabled(features))
        features.ffGamification = true
        XCTAssertTrue(GamificationLogic.gamificationEnabled(features))
    }

    func testBadgeLabel() {
        XCTAssertEqual(GamificationLogic.badgeLabel("streak_7"), L.text("mobile.gamification.badge.streak7"))
        XCTAssertEqual(GamificationLogic.badgeLabel("custom_badge"), "Custom badge")
    }

    func testLeaderboardVisibility() {
        var profile = GamificationProfile(
            xpTotal: 0,
            level: 0,
            xpToNextLevel: 10,
            levelProgressPct: 0,
            currentStreak: 0,
            longestStreak: 0,
            streakFreezes: 0,
            streakAtRisk: false,
            leaderboardVisible: false,
            badges: nil,
            recentBadges: nil
        )
        XCTAssertFalse(GamificationLogic.shouldShowLeaderboard(profile: profile))
        profile.leaderboardVisible = true
        XCTAssertTrue(GamificationLogic.shouldShowLeaderboard(profile: profile))
    }
}