import Foundation

/// Client-side rollout flag for the M0.5 information-architecture redesign.
/// Default off so the legacy 5-tab shell remains the rollback path (AC-6).
enum MobileIaPreferences {
    private static let key = "mobile_ia_redesign"
    private static let universalSearchKey = "mobile_universal_search"

    static var isRedesignEnabled: Bool {
        get { UserDefaults.standard.bool(forKey: key) }
        set { UserDefaults.standard.set(newValue, forKey: key) }
    }

    static var isUniversalSearchEnabled: Bool {
        get { UserDefaults.standard.bool(forKey: universalSearchKey) }
        set { UserDefaults.standard.set(newValue, forKey: universalSearchKey) }
    }

    private static let roleContextKey = "mobile_ia_role_context"

    static func loadRoleContext() -> MobileRoleContext? {
        guard let raw = UserDefaults.standard.string(forKey: roleContextKey) else { return nil }
        return MobileRoleContext(rawValue: raw)
    }

    static func saveRoleContext(_ context: MobileRoleContext) {
        UserDefaults.standard.set(context.rawValue, forKey: roleContextKey)
    }

    private static let selectedChildKey = "mobile_parent_selected_child"

    static func loadSelectedChildId() -> String? {
        UserDefaults.standard.string(forKey: selectedChildKey)
    }

    static func saveSelectedChildId(_ studentId: String) {
        UserDefaults.standard.set(studentId, forKey: selectedChildKey)
    }
}