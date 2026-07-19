import XCTest
@testable import Lextures

final class LexturesMotionTests: XCTestCase {
    func testDurationScaleMatchesAN1Spec() {
        XCTAssertEqual(LexturesMotion.instant, 0.100, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.fast, 0.150, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.base, 0.220, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.slow, 0.320, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.deliberate, 0.480, accuracy: 0.0001)
    }

    func testDistanceAndStaggerTokens() {
        XCTAssertEqual(LexturesMotion.enterTranslate, 8)
        XCTAssertEqual(LexturesMotion.enterScaleFrom, 0.97, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.pressScale, 0.97, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.staggerStep, 0.040, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.staggerMaxItems, 8)
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 0), 0)
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 3), 0.120, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 99), 0.280, accuracy: 0.0001)
    }

    func testResolveUsesShortOpacityFriendlyAnimationWhenReduced() {
        let reduced = LexturesMotion.resolve(LexturesMotion.bubble, reduceMotion: true)
        XCTAssertNotNil(reduced)
        let full = LexturesMotion.resolve(LexturesMotion.bubble, reduceMotion: false)
        XCTAssertNotNil(full)
    }

    func testNavigationDurationsHonorReducedMotionAndKillSwitch() {
        XCTAssertEqual(LexturesMotion.navigationDuration(reduceMotion: false), LexturesMotion.base, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.navigationDuration(reduceMotion: true), LexturesMotion.instant, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.navigationDuration(reduceMotion: false, enabled: false), 0, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.phaseDuration(reduceMotion: false), LexturesMotion.deliberate, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.phaseDuration(reduceMotion: true), LexturesMotion.instant, accuracy: 0.0001)
    }

    /// AN.3: stagger delay caps at maxItems; reduced path uses zero delay via caller.
    /// LXLoadReveal hosts content in a VStack (not ZStack) so dashboard cards do not overlay.
    func testStaggerRevealDelayCapsAtMaxItems() {
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 0), 0, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 7), 0.280, accuracy: 0.0001)
        XCTAssertEqual(LexturesMotion.staggerDelay(for: 50), LexturesMotion.staggerDelay(for: 7), accuracy: 0.0001)
        // Total choreography budget: max delay + base enter (8×40ms + 220ms).
        let maxDelay = LexturesMotion.staggerDelay(for: 99)
        XCTAssertLessThanOrEqual(maxDelay + LexturesMotion.base, 0.500 + 0.001)
    }

    func testAccessibilityPreferencesPersistsReducedMotion() {
        let prefs = AccessibilityPreferences.shared
        let previous = prefs.reducedMotionEnabled
        prefs.reducedMotionEnabled = true
        XCTAssertTrue(prefs.reducedMotionEnabled)
        prefs.reducedMotionEnabled = false
        XCTAssertFalse(prefs.reducedMotionEnabled)
        prefs.reducedMotionEnabled = previous
    }

    /// AN.4: concurrent animation budget and kill-switch.
    func testListMotionShouldAnimateRespectsBudgetAndKillSwitch() {
        XCTAssertEqual(LXListMotion.maxConcurrent, 12)
        XCTAssertEqual(LXListMotion.dragLiftScale, 1.03, accuracy: 0.0001)
        XCTAssertTrue(LXListMotion.shouldAnimate(index: 0, reduceMotion: false, enabled: true))
        XCTAssertFalse(LXListMotion.shouldAnimate(index: 99, reduceMotion: false, enabled: true))
        XCTAssertFalse(LXListMotion.shouldAnimate(index: 0, reduceMotion: false, enabled: false))
        XCTAssertTrue(LXListMotion.shouldAnimate(index: 99, reduceMotion: true, enabled: true))
    }

    /// AN.5: sheet drag dismiss threshold and kill-switch animations.
    func testOverlaySheetDismissThresholdAndAnimations() {
        XCTAssertEqual(LXOverlayMotion.sheetDismissThreshold, 0.28, accuracy: 0.0001)
        XCTAssertFalse(LXOverlayMotion.shouldDismissSheetDrag(offset: 100, sheetHeight: 400))
        XCTAssertTrue(LXOverlayMotion.shouldDismissSheetDrag(offset: 120, sheetHeight: 400))
        XCTAssertTrue(LXOverlayMotion.shouldDismissSheetDrag(offset: 10, sheetHeight: 400, velocity: 900))
        XCTAssertNil(LXOverlayMotion.dialogAnimation(reduceMotion: false, enabled: false))
        XCTAssertNotNil(LXOverlayMotion.dialogAnimation(reduceMotion: true, enabled: true))
        XCTAssertNotNil(LXOverlayMotion.sheetAnimation(reduceMotion: false, enabled: true, exiting: true))
    }
}
