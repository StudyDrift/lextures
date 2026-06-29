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
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var allowMultiline: Bool

    func body(content: Content) -> some View {
        content
            .font(LexturesType.body(dyslexia: preferences.dyslexiaDisplayEnabled))
            .tracking(preferences.dyslexiaDisplayEnabled ? LexturesType.dyslexiaTracking : 0)
            .lineSpacing(preferences.dyslexiaDisplayEnabled ? LexturesType.dyslexiaLineSpacing : 2)
            .lineLimit(allowMultiline ? nil : dynamicTypeSize.isAccessibilitySize ? 6 : 3)
            .minimumScaleFactor(allowMultiline ? 1 : 0.85)
            .fixedSize(horizontal: false, vertical: allowMultiline)
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
