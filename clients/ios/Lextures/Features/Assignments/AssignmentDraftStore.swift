import Foundation

/// Persists unsubmitted assignment text locally (M5.1).
enum AssignmentDraftStore {
    private static let defaults = UserDefaults.standard

    static func load(key: String) -> String {
        defaults.string(forKey: key) ?? ""
    }

    static func save(key: String, text: String) {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            defaults.removeObject(forKey: key)
        } else {
            defaults.set(text, forKey: key)
        }
    }

    static func clear(key: String) {
        defaults.removeObject(forKey: key)
    }
}
