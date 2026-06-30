package com.lextures.android.core.lms

data class RequirementRow(
    val id: String,
    val title: String,
    val detail: String? = null,
    val met: Boolean,
    val navigateItemId: String? = null,
)

data class RequirementsSummary(
    val rows: List<RequirementRow>,
    val nextRequiredItemId: String? = null,
) {
    val metCount: Int get() = rows.count { it.met }
    val totalCount: Int get() = rows.size
}

object RequirementsLogic {
    fun findItem(id: String, groups: List<ModuleGroup>): CourseStructureItem? =
        groups.flatMap { it.items }.firstOrNull { it.id == id }

    fun parentModule(itemId: String, groups: List<ModuleGroup>): ModuleGroup? =
        groups.firstOrNull { group -> group.items.any { it.id == itemId } }

    fun buildRequirements(
        targetItem: CourseStructureItem,
        groups: List<ModuleGroup>,
        progress: ModulesProgressSnapshot?,
    ): RequirementsSummary {
        val rows = mutableListOf<RequirementRow>()
        val seen = mutableSetOf<String>()

        fun appendRow(row: RequirementRow) {
            if (seen.add(row.id)) rows += row
        }

        val parent = parentModule(targetItem.id, groups)
        val moduleState = parent?.let { ModuleContentLogic.moduleLockState(progress, it.id) }
        val itemState = ModuleContentLogic.itemLockState(progress, targetItem.id)

        if (moduleState?.locked == true && moduleState.reason != null) {
            appendModuleRequirements(moduleState.reason!!, groups, progress, ::appendRow)
        } else if (itemState?.locked == true) {
            appendItemRequirements(
                targetItem = targetItem,
                reason = itemState.reason,
                parent = parent,
                groups = groups,
                progress = progress,
                appendRow = ::appendRow,
            )
        }

        if (rows.isEmpty()) {
            appendRow(
                RequirementRow(
                    id = "fallback",
                    title = targetItem.title,
                    detail = itemState?.reason?.message ?: moduleState?.reason?.message,
                    met = false,
                    navigateItemId = navigableItemId(itemState?.reason?.itemId, groups),
                ),
            )
        }

        val nextRequiredItemId = rows.firstOrNull { !it.met && it.navigateItemId != null }?.navigateItemId
            ?: navigableItemId(itemState?.reason?.itemId, groups)

        return RequirementsSummary(rows = rows, nextRequiredItemId = nextRequiredItemId)
    }

    private fun appendModuleRequirements(
        reason: LockReason,
        groups: List<ModuleGroup>,
        progress: ModulesProgressSnapshot?,
        appendRow: (RequirementRow) -> Unit,
    ) {
        when (reason.code) {
            "module_prerequisite" -> {
                val prereq = prerequisiteModule(reason, groups, progress)
                if (prereq != null) {
                    appendRow(
                        RequirementRow(
                            id = "module:${prereq.moduleId}",
                            title = prereq.title,
                            detail = reason.message,
                            met = prereq.complete,
                            navigateItemId = firstIncompleteNavigableItem(prereq.moduleId, groups, progress)?.id,
                        ),
                    )
                    groups.firstOrNull { it.id == prereq.moduleId }?.items.orEmpty()
                        .filter { ModuleContentLogic.isNavigable(it.kind) }
                        .forEach { item ->
                            val complete = ModuleContentLogic.isComplete(progress, item.id)
                            appendRow(
                                RequirementRow(
                                    id = "item:${item.id}",
                                    title = item.title,
                                    detail = if (complete) null else null,
                                    met = complete,
                                    navigateItemId = if (complete) null else item.id,
                                ),
                            )
                        }
                } else {
                    appendRow(
                        RequirementRow(
                            id = "module-prereq",
                            title = reason.title ?: reason.message,
                            detail = reason.message,
                            met = false,
                        ),
                    )
                }
            }
            "unlock_date" -> appendRow(
                RequirementRow(id = "unlock-date", title = reason.message, met = false),
            )
            else -> appendRow(
                RequirementRow(
                    id = "module-lock",
                    title = reason.title ?: reason.message,
                    detail = reason.message,
                    met = false,
                ),
            )
        }
    }

    private fun appendItemRequirements(
        targetItem: CourseStructureItem,
        reason: LockReason?,
        parent: ModuleGroup?,
        groups: List<ModuleGroup>,
        progress: ModulesProgressSnapshot?,
        appendRow: (RequirementRow) -> Unit,
    ) {
        if (reason?.code == "sequential_order" && parent != null) {
            for (item in parent.items) {
                if (item.id == targetItem.id) break
                if (!ModuleContentLogic.isNavigable(item.kind)) continue
                val complete = ModuleContentLogic.isComplete(progress, item.id)
                appendRow(
                    RequirementRow(
                        id = "item:${item.id}",
                        title = item.title,
                        detail = null,
                        met = complete,
                        navigateItemId = if (complete) null else item.id,
                    ),
                )
                if (item.id == reason.itemId) break
            }
        }

        if (reason?.code != "sequential_order") {
            var currentReason = reason
            var guardCount = 0
            while (currentReason != null && guardCount < 12) {
                guardCount += 1
                val itemId = currentReason.itemId?.takeIf { it.isNotEmpty() }
                if (itemId != null) {
                    val item = findItem(itemId, groups)
                    val complete = ModuleContentLogic.isComplete(progress, itemId)
                    appendRow(
                        RequirementRow(
                            id = "item:$itemId",
                            title = currentReason.title ?: item?.title ?: currentReason.message,
                            detail = currentReason.message,
                            met = complete,
                            navigateItemId = if (complete) null else navigableItemId(itemId, groups),
                        ),
                    )
                    if (complete) break
                    currentReason = ModuleContentLogic.itemLockState(progress, itemId)?.reason
                } else if (currentReason.code != "sequential_order") {
                    appendRow(
                        RequirementRow(
                            id = "reason:${currentReason.code}",
                            title = currentReason.title ?: currentReason.message,
                            detail = if (currentReason.title == null) null else currentReason.message,
                            met = false,
                            navigateItemId = navigableItemId(currentReason.itemId, groups),
                        ),
                    )
                    break
                } else {
                    break
                }
            }
        }
    }

    private fun prerequisiteModule(
        reason: LockReason,
        groups: List<ModuleGroup>,
        progress: ModulesProgressSnapshot?,
    ): ModuleLockState? {
        reason.title?.let { title ->
            progress?.modules?.firstOrNull { it.title == title }?.let { return it }
        }
        return progress?.modules?.firstOrNull { !it.complete }
    }

    private fun firstIncompleteNavigableItem(
        moduleId: String,
        groups: List<ModuleGroup>,
        progress: ModulesProgressSnapshot?,
    ): CourseStructureItem? =
        groups.firstOrNull { it.id == moduleId }?.items?.firstOrNull { item ->
            ModuleContentLogic.isNavigable(item.kind) && !ModuleContentLogic.isComplete(progress, item.id)
        }

    private fun navigableItemId(itemId: String?, groups: List<ModuleGroup>): String? {
        if (itemId.isNullOrEmpty()) return null
        val item = findItem(itemId, groups) ?: return null
        return if (ModuleContentLogic.isNavigable(item.kind)) itemId else null
    }
}
