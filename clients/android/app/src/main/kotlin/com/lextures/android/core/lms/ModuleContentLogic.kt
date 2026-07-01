package com.lextures.android.core.lms

data class ModuleGroup(
    val id: String,
    val title: String,
    val items: List<CourseStructureItem>,
)

enum class ModuleItemDestination {
    ContentPage,
    Quiz,
    Assignment,
    ExternalLink,
    WebContent,
    Interactive,
    VibeActivity,
    LibraryResource,
    File,
    Unsupported,
}

object ModuleContentLogic {
    fun buildModuleGroups(items: List<CourseStructureItem>): List<ModuleGroup> {
        val modules = items.filter { it.isModule }.sortedBy { it.sortOrder }
        val children = items.filter { !it.isModule && it.parentId != null }.groupBy { it.parentId }
        val grouped = modules.map { module ->
            ModuleGroup(module.id, module.title, (children[module.id] ?: emptyList()).sortedBy { it.sortOrder })
        }
        val orphans = items
            .filter { !it.isModule && it.parentId == null && it.kind != "heading" }
            .sortedBy { it.sortOrder }
        return if (orphans.isEmpty()) grouped else grouped + ModuleGroup("__orphans__", "Other items", orphans)
    }

    fun destination(kind: String): ModuleItemDestination = when (kind) {
        "content_page" -> ModuleItemDestination.ContentPage
        "quiz" -> ModuleItemDestination.Quiz
        "assignment" -> ModuleItemDestination.Assignment
        "external_link" -> ModuleItemDestination.ExternalLink
        "library_resource" -> ModuleItemDestination.LibraryResource
        "textbook_resource" -> ModuleItemDestination.WebContent
        "h5p", "scorm", "lti_link" -> ModuleItemDestination.Interactive
        "vibe_activity" -> ModuleItemDestination.VibeActivity
        "file", "file_item" -> ModuleItemDestination.File
        else -> ModuleItemDestination.Unsupported
    }

    fun isNavigable(kind: String): Boolean = destination(kind) != ModuleItemDestination.Unsupported

    fun itemLockState(progress: ModulesProgressSnapshot?, itemId: String): ItemLockState? {
        progress?.modules.orEmpty().forEach { module ->
            module.items.orEmpty().firstOrNull { it.itemId == itemId }?.let { return it }
        }
        return null
    }

    fun moduleLockState(progress: ModulesProgressSnapshot?, moduleId: String): ModuleLockState? =
        progress?.modules.orEmpty().firstOrNull { it.moduleId == moduleId }

    fun isLocked(progress: ModulesProgressSnapshot?, itemId: String): Boolean =
        itemLockState(progress, itemId)?.locked == true

    fun isComplete(progress: ModulesProgressSnapshot?, itemId: String): Boolean =
        itemLockState(progress, itemId)?.complete == true
}
