import Foundation

/// Mirrors web `student-notebook-storage` (device-local notebooks, format v2).
struct NotebookPage: Codable, Identifiable, Hashable {
    var id: String
    var title: String
    var parentId: String?
    var sortOrder: Int
    var kind: String // "page" | "group"
    var contentMd: String

    static func new(title: String = "Untitled", sortOrder: Int = 0) -> NotebookPage {
        NotebookPage(
            id: UUID().uuidString.lowercased(),
            title: title,
            parentId: nil,
            sortOrder: sortOrder,
            kind: "page",
            contentMd: ""
        )
    }
}

struct CourseNotebook: Codable {
    var formatVersion: Int
    var updatedAt: String
    var courseTitle: String?
    var pages: [NotebookPage]
    var activePageId: String?

    static func empty() -> CourseNotebook {
        let page = NotebookPage.new()
        return CourseNotebook(
            formatVersion: 2,
            updatedAt: ISO8601DateFormatter().string(from: Date()),
            courseTitle: nil,
            pages: [page],
            activePageId: page.id
        )
    }

    var previewText: String {
        pages.map(\.contentMd).joined(separator: "\n\n").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

/// Device-local notebook persistence keyed per signed-in user (JWT `sub`), parity with web localStorage.
@MainActor
final class NotebookStore {
    /// Learner-wide notebook key — must not collide with real course codes (same value as web).
    static let globalKey = "__lextures_global__"
    static let globalTitle = "Global notebook"

    private let defaults = UserDefaults.standard
    private let ownerKey: String

    init(accessToken: String?) {
        ownerKey = NotebookStore.jwtSubject(from: accessToken) ?? "anonymous"
    }

    private var storageKey: String {
        "lextures.studentNotebooks.v1:\(ownerKey)"
    }

    private func readAll() -> [String: CourseNotebook] {
        guard let data = defaults.data(forKey: storageKey) else { return [:] }
        return (try? JSONDecoder().decode([String: CourseNotebook].self, from: data)) ?? [:]
    }

    private func writeAll(_ notebooks: [String: CourseNotebook]) {
        guard let data = try? JSONEncoder().encode(notebooks) else { return }
        defaults.set(data, forKey: storageKey)
    }

    func load(courseCode: String) -> CourseNotebook {
        readAll()[courseCode] ?? .empty()
    }

    func save(courseCode: String, notebook: CourseNotebook) {
        var all = readAll()
        var next = notebook
        next.updatedAt = ISO8601DateFormatter().string(from: Date())
        all[courseCode] = next
        writeAll(all)
    }

    func exists(courseCode: String) -> Bool {
        readAll()[courseCode] != nil
    }

    /// Every stored course code, including the global key.
    func allCourseCodes() -> [String] {
        Array(readAll().keys)
    }

    /// Write a server copy verbatim — keeps the server `updatedAt` so last-write-wins stays stable.
    func saveFromServer(courseCode: String, notebook: CourseNotebook) {
        var all = readAll()
        all[courseCode] = notebook
        writeAll(all)
    }

    /// Course-scoped notebooks with content (excludes the global key).
    func listCourseNotebooks() -> [String: CourseNotebook] {
        readAll().filter { key, notebook in
            key != NotebookStore.globalKey && !notebook.previewText.isEmpty
        }
    }

    static func jwtSubject(from token: String?) -> String? {
        guard let token else { return nil }
        let parts = token.split(separator: ".")
        guard parts.count >= 2 else { return nil }
        var payload = String(parts[1])
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        while payload.count % 4 != 0 { payload.append("=") }
        guard
            let data = Data(base64Encoded: payload),
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let sub = json["sub"] as? String, !sub.isEmpty
        else { return nil }
        return sub
    }
}
