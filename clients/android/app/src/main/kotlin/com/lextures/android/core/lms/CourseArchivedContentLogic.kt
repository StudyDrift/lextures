package com.lextures.android.core.lms

/** Archived module item list + restore helpers (M13.12). */
object CourseArchivedContentLogic {
    private val restorableKinds = setOf(
        "heading",
        "content_page",
        "assignment",
        "quiz",
        "external_link",
    )

    data class ArchivedContentRow(
        val id: String,
        val title: String,
        val kind: String,
        val kindLabelKey: String,
        val moduleTitle: String,
        val archivedAt: String?,
    )

    fun canViewArchivedContent(courseCode: String, permissions: List<String>): Boolean =
        CourseSettingsLogic.canManageCourse(courseCode, permissions)

    fun cacheKeyArchivedStructure(courseCode: String): String = "course:$courseCode:archived-structure"

    fun moduleTitleById(items: List<CourseStructureItem>): Map<String, String> =
        items
            .asSequence()
            .filter { it.kind == "module" && it.parentId == null }
            .associate { it.id to it.title }

    fun archivedRows(items: List<CourseStructureItem>): List<ArchivedContentRow> {
        val modules = moduleTitleById(items)
        return items
            .asSequence()
            .filter { item ->
                item.archived == true &&
                    item.parentId != null &&
                    item.kind in restorableKinds
            }
            .sortedWith(
                compareBy<CourseStructureItem> { it.sortOrder }
                    .thenBy(String.CASE_INSENSITIVE_ORDER) { it.title },
            )
            .map { item ->
                ArchivedContentRow(
                    id = item.id,
                    title = item.title.trim().ifEmpty { "—" },
                    kind = item.kind,
                    kindLabelKey = kindLabelKey(item.kind),
                    moduleTitle = item.parentId?.let(modules::get) ?: "—",
                    archivedAt = item.updatedAt,
                )
            }
            .toList()
    }

    fun itemsAfterRestore(items: List<CourseStructureItem>, removedId: String): List<CourseStructureItem> =
        items.filter { it.id != removedId }

    fun kindLabelKey(kind: String): String = when (kind) {
        "heading" -> "mobile.courseSettings.archivedContent.kind.heading"
        "content_page" -> "mobile.courseSettings.archivedContent.kind.contentPage"
        "assignment" -> "mobile.courseSettings.archivedContent.kind.assignment"
        "quiz" -> "mobile.courseSettings.archivedContent.kind.quiz"
        "external_link" -> "mobile.courseSettings.archivedContent.kind.externalLink"
        else -> "mobile.courseSettings.archivedContent.kind.other"
    }

    fun formatArchivedAt(raw: String?): String {
        val formatted = LmsDates.shortDateTime(raw)
        return formatted.ifEmpty { "—" }
    }

    fun userFacingError(error: Throwable): String = error.message ?: "generic"
}
