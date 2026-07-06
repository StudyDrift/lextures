import XCTest
@testable import Lextures

final class UIModeLogicTests: XCTestCase {
    func testGradeToUIModeMapsKThroughTwo() {
        XCTAssertEqual(UIModeLogic.gradeToUIMode("K"), .k2)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("1"), .k2)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("2"), .k2)
    }

    func testGradeToUIModeMapsThreeThroughFive() {
        XCTAssertEqual(UIModeLogic.gradeToUIMode("3"), .elementary)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("4"), .elementary)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("5"), .elementary)
    }

    func testGradeToUIModeDefaultsToStandard() {
        XCTAssertEqual(UIModeLogic.gradeToUIMode(nil), .standard)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("6"), .standard)
        XCTAssertEqual(UIModeLogic.gradeToUIMode("12"), .standard)
    }

    func testServerOverrideBeatsLocalPreference() {
        let mode = UIModeLogic.effectiveMode(
            featureEnabled: true,
            roleContext: .learning,
            serverOverride: "k2",
            serverEffective: "standard",
            localPreference: .standard
        )
        XCTAssertEqual(mode, .k2)
    }

    func testLocalPreferenceBeatsServerEffective() {
        let mode = UIModeLogic.effectiveMode(
            featureEnabled: true,
            roleContext: .learning,
            serverOverride: nil,
            serverEffective: "standard",
            localPreference: .elementary
        )
        XCTAssertEqual(mode, .elementary)
    }

    func testTeachingContextAlwaysStandard() {
        let mode = UIModeLogic.effectiveMode(
            featureEnabled: true,
            roleContext: .teaching,
            serverOverride: "k2",
            serverEffective: "k2",
            localPreference: .k2
        )
        XCTAssertEqual(mode, .standard)
    }

    func testK2DrawerHasFewerGroupsThanStandard() {
        let platform = MobilePlatformFeatures()
        let standard = MobileDestinations.globalDrawerGroups(context: .learning, platform: platform, uiMode: .standard)
        let k2 = MobileDestinations.globalDrawerGroups(context: .learning, platform: platform, uiMode: .k2)
        XCTAssertGreaterThan(standard.count, k2.count)
        XCTAssertFalse(k2.flatMap(\.items).contains(.notebooks))
    }

    func testK2MoreHubHidesAdvancedDestinations() {
        var platform = MobilePlatformFeatures()
        platform.ffLibrary = true
        platform.ffPeerReview = true
        platform.ffGamification = true
        let k2 = MobileDestinations.moreDestinations(context: .learning, platform: platform, uiMode: .k2)
        XCTAssertTrue(k2.contains(.reading))
        XCTAssertFalse(k2.contains(.peerReviews))
        XCTAssertFalse(k2.contains(.gamification))
    }
}
