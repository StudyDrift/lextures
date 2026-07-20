import SwiftUI

/// AN.1 — Lextures motion tokens & View helpers.
///
/// Shared vocabulary: durations, the signature bubble spring, enter distances,
/// stagger, and a unified reduced-motion signal (`lxReduceMotion`).
enum LexturesMotion {
    // MARK: - Durations (seconds)

    static let instant: TimeInterval = 0.100
    static let fast: TimeInterval = 0.150
    static let base: TimeInterval = 0.220
    static let slow: TimeInterval = 0.320
    static let deliberate: TimeInterval = 0.480

    // MARK: - Springs & curves

    /// Signature bubble spring — response 0.5s, damping 0.72 (matches web/Android).
    static let bubble: Animation = .spring(response: 0.5, dampingFraction: 0.72)

    static let standard: Animation = .timingCurve(0.2, 0, 0, 1, duration: base)
    static let exit: Animation = .timingCurve(0.3, 0, 1, 1, duration: fast)
    static let emphasized: Animation = .timingCurve(0.2, 0, 0, 1, duration: base)

    // MARK: - Navigation (AN.2)

    /// Splash → first screen handoff (capped at deliberate; FR-2).
    static let phaseTransition: Animation = .timingCurve(0.2, 0, 0, 1, duration: deliberate)
    /// Root pane / push-style section change.
    static let navigation: Animation = .timingCurve(0.2, 0, 0, 1, duration: base)
    /// Tab indicator / lateral section switch (crossfade-friendly).
    static let tabSwitch: Animation = .timingCurve(0.2, 0, 0, 1, duration: base)

    // MARK: - Distances / scale

    static let enterTranslate: CGFloat = 8
    static let enterScaleFrom: CGFloat = 0.97
    static let pressScale: CGFloat = 0.97

    // MARK: - Stagger

    static let staggerStep: TimeInterval = 0.040
    static let staggerMaxItems: Int = 8

    static func staggerDelay(for index: Int) -> TimeInterval {
        let clampedIndex = min(max(0, index), staggerMaxItems - 1)
        return TimeInterval(clampedIndex) * staggerStep
    }

    /// Resolve an animation under reduced motion → opacity-friendly short ease (FR-7).
    static func resolve(_ animation: Animation, reduceMotion: Bool) -> Animation? {
        if reduceMotion {
            return .easeOut(duration: instant)
        }
        return animation
    }

    /// Navigation duration in seconds (for Task.sleep completion), reduced → instant.
    static func navigationDuration(reduceMotion: Bool, enabled: Bool = true) -> TimeInterval {
        if !enabled { return 0 }
        return reduceMotion ? instant : base
    }

    /// Splash handoff duration (capped at deliberate).
    static func phaseDuration(reduceMotion: Bool, enabled: Bool = true) -> TimeInterval {
        if !enabled { return 0 }
        return reduceMotion ? instant : deliberate
    }
}

// MARK: - Environment: unified reduce-motion

private struct LXReduceMotionKey: EnvironmentKey {
    static let defaultValue = false
}

extension EnvironmentValues {
    /// OS `accessibilityReduceMotion` OR in-app reduce-motion preference (FR-6).
    var lxReduceMotion: Bool {
        get { self[LXReduceMotionKey.self] }
        set { self[LXReduceMotionKey.self] = newValue }
    }
}

/// Injects `lxReduceMotion` from accessibility + app setting. Apply near the app root.
struct LXReduceMotionProvider: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var accessibilityReduceMotion
    @Environment(\.accessibilityPreferences) private var accessibilityPreferences

    func body(content: Content) -> some View {
        content.environment(
            \.lxReduceMotion,
            accessibilityReduceMotion || accessibilityPreferences.reducedMotionEnabled
        )
    }
}

extension View {
    func lxReduceMotionEnvironment() -> some View {
        modifier(LXReduceMotionProvider())
    }
}

// MARK: - View helpers

private struct LXBubbleInModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let active: Bool

    func body(content: Content) -> some View {
        content
            .opacity(active ? 1 : 0)
            .scaleEffect(reduceMotion ? 1 : (active ? 1 : LexturesMotion.enterScaleFrom))
            .offset(y: reduceMotion ? 0 : (active ? 0 : LexturesMotion.enterTranslate))
            .animation(
                LexturesMotion.resolve(LexturesMotion.bubble, reduceMotion: reduceMotion),
                value: active
            )
    }
}

private struct LXEnterModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let active: Bool

    func body(content: Content) -> some View {
        content
            .opacity(active ? 1 : 0)
            .offset(y: reduceMotion ? 0 : (active ? 0 : LexturesMotion.enterTranslate))
            .animation(
                LexturesMotion.resolve(LexturesMotion.standard, reduceMotion: reduceMotion),
                value: active
            )
    }
}

extension View {
    /// Bubble-spring enter. Reduced motion → opacity only ≤100ms (AC-2).
    func lxBubbleIn(_ active: Bool = true) -> some View {
        modifier(LXBubbleInModifier(active: active))
    }

    /// Standard enter (emphasized decelerate). Reduced motion → opacity only.
    func lxEnter(_ active: Bool = true) -> some View {
        modifier(LXEnterModifier(active: active))
    }

    /// AN.3 — staggered bubble reveal for peer items (cards/rows/stats).
    /// Runs once when `appeared` becomes true; subsequent toggles do not re-animate.
    func lxStaggeredReveal(index: Int, appeared: Bool, enabled: Bool = true) -> some View {
        modifier(LXStaggeredRevealModifier(index: index, appeared: appeared, enabled: enabled))
    }
}

// MARK: - AN.3 Load choreography

private struct LXStaggeredRevealModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let index: Int
    let appeared: Bool
    let enabled: Bool

    @State private var visible = false
    @State private var hasRevealed = false

    func body(content: Content) -> some View {
        content
            .opacity(revealOpacity)
            .scaleEffect(revealScale)
            .offset(y: revealOffset)
            .animation(revealAnimation, value: visible)
            .onChange(of: appeared) { _, ready in
                guard enabled, ready, !hasRevealed else { return }
                hasRevealed = true
                let delay = reduceMotion ? 0 : LexturesMotion.staggerDelay(for: index)
                if delay <= 0 {
                    visible = true
                } else {
                    DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                        visible = true
                    }
                }
            }
            .onAppear {
                if !enabled {
                    visible = true
                    return
                }
                guard appeared, !hasRevealed else { return }
                hasRevealed = true
                let delay = reduceMotion ? 0 : LexturesMotion.staggerDelay(for: index)
                if delay <= 0 {
                    visible = true
                } else {
                    DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                        visible = true
                    }
                }
            }
    }

    private var revealOpacity: Double {
        if !enabled { return 1 }
        return visible ? 1 : 0
    }

    private var revealScale: CGFloat {
        if !enabled || reduceMotion { return 1 }
        return visible ? 1 : LexturesMotion.enterScaleFrom
    }

    private var revealOffset: CGFloat {
        if !enabled || reduceMotion { return 0 }
        return visible ? 0 : LexturesMotion.enterTranslate
    }

    private var revealAnimation: Animation? {
        if !enabled { return nil }
        if reduceMotion {
            return .easeOut(duration: LexturesMotion.instant)
        }
        return LexturesMotion.bubble
    }
}

/// Crossfade container: skeleton ↔ content. Tracks "has revealed" so refresh does not re-swap.
///
/// Uses a `VStack` (not `ZStack`) so multi-child `@ViewBuilder` content — `Group` / tuple views —
/// lays out vertically. A `ZStack` flattens those children and stacks every card on one spot.
struct LXLoadReveal<Skeleton: View, Content: View>: View {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let ready: Bool
    let enabled: Bool
    var spacing: CGFloat = 16
    @ViewBuilder let skeleton: () -> Skeleton
    @ViewBuilder let content: () -> Content

    @State private var hasRevealed = false
    @State private var showContent = false

    var body: some View {
        Group {
            if showContent {
                // Inner VStack is the layout boundary: multi-child `@ViewBuilder`
                // content flattens here vertically. Do not put that content in a
                // ZStack or apply modifiers that re-wrap the tuple as one overlay.
                VStack(alignment: .leading, spacing: spacing) {
                    content()
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .transition(contentTransition)
            } else {
                skeleton()
            }
        }
        .animation(crossfadeAnimation, value: showContent)
        .onChange(of: ready) { _, isReady in
            guard isReady, !hasRevealed else { return }
            hasRevealed = true
            showContent = true
        }
        .onAppear {
            if ready, !hasRevealed {
                hasRevealed = true
                showContent = true
            }
        }
        .accessibilityElement(children: .contain)
    }

    private var contentTransition: AnyTransition {
        if enabled && !reduceMotion {
            return .opacity.combined(with: .scale(scale: LexturesMotion.enterScaleFrom))
        }
        return .opacity
    }

    private var crossfadeAnimation: Animation? {
        if !enabled { return nil }
        if reduceMotion {
            return .easeOut(duration: LexturesMotion.instant)
        }
        return .timingCurve(0.2, 0, 0, 1, duration: LexturesMotion.base)
    }
}

// MARK: - AN.4 List / collection motion

/// Max simultaneous list mutation animations (FR-9).
enum LXListMotion {
    static let maxConcurrent = 12
    static let dragLiftScale: CGFloat = 1.03

    /// Whether a mutation at `index` should animate given the concurrent budget.
    static func shouldAnimate(index: Int, reduceMotion: Bool, enabled: Bool) -> Bool {
        if !enabled { return false }
        if reduceMotion { return true } // opacity-only path still runs
        return index < maxConcurrent
    }
}

private struct LXListMotionModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let enabled: Bool

    func body(content: Content) -> some View {
        content
            .transition(listTransition)
            .animation(listAnimation, value: enabled)
    }

