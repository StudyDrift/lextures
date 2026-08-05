import Foundation

/// Bundled CC.8 native target table (anchor → iOS destination).
enum CourseChecklistTargetTable {
    private static var cached: [String: String]?

    static var shared: [String: String] {
        if let cached { return cached }
        let map = load()
        cached = map
        return map
    }

    static func load(bundle: Bundle = .main) -> [String: String] {
        if let url = bundle.url(forResource: "checklist-targets", withExtension: "json"),
           let data = try? Data(contentsOf: url) {
            return CourseChecklistLogic.loadTargetTable(from: data)
        }
        // Fallback: packages path when running tests without app resources.
        let thisFile = URL(fileURLWithPath: #filePath)
        let candidates = [
            thisFile
                .deletingLastPathComponent()
                .deletingLastPathComponent()
                .deletingLastPathComponent()
                .deletingLastPathComponent()
                .appendingPathComponent("packages/checklist-targets.json"),
            URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
                .appendingPathComponent("clients/packages/checklist-targets.json"),
        ]
        for url in candidates {
            if let data = try? Data(contentsOf: url) {
                return CourseChecklistLogic.loadTargetTable(from: data)
            }
        }
        return [:]
    }

    #if DEBUG
    static func resetCacheForTests() {
        cached = nil
    }
    #endif
}
