package com.lextures.android.core.lms

import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobileRoleContext
import java.net.URLEncoder
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.nio.charset.StandardCharsets

/** Pure course-checklist helpers (CC.9 FR-23). No networking / disk I/O. */
object CourseChecklistLogic {
    const val SUMMARY_MEMO_SECONDS = 60L
    const val HIGHLIGHT_DURATION_MS = 4_000L
    const val MAX_DISMISS_NOTE_LENGTH = 500
    const val BADGE_CAP = 99
    const val OFFLINE_CACHE_KEY_PREFIX = "checklist:"
    const val RATE_LIMITED_MESSAGE = "Just checked — try again in a moment"
    const val OFFLINE_MESSAGE = "Connect to see your checklist"

    fun shouldShowWorkspaceSection(
        viewerIsStaff: Boolean,
        roleContext: MobileRoleContext,
    ): Boolean = viewerIsStaff && roleContext == MobileRoleContext.Teaching

    fun normalizeStatus(raw: String): ChecklistStatus = when (raw) {
        "done" -> ChecklistStatus.Done
        "todo" -> ChecklistStatus.Todo
        "in_progress" -> ChecklistStatus.InProgress
        "not_applicable" -> ChecklistStatus.NotApplicable
        else -> ChecklistStatus.Unknown
    }

    fun isOutstanding(status: String): Boolean = when (normalizeStatus(status)) {
        ChecklistStatus.Todo, ChecklistStatus.InProgress, ChecklistStatus.Unknown -> true
        ChecklistStatus.Done, ChecklistStatus.NotApplicable -> false
    }

    fun isDone(status: String): Boolean = normalizeStatus(status) == ChecklistStatus.Done

    fun accessibilityStatusValue(status: String): String = when (normalizeStatus(status)) {
        ChecklistStatus.Done -> "Completed"
        ChecklistStatus.Todo -> "To do"
        ChecklistStatus.InProgress -> "In progress"
        ChecklistStatus.NotApplicable -> "Not applicable"
        ChecklistStatus.Unknown -> "Unknown"
    }

    data class BadgePresentation(
        val visible: Boolean,
        val text: String,
        val count: Int,
        val accessibilityLabel: String,
    )

    fun badgePresentation(outstandingEssential: Int): BadgePresentation {
        if (outstandingEssential <= 0) {
            return BadgePresentation(false, "", 0, "")
        }
        val text = if (outstandingEssential > BADGE_CAP) "99+" else outstandingEssential.toString()
        val label = if (outstandingEssential == 1) {
            "1 checklist item needs attention"
        } else {
            "$outstandingEssential checklist items need attention"
        }
        return BadgePresentation(true, text, outstandingEssential, label)
    }

    fun progressFraction(done: Int, total: Int): Double {
        if (total <= 0) return 0.0
        return (done.toDouble() / total.toDouble()).coerceIn(0.0, 1.0)
    }

    fun progressLabel(done: Int, total: Int): String = "$done of $total done"

    fun outstandingCount(category: ChecklistCategory): Int =
        category.items.count { isOutstanding(it.status) }

    fun visibleItems(category: ChecklistCategory, showCompleted: Boolean): List<ChecklistItem> =
        if (showCompleted) category.items else category.items.filter { !isDone(it.status) }

    enum class TargetKind { Native, Web, Unresolved }

    enum class NativeDestination(val wire: String) {
        CourseSettings("CourseSettings"),
        Syllabus("Syllabus"),
        Modules("Modules"),
        CourseFeed("CourseFeed"),
        Enrollments("Enrollments"),
        Discussions("Discussions"),
        OfficeHours("OfficeHours"),
        Groups("Groups"),
        Files("Files"),
        WebOnly("web-only"),
        ;

        companion object {
            fun fromWire(value: String): NativeDestination? =
                entries.firstOrNull { it.wire == value }
        }
    }

    data class ResolvedTarget(
        val kind: TargetKind,
        val native: NativeDestination? = null,
        val workspaceSection: CourseWorkspaceSection? = null,
        val webPath: String? = null,
        val focusAnchor: String? = null,
        val focusEntity: String? = null,
    )

