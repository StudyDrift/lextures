import SwiftUI

/// Scaling-aware typography and dyslexia-friendly presets.
enum LexturesType {
    static let dyslexiaTracking: CGFloat = 0.6
    static let dyslexiaLineSpacing: CGFloat = 6

    static func body(_ style: Font.TextStyle = .body, dyslexia: Bool = false) -> Font {
        dyslexia ? .system(style, design: .rounded) : .system(style)
    }

    static func display(_ style: Font.TextStyle = .title2, weight: Font.Weight = .semibold, dyslexia: Bool = false) -> Font {
        dyslexia
            ? .system(style, design: .rounded, weight: weight)
            : .system(style, design: .serif, weight: weight)
    }

    static func caption(dyslexia: Bool = false) -> Font {
        dyslexia ? .system(.caption, design: .rounded) : .caption
    }
}

struct LexturesReadableText: ViewModifier {
    @Environment(\.accessibilityPreferences) private var preferences
    @Environment(\.readingPreferencesStore) private var readingStore
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var allowMultiline: Bool

    func body(content: Content) -> some View {
        let dyslexia = preferences.dyslexiaDisplayEnabled || readingStore.usesDyslexiaFont
        let letterSpacing = ReaderTypography.letterSpacing(readingStore.prefs.letterSpacing, dyslexia: dyslexia)
        let wordSpacing = ReaderTypography.wordSpacing(readingStore.prefs.wordSpacing)
        let lineSpacing = ReaderTypography.lineSpacing(readingStore.prefs.lineHeight, dyslexia: dyslexia)
        content
            .font(ReaderTypography.font(face: readingStore.prefs.fontFace, dyslexia: dyslexia))
            .tracking(letterSpacing)
            .kerning(wordSpacing)
            .lineSpacing(lineSpacing)
            .lineLimit(allowMultiline ? nil : dynamicTypeSize.isAccessibilitySize ? 6 : 3)
            .minimumScaleFactor(allowMultiline ? 1 : 0.85)
            .fixedSize(horizontal: false, vertical: allowMultiline)
    }
}

enum ReaderTypography {
    static func font(face: String, dyslexia: Bool) -> Font {
        switch face {
        case "open-dyslexic", "atkinson":
            return .system(.body, design: .rounded)
        case "system":
            return .body
        default:
            return LexturesType.body(dyslexia: dyslexia)
        }
    }

    static func letterSpacing(_ value: String, dyslexia: Bool) -> CGFloat {
        if dyslexia { return LexturesType.dyslexiaTracking }
        switch value {
        case "wide": return 0.8
        case "wider": return 1.6
        default: return 0
        }
    }

    static func wordSpacing(_ value: String) -> CGFloat {
        switch value {
        case "wide": return 1.0
        case "wider": return 2.0
        default: return 0
        }
    }

    static func lineSpacing(_ value: String, dyslexia: Bool) -> CGFloat {
        if dyslexia { return LexturesType.dyslexiaLineSpacing }
        switch value {
        case "tall": return 6
        case "taller": return 10
        default: return 2
        }
    }
}

struct MinimumTapTargetModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .frame(minWidth: AccessibilitySupport.minimumTapTarget, minHeight: AccessibilitySupport.minimumTapTarget)
            .contentShape(Rectangle())
    }
}

extension View {
    func lexturesReadableText(allowMultiline: Bool = true) -> some View {
        modifier(LexturesReadableText(allowMultiline: allowMultiline))
    }

    func minimumTapTarget() -> some View {
        modifier(MinimumTapTargetModifier())
    }

    /// Applies animation only when the user has not enabled Reduce Motion.
    func lexturesAnimation<V: Equatable>(_ animation: Animation, value: V) -> some View {
        modifier(ReducedMotionAnimationModifier(animation: animation, value: value))
    }
}

private struct ReducedMotionAnimationModifier<V: Equatable>: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let animation: Animation
    let value: V

    func body(content: Content) -> some View {
        content.animation(reduceMotion ? nil : animation, value: value)
    }
}
