import XCTest
@testable import Lextures

final class HomeschoolSubscribeGateLogicTests: XCTestCase {
    func testSelfLearnerHostByKind() {
        XCTAssertTrue(HomeschoolSubscribeGateLogic.isSelfLearnerHost(kind: .homeschool, apiBaseURLString: nil))
        XCTAssertFalse(HomeschoolSubscribeGateLogic.isSelfLearnerHost(kind: .school, apiBaseURLString: "https://acme.lextures.com"))
    }

    func testSelfLearnerHostByURL() {
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isSelfLearnerHost(
                kind: .school,
                apiBaseURLString: "https://self.lextures.com"
            )
        )
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isSelfLearnerHost(
                kind: nil,
                apiBaseURLString: "https://self.lextures.com/api"
            )
        )
    }

    func testEligibleRequiresAllGates() {
        XCTAssertTrue(
            HomeschoolSubscribeGateLogic.isEligible(
                isSelfLearnerHost: true,
                billingEnabled: true,
                hasActiveSubscription: false
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isSelfLearnerHost: true,
                billingEnabled: true,
                hasActiveSubscription: true
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isSelfLearnerHost: false,
                billingEnabled: true,
                hasActiveSubscription: false
            )
        )
        XCTAssertFalse(
            HomeschoolSubscribeGateLogic.isEligible(
                isSelfLearnerHost: true,
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
