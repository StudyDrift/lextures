import Foundation

// MARK: - Module grouping

struct ModuleGroup: Identifiable, Equatable {
    let id: String
    let title: String
    let items: [CourseStructureItem]
}

enum ModuleItemDestination: Equatable {
    case contentPage
    case quiz
    case assignment
    case externalLink
    case webContent
    case interactive
    case file
    case unsupported
}

enum ModuleContentLogic {
    static func buildModuleGroups(from items: [CourseStructureItem]) -> [ModuleGroup] {
        let modules = items.filter(\.isModule).sorted { $0.sortOrder < $1.sortOrder }
        let children = Dictionary(grouping: items.filter { !$0.isModule && $0.parentId != nil }) { $0.parentId! }
        var groups = modules.map { module in
            ModuleGroup(
                id: module.id,
                title: module.title,
                items: (children[module.id] ?? []).sorted { $0.sortOrder < $1.sortOrder }
            )
        }
        let orphans = items
            .filter { !$0.isModule && $0.parentId == nil && $0.kind != "heading" }
            .sorted { $0.sortOrder < $1.sortOrder }
        if !orphans.isEmpty {
            groups.append(ModuleGroup(id: "__orphans__", title: "Other items", items: orphans))
        }
        return groups
    }

    static func destination(for kind: String) -> ModuleItemDestination {
        switch kind {
        case "content_page": return .contentPage
        case "quiz": return .quiz
        case "assignment": return .assignment
        case "external_link": return .externalLink
        case "textbook_resource", "library_resource": return .webContent
        case "h5p", "scorm", "lti_link", "vibe_activity": return .interactive
        case "file", "file_item": return .file
        default: return .unsupported
        }
    }

    static func isNavigable(_ kind: String) -> Bool {
        switch destination(for: kind) {
        case .unsupported: return false
        default: return true
        }
    }

    static func itemLockState(in progress: ModulesProgressSnapshot?, itemId: String) -> ItemLockState? {
        guard let progress else { return nil }
        for module in progress.modules {
            for item in module.items ?? [] where item.itemId == itemId {
                return item
            }
        }
        return nil
    }

    static func moduleLockState(in progress: ModulesProgressSnapshot?, moduleId: String) -> ModuleLockState? {
        progress?.modules.first { $0.moduleId == moduleId }
    }

    static func isLocked(in progress: ModulesProgressSnapshot?, itemId: String) -> Bool {
        itemLockState(in: progress, itemId: itemId)?.locked == true
    }

    static func isComplete(in progress: ModulesProgressSnapshot?, itemId: String) -> Bool {
        itemLockState(in: progress, itemId: itemId)?.complete == true
    }

    static func accessibilityLabel(for item: CourseStructureItem, progress: ModulesProgressSnapshot?) -> String {
        var parts = [ItemKind.label(for: item.kind), item.title]
        if isComplete(in: progress, itemId: item.id) {
            parts.append(L.text("mobile.modules.complete"))
        }
        if isLocked(in: progress, itemId: item.id),
           let reason = itemLockState(in: progress, itemId: item.id)?.reason?.message,
           !reason.isEmpty {
            parts.append(reason)
        }
        return parts.joined(separator: ", ")
    }
}
