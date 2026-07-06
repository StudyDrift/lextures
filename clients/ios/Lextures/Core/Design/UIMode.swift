import SwiftUI

/// Age-appropriate UI mode (plan 13.11 / M10.4).
enum UIMode: String, CaseIterable, Equatable, Identifiable {
    case standard
    case elementary
    case k2

    var id: String { rawValue }

    var isYoung: Bool { self != .standard }

    /// Minimum interactive height (pt). K2 exceeds WCAG; elementary matches M0.3 baseline.
    var minimumTapTarget: CGFloat {
        switch self {
        case .k2: return 48
        case .elementary, .standard: return AccessibilitySupport.minimumTapTarget
        }
    }

    var baseBodyPointSize: CGFloat {
        switch self {
        case .k2: return 24
        case .elementary: return 18
        case .standard: return 17
        }
    }

    var drawerIconPointSize: CGFloat {
        switch self {
        case .k2: return 28
        case .elementary: return 22
        case .standard: return 16
        }
    }

    var drawerRowVerticalPadding: CGFloat {
        switch self {
        case .k2: return 16
        case .elementary: return 13
        case .standard: return 11
        }
    }

    var choiceButtonMinHeight: CGFloat {
        switch self {
        case .k2: return 56
        case .elementary: return 48
        case .standard: return 44
        }
    }
}

/// User-facing override stored on device. `auto` follows the server-derived mode.
enum UIModePreference: String, CaseIterable, Equatable, Identifiable {
    case auto
    case k2
    case elementary
    case standard

    var id: String { rawValue }

    var label: String {
        switch self {
        case .auto: return L.text("mobile.uiMode.preference.auto")
        case .k2: return L.text("mobile.uiMode.preference.k2")
        case .elementary: return L.text("mobile.uiMode.preference.elementary")
        case .standard: return L.text("mobile.uiMode.preference.standard")
        }
    }

    var resolvedMode: UIMode? {
        switch self {
        case .auto: return nil
        case .k2: return .k2
        case .elementary: return .elementary
        case .standard: return .standard
        }
    }
}

/// Derives effective UI mode from grade level and overrides (mirrors server `readingprefs` package).
enum UIModeLogic {
    static func gradeToUIMode(_ gradeLevel: String?) -> UIMode {
        guard let gradeLevel else { return .standard }
        switch gradeLevel {
        case "K", "1", "2": return .k2
        case "3", "4", "5": return .elementary
        default: return .standard
        }
    }

    static func parseMode(_ raw: String?) -> UIMode? {
        guard let raw else { return nil }
        return UIMode(rawValue: raw)
    }

    /// Admin/server override beats local preference; local preference beats grade-derived mode.
    static func effectiveMode(
        featureEnabled: Bool,
        roleContext: MobileRoleContext,
        serverOverride: String?,
        serverEffective: String?,
        localPreference: UIModePreference
    ) -> UIMode {
        guard featureEnabled, roleContext == .learning else { return .standard }
        if let serverOverride, let mode = parseMode(serverOverride) { return mode }
        if let local = localPreference.resolvedMode { return local }
        if let serverEffective, let mode = parseMode(serverEffective) { return mode }
        return .standard
    }

    static func drawerLabel(for destination: RootDestination, mode: UIMode) -> String {
        if mode == .k2, let key = k2DrawerLabelKey(destination) {
            return L.text(String.LocalizationValue(key))
        }
        if mode == .elementary, let key = elementaryDrawerLabelKey(destination) {
            return L.text(String.LocalizationValue(key))
        }
        return destination.label
    }

    static func moreLabel(for destination: MoreDestination, mode: UIMode) -> String {
        if mode == .k2, let key = k2MoreLabelKey(destination) {
            return L.text(String.LocalizationValue(key))
        }
        return destination.label
    }

