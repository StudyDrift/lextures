import Foundation

/// Client-only Canvas import funnel counters (MOB.2). Never records the Canvas token.
enum CanvasImportObservability {
    private static let defaults = UserDefaults.standard

    static func recordListed(courseCount: Int) {
        bump("canvas_import_listed")
        defaults.set(courseCount, forKey: "canvas_import_listed.last_count")
    }

    static func recordStarted(include: CanvasImportLogic.Include) {
        bump("canvas_import_started")
        defaults.set(include.enabledCategoryCounts, forKey: "canvas_import_started.categories")
    }

    static func recordProgress() {
        bump("canvas_import_progress")
    }

    static func recordSucceeded(include: CanvasImportLogic.Include) {
        bump("canvas_import_succeeded")
        defaults.set(include.enabledCategoryCounts, forKey: "canvas_import_succeeded.categories")
    }

    static func recordFailed() {
        bump("canvas_import_failed")
    }

    static func recordCancelled() {
        bump("canvas_import_cancelled")
    }

    private static func bump(_ key: String) {
        let prev = defaults.integer(forKey: key)
        defaults.set(prev + 1, forKey: key)
    }
}
