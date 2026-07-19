import Foundation

/// Persists in-progress create-wizard UI state across backgrounding (MOB.1 FR-10).
enum CourseCreateDraftStore {
    private static let defaults = UserDefaults.standard
    private static let prefix = "course_create_draft."

    struct Draft: Codable, Equatable {
        var step: Int
        var title: String
        var description: String
        var courseMode: String
        var selectedTermId: String
        var selectedGradeLevel: String
        var selectedTemplateId: String
        var firstModuleTitle: String
        var createdCourseCode: String?
        var competencies: [CourseCreateLogic.CompetencyDraft]
        var createSource: String?
    }

    static func storageKey(userId: String?, orgId: String?) -> String {
        let user = (userId?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "anon"
        let org = (orgId?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "org"
        return prefix + user + "." + org
    }

    static func load(key: String) -> Draft? {
        guard let data = defaults.data(forKey: key) else { return nil }
        return try? JSONDecoder().decode(Draft.self, from: data)
    }

    static func save(key: String, draft: Draft) {
        guard let data = try? JSONEncoder().encode(draft) else { return }
        defaults.set(data, forKey: key)
    }

    static func clear(key: String) {
        defaults.removeObject(forKey: key)
    }
}
