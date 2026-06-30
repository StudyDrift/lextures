import XCTest
@testable import Lextures

final class ModuleContentModelsTests: XCTestCase {
    func testBuildModuleGroupsOrdersChildren() {
        let items = [
            CourseStructureItem(
                id: "m1", sortOrder: 0, kind: "module", title: "Module 1",
                parentId: nil, published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil
            ),
            CourseStructureItem(
                id: "p2", sortOrder: 2, kind: "content_page", title: "Reading 2",
                parentId: "m1", published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil
            ),
            CourseStructureItem(
                id: "p1", sortOrder: 1, kind: "content_page", title: "Reading 1",
                parentId: "m1", published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil
            ),
        ]
        let groups = ModuleContentLogic.buildModuleGroups(from: items)
        XCTAssertEqual(groups.count, 1)
        XCTAssertEqual(groups[0].items.map(\.id), ["p1", "p2"])
    }

    func testItemLockStateLookup() {
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
                        ItemLockState(
                            itemId: "q1",
                            locked: true,
                            complete: false,
                            reason: LockReason(code: "prerequisite", message: "Complete Reading 1", itemId: "p1", title: "Reading 1")
                        ),
                    ]
                ),
            ]
        )
        let lock = ModuleContentLogic.itemLockState(in: progress, itemId: "q1")
        XCTAssertEqual(lock?.locked, true)
        XCTAssertEqual(lock?.reason?.message, "Complete Reading 1")
        XCTAssertTrue(ModuleContentLogic.isLocked(in: progress, itemId: "q1"))
    }

    func testDestinationRouting() {
        XCTAssertEqual(ModuleContentLogic.destination(for: "content_page"), .contentPage)
        XCTAssertEqual(ModuleContentLogic.destination(for: "quiz"), .quiz)
        XCTAssertEqual(ModuleContentLogic.destination(for: "h5p"), .interactive)
        XCTAssertTrue(ModuleContentLogic.isNavigable("external_link"))
        XCTAssertFalse(ModuleContentLogic.isNavigable("heading"))
    }
}
