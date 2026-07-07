import XCTest
@testable import Lextures

final class CourseArchivedContentLogicTests: XCTestCase {
    func testCanViewRequiresItemCreatePermission() {
        XCTAssertFalse(
            CourseArchivedContentLogic.canViewArchivedContent(
                courseCode: "C-1",
                permissions: []
            )
        )
        XCTAssertTrue(
            CourseArchivedContentLogic.canViewArchivedContent(
                courseCode: "C-1",
                permissions: [CourseSettingsLogic.courseItemCreatePermission(courseCode: "C-1")]
            )
        )
    }

    func testArchivedRowsFiltersRestorableItems() {
        let items = [
            CourseStructureItem(
                id: "m1",
                sortOrder: 0,
                kind: "module",
                title: "Week 1",
                parentId: nil,
                published: true,
                archived: false
            ),
            CourseStructureItem(
                id: "p1",
                sortOrder: 1,
                kind: "content_page",
                title: "Reading",
                parentId: "m1",
                published: false,
                archived: true,
                updatedAt: "2026-01-15T12:00:00Z"
            ),
            CourseStructureItem(
                id: "q1",
                sortOrder: 2,
                kind: "quiz",
                title: "Quiz",
                parentId: "m1",
                published: true,
                archived: true
            ),
            CourseStructureItem(
                id: "x1",
                sortOrder: 3,
                kind: "scorm",
                title: "Package",
                parentId: "m1",
                published: true,
                archived: true
            ),
        ]

        let rows = CourseArchivedContentLogic.archivedRows(from: items)
        XCTAssertEqual(rows.map(\.id), ["p1", "q1"])
        XCTAssertEqual(rows.first?.moduleTitle, "Week 1")
        XCTAssertEqual(rows.first?.archivedAt, "2026-01-15T12:00:00Z")
    }

    func testItemsAfterRestoreRemovesRow() {
        let items = [
            CourseStructureItem(
                id: "a",
                sortOrder: 0,
                kind: "content_page",
                title: "A",
                parentId: "m1",
                published: true,
                archived: true
            ),
            CourseStructureItem(
                id: "b",
                sortOrder: 1,
                kind: "assignment",
                title: "B",
                parentId: "m1",
                published: true,
                archived: true
            ),
        ]
        let updated = CourseArchivedContentLogic.itemsAfterRestore(items: items, removedId: "a")
        XCTAssertEqual(updated.map(\.id), ["b"])
    }

    func testKindLabelKeys() {
        XCTAssertEqual(
            CourseArchivedContentLogic.kindLabelKey(for: "content_page"),
            "mobile.courseSettings.archivedContent.kind.contentPage"
        )
        XCTAssertEqual(
            CourseArchivedContentLogic.kindLabelKey(for: "unknown"),
            "mobile.courseSettings.archivedContent.kind.other"
        )
    }

    func testCacheKey() {
        XCTAssertEqual(
            CourseArchivedContentLogic.cacheKeyArchivedStructure(courseCode: "C-1"),
            "course:C-1:archived-structure"
        )
    }
}
