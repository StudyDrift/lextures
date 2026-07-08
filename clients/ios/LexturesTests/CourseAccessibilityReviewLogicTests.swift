import XCTest
@testable import Lextures

final class CourseAccessibilityReviewLogicTests: XCTestCase {
    func testCoveragePercentEmptyTotalIs100() {
        XCTAssertEqual(CourseAccessibilityReviewLogic.coveragePercent(withAlt: 0, total: 0), 100)
    }

    func testCoveragePercentRoundsCorrectly() {
        XCTAssertEqual(CourseAccessibilityReviewLogic.coveragePercent(withAlt: 1, total: 3), 33)
        XCTAssertEqual(CourseAccessibilityReviewLogic.coveragePercent(withAlt: 2, total: 2), 100)
    }

    func testScanMarkdownImagesFindsMissingAlt() {
        let markdown = "# Title\n![](/img/a.png)\n![ok](/img/b.png \"lex-decorative\")"
        let images = CourseAccessibilityReviewLogic.scanMarkdownImages(markdown)
        XCTAssertEqual(images.count, 2)
        XCTAssertFalse(images[0].hasValidAlt)
        XCTAssertTrue(images[1].decorative)
        XCTAssertTrue(images[1].hasValidAlt)
        XCTAssertEqual(CourseAccessibilityReviewLogic.missingImages(markdown).count, 1)
    }

    func testApplyAltTextUpdateAddsDescription() {
        let markdown = "![](/files/chart.png)"
        let updated = CourseAccessibilityReviewLogic.applyAltTextUpdate(
            in: markdown,
            imageIndex: 0,
            alt: "Bar chart of enrollment",
            decorative: false
        )
        XCTAssertEqual(updated, "![Bar chart of enrollment](/files/chart.png)")
    }

    func testApplyAltTextUpdateMarksDecorative() {
        let markdown = "![](/files/icon.png)"
        let updated = CourseAccessibilityReviewLogic.applyAltTextUpdate(
            in: markdown,
            imageIndex: 0,
            alt: "",
            decorative: true
        )
        XCTAssertEqual(updated, "![](/files/icon.png \"lex-decorative\")")
    }

    func testSupportsInlineEditOnlyForMarkdownItems() {
        XCTAssertTrue(CourseAccessibilityReviewLogic.supportsInlineEdit(kind: "content_page"))
        XCTAssertTrue(CourseAccessibilityReviewLogic.supportsInlineEdit(kind: "assignment"))
        XCTAssertFalse(CourseAccessibilityReviewLogic.supportsInlineEdit(kind: "quiz"))
    }

    func testPaginatedUncoveredItems() {
        let items = (0 ..< 25).map {
            UncoveredAccessibilityItem(
                itemId: "item-\($0)",
                title: "Item \($0)",
                kind: "content_page",
                withAlt: 0,
                total: 1,
                missing: 1
            )
        }
        XCTAssertEqual(CourseAccessibilityReviewLogic.paginatedUncoveredItems(items, page: 0).count, 20)
        XCTAssertTrue(CourseAccessibilityReviewLogic.hasMorePages(items: items, page: 0))
        XCTAssertFalse(CourseAccessibilityReviewLogic.hasMorePages(items: items, page: 1))
    }
}
