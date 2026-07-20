package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.util.Locale
import kotlin.math.min
import kotlin.math.pow

/** MOB.8 helpers — templates, export polling, present ordering, governance gating. */
object BoardsAdvancedLogic {
    fun isAdvancedEnabled(courseEnabled: Boolean, features: MobilePlatformFeatures): Boolean =
        courseEnabled && features.ffMobileBoardsAdvanced

    fun canUseTemplates(
        courseEnabled: Boolean,
        features: MobilePlatformFeatures,
        canCreate: Boolean,
    ): Boolean = isAdvancedEnabled(courseEnabled, features) && canCreate

    fun canExportOrPresent(
        courseEnabled: Boolean,
        features: MobilePlatformFeatures,
        canManage: Boolean,
    ): Boolean = isAdvancedEnabled(courseEnabled, features) && canManage

    fun canViewBoardAnalytics(
        courseEnabled: Boolean,
        features: MobilePlatformFeatures,
        canManage: Boolean,
    ): Boolean = isAdvancedEnabled(courseEnabled, features) && canManage

    fun filterTemplates(
        templates: List<BoardTemplate>,
        scope: BoardTemplateScope?,
        query: String,
    ): List<BoardTemplate> {
        val needle = query.trim().lowercase(Locale.US)
        return templates.filter { template ->
            if (scope != null && !template.scope.equals(scope.apiValue, ignoreCase = true)) return@filter false
            if (needle.isEmpty()) return@filter true
            val hay = listOf(template.title, template.description, template.tags.joinToString(" "))
                .joinToString(" ")
                .lowercase(Locale.US)
            hay.contains(needle)
        }
    }

    fun pollDelaySeconds(attempt: Int, cap: Double = 8.0): Double {
        val base = 0.5 * 2.0.pow(maxOf(0, attempt).toDouble())
        return min(cap, base)
    }

    fun isExportTerminal(status: String): Boolean {
        val s = status.lowercase(Locale.US)
        return s == "done" || s == "failed"
    }

    fun isCopyTerminal(status: String): Boolean {
        val s = status.lowercase(Locale.US)
        return s == "completed" || s == "failed"
    }

    fun exportFileExtension(format: BoardExportFormat): String = when (format) {
        BoardExportFormat.Pdf -> "pdf"
        BoardExportFormat.Csv -> "csv"
        BoardExportFormat.Image -> "png"
    }

    fun exportMimeType(format: BoardExportFormat): String = when (format) {
        BoardExportFormat.Pdf -> "application/pdf"
        BoardExportFormat.Csv -> "text/csv"
        BoardExportFormat.Image -> "image/png"
    }

    fun orderedPostsForPresent(posts: List<BoardPost>, sections: List<BoardSection>): List<BoardPost> {
        val secOrder = sections.associate { it.id to it.sortIndex }
        return posts.sortedWith(
            compareBy<BoardPost> { post ->
                post.sectionId?.let { secOrder[it] } ?: Double.POSITIVE_INFINITY
            }.thenBy { it.sortIndex },
        )
    }

    fun postBodyText(post: BoardPost): String {
        val text = post.body?.text?.trim().orEmpty()
        if (text.isNotEmpty()) return text
        val html = post.body?.html.orEmpty()
        if (html.isNotEmpty()) {
            return html
                .replace(Regex("<[^>]+>"), "")
                .replace("&nbsp;", " ")
                .replace("&amp;", "&")
                .replace("&lt;", "<")
                .replace("&gt;", ">")
                .trim()
        }
        return post.title.trim()
    }

    fun formatStorageBytes(bytes: Long): String = BoardsLogic.formatFileSize(bytes)

    fun parseBoardCapDraft(raw: String): Int? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null
        return trimmed.toIntOrNull()
    }
}

object BoardsAdvancedObservability {
    private val counters = mutableMapOf<String, Int>()
    private val lock = Any()

    fun record(event: String, attributes: Map<String, String> = emptyMap()) {
        synchronized(lock) {
            val key = if (attributes.isEmpty()) {
                event
            } else {
                event + "|" + attributes.keys.sorted().joinToString(",") { "$it=${attributes[it]}" }
            }
            counters[key] = (counters[key] ?: 0) + 1
        }
    }

    fun count(event: String): Int = synchronized(lock) {
        counters.filter { it.key == event || it.key.startsWith("$event|") }.values.sum()
    }

    fun resetForTests() {
        synchronized(lock) { counters.clear() }
    }
}