    private static func k2DrawerLabelKey(_ destination: RootDestination) -> String? {
        switch destination {
        case .dashboard: return "mobile.uiMode.young.dashboard"
        case .courses: return "mobile.uiMode.young.courses"
        case .todos: return "mobile.uiMode.young.todos"
        case .calendar: return "mobile.uiMode.young.calendar"
        case .inbox: return "mobile.uiMode.young.inbox"
        case .settings: return "mobile.uiMode.young.settings"
        default: return nil
        }
    }

    private static func elementaryDrawerLabelKey(_ destination: RootDestination) -> String? {
        switch destination {
        case .dashboard: return "mobile.uiMode.elementary.dashboard"
        case .courses: return "mobile.uiMode.elementary.courses"
        case .todos: return "mobile.uiMode.elementary.todos"
        default: return nil
        }
    }

    private static func k2MoreLabelKey(_ destination: MoreDestination) -> String? {
        switch destination {
        case .reading: return "mobile.uiMode.young.reading"
        case .settings: return "mobile.uiMode.young.settings"
        default: return nil
        }
    }
}

/// Persists a device override and mirrors server `effectiveUiMode` / `uiModeOverride`.
@MainActor
@Observable
final class UIModeStore {
    static let shared = UIModeStore()

    private enum Keys {
        static let preference = "lextures.uiMode.preference"
        static let serverEffective = "lextures.uiMode.serverEffective"
        static let serverOverride = "lextures.uiMode.serverOverride"
    }

    private(set) var featureEnabled = false
    private(set) var serverEffectiveMode: UIMode = .standard
    private(set) var serverOverrideMode: UIMode?
    private(set) var lastRoleContext: MobileRoleContext = .learning
    var localPreference: UIModePreference {
        didSet { UserDefaults.standard.set(localPreference.rawValue, forKey: Keys.preference) }
    }

    private init() {
        let stored = UserDefaults.standard.string(forKey: Keys.preference) ?? UIModePreference.auto.rawValue
        localPreference = UIModePreference(rawValue: stored) ?? .auto
        serverEffectiveMode = UIMode(rawValue: UserDefaults.standard.string(forKey: Keys.serverEffective) ?? "") ?? .standard
        if let override = UserDefaults.standard.string(forKey: Keys.serverOverride) {
            serverOverrideMode = UIMode(rawValue: override)
        }
    }

    func effectiveMode(roleContext: MobileRoleContext) -> UIMode {
        lastRoleContext = roleContext
        return UIModeLogic.effectiveMode(
            featureEnabled: featureEnabled,
            roleContext: roleContext,
            serverOverride: serverOverrideMode?.rawValue,
            serverEffective: serverEffectiveMode.rawValue,
            localPreference: localPreference
        )
    }

    /// Last resolved mode for views that cannot access [AppShellModel].
    var resolvedMode: UIMode {
        effectiveMode(roleContext: lastRoleContext)
    }

    func updatePlatform(featureEnabled: Bool) {
        self.featureEnabled = featureEnabled
    }

    func applyReadingPreferences(
        effectiveUiMode: String?,
        uiModeOverride: String?,
        featureEnabled: Bool
    ) {
        self.featureEnabled = featureEnabled
        if let effectiveUiMode, let mode = UIMode(rawValue: effectiveUiMode) {
            serverEffectiveMode = mode
            UserDefaults.standard.set(mode.rawValue, forKey: Keys.serverEffective)
        }
        if let uiModeOverride, let mode = UIMode(rawValue: uiModeOverride) {
            serverOverrideMode = mode
            UserDefaults.standard.set(mode.rawValue, forKey: Keys.serverOverride)
        } else if uiModeOverride == nil {
            serverOverrideMode = nil
            UserDefaults.standard.removeObject(forKey: Keys.serverOverride)
        }
    }

    var hasAdminOverride: Bool { serverOverrideMode != nil }
}

private struct UIModeStoreKey: EnvironmentKey {
    static let defaultValue = UIModeStore.shared
}

extension EnvironmentValues {
    var uiModeStore: UIModeStore {
        get { self[UIModeStoreKey.self] }
        set { self[UIModeStoreKey.self] = newValue }
    }
}
