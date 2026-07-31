import XCTest
@testable import Lextures

final class AccessibilitySupportTests: XCTestCase {
    func testChunkSentencesSplitsOnPunctuation() {
        let sentences = AccessibilitySupport.chunkSentences("Hello world. How are you? Fine!")
        XCTAssertEqual(sentences, ["Hello world.", "How are you?", "Fine!"])
    }

    func testPlainTextFromMarkdownStripsFormatting() {
        let plain = AccessibilitySupport.plainText(fromMarkdown: "# Title\n\n**Bold** text with [link](https://example.com).")
        XCTAssertEqual(plain, "Title Bold text with link.")
    }

    func testPlainTextFromMarkdownLinearisesTablesWithoutPipes() {
        let md = """
        | Week | Topic |
        | --- | --- |
        | 1 | Intro |
        | 2 | Labs |
        """
        let plain = AccessibilitySupport.plainText(fromMarkdown: md)
        XCTAssertFalse(plain.contains("|"))
        XCTAssertTrue(plain.contains("Week"))
        XCTAssertTrue(plain.contains("Intro"))
    }

    func testPlainTextFromMarkdownSpeaksCodeWithoutFences() {
        let plain = AccessibilitySupport.plainText(fromMarkdown: "```python\nprint(1)\n```")
        XCTAssertFalse(plain.contains("```"))
        XCTAssertTrue(plain.contains("print(1)"))
    }

    func testContrastRatioMeetsWCAGAAForBrandText() {
        XCTAssertTrue(LexturesTheme.primaryTextContrastMeetsAA)
        let ratio = AccessibilitySupport.contrastRatio(
            foreground: ColorComponents(hex: 0x1F2D2A),
            background: ColorComponents(hex: 0xFAF5EA)
        )
        XCTAssertTrue(AccessibilitySupport.meetsWCAGAA(ratio: ratio))
    }

    func testMeetsWCAGAARequiresHigherRatioForBodyText() {
        XCTAssertTrue(AccessibilitySupport.meetsWCAGAA(ratio: 4.5))
        XCTAssertFalse(AccessibilitySupport.meetsWCAGAA(ratio: 4.0))
        XCTAssertTrue(AccessibilitySupport.meetsWCAGAA(ratio: 3.0, isLargeText: true))
    }
}
