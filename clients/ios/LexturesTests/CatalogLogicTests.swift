import XCTest
@testable import Lextures

final class CatalogLogicTests: XCTestCase {
    func testIsPaidAndFree() {
        XCTAssertTrue(CatalogLogic.isPaid(priceCents: 1999))
        XCTAssertFalse(CatalogLogic.isPaid(priceCents: 0))
        XCTAssertTrue(CatalogLogic.isFree(priceCents: 0))
        XCTAssertFalse(CatalogLogic.isFree(priceCents: 500))
    }

    func testIsEnrolledMatchesCourseCodeCaseInsensitively() {
        let courses = [
            CourseSummary(
                id: "1",
                courseCode: "SPAN101",
                title: "Spanish",
                description: ""
            ),
        ]
        XCTAssertTrue(CatalogLogic.isEnrolled(courseCode: "span101", in: courses))
        XCTAssertFalse(CatalogLogic.isEnrolled(courseCode: "FRENCH101", in: courses))
    }

    func testPreviewParagraphsTrimsAndLimits() {
        let text = "Learn greetings.\n\nPractice daily.\n\n\nSpeak with confidence.\nExtra."
        XCTAssertEqual(CatalogLogic.previewParagraphs(from: text, limit: 2), [
            "Learn greetings.",
            "Practice daily.",
        ])
    }

    func testCacheKeyIncludesFilters() {
        let key = CatalogLogic.cacheKey(
            query: "spanish",
            category: "Languages",
            level: .beginner,
            price: .free,
            sort: .relevance
        )
        XCTAssertTrue(key.contains("spanish"))
        XCTAssertTrue(key.contains("beginner"))
        XCTAssertTrue(key.contains("free"))
    }

    func testCatalogWebPath() {
        XCTAssertEqual(CatalogLogic.catalogWebPath(slug: "spanish-a1"), "/explore/spanish-a1")
    }
}