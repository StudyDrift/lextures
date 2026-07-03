import XCTest
@testable import Lextures

final class CourseReviewLogicTests: XCTestCase {
    func testShouldShowComposer() {
        let eligible = ReviewEligibility(eligible: true, progressPercent: 50, hasReview: false, canEdit: true)
        XCTAssertTrue(CourseReviewLogic.shouldShowComposer(eligible))

        let ineligible = ReviewEligibility(eligible: false, progressPercent: 5, hasReview: false, canEdit: false)
        XCTAssertFalse(CourseReviewLogic.shouldShowComposer(ineligible))
    }

    func testValidateRating() {
        XCTAssertNil(CourseReviewLogic.validateRating(5))
        XCTAssertNotNil(CourseReviewLogic.validateRating(0))
    }

    func testValidateReviewText() {
        XCTAssertNil(CourseReviewLogic.validateReviewText("Great course"))
        let long = String(repeating: "a", count: CourseReviewLogic.maxReviewTextLength + 1)
        XCTAssertNotNil(CourseReviewLogic.validateReviewText(long))
    }
}