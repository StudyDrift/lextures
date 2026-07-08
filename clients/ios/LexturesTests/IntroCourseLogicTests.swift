import XCTest
@testable import Lextures

final class IntroCourseLogicTests: XCTestCase {
    func testIntroCourseEnabledRespectsPlatformFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(IntroCourseLogic.introCourseEnabled(features))
        features.introCourseEnabled = true
        XCTAssertTrue(IntroCourseLogic.introCourseEnabled(features))
    }

    func testCardStateMachine() {
        XCTAssertEqual(IntroCourseLogic.cardState(progress: nil, loading: true, error: false), .loading)
        XCTAssertEqual(IntroCourseLogic.cardState(progress: nil, loading: false, error: true), .error)
        let hidden = IntroCourseProgress(
            enrolled: false,
            modulesComplete: 0,
            modulesTotal: 7,
            percent: 0
        )
        XCTAssertEqual(IntroCourseLogic.cardState(progress: hidden, loading: false, error: false), .hidden)
        let notStarted = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 0,
            modulesTotal: 7,
            percent: 0
        )
        XCTAssertEqual(IntroCourseLogic.cardState(progress: notStarted, loading: false, error: false), .notStarted)
        let inProgress = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 2,
            modulesTotal: 7,
            percent: 28
        )
        XCTAssertEqual(IntroCourseLogic.cardState(progress: inProgress, loading: false, error: false), .inProgress)
        let completed = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 7,
            modulesTotal: 7,
            percent: 100,
            completedAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertEqual(IntroCourseLogic.cardState(progress: completed, loading: false, error: false), .completed)
    }

    func testShouldShowCelebration() {
        let incomplete = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 1,
            modulesTotal: 7,
            percent: 10
        )
        XCTAssertFalse(IntroCourseLogic.shouldShowCelebration(incomplete))
        let doneSeen = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 7,
            modulesTotal: 7,
            percent: 100,
            completedAt: "2026-01-01T00:00:00Z",
            celebrationSeen: true
        )
        XCTAssertFalse(IntroCourseLogic.shouldShowCelebration(doneSeen))
        let doneFresh = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 7,
            modulesTotal: 7,
            percent: 100,
            completedAt: "2026-01-01T00:00:00Z",
            celebrationSeen: false
        )
        XCTAssertTrue(IntroCourseLogic.shouldShowCelebration(doneFresh))
    }

    func testCtaRoutePrefersNextItem() {
        let progress = IntroCourseProgress(
            enrolled: true,
            modulesComplete: 1,
            modulesTotal: 7,
            percent: 10,
            nextItem: IntroCourseNextItem(
                slug: "m1.welcome.dashboard",
                title: "Dashboard tour",
                route: "/courses/C-WLCOME/modules/content/abc"
            )
        )
        XCTAssertEqual(
            IntroCourseLogic.ctaRoute(for: progress),
            "/courses/C-WLCOME/modules/content/abc"
        )
    }

    func testDeepLinkResolvesIntroModuleItem() {
        let destination = DeepLinkRouter.resolve(
            "/courses/C-WLCOME/modules/content/a0000000-0000-4000-8000-000000000099"
        )
        guard case let .course(code, section, itemId) = destination else {
            return XCTFail("expected course deep link")
        }
        XCTAssertEqual(code, "C-WLCOME")
        XCTAssertEqual(section, .modules)
        XCTAssertEqual(itemId, "a0000000-0000-4000-8000-000000000099")
    }

    func testDeepLinkResolvesNotificationSettings() {
        guard case let .settings(section) = DeepLinkRouter.resolve("/settings/notifications") else {
            return XCTFail("expected settings deep link")
        }
        XCTAssertEqual(section, .notifications)
    }
}