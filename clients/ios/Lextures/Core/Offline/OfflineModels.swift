import Foundation

/// Typed cache keys for LMS read responses (resource + params).
enum OfflineCacheKey {
    static func courses() -> String { "courses" }
    static func course(_ courseCode: String) -> String { "course:\(courseCode)" }
    static func courseStructure(_ courseCode: String) -> String { "course:\(courseCode):structure" }
    static func myGrades(_ courseCode: String) -> String { "course:\(courseCode):my-grades" }
    static func itemDetail(courseCode: String, itemId: String) -> String {
        "course:\(courseCode):item:\(itemId)"
    }
    static func modulesProgress(_ courseCode: String) -> String {
        "course:\(courseCode):modules-progress"
    }
    static func contentPage(courseCode: String, itemId: String) -> String {
        "course:\(courseCode):content-page:\(itemId)"
    }
    static func vibeActivity(courseCode: String, itemId: String) -> String {
        "course:\(courseCode):vibe-activity:\(itemId)"
    }
    static func courseFiles(courseCode: String, folderId: String?) -> String {
        CourseFileLogic.courseFilesCacheKey(courseCode: courseCode, folderId: folderId)
    }
    static func plannerSnapshot() -> String { "planner:snapshot" }
    static func notificationsPage() -> String { "notifications:page" }
    static func notificationPreferences() -> String { "notifications:preferences" }
    static func officeHours(_ courseCode: String) -> String { "course:\(courseCode):office-hours" }
    static func discussionForums(_ courseCode: String) -> String { "course:\(courseCode):discussion-forums" }
    static func discussionThreads(courseCode: String, forumId: String) -> String {
        "course:\(courseCode):discussion-threads:\(forumId)"
    }
    static func discussionThread(courseCode: String, threadId: String) -> String {
        "course:\(courseCode):discussion-thread:\(threadId)"
    }
    static func discussionPosts(courseCode: String, threadId: String) -> String {
        "course:\(courseCode):discussion-posts:\(threadId)"
    }
}

/// A cached value plus freshness metadata for read screens.
struct Cached<T> {
    let value: T
    let fetchedAt: Date

    /// True when the UI should show a staleness chip (offline replay or aged cache).
    func isStale(isOnline: Bool, maxFreshAge: TimeInterval = 5 * 60) -> Bool {
        !isOnline || Date().timeIntervalSince(fetchedAt) > maxFreshAge
    }

    var lastUpdatedLabel: String {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .abbreviated
        return "Last updated \(formatter.localizedString(for: fetchedAt, relativeTo: Date()))"
    }
}

enum OutboxStatus: String, Codable, CaseIterable {
    case queued
    case syncing
    case synced
    case failed
    case conflict

    var userLabel: String {
        switch self {
        case .queued: return "Saved locally — will sync"
        case .syncing: return "Syncing…"
        case .synced: return "Synced"
        case .failed: return "Sync failed — retry"
        case .conflict: return "Conflict — review required"
        }
    }
}

/// One queued mutation replayed in order on reconnect.
struct OutboxItem: Codable, Identifiable, Equatable {
    let id: String
    let createdAt: Date
    let sequence: Int
    let method: String
    let path: String
    let bodyJSON: String?
    let label: String
    var status: OutboxStatus
    var lastError: String?

    var idempotencyKey: String { id }
}

enum OfflineStorageBudget {
    /// Shared LRU budget for read cache + downloads (bytes).
    static let defaultMaxBytes: Int = 50 * 1024 * 1024
}

/// Non-PII sync counters for diagnostics (plan 17.7).
struct OfflineSyncMetrics: Codable, Equatable {
    var successCount: Int = 0
    var failureCount: Int = 0
    var conflictCount: Int = 0
    var lastSyncAt: Date?
}
