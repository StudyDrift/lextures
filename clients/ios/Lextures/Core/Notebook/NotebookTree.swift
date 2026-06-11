import Foundation

/// Page-tree helpers over the flat `NotebookPage` list (parity with web `course-notebook-tree`).
enum NotebookTree {
    struct FlatRow: Identifiable, Equatable {
        let page: NotebookPage
        let depth: Int
        var id: String { page.id }
    }

    static func isGroup(_ page: NotebookPage) -> Bool {
        page.kind == "group"
    }

    static func sortedChildren(_ pages: [NotebookPage], parentId: String?) -> [NotebookPage] {
        pages
            .filter { $0.parentId == parentId }
            .sorted { ($0.sortOrder, $0.id) < ($1.sortOrder, $1.id) }
    }

    /// Depth-first flattening for list rendering (groups followed by their children).
    static func flatten(_ pages: [NotebookPage]) -> [FlatRow] {
        var rows: [FlatRow] = []
        func walk(parentId: String?, depth: Int) {
            for page in sortedChildren(pages, parentId: parentId) {
                rows.append(FlatRow(page: page, depth: depth))
                walk(parentId: page.id, depth: depth + 1)
            }
        }
        walk(parentId: nil, depth: 0)
        return rows
    }

    /// True if `targetId` is `ancestorId` or nested under it.
    static func isUnderAncestor(_ pages: [NotebookPage], ancestorId: String, targetId: String) -> Bool {
        let map = Dictionary(uniqueKeysWithValues: pages.map { ($0.id, $0) })
        var cur: String? = targetId
        while let id = cur {
            if id == ancestorId { return true }
            cur = map[id]?.parentId
        }
        return false
    }

    private static func nextSortOrder(_ pages: [NotebookPage], parentId: String?) -> Int {
        (sortedChildren(pages, parentId: parentId).map(\.sortOrder).max() ?? -1) + 1
    }

    static func addPage(_ pages: [NotebookPage], parentId: String?, title: String = "Untitled") -> (pages: [NotebookPage], newId: String) {
        var page = NotebookPage.new(title: title, sortOrder: nextSortOrder(pages, parentId: parentId))
        page.parentId = parentId
        return (pages + [page], page.id)
    }

    static func addGroup(_ pages: [NotebookPage], parentId: String?, title: String = "Untitled group") -> (pages: [NotebookPage], newId: String) {
        var group = NotebookPage.new(title: title, sortOrder: nextSortOrder(pages, parentId: parentId))
        group.parentId = parentId
        group.kind = "group"
        return (pages + [group], group.id)
    }

    /// Delete a page or group along with all descendants.
    static func delete(_ pages: [NotebookPage], pageId: String) -> [NotebookPage] {
        var toRemove = Set<String>()
        func walk(_ id: String) {
            toRemove.insert(id)
            for child in pages.filter({ $0.parentId == id }) {
                walk(child.id)
            }
        }
        walk(pageId)
        return pages.filter { !toRemove.contains($0.id) }
    }

    static func rename(_ pages: [NotebookPage], pageId: String, title: String) -> [NotebookPage] {
        pages.map { page in
            var next = page
            if page.id == pageId { next.title = title }
            return next
        }
    }

    static func updateContent(_ pages: [NotebookPage], pageId: String, contentMd: String) -> [NotebookPage] {
        pages.map { page in
            var next = page
            if page.id == pageId { next.contentMd = contentMd }
            return next
        }
    }

    /// Move a page or group under `newParentId` (nil = top level), appended last. Nil on invalid move.
    static func moveToParent(_ pages: [NotebookPage], pageId: String, newParentId: String?) -> [NotebookPage]? {
        guard pageId != newParentId else { return nil }
        if let newParentId, isUnderAncestor(pages, ancestorId: pageId, targetId: newParentId) {
            return nil
        }
        guard pages.contains(where: { $0.id == pageId }) else { return nil }
        let order = nextSortOrder(pages, parentId: newParentId)
        return pages.map { page in
            var next = page
            if page.id == pageId {
                next.parentId = newParentId
                next.sortOrder = order
            }
            return next
        }
    }

    /// Groups the page can be moved into (excludes self and own descendants).
    static func groupMoveTargets(_ pages: [NotebookPage], pageId: String) -> [NotebookPage] {
        pages
            .filter { isGroup($0) && $0.id != pageId }
            .filter { !isUnderAncestor(pages, ancestorId: pageId, targetId: $0.id) }
            .sorted { pathLabel(pages, pageId: $0.id) < pathLabel(pages, pageId: $1.id) }
    }

    static func pathLabel(_ pages: [NotebookPage], pageId: String) -> String {
        let map = Dictionary(uniqueKeysWithValues: pages.map { ($0.id, $0) })
        var parts: [String] = []
        var cur: String? = pageId
        while let id = cur, let row = map[id] {
            parts.insert(row.title.isEmpty ? "Untitled" : row.title, at: 0)
            cur = row.parentId
        }
        return parts.joined(separator: " / ")
    }
}
