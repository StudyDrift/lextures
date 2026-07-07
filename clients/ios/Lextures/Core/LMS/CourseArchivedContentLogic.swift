import Foundation

/// Archived module item list + restore helpers (M13.12).
enum CourseArchivedContentLogic {
    struct ArchivedContentRow: Identifiable, Hashable, Equatable {
        var id: String
        var title: String
        var kind: String
        var kindLabelKey: String
        var moduleTitle: String
        var archivedAt: String?
    }

    private static let restorableKinds: Set<String> = [
        "heading",
        "content_page",
        "assignment",
        "quiz",
        "external_link",
    ]

    static func canViewArchivedContent(courseCode: String, permissions: [String]) -> Bool {
        CourseSettingsLogic.canManageCourse(courseCode: courseCode, permissions: permissions)
    }

    static func cacheKeyArchivedStructure(courseCode: String) -> String {
        "course:\(courseCode):archived-structure"
    }

    static func moduleTitleById(from items: [CourseStructureItem]) -> [String: String] {
        var titles: [String: String] = [:]
        for item in items where item.kind == "module" && item.parentId == nil {
            titles[item.id] = item.title
        }
        return titles
    }

    static func archivedRows(from items: [CourseStructureItem]) -> [ArchivedContentRow] {
        let modules = moduleTitleById(from: items)
        return items
            .filter { item in
                item.archived == true
                    && item.parentId != nil
                    && restorableKinds.contains(item.kind)
            }
            .sorted { lhs, rhs in
                if lhs.sortOrder != rhs.sortOrder {
                    return lhs.sortOrder < rhs.sortOrder
                }
                return lhs.title.localizedCaseInsensitiveCompare(rhs.title) == .orderedAscending
            }
            .map { item in
                ArchivedContentRow(
                    id: item.id,
                    title: item.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        ? L.text("mobile.emDash")
                        : item.title,
                    kind: item.kind,
                    kindLabelKey: kindLabelKey(for: item.kind),
                    moduleTitle: item.parentId.flatMap { modules[$0] } ?? L.text("mobile.emDash"),
                    archivedAt: item.updatedAt
                )
            }
    }

    static func itemsAfterRestore(items: [CourseStructureItem], removedId: String) -> [CourseStructureItem] {
        items.filter { $0.id != removedId }
    }

    static func kindLabelKey(for kind: String) -> String {
        switch kind {
        case "heading":
            return "mobile.courseSettings.archivedContent.kind.heading"
        case "content_page":
            return "mobile.courseSettings.archivedContent.kind.contentPage"
        case "assignment":
            return "mobile.courseSettings.archivedContent.kind.assignment"
        case "quiz":
            return "mobile.courseSettings.archivedContent.kind.quiz"
        case "external_link":
            return "mobile.courseSettings.archivedContent.kind.externalLink"
        default:
            return "mobile.courseSettings.archivedContent.kind.other"
        }
    }

    static func formatArchivedAt(_ iso: String?) -> String {
        let formatted = DateFormatting.formatDateTime(iso)
        return formatted.isEmpty ? L.text("mobile.emDash") : formatted
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError {
            return apiError.errorDescription ?? L.text("mobile.courseSettings.archivedContent.genericError")
        }
        return error.localizedDescription
    }
}

struct CourseStructureItemPatch: Encodable {
    var archived: Bool?
}
