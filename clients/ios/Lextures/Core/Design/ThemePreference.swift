import SwiftUI

/// App-wide appearance override (system / light / dark), persisted on-device.
///
/// Theme is a device-only preference (FR-5): it is not synced to the server, so
/// each device keeps its own look. The selected value is applied at the app root
/// via `preferredColorScheme`, which propagates through `@Environment(\.colorScheme)`.
@Observable
final class ThemePreference {
    static let shared = ThemePreference()

    enum Appearance: String, CaseIterable, Identifiable {
        case system
        case light
        case dark

        var id: String { rawValue }

        /// nil = follow the system setting.
        var colorScheme: ColorScheme? {
            switch self {
            case .system: return nil
            case .light: return .light
            case .dark: return .dark
            }
        }

        /// Localized label key for the picker.
        var labelKey: String.LocalizationValue {
            switch self {
            case .system: return "mobile.settings.theme.system"
            case .light: return "mobile.settings.theme.light"
            case .dark: return "mobile.settings.theme.dark"
            }
        }
    }

    private enum Keys {
        static let appearance = "lextures.theme.appearance"
    }

    var appearance: Appearance {
        didSet { UserDefaults.standard.set(appearance.rawValue, forKey: Keys.appearance) }
    }

    /// Convenience for applying at the root view.
    var colorScheme: ColorScheme? { appearance.colorScheme }

    private init() {
        let stored = UserDefaults.standard.string(forKey: Keys.appearance)
        appearance = stored.flatMap(Appearance.init(rawValue:)) ?? .system
    }

    func reset() {
        appearance = .system
    }
}

private struct ThemePreferenceKey: EnvironmentKey {
    static let defaultValue = ThemePreference.shared
}

extension EnvironmentValues {
    var themePreference: ThemePreference {
        get { self[ThemePreferenceKey.self] }
        set { self[ThemePreferenceKey.self] = newValue }
    }
}
