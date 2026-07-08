package com.lextures.android.core.lms

import kotlin.math.roundToInt

/** Course accessibility / alt-text review helpers (M13.8). */
object CourseAccessibilityReviewLogic {
    const val DECORATIVE_TITLE_MARKER = "lex-decorative"
    const val PAGE_SIZE = 20

    private val markdownImagePattern =
        Regex("""!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)""")

    data class MarkdownImageRef(
        val globalIndex: Int,
        val alt: String,
        val src: String,
        val title: String?,
        val decorative: Boolean,
        val hasValidAlt: Boolean,
        val line: Int,
    )

    data class ImageAltDraft(
        val alt: String,
        val decorative: Boolean,
    )

    data class AltTextUpdate(
        val imageIndex: Int,
        val alt: String,
        val decorative: Boolean,
    )

    fun cacheKey(courseCode: String): String = "course:$courseCode:accessibility"

    fun saveMarkdownIdempotencyKey(courseCode: String, itemId: String, kind: String): String =
        "course-accessibility:$courseCode:$kind:$itemId:markdown"

    fun markdownPatchPath(courseCode: String, itemId: String, kind: String): String? = when (kind) {
        "content_page" -> "/api/v1/courses/$courseCode/content-pages/$itemId"
        "assignment" -> "/api/v1/courses/$courseCode/assignments/$itemId"
        else -> null
    }

    fun supportsInlineEdit(kind: String): Boolean = kind == "content_page" || kind == "assignment"

    fun coveragePercent(withAlt: Int, total: Int): Int {
        if (total <= 0) return 100
        return ((withAlt.toDouble() / total.toDouble()) * 100).roundToInt()
    }

    fun paginatedUncoveredItems(items: List<UncoveredAccessibilityItem>, page: Int): List<UncoveredAccessibilityItem> {
        val end = minOf(items.size, maxOf(0, page + 1) * PAGE_SIZE)
        return items.take(end)
    }

    fun hasMorePages(items: List<UncoveredAccessibilityItem>, page: Int): Boolean =
        items.size > (page + 1) * PAGE_SIZE

    fun scanMarkdownImages(markdown: String): List<MarkdownImageRef> {
        val refs = mutableListOf<MarkdownImageRef>()
        var globalIndex = 0
        markdown.split("\n").forEachIndexed { lineOffset, line ->
            var searchStart = 0
            while (searchStart < line.length) {
                val match = markdownImagePattern.find(line, searchStart) ?: break
                val alt = match.groupValues[1]
                val src = match.groupValues[2]
                val title = match.groups[3]?.value
                val decorative = title == DECORATIVE_TITLE_MARKER
                val hasValidAlt = decorative || alt.trim().isNotEmpty()
                refs += MarkdownImageRef(
                    globalIndex = globalIndex,
                    alt = alt,
                    src = src,
                    title = title,
                    decorative = decorative,
                    hasValidAlt = hasValidAlt,
                    line = lineOffset + 1,
                )
                globalIndex += 1
                searchStart = match.range.last + 1
            }
        }
        return refs
    }

    fun missingImages(markdown: String): List<MarkdownImageRef> =
        scanMarkdownImages(markdown).filter { !it.hasValidAlt }

    fun applyAltTextUpdate(
        markdown: String,
        imageIndex: Int,
        alt: String,
        decorative: Boolean,
    ): String? = applyAltTextUpdates(markdown, listOf(AltTextUpdate(imageIndex, alt, decorative)))

    fun applyAltTextUpdates(markdown: String, updates: List<AltTextUpdate>): String? {
        if (updates.isEmpty()) return markdown
        val matches = markdownImagePattern.findAll(markdown).toList()
        var result = markdown
        for (update in updates.sortedByDescending { it.imageIndex }) {
            if (update.imageIndex !in matches.indices) return null
            val match = matches[update.imageIndex]
            val src = match.groupValues[2]
            val replacement = if (update.decorative) {
                "![]($src \"$DECORATIVE_TITLE_MARKER\")"
            } else {
                val trimmedAlt = update.alt.trim().replace("]", "\\]")
                "![$trimmedAlt]($src)"
            }
            result = result.replaceRange(match.range, replacement)
        }
        return result
    }

    fun kindLabelRes(kind: String): Int = when (kind) {
        "assignment" -> com.lextures.android.R.string.mobile_courseSettings_accessibility_kind_assignment
        "content_page" -> com.lextures.android.R.string.mobile_courseSettings_accessibility_kind_contentPage
        else -> com.lextures.android.R.string.mobile_courseSettings_accessibility_kind_other
    }

    fun drafts(images: List<MarkdownImageRef>): Map<Int, ImageAltDraft> =
        images.associate { it.globalIndex to ImageAltDraft(it.alt, it.decorative) }

    fun pendingUpdates(
        images: List<MarkdownImageRef>,
        drafts: Map<Int, ImageAltDraft>,
    ): List<AltTextUpdate> = images.mapNotNull { image ->
        val draft = drafts[image.globalIndex] ?: return@mapNotNull null
        val trimmedAlt = draft.alt.trim()
        val resolved = draft.decorative || trimmedAlt.isNotEmpty()
        if (!resolved) return@mapNotNull null
        if (draft.decorative == image.decorative && trimmedAlt == image.alt.trim()) return@mapNotNull null
        AltTextUpdate(image.globalIndex, trimmedAlt, draft.decorative)
    }
}
