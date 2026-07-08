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
    static func liveMeetings(_ courseCode: String) -> String { "course:\(courseCode):live-meetings" }
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
    static func reviewQueue() -> String { "review:queue" }
    static func reviewStats() -> String { "review:stats" }
    static func feedChannels(_ courseCode: String) -> String { "course:\(courseCode):feed-channels" }
    static func feedMessages(courseCode: String, channelId: String) -> String {
        "course:\(courseCode):feed-messages:\(channelId)"
    }
    static func myGroups(_ courseCode: String) -> String { "course:\(courseCode):my-groups" }
    static func groupFeedChannels(courseCode: String, groupId: String) -> String {
        "course:\(courseCode):group:\(groupId):feed-channels"
    }
    static func groupFeedMessages(courseCode: String, groupId: String, channelId: String) -> String {
        "course:\(courseCode):group:\(groupId):feed-messages:\(channelId)"
    }
    static func collabDocs(_ courseCode: String) -> String { "course:\(courseCode):collab-docs" }
    static func collabDoc(courseCode: String, docId: String) -> String {
        "course:\(courseCode):collab-doc:\(docId)"
    }
    static func courseEnrollments(_ courseCode: String) -> String {
        "course:\(courseCode):enrollments"
    }
    static func courseAtRisk(_ courseCode: String) -> String {
        "course:\(courseCode):at-risk"
    }
    static func courseInstructorInsights(_ courseCode: String) -> String {
        "course:\(courseCode):instructor-insights"
    }
    static func studentProgress(courseCode: String, enrollmentId: String) -> String {
        "course:\(courseCode):student-progress:\(enrollmentId)"
    }
    static func myPaths() -> String { "paths:my" }
    static func pathProgress(_ pathId: String) -> String { "paths:progress:\(pathId)" }
    static func catalogPaths(query: String) -> String { "paths:catalog:\(query)" }
    static func catalogCourses(key: String) -> String { "catalog:courses:\(key)" }
    static func catalogCourseDetail(slug: String) -> String { "catalog:course:\(slug)" }
    static func catalogCategories() -> String { "catalog:categories" }
    static func studyStats() -> String { "insights:study-stats" }
    static func reflectionJournal() -> String { "insights:reflection-journal" }
    static func coachingTips() -> String { "insights:coaching-tips" }
    static func readingLog() -> String { "reading:log" }
    static func libraryBooks(orgId: String, gradeBand: String) -> String {
        "reading:library:\(orgId):\(gradeBand)"
    }
    static func credentialsList() -> String { "credentials:list" }
    static func walletCCR() -> String { "wallet:ccr" }
    static func walletCETranscript() -> String { "wallet:ce-transcript" }
    static func walletTranscriptRequests() -> String { "wallet:transcript-requests" }
    static func portfolioList() -> String { "portfolio:list" }
    static func portfolioDetail(portfolioId: String) -> String { "portfolio:\(portfolioId)" }
    static func gamificationProfile() -> String { "gamification:profile" }
    static func gamificationLeaderboard(courseCode: String) -> String {
        "gamification:leaderboard:\(courseCode)"
    }
    static func advisingNotes() -> String { "advising:notes" }
    static func degreeProgress() -> String { "advising:degree-progress" }
    static func evaluationStatus(_ courseCode: String) -> String { "evaluation:status:\(courseCode)" }
    static func evaluationResults(_ courseCode: String) -> String { "evaluation:results:\(courseCode)" }
    static func conferenceSlots(teacherId: String, date: String) -> String {
        "parent:conference-slots:\(teacherId):\(date)"
    }
    static func learnerProfile() -> String { "learner-profile:summary" }
    static func learnerProfileEvidence(_ facetKey: String) -> String { "learner-profile:evidence:\(facetKey)" }
    static func introCourseProgress() -> String { "intro-course:progress" }
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
