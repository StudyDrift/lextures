import Foundation
import Observation

/// Persists the tenant / environment chosen on the get-started screen.
/// The resolved API base URL drives all network traffic via `AppConfiguration`.
@Observable
final class EnvironmentStore {
    static let shared = EnvironmentStore()

    private enum Keys {
        static let apiBaseURL = "lextures.environment.apiBaseURL"
        static let kind = "lextures.environment.kind"
        static let schoolCode = "lextures.environment.schoolCode"
    }

    enum Kind: String {
        // DO NOT RENAME — persisted in UserDefaults as "selfLearner"; pin keeps upgrades in place.
        case homeschool = "selfLearner"
        case school
    }

    private let defaults: UserDefaults

    private(set) var apiBaseURLString: String?
    private(set) var kind: Kind?
    private(set) var schoolCode: String?

    var hasSelection: Bool {
        guard let apiBaseURLString, !apiBaseURLString.isEmpty else { return false }
        return true
    }

    var apiBaseURL: URL? {
        guard let apiBaseURLString, let url = URL(string: apiBaseURLString) else { return nil }
        return url
    }

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        apiBaseURLString = defaults.string(forKey: Keys.apiBaseURL)
        if let raw = defaults.string(forKey: Keys.kind) {
            kind = Kind(rawValue: raw)
        }
        schoolCode = defaults.string(forKey: Keys.schoolCode)
    }

    func selectHomeschool() {
        persist(kind: .homeschool, schoolCode: nil, apiBaseURL: SchoolCodeLogic.homeschoolAPIBase)
    }

    func selectSchool(code: String) {
        let normalized = SchoolCodeLogic.normalize(code)
        persist(
            kind: .school,
            schoolCode: normalized,
            apiBaseURL: SchoolCodeLogic.apiBaseURL(schoolCode: normalized)
        )
    }

    /// Clears the selection so the get-started flow is shown again.
    func clearSelection() {
        apiBaseURLString = nil
        kind = nil
        schoolCode = nil
        defaults.removeObject(forKey: Keys.apiBaseURL)
        defaults.removeObject(forKey: Keys.kind)
        defaults.removeObject(forKey: Keys.schoolCode)
    }

    private func persist(kind: Kind, schoolCode: String?, apiBaseURL: String) {
        let trimmed = apiBaseURL.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        self.kind = kind
        self.schoolCode = schoolCode
        self.apiBaseURLString = trimmed
        defaults.set(trimmed, forKey: Keys.apiBaseURL)
        defaults.set(kind.rawValue, forKey: Keys.kind)
        if let schoolCode {
            defaults.set(schoolCode, forKey: Keys.schoolCode)
        } else {
            defaults.removeObject(forKey: Keys.schoolCode)
        }
    }
}
