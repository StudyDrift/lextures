import CoreGraphics
import Foundation

/// Spacing and typography tokens for the shared markdown renderer (CT.M2 FR-11 / FR-13).
enum MarkdownRenderStyle {
    static func blockSpacing(compact: Bool) -> CGFloat { compact ? 6 : 12 }

    static func headingPointSize(level: Int, compact: Bool) -> CGFloat {
        if compact {
            switch level {
            case 1: return 18
            case 2: return 16
            default: return 15
            }
        }
        switch level {
        case 1: return 24
        case 2: return 19
        default: return 16
        }
    }

    static func headingTopPadding(level: Int, compact: Bool) -> CGFloat {
        if compact { return level == 1 ? 2 : 0 }
        return level == 1 ? 6 : 2
    }

    /// FR-13: copy, table expand, and link navigation are suppressed during lockdown quizzes.
    static func allowsAffordances(suppressAffordances: Bool) -> Bool { !suppressAffordances }

    /// Strip link destinations so AttributedString does not create tappable runs (choices / lockdown).
    static func sourceForInlineRendering(_ text: String, suppressLinks: Bool) -> String {
        guard suppressLinks else { return text }
        return text.replacingOccurrences(
            of: #"\[([^\]]+)\]\([^)]*\)"#,
            with: "$1",
            options: .regularExpression
        )
    }

    static func inlineAttributedString(_ text: String, suppressLinks: Bool = false) -> AttributedString {
        let source = sourceForInlineRendering(text, suppressLinks: suppressLinks)
        return (try? AttributedString(
            markdown: source,
            options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        )) ?? AttributedString(source)
    }

    /// Plain label for a11y / merged choice semantics (formatting must not fragment the name).
    static func plainLabel(_ text: String) -> String {
        AccessibilitySupport.plainText(fromMarkdown: text)
    }
}
