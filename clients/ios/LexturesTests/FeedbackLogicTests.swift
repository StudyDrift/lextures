import XCTest
@testable import Lextures

final class FeedbackLogicTests: XCTestCase {
    func testFeedbackEnabledDefaultsOn() {
        XCTAssertTrue(FeedbackLogic.feedbackEnabled(MobilePlatformFeatures()))
        var off = MobilePlatformFeatures()
        off.ffFeedback = false
        XCTAssertFalse(FeedbackLogic.feedbackEnabled(off))
    }

    func testMessageValid() {
        XCTAssertFalse(FeedbackLogic.messageValid(""))
        XCTAssertFalse(FeedbackLogic.messageValid("   "))
        XCTAssertTrue(FeedbackLogic.messageValid("note"))
    }

    func testBuildSubmitRequest() {
        let request = FeedbackLogic.buildSubmitRequest(
            message: "  Hello world  ",
            category: "bug",
            route: "profile",
            locale: "en",
            viewport: "390x844"
        )
        XCTAssertEqual(request.message, "Hello world")
        XCTAssertEqual(request.source, "ios")
        XCTAssertFalse(request.appVersion.isEmpty)
        XCTAssertEqual(request.context.route, "profile")
        XCTAssertEqual(request.context.locale, "en")
        XCTAssertEqual(request.context.viewport, "390x844")
        XCTAssertEqual(request.category, "bug")
    }

    func testBuildSubmitRequestOmitsEmptyCategory() {
        let request = FeedbackLogic.buildSubmitRequest(
            message: "Hi",
            category: "",
            route: "profile",
            locale: nil,
            viewport: nil
        )
        XCTAssertEqual(request.category, "")
    }

    func testMapSubmitError() {
        XCTAssertEqual(
            FeedbackLogic.mapSubmitError(APIError.httpStatus(429, message: nil), isOnline: true),
            .rateLimited
        )
        XCTAssertEqual(
            FeedbackLogic.mapSubmitError(APIError.transport(URLError(.notConnectedToInternet)), isOnline: true),
            .offline
        )
        XCTAssertEqual(
            FeedbackLogic.mapSubmitError(APIError.httpStatus(500, message: nil), isOnline: true),
            .error
        )
        XCTAssertEqual(
            FeedbackLogic.mapSubmitError(APIError.httpStatus(500, message: nil), isOnline: false),
            .offline
        )
    }

    func testCategoryLabelKeys() {
        XCTAssertEqual(FeedbackLogic.categoryLabelKey(""), "mobile.feedback.category.none")
        XCTAssertEqual(FeedbackLogic.categoryLabelKey("bug"), "mobile.feedback.category.bug")
    }
}