    fun loadTargetTable(json: String): Map<String, String> {
        // Avoid org.json so JVM unit tests work without Android stubs.
        return runCatching {
            val root = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }
                .parseToJsonElement(json)
                .jsonObject
            val arr = root["targets"]?.jsonArray ?: return emptyMap()
            buildMap {
                for (el in arr) {
                    val row = el.jsonObject
                    val id = row["id"]?.jsonPrimitive?.contentOrNull ?: continue
                    val dest = row["android"]?.jsonPrimitive?.contentOrNull
                        ?: row["ios"]?.jsonPrimitive?.contentOrNull
                        ?: "web-only"
                    put(id, dest)
                }
            }
        }.getOrDefault(emptyMap())
    }

    fun resolveTarget(
        target: ChecklistNavTarget?,
        courseCode: String,
        table: Map<String, String>,
    ): ResolvedTarget {
        if (target == null) {
            return ResolvedTarget(TargetKind.Unresolved)
        }
        val anchorKey = normalizeAnchorKey(target.anchor)
        val mapped = anchorKey?.let { table[it] }
        val native = mapped?.let { NativeDestination.fromWire(it) }

        if (native != null && native != NativeDestination.WebOnly) {
            return ResolvedTarget(
                kind = TargetKind.Native,
                native = native,
                workspaceSection = workspaceSection(native),
                focusAnchor = target.anchor,
                focusEntity = target.entityKey,
            )
        }

        val path = webPath(target.route, courseCode, target.anchor, target.entityKey)
        if (mapped == NativeDestination.WebOnly.wire) {
            return ResolvedTarget(
                kind = TargetKind.Web,
                native = NativeDestination.WebOnly,
                workspaceSection = fallbackSection(target.route),
                webPath = path,
                focusAnchor = target.anchor,
                focusEntity = target.entityKey,
            )
        }
        return ResolvedTarget(
            kind = TargetKind.Unresolved,
            workspaceSection = fallbackSection(target.route) ?: CourseWorkspaceSection.Overview,
            webPath = path,
        )
    }

    fun normalizeAnchorKey(anchor: String?): String? {
        if (anchor.isNullOrBlank()) return null
        val colon = anchor.indexOf(':')
        return if (colon >= 0) anchor.substring(0, colon) else anchor
    }

    fun workspaceSection(destination: NativeDestination): CourseWorkspaceSection? = when (destination) {
        NativeDestination.CourseSettings -> CourseWorkspaceSection.Settings
        NativeDestination.Syllabus -> CourseWorkspaceSection.Overview
        NativeDestination.Modules -> CourseWorkspaceSection.Modules
        NativeDestination.CourseFeed -> CourseWorkspaceSection.Feed
        NativeDestination.Enrollments -> CourseWorkspaceSection.People
        NativeDestination.Discussions -> CourseWorkspaceSection.Discussions
        NativeDestination.OfficeHours -> CourseWorkspaceSection.OfficeHours
        NativeDestination.Groups -> CourseWorkspaceSection.Groups
        NativeDestination.Files -> CourseWorkspaceSection.Files
        NativeDestination.WebOnly -> null
    }

    fun fallbackSection(route: String): CourseWorkspaceSection? {
        val lower = route.lowercase()
        return when {
            "settings" in lower -> CourseWorkspaceSection.Settings
            "modules" in lower -> CourseWorkspaceSection.Modules
            "syllabus" in lower || "overview" in lower -> CourseWorkspaceSection.Overview
            "enroll" in lower || "people" in lower -> CourseWorkspaceSection.People
            "feed" in lower || "announce" in lower -> CourseWorkspaceSection.Feed
            "discussion" in lower -> CourseWorkspaceSection.Discussions
            "office" in lower -> CourseWorkspaceSection.OfficeHours
            "group" in lower -> CourseWorkspaceSection.Groups
            "file" in lower -> CourseWorkspaceSection.Files
            "grading" in lower -> CourseWorkspaceSection.Grading
            else -> CourseWorkspaceSection.Overview
        }
    }

    fun webPath(
        route: String,
        courseCode: String,
        anchor: String?,
        entityKey: String?,
    ): String {
        var path = route.replace("{courseCode}", courseCode)
        if (!path.startsWith("/")) path = "/$path"
        val params = buildList {
            if (!anchor.isNullOrBlank()) add("focus=${enc(anchor)}")
            if (!entityKey.isNullOrBlank()) add("focusEntity=${enc(entityKey)}")
        }
        return if (params.isEmpty()) path else "$path?${params.joinToString("&")}"
    }

    private fun enc(value: String): String =
        URLEncoder.encode(value, StandardCharsets.UTF_8).replace("+", "%20")

    data class SummaryCacheEntry(
        val summary: CourseChecklistSummary,
        val catalogVersion: String? = null,
        val fetchedAtMs: Long,
    )

    fun isSummaryFresh(entry: SummaryCacheEntry?, nowMs: Long = System.currentTimeMillis()): Boolean {
        if (entry == null) return false
        return nowMs - entry.fetchedAtMs < SUMMARY_MEMO_SECONDS * 1000
    }

    fun shouldInvalidate(cachedCatalogVersion: String?, responseCatalogVersion: String): Boolean {
        if (cachedCatalogVersion.isNullOrBlank()) return false
        return cachedCatalogVersion != responseCatalogVersion
    }

    fun clampedNote(note: String?): String {
        val trimmed = note?.trim().orEmpty()
        return if (trimmed.length <= MAX_DISMISS_NOTE_LENGTH) {
            trimmed
        } else {
            trimmed.take(MAX_DISMISS_NOTE_LENGTH)
        }
    }

    data class ItemPresentation(
        val id: String,
        val title: String,
        val status: String,
        val accessibilityValue: String,
        val isDone: Boolean,
        val isOutstanding: Boolean,
        val targetKind: String,
    )

    fun presentItem(
        item: ChecklistItem,
        courseCode: String,
        table: Map<String, String>,
    ): ItemPresentation {
        val resolved = resolveTarget(item.target, courseCode, table)
        val kind = when (resolved.kind) {
            TargetKind.Native -> "native"
            TargetKind.Web -> "web"
            TargetKind.Unresolved -> "unresolved"
        }
        return ItemPresentation(
            id = item.id,
            title = item.title,
            status = when (normalizeStatus(item.status)) {
                ChecklistStatus.Done -> "done"
                ChecklistStatus.Todo -> "todo"
                ChecklistStatus.InProgress -> "in_progress"
                ChecklistStatus.NotApplicable -> "not_applicable"
                ChecklistStatus.Unknown -> "unknown"
            },
            accessibilityValue = accessibilityStatusValue(item.status),
            isDone = isDone(item.status),
            isOutstanding = isOutstanding(item.status),
            targetKind = kind,
        )
    }

    fun presentChecklist(
        checklist: CourseChecklist,
        table: Map<String, String>,
    ): List<ItemPresentation> {
        val out = mutableListOf<ItemPresentation>()
        for (cat in checklist.categories) {
            for (item in cat.items) {
                out += presentItem(item, checklist.courseCode, table)
            }
        }
        for (item in checklist.dismissed) {
            out += presentItem(item, checklist.courseCode, table)
        }
        return out
    }
}
