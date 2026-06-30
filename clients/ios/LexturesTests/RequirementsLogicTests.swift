import XCTest
@testable import Lextures

final class RequirementsLogicTests: XCTestCase {
    private func item(_ id: String, sort: Int, kind: String = "content_page", title: String, parent: String) -> CourseStructureItem {
        CourseStructureItem(
            id: id, sortOrder: sort, kind: kind, title: title,
            parentId: parent, published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil
        )
    }

    private func moduleItem(_ id: String, sort: Int, title: String) -> CourseStructureItem {
        CourseStructureItem(
            id: id, sortOrder: sort, kind: "module", title: title,
            parentId: nil, published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil
        )
    }

    func testSequentialRequirementsListsPriorSteps() {
        let items = [
            moduleItem("m1", sort: 0, title: "Module 1"),
            item("p1", sort: 1, title: "Reading 1", parent: "m1"),
            item("p2", sort: 2, title: "Reading 2", parent: "m1"),
            item("q1", sort: 3, kind: "quiz", title: "Quiz 1", parent: "m1"),
        ]
        let groups = ModuleContentLogic.buildModuleGroups(from: items)
        let progress = ModulesProgressSnapshot(
            enrollmentId: "e1",
            modules: [
                ModuleLockState(
                    moduleId: "m1",
                    title: "Module 1",
                    sortOrder: 0,
                    locked: false,
                    complete: false,
                    reason: nil,
                    items: [
                        ItemLockState(itemId: "p1", locked: false, complete: true, reason: nil),
                        ItemLockState(
                            itemId: "p2",
                            locked: true,
                            complete: false,
                            reason: LockReason(code: "sequential_order", message: "Complete \"Reading 1\" first.", itemId: "p1", title: "Reading 1")
                        ),
                        ItemLockState(
                            itemId: "q1",
                            locked: true,
                            complete: false,
                            reason: LockReason(code: "sequential_order", message: "Complete \"Reading 2\" first.", itemId: "p2", title: "Reading 2")
                        ),
                    ]
                ),
            ]
        )

        let target = items.first { $0.id == "q1" }!
        let summary = RequirementsLogic.buildRequirements(for: target, groups: groups, progress: progress)

        XCTAssertEqual(summary.metCount, 1)
        XCTAssertEqual(summary.totalCount, 2)
        XCTAssertEqual(summary.nextRequiredItemId, "p2")
        XCTAssertTrue(summary.rows.contains { $0.id == "item:p1" && $0.met })
        XCTAssertTrue(summary.rows.contains { $0.id == "item:p2" && !$0.met })
    }

    func testModulePrerequisiteRequirementsIncludeIncompleteItems() {
        let items = [
            moduleItem("m1", sort: 0, title: "Module A"),
            item("a1", sort: 1, title: "Lesson A", parent: "m1"),
            moduleItem("m2", sort: 2, title: "Module B"),
            item("b1", sort: 3, title: "Lesson B", parent: "m2"),
        ]
        let groups = ModuleContentLogic.buildModuleGroups(from: items)
        let progress = ModulesProgressSnapshot(
            enrollmentId: "e1",
            modules: [
                ModuleLockState(
                    moduleId: "m1",
                    title: "Module A",
                    sortOrder: 0,
                    locked: false,
                    complete: false,
                    reason: nil,
                    items: [
                        ItemLockState(itemId: "a1", locked: false, complete: false, reason: nil),
                    ]
                ),
                ModuleLockState(
                    moduleId: "m2",
                    title: "Module B",
                    sortOrder: 1,
                    locked: true,
                    complete: false,
                    reason: LockReason(code: "module_prerequisite", message: "Complete module \"Module A\" to unlock.", title: "Module A"),
                    items: [
                        ItemLockState(
                            itemId: "b1",
                            locked: true,
                            complete: false,
                            reason: LockReason(code: "module_prerequisite", message: "Complete module \"Module A\" to unlock.", title: "Module A")
                        ),
                    ]
                ),
            ]
        )

        let target = items.first { $0.id == "b1" }!
        let summary = RequirementsLogic.buildRequirements(for: target, groups: groups, progress: progress)

        XCTAssertEqual(summary.nextRequiredItemId, "a1")
        XCTAssertTrue(summary.rows.contains { $0.id == "module:m1" && !$0.met })
        XCTAssertTrue(summary.rows.contains { $0.id == "item:a1" && !$0.met })
    }

    func testFindItemReturnsNavigableMatch() {
        let items = [
            moduleItem("m1", sort: 0, title: "Module 1"),
            item("p1", sort: 1, title: "Reading 1", parent: "m1"),
        ]
        let groups = ModuleContentLogic.buildModuleGroups(from: items)
        XCTAssertEqual(RequirementsLogic.findItem(id: "p1", in: groups)?.title, "Reading 1")
    }
}
