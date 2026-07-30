import XCTest
@testable import Lextures

final class MarkdownRenderStyleTests: XCTestCase {
    func testCompactSpacingIsTighterThanDefault() {
        XCTAssertEqual(MarkdownRenderStyle.blockSpacing(compact: false), 12)
        XCTAssertEqual(MarkdownRenderStyle.blockSpacing(compact: true), 6)
        XCTAssertLessThan(
            MarkdownRenderStyle.headingPointSize(level: 1, compact: true),
            MarkdownRenderStyle.headingPointSize(level: 1, compact: false)
        )
    }

    func testLockdownSuppressesAffordances() {
        XCTAssertTrue(MarkdownRenderStyle.allowsAffordances(suppressAffordances: false))
        XCTAssertFalse(MarkdownRenderStyle.allowsAffordances(suppressAffordances: true))
    }

    func testInlineSourceStripsLinksWhenSuppressed() {
        let raw = "See [docs](https://example.com) and **bold**"
        let suppressed = MarkdownRenderStyle.sourceForInlineRendering(raw, suppressLinks: true)
        XCTAssertEqual(suppressed, "See docs and **bold**")
        XCTAssertEqual(MarkdownRenderStyle.sourceForInlineRendering(raw, suppressLinks: false), raw)
    }

    func testPlainLabelMergesChoiceText() {
        let label = MarkdownRenderStyle.plainLabel("Use `print()` and **bold**")
        XCTAssertEqual(label, "Use print() and bold")
        XCTAssertFalse(label.contains("`"))
        XCTAssertFalse(label.contains("**"))
    }
}
