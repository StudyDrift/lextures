import XCTest
@testable import Lextures

final class HomeschoolSubscribeGateLogicTests: XCTestCase {
    func testHomeschoolHostByKind() {
        XCTAssertTrue(HomeschoolSubscribeGateLogic.isHomeschoolHost(kind: .homeschool, apiBaseURLString: nil))
        XCTAssertFalse(HomeschoolSubscribeGateLogic.isHomeschoolHost(kind: .school, apiBaseURLString: "https://acme.lextures.com"))
    }

    func testHomeschoolHostByURL() {
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isHomeschoolHost(
                kind: .school,
                apiBaseURLString: "https://self.lextures.com"
            )
        )
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isHomeschoolHost(
                kind: nil,
                apiBaseURLString: "https://self.lextures.com/api"
            )
        )
    }

    func testEligibleRequiresAllGates() {
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isEligible(
                isHomeschoolHost: true,
                billingEnabled: true,
                hasActiveSubscription: false
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isHomeschoolHost: true,
                billingEnabled: true,
                hasActiveSubscription: true
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isHomeschoolHost: false,
                billingEnabled: true,
                hasActiveSubscription: false
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isHomeschoolHost: true,
                billingEnabled: false,
                hasActiveSubscription: false
            )
        )
    }

    func testPresentationDelayIsFiveSeconds() {
        XCTAssertEqual(HomeschoolSubscribeGateLogic.presentationDelaySeconds, 5, accuracy: 0.001)
    }

    func testPaywallAfterDelayWhenEligible() {
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.shouldPresentPaywall(
                eligible: true,
                delayElapsed: false,
                dismissedForSession: false
            )
        )
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.shouldPresentPaywall(
                eligible: true,
                delayElapsed: true,
                dismissedForSession: false
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.shouldPresentPaywall(
                eligible: true,
                delayElapsed: true,
                dismissedForSession: true
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.shouldPresentPaywall(
                eligible: false,
                delayElapsed: true,
                dismissedForSession: false
            )
        )
    }
}
