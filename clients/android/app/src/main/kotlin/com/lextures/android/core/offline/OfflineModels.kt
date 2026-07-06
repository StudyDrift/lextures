package com.lextures.android.core.offline

import java.time.Instant
import java.time.temporal.ChronoUnit
import java.util.UUID
import kotlinx.serialization.Serializable

object OfflineCacheKey {
    fun courses(): String = "courses"
    fun course(courseCode: String): String = "course:$courseCode"
    fun courseStructure(courseCode: String): String = "course:$courseCode:structure"
    fun myGrades(courseCode: String): String = "course:$courseCode:my-grades"
    fun itemDetail(courseCode: String, itemId: String): String = "course:$courseCode:item:$itemId"
    fun modulesProgress(courseCode: String): String = "course:$courseCode:modules-progress"
    fun contentPage(courseCode: String, itemId: String): String = "course:$courseCode:content-page:$itemId"
    fun vibeActivity(courseCode: String, itemId: String): String = "course:$courseCode:vibe-activity:$itemId"
    fun courseFiles(courseCode: String, folderId: String?): String =
        if (!folderId.isNullOrEmpty()) "course:$courseCode:files:folder:$folderId"
        else "course:$courseCode:files:root"
    fun plannerSnapshot(): String = "planner:snapshot"
    fun notificationsPage(): String = "notifications:page"
    fun notificationPreferences(): String = "notifications:preferences"
    fun officeHours(courseCode: String): String = "course:$courseCode:office-hours"
    fun liveMeetings(courseCode: String): String = "course:$courseCode:live-meetings"
    fun conferenceSlots(teacherId: String, date: String): String = "parent:conference-slots:$teacherId:$date"
    fun discussionForums(courseCode: String): String = "course:$courseCode:discussion-forums"
    fun discussionThreads(courseCode: String, forumId: String): String =
        "course:$courseCode:discussion-threads:$forumId"
    fun discussionThread(courseCode: String, threadId: String): String =
        "course:$courseCode:discussion-thread:$threadId"
    fun discussionPosts(courseCode: String, threadId: String): String =
        "course:$courseCode:discussion-posts:$threadId"
    fun reviewQueue(): String = "review:queue"
    fun reviewStats(): String = "review:stats"
    fun feedChannels(courseCode: String): String = "course:$courseCode:feed-channels"
    fun feedMessages(courseCode: String, channelId: String): String =
        "course:$courseCode:feed-messages:$channelId"
    fun courseEnrollments(courseCode: String): String = "course:$courseCode:enrollments"
    fun myGroups(courseCode: String): String = "course:$courseCode:my-groups"
    fun groupFeedChannels(courseCode: String, groupId: String): String =
        "course:$courseCode:group:$groupId:feed-channels"
    fun groupFeedMessages(courseCode: String, groupId: String, channelId: String): String =
        "course:$courseCode:group:$groupId:feed-messages:$channelId"
    fun collabDocs(courseCode: String): String = "course:$courseCode:collab-docs"
    fun myPaths(): String = "paths:my"
    fun pathProgress(pathId: String): String = "paths:progress:$pathId"
    fun catalogPaths(query: String): String = "paths:catalog:$query"
    fun catalogCourses(key: String): String = "catalog:courses:$key"
    fun catalogCourseDetail(slug: String): String = "catalog:course:$slug"
    fun catalogCategories(): String = "catalog:categories"
    fun studyStats(): String = "insights:study-stats"
    fun reflectionJournal(): String = "insights:reflection-journal"
    fun coachingTips(): String = "insights:coaching-tips"
    fun readingLog(): String = "reading:log"
    fun libraryBooks(orgId: String, gradeBand: String): String = "reading:library:$orgId:$gradeBand"
    fun credentialsList(): String = "credentials:list"
    fun portfolioList(): String = "portfolio:list"
    fun portfolioDetail(id: String): String = "portfolio:$id"
    fun gamificationProfile(): String = "gamification:profile"
    fun gamificationLeaderboard(courseCode: String): String = "gamification:leaderboard:$courseCode"
    fun advisingNotes(): String = "advising:notes"
    fun degreeProgress(): String = "advising:degree-progress"
    fun evaluationStatus(courseCode: String): String = "evaluation:status:$courseCode"
    fun evaluationResults(courseCode: String): String = "evaluation:results:$courseCode"
}

data class Cached<T>(
    val value: T,
    val fetchedAt: Instant,
) {
    fun isStale(isOnline: Boolean, maxFreshMinutes: Long = 5): Boolean =
        !isOnline || ChronoUnit.MINUTES.between(fetchedAt, Instant.now()) > maxFreshMinutes

    fun lastUpdatedLabel(): String {
        val minutes = ChronoUnit.MINUTES.between(fetchedAt, Instant.now()).coerceAtLeast(0)
        return when {
            minutes < 1 -> "Last updated just now"
            minutes < 60 -> "Last updated $minutes min ago"
            else -> "Last updated ${ChronoUnit.HOURS.between(fetchedAt, Instant.now())} hr ago"
        }
    }
}

enum class OutboxStatus(val userLabel: String) {
    Queued("Saved locally — will sync"),
    Syncing("Syncing…"),
    Synced("Synced"),
    Failed("Sync failed — retry"),
    Conflict("Conflict — review required"),
}

@Serializable
data class OutboxItem(
    val id: String = UUID.randomUUID().toString(),
    val createdAtEpochMs: Long = System.currentTimeMillis(),
    val sequence: Int = 0,
    val method: String,
    val path: String,
    val bodyJson: String? = null,
    val label: String,
    val status: String = OutboxStatus.Queued.name,
    val lastError: String? = null,
) {
    val idempotencyKey: String get() = id

    fun outboxStatus(): OutboxStatus =
        runCatching { OutboxStatus.valueOf(status) }.getOrDefault(OutboxStatus.Queued)
}

object OfflineStorageBudget {
    const val DEFAULT_MAX_BYTES: Long = 50L * 1024 * 1024
}

@Serializable
data class OfflineSyncMetrics(
    val successCount: Int = 0,
    val failureCount: Int = 0,
    val conflictCount: Int = 0,
    val lastSyncEpochMs: Long? = null,
)
