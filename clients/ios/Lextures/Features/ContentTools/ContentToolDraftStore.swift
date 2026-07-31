import Foundation

/// Local free-text draft store for CT.M6 composers.
/// Keyed separately from tool state so a draft is never mistaken for saved work (FR-2).
enum ContentToolDraftStore {
    private static let defaults = UserDefaults.standard

    static func load(key: String) -> String {
        defaults.string(forKey: key) ?? ""
    }

    static func save(key: String, text: String) {
        if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            defaults.removeObject(forKey: key)
        } else {
            defaults.set(text, forKey: key)
        }
    }

    static func clear(key: String) {
        defaults.removeObject(forKey: key)
    }
}
