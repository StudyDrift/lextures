import SwiftUI

/// Inline-only markdown label for quiz choices and compact rows (CT.M2 FR-3).
/// Renders bold/italic/code/links as styled text inside a single accessibility element.
struct InlineMarkdownText: View {
    let markdown: String
    var suppressLinks = false
    var font: Font = .subheadline

    var body: some View {
        Text(MarkdownRenderStyle.inlineAttributedString(markdown, suppressLinks: suppressLinks))
            .font(font)
            .multilineTextAlignment(.leading)
            .accessibilityLabel(MarkdownRenderStyle.plainLabel(markdown))
    }
}
