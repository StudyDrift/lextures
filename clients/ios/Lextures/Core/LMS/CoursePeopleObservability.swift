import Foundation

/// Client-only enrollment management counters (MOB.4). Never records emails/PII.
enum CoursePeopleObservability {
    private static let defaults = UserDefaults.standard

    static func recordAdded(role: String, addedCount: Int, alreadyCount: Int, notFoundCount: Int) {
        if addedCount > 0 {
            bump("enrollment_added")
            defaults.set(role, forKey: "enrollment_added.last_role")
            defaults.set(addedCount, forKey: "enrollment_added.last_count")
        }
        if alreadyCount > 0 {
            bump("enrollment_add_already")
        }
        if notFoundCount > 0 {
            bump("enrollment_add_not_found")
        }
    }

    static func recordStateChanged(role: String, state: String) {
        bump("enrollment_state_changed")
        defaults.set(role, forKey: "enrollment_state_changed.last_role")
        defaults.set(state, forKey: "enrollment_state_changed.last_state")
    }

    static func recordRemoved(role: String) {
        bump("enrollment_removed")
        defaults.set(role, forKey: "enrollment_removed.last_role")
    }

    private static func bump(_ key: String) {
        let prev = defaults.integer(forKey: key)
        defaults.set(prev + 1, forKey: key)
    }
}