    private var listTransition: AnyTransition {
        if !enabled {
            return .identity
        }
        if reduceMotion {
            return .opacity
        }
        return .asymmetric(
            insertion: .opacity
                .combined(with: .scale(scale: LexturesMotion.enterScaleFrom))
                .combined(with: .offset(y: LexturesMotion.enterTranslate)),
            removal: .opacity.combined(with: .scale(scale: LexturesMotion.enterScaleFrom))
        )
    }

    private var listAnimation: Animation? {
        if !enabled { return nil }
        if reduceMotion {
            return .easeOut(duration: LexturesMotion.instant)
        }
        return LexturesMotion.bubble
    }
}

private struct LXListDragLiftModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let isDragging: Bool
    let enabled: Bool

    func body(content: Content) -> some View {
        content
            .scaleEffect(liftScale)
            .shadow(
                color: .black.opacity(isDragging ? (reduceMotion || !enabled ? 0.12 : 0.18) : 0),
                radius: isDragging ? (reduceMotion || !enabled ? 6 : 12) : 0,
                y: isDragging ? 4 : 0
            )
            .animation(
                enabled
                    ? (reduceMotion
                        ? .easeOut(duration: LexturesMotion.instant)
                        : LexturesMotion.bubble)
                    : nil,
                value: isDragging
            )
    }

    private var liftScale: CGFloat {
        if !enabled || !isDragging { return 1 }
        if reduceMotion { return 1 }
        return LXListMotion.dragLiftScale
    }
}

extension View {
    /// AN.4 — insert/remove transition for list rows (ForEach identity required).
    func lxListMotion(enabled: Bool = true) -> some View {
        modifier(LXListMotionModifier(enabled: enabled))
    }

    /// AN.4 — drag lift (scale + shadow); reduced motion → elevation only.
    func lxListDragLift(isDragging: Bool, enabled: Bool = true) -> some View {
        modifier(LXListDragLiftModifier(isDragging: isDragging, enabled: enabled))
    }

    /// AN.5 — dialog enter (scale+fade bubble) / exit; reduced → fade only.
    func lxDialog(enabled: Bool = true) -> some View {
        modifier(LXDialogMotionModifier(enabled: enabled))
    }

    /// AN.5 — sheet/drawer presentation polish + interactive dismiss threshold helper.
    func lxSheet(enabled: Bool = true) -> some View {
        modifier(LXSheetMotionModifier(enabled: enabled))
    }
}

// MARK: AN.5 — Overlay / surface motion

enum LXOverlayMotion {
    /// Drag past this fraction of sheet height dismisses (FR-2 / AC-2).
    static let sheetDismissThreshold: CGFloat = 0.28

    static func shouldDismissSheetDrag(
        offset: CGFloat,
        sheetHeight: CGFloat,
        velocity: CGFloat = 0
    ) -> Bool {
        if sheetHeight <= 0 { return false }
        if velocity > 800 { return true }
        return offset / sheetHeight >= sheetDismissThreshold
    }

    static func dialogAnimation(reduceMotion: Bool, enabled: Bool) -> Animation? {
        if !enabled { return nil }
        if reduceMotion { return .easeOut(duration: LexturesMotion.instant) }
        return LexturesMotion.bubble
    }

    static func sheetAnimation(reduceMotion: Bool, enabled: Bool, exiting: Bool = false) -> Animation? {
        if !enabled { return nil }
        if reduceMotion { return .easeOut(duration: LexturesMotion.instant) }
        return exiting ? LexturesMotion.exit : LexturesMotion.bubble
    }
}

private struct LXDialogMotionModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let enabled: Bool

    func body(content: Content) -> some View {
        content
            .transition(dialogTransition)
            .animation(LXOverlayMotion.dialogAnimation(reduceMotion: reduceMotion, enabled: enabled), value: enabled)
    }

    private var dialogTransition: AnyTransition {
        if !enabled { return .identity }
        if reduceMotion { return .opacity }
        return .asymmetric(
            insertion: .opacity.combined(with: .scale(scale: LexturesMotion.enterScaleFrom)),
            removal: .opacity.combined(with: .scale(scale: LexturesMotion.enterScaleFrom))
        )
    }
}

private struct LXSheetMotionModifier: ViewModifier {
    @Environment(\.lxReduceMotion) private var reduceMotion
    let enabled: Bool

    func body(content: Content) -> some View {
        content
            .transition(sheetTransition)
            .animation(LXOverlayMotion.sheetAnimation(reduceMotion: reduceMotion, enabled: enabled), value: enabled)
            // Interactive dismiss remains platform-native; threshold documented for custom drags.
            .presentationDragIndicator(enabled ? .visible : .automatic)
    }

    private var sheetTransition: AnyTransition {
        if !enabled { return .identity }
        if reduceMotion { return .opacity }
        return .asymmetric(
            insertion: .move(edge: .bottom).combined(with: .opacity),
            removal: .move(edge: .bottom).combined(with: .opacity)
        )
    }
}
