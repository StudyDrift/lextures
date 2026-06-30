import Foundation

struct RequirementRow: Identifiable, Hashable {
    let id: String
    let title: String
    let detail: String?
    let met: Bool
    let navigateItemId: String?
}

struct RequirementsSummary: Hashable {
    let rows: [RequirementRow]
    let nextRequiredItemId: String?

    var metCount: Int { rows.filter(\.met).count }
    var totalCount: Int { rows.count }
}

enum RequirementsLogic {
    static func findItem(id: String, in groups: [ModuleGroup]) -> CourseStructureItem? {
        for group in groups {
            if let item = group.items.first(where: { $0.id == id }) {
                return item
            }
        }
        return nil
    }

    static func parentModule(for itemId: String, in groups: [ModuleGroup]) -> ModuleGroup? {
        groups.first { group in group.items.contains { $0.id == itemId } }
    }

    static func buildRequirements(
        for targetItem: CourseStructureItem,
        groups: [ModuleGroup],
        progress: ModulesProgressSnapshot?
    ) -> RequirementsSummary {
        var rows: [RequirementRow] = []
        var seen = Set<String>()

        func appendRow(_ row: RequirementRow) {
            guard !seen.contains(row.id) else { return }
            seen.insert(row.id)
            rows.append(row)
        }

        let parent = parentModule(for: targetItem.id, in: groups)
        let moduleState = parent.flatMap { ModuleContentLogic.moduleLockState(in: progress, moduleId: $0.id) }
        let itemState = ModuleContentLogic.itemLockState(in: progress, itemId: targetItem.id)

        if moduleState?.locked == true, let reason = moduleState?.reason {
            appendModuleRequirements(reason: reason, groups: groups, progress: progress, appendRow: appendRow)
        } else if itemState?.locked == true {
            appendItemRequirements(
                targetItem: targetItem,
                reason: itemState?.reason,
                parent: parent,
                groups: groups,
                progress: progress,
                appendRow: appendRow
            )
        }

        if rows.isEmpty {
            appendRow(
                RequirementRow(
                    id: "fallback",
                    title: targetItem.title,
                    detail: itemState?.reason?.message ?? moduleState?.reason?.message,
                    met: false,
                    navigateItemId: navigableItemId(itemState?.reason?.itemId, groups: groups)
                )
            )
        }

        let nextRequiredItemId =
            rows.first(where: { !$0.met && $0.navigateItemId != nil })?.navigateItemId
            ?? navigableItemId(itemState?.reason?.itemId, groups: groups)

        return RequirementsSummary(rows: rows, nextRequiredItemId: nextRequiredItemId)
    }

    private static func appendModuleRequirements(
        reason: LockReason,
        groups: [ModuleGroup],
        progress: ModulesProgressSnapshot?,
        appendRow: (RequirementRow) -> Void
    ) {
        switch reason.code {
        case "module_prerequisite":
            if let prereq = prerequisiteModule(for: reason, groups: groups, progress: progress) {
                appendRow(
                    RequirementRow(
                        id: "module:\(prereq.moduleId)",
                        title: prereq.title,
                        detail: reason.message,
                        met: prereq.complete,
                        navigateItemId: firstIncompleteNavigableItem(
                            moduleId: prereq.moduleId,
                            groups: groups,
                            progress: progress
                        )?.id
                    )
                )
                if let group = groups.first(where: { $0.id == prereq.moduleId }) {
                    for item in group.items where ModuleContentLogic.isNavigable(item.kind) {
                        let complete = ModuleContentLogic.isComplete(in: progress, itemId: item.id)
                        appendRow(
                            RequirementRow(
                                id: "item:\(item.id)",
                                title: item.title,
                                detail: complete ? nil : nil,
                                met: complete,
                                navigateItemId: complete ? nil : item.id
                            )
                        )
                    }
                }
            } else {
                appendRow(
                    RequirementRow(
                        id: "module-prereq",
                        title: reason.title ?? reason.message,
                        detail: reason.message,
                        met: false,
                        navigateItemId: nil
                    )
                )
            }
        case "unlock_date":
            appendRow(
                RequirementRow(
                    id: "unlock-date",
                    title: reason.message,
                    detail: nil,
                    met: false,
                    navigateItemId: nil
                )
            )
        default:
            appendRow(
                RequirementRow(
                    id: "module-lock",
                    title: reason.title ?? reason.message,
                    detail: reason.message,
                    met: false,
                    navigateItemId: nil
                )
            )
        }
    }

