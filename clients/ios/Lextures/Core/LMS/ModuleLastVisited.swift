import Foundation

struct LastVisitedModuleEntry: Codable, Equatable {
    var itemId: String
    var kind: String
    var title: String
    var openedAt: String
}

/// Per-course last opened module item (parity with web `last-visited-module-item.ts`).
enum ModuleLastVisited {
    private static let storageKey = "lextures:last-module-item:v1"

    static func record(courseCode: String, itemId: String, kind: String, title: String) {
        let code = courseCode.trimmingCharacters(in: .whitespacesAndNewlines)
        let id = itemId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !code.isEmpty, !id.isEmpty else { return }

        var store = readStore()
        store[code] = LastVisitedModuleEntry(
            itemId: id,
            kind: kind,
            title: title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Untitled" : title,
            openedAt: ISO8601DateFormatter().string(from: Date())
        )
        writeStore(store)
    }

    static func entry(for courseCode: String) -> LastVisitedModuleEntry? {
        let code = courseCode.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !code.isEmpty else { return nil }
        return readStore()[code]
    }

    private static func readStore() -> [String: LastVisitedModuleEntry] {
        guard let data = UserDefaults.standard.data(forKey: storageKey) else { return [:] }
        return (try? JSONDecoder().decode([String: LastVisitedModuleEntry].self, from: data)) ?? [:]
    }

    private static func writeStore(_ store: [String: LastVisitedModuleEntry]) {
        guard let data = try? JSONEncoder().encode(store) else { return }
        UserDefaults.standard.set(data, forKey: storageKey)
    }
}