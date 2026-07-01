import Foundation

/// Client rollout flag for M1.5 profile depth sections.
enum MobileProfileDepthPreferences {
    private static let key = "mobile_profile_depth"

    static var isEnabled: Bool {
        get {
            if UserDefaults.standard.object(forKey: key) == nil {
                return true
            }
            return UserDefaults.standard.bool(forKey: key)
        }
        set { UserDefaults.standard.set(newValue, forKey: key) }
    }
}