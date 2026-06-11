package com.lextures.android.core.notebook

import java.util.UUID

/** Page-tree helpers over the flat [NotebookPage] list (parity with web `course-notebook-tree`). */
object NotebookTree {
    data class FlatRow(val page: NotebookPage, val depth: Int)

    fun isGroup(page: NotebookPage): Boolean = page.kind == "group"

    fun sortedChildren(pages: List<NotebookPage>, parentId: String?): List<NotebookPage> =
        pages.filter { it.parentId == parentId }.sortedWith(compareBy({ it.sortOrder }, { it.id }))

    /** Depth-first flattening for list rendering (groups followed by their children). */
    fun flatten(pages: List<NotebookPage>): List<FlatRow> {
        val rows = mutableListOf<FlatRow>()
        fun walk(parentId: String?, depth: Int) {
            for (page in sortedChildren(pages, parentId)) {
                rows.add(FlatRow(page, depth))
                walk(page.id, depth + 1)
            }
        }
        walk(null, 0)
        return rows
    }

    /** True if [targetId] is [ancestorId] or nested under it. */
    fun isUnderAncestor(pages: List<NotebookPage>, ancestorId: String, targetId: String): Boolean {
        val map = pages.associateBy { it.id }
        var cur: String? = targetId
        while (cur != null) {
            if (cur == ancestorId) return true
            cur = map[cur]?.parentId
        }
        return false
    }

    private fun nextSortOrder(pages: List<NotebookPage>, parentId: String?): Int =
        (sortedChildren(pages, parentId).maxOfOrNull { it.sortOrder } ?: -1) + 1

    fun addPage(pages: List<NotebookPage>, parentId: String?, title: String = "Untitled"): Pair<List<NotebookPage>, String> {
        val page = NotebookPage(
            id = UUID.randomUUID().toString(),
            title = title,
            parentId = parentId,
            sortOrder = nextSortOrder(pages, parentId),
        )
        return (pages + page) to page.id
    }

    fun addGroup(pages: List<NotebookPage>, parentId: String?, title: String = "Untitled group"): Pair<List<NotebookPage>, String> {
        val group = NotebookPage(
            id = UUID.randomUUID().toString(),
            title = title,
            parentId = parentId,
            sortOrder = nextSortOrder(pages, parentId),
            kind = "group",
        )
        return (pages + group) to group.id
    }

    /** Delete a page or group along with all descendants. */
    fun delete(pages: List<NotebookPage>, pageId: String): List<NotebookPage> {
        val toRemove = mutableSetOf<String>()
        fun walk(id: String) {
            toRemove.add(id)
            pages.filter { it.parentId == id }.forEach { walk(it.id) }
        }
        walk(pageId)
        return pages.filterNot { it.id in toRemove }
    }

    fun rename(pages: List<NotebookPage>, pageId: String, title: String): List<NotebookPage> =
        pages.map { if (it.id == pageId) it.copy(title = title) else it }

    fun updateContent(pages: List<NotebookPage>, pageId: String, contentMd: String): List<NotebookPage> =
        pages.map { if (it.id == pageId) it.copy(contentMd = contentMd) else it }

    /** Move a page or group under [newParentId] (null = top level), appended last. Null on invalid move. */
    fun moveToParent(pages: List<NotebookPage>, pageId: String, newParentId: String?): List<NotebookPage>? {
        if (pageId == newParentId) return null
        if (newParentId != null && isUnderAncestor(pages, ancestorId = pageId, targetId = newParentId)) return null
        if (pages.none { it.id == pageId }) return null
        val order = nextSortOrder(pages, newParentId)
        return pages.map {
            if (it.id == pageId) it.copy(parentId = newParentId, sortOrder = order) else it
        }
    }

    /** Groups the page can be moved into (excludes self and own descendants). */
    fun groupMoveTargets(pages: List<NotebookPage>, pageId: String): List<NotebookPage> =
        pages
            .filter { isGroup(it) && it.id != pageId }
            .filterNot { isUnderAncestor(pages, ancestorId = pageId, targetId = it.id) }
            .sortedBy { pathLabel(pages, it.id) }

    fun pathLabel(pages: List<NotebookPage>, pageId: String): String {
        val map = pages.associateBy { it.id }
        val parts = mutableListOf<String>()
        var cur: String? = pageId
        while (cur != null) {
            val row = map[cur] ?: break
            parts.add(0, row.title.ifBlank { "Untitled" })
            cur = row.parentId
        }
        return parts.joinToString(" / ")
    }
}
