import SwiftUI

/// App-wide reading/display preferences (dyslexia preset, TTS speed).
@Observable
final class AccessibilityPreferences {
    static let shared = AccessibilityPreferences()

    private enum Keys {
        static let dyslexiaDisplay = "lextures.a11y.dyslexiaDisplay"
        static let ttsSpeed = "lextures.a11y.ttsSpeed"
    }

    var dyslexiaDisplayEnabled: Bool {
        didSet { UserDefaults.standard.set(dyslexiaDisplayEnabled, forKey: Keys.dyslexiaDisplay) }
    }

    /// AVSpeechUtterance rate multiplier (0.5 = slow, 1.0 = default, 1.5 = fast).
    var ttsSpeed: Float {
        didSet { UserDefaults.standard.set(Double(ttsSpeed), forKey: Keys.ttsSpeed) }
    }

    private init() {
        dyslexiaDisplayEnabled = UserDefaults.standard.bool(forKey: Keys.dyslexiaDisplay)
        let stored = UserDefaults.standard.object(forKey: Keys.ttsSpeed) as? Double
        ttsSpeed = Float(stored ?? 1.0)
    }

    func reset() {
        dyslexiaDisplayEnabled = false
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
