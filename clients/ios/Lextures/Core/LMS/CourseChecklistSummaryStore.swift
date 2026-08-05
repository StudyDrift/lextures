import Foundation

/// In-memory-only checklist summary memo (FR-8, FR-11). Never writes to disk.
final class CourseChecklistSummaryStore: @unchecked Sendable {
    static let shared = CourseChecklistSummaryStore()

    private let lock = NSLock()
    private var entries: [String: CourseChecklistLogic.SummaryCacheEntry] = [:]
    private var hiddenBy403: Set<String> = []

    func cached(courseCode: String) -> CourseChecklistSummary? {
        lock.lock()
        defer { lock.unlock() }
        guard let entry = entries[courseCode],
              CourseChecklistLogic.isSummaryFresh(entry) else { return nil }
        return entry.summary
    }

    func outstandingEssential(courseCode: String) -> Int {
        cached(courseCode: courseCode)?.outstandingEssential ?? 0
    }

    func isHiddenBy403(courseCode: String) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return hiddenBy403.contains(courseCode)
    }

    func put(
        courseCode: String,
        summary: CourseChecklistSummary,
        catalogVersion: String? = nil
    ) {
        lock.lock()
        defer { lock.unlock() }
        hiddenBy403.remove(courseCode)
        entries[courseCode] = CourseChecklistLogic.SummaryCacheEntry(
            summary: summary,
            catalogVersion: catalogVersion,
            fetchedAt: Date()
        )
    }

    func markForbidden(courseCode: String) {
        lock.lock()
        defer { lock.unlock() }
        hiddenBy403.insert(courseCode)
        entries.removeValue(forKey: courseCode)
    }

    func invalidate(courseCode: String) {
        lock.lock()
        defer { lock.unlock() }
        entries.removeValue(forKey: courseCode)
    }

    func clearAll() {
        lock.lock()
        defer { lock.unlock() }
        entries.removeAll()
        hiddenBy403.removeAll()
    }

    /// Apply a full checklist response (updates summary + catalog version).
    func applyChecklist(_ checklist: CourseChecklist) {
        put(
            courseCode: checklist.courseCode,
            summary: checklist.summary,
            catalogVersion: checklist.catalogVersion
        )
    }
}
