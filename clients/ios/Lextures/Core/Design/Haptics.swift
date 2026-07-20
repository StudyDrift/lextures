import UIKit

/// AN.6 — Standardized haptics mapping for control interactions (FR-5).
///
/// Respects the OS haptics / reduce-motion settings via `UIFeedbackGenerator`.
/// Never call from the main action path in a way that gates the handler (FR-9).
enum Haptics {
    enum Kind: String, CaseIterable {
        case tap
        case selection
        case success
        case error
    }

    /// System feedback type name used for tests / logging (AC mapping table).
    static func systemName(for kind: Kind) -> String {
        switch kind {
        case .tap: return "lightImpact"
        case .selection: return "selection"
        case .success: return "notificationSuccess"
        case .error: return "notificationError"
        }
    }

    /// Fire a haptic if `enabled` and the OS allows feedback.
    static func trigger(_ kind: Kind, enabled: Bool = true) {
        guard enabled else { return }
        switch kind {
        case .tap:
            let gen = UIImpactFeedbackGenerator(style: .light)
            gen.prepare()
            gen.impactOccurred()
        case .selection:
            let gen = UISelectionFeedbackGenerator()
            gen.prepare()
            gen.selectionChanged()
        case .success:
            let gen = UINotificationFeedbackGenerator()
            gen.prepare()
            gen.notificationOccurred(.success)
        case .error:
            let gen = UINotificationFeedbackGenerator()
            gen.prepare()
            gen.notificationOccurred(.error)
        }
    }
}
