import SwiftUI

/// App-wide reading/display preferences (dyslexia preset, TTS speed, reduce motion).
@Observable
final class AccessibilityPreferences {
    static let shared = AccessibilityPreferences()

    private enum Keys {
        static let dyslexiaDisplay = "lextures.a11y.dyslexiaDisplay"
        static let ttsSpeed = "lextures.a11y.ttsSpeed"
        static let reducedMotion = "lextures.a11y.reducedMotion"
    }

    var dyslexiaDisplayEnabled: Bool {
        didSet { UserDefaults.standard.set(dyslexiaDisplayEnabled, forKey: Keys.dyslexiaDisplay) }
    }

    /// In-app reduce-motion override (AN.1). Combined with OS accessibilityReduceMotion via `lxReduceMotion`.
    var reducedMotionEnabled: Bool {
        didSet { UserDefaults.standard.set(reducedMotionEnabled, forKey: Keys.reducedMotion) }
    }

    /// AVSpeechUtterance rate multiplier (0.5 = slow, 1.0 = default, 1.5 = fast).
    var ttsSpeed: Float {
        didSet { UserDefaults.standard.set(Double(ttsSpeed), forKey: Keys.ttsSpeed) }
    }

    private init() {
        dyslexiaDisplayEnabled = UserDefaults.standard.bool(forKey: Keys.dyslexiaDisplay)
        reducedMotionEnabled = UserDefaults.standard.bool(forKey: Keys.reducedMotion)
        let stored = UserDefaults.standard.object(forKey: Keys.ttsSpeed) as? Double
        ttsSpeed = Float(stored ?? 1.0)
    }

    func reset() {
        dyslexiaDisplayEnabled = false
        reducedMotionEnabled = false
        ttsSpeed = 1.0
    }
}

private struct AccessibilityPreferencesKey: EnvironmentKey {
    static let defaultValue = AccessibilityPreferences.shared
}

extension EnvironmentValues {
    var accessibilityPreferences: AccessibilityPreferences {
        get { self[AccessibilityPreferencesKey.self] }
        set { self[AccessibilityPreferencesKey.self] = newValue }
    }
}
