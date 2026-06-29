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