    private static func appendItemRequirements(
        targetItem: CourseStructureItem,
        reason: LockReason?,
        parent: ModuleGroup?,
        groups: [ModuleGroup],
        progress: ModulesProgressSnapshot?,
        appendRow: (RequirementRow) -> Void
    ) {
        if reason?.code == "sequential_order", let parent {
            for item in parent.items {
                if item.id == targetItem.id { break }
                guard ModuleContentLogic.isNavigable(item.kind) else { continue }
                let complete = ModuleContentLogic.isComplete(in: progress, itemId: item.id)
                appendRow(
                    RequirementRow(
                        id: "item:\(item.id)",
                        title: item.title,
                        detail: nil,
                        met: complete,
                        navigateItemId: complete ? nil : item.id
                    )
                )
                if item.id == reason?.itemId { break }
            }
        }

        if reason?.code != "sequential_order" {
            var currentReason = reason
            var guardCount = 0
            while let activeReason = currentReason, guardCount < 12 {
                guardCount += 1
                if let itemId = activeReason.itemId, !itemId.isEmpty {
                    let item = findItem(id: itemId, in: groups)
                    let complete = ModuleContentLogic.isComplete(in: progress, itemId: itemId)
                    appendRow(
                        RequirementRow(
                            id: "item:\(itemId)",
                            title: activeReason.title ?? item?.title ?? activeReason.message,
                            detail: activeReason.message,
                            met: complete,
                            navigateItemId: complete ? nil : navigableItemId(itemId, groups: groups)
                        )
                    )
                    if complete {
                        break
                    }
                    currentReason = ModuleContentLogic.itemLockState(in: progress, itemId: itemId)?.reason
                } else if activeReason.code != "sequential_order" {
                    appendRow(
                        RequirementRow(
                            id: "reason:\(activeReason.code)",
                            title: activeReason.title ?? activeReason.message,
                            detail: activeReason.title == nil ? nil : activeReason.message,
                            met: false,
                            navigateItemId: navigableItemId(activeReason.itemId, groups: groups)
                        )
                    )
                    break
                } else {
                    break
                }
            }
        }
    }

    private static func prerequisiteModule(
        for reason: LockReason,
        groups: [ModuleGroup],
        progress: ModulesProgressSnapshot?
    ) -> ModuleLockState? {
        if let title = reason.title,
           let match = progress?.modules.first(where: { $0.title == title }) {
            return match
        }
        return progress?.modules.first { module in
            !module.complete && module.moduleId != groups.last(where: { _ in true })?.id
        }
    }

    private static func firstIncompleteNavigableItem(
        moduleId: String,
        groups: [ModuleGroup],
        progress: ModulesProgressSnapshot?
    ) -> CourseStructureItem? {
        guard let group = groups.first(where: { $0.id == moduleId }) else { return nil }
        return group.items.first { item in
            ModuleContentLogic.isNavigable(item.kind)
                && !ModuleContentLogic.isComplete(in: progress, itemId: item.id)
        }
    }

    private static func navigableItemId(_ itemId: String?, groups: [ModuleGroup]) -> String? {
        guard let itemId, let item = findItem(id: itemId, in: groups) else { return nil }
        return ModuleContentLogic.isNavigable(item.kind) ? itemId : nil
    }
}
