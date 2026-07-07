import XCTest
@testable import Lextures

final class CourseBlueprintLogicTests: XCTestCase {
    func testCanManageBlueprintRequiresOrgAndPermission() {
        let course = sampleCourse(orgId: "org-1")
        XCTAssertFalse(CourseBlueprintLogic.canManageBlueprint(course: course, permissions: []))
        XCTAssertTrue(
            CourseBlueprintLogic.canManageBlueprint(
                course: course,
                permissions: [CourseBlueprintLogic.globalAdminPermission]
            )
        )
        XCTAssertTrue(
            CourseBlueprintLogic.canManageBlueprint(
                course: course,
                permissions: [CourseBlueprintLogic.orgUnitsAdminPermission]
            )
        )
        XCTAssertFalse(
            CourseBlueprintLogic.canManageBlueprint(
                course: sampleCourse(orgId: nil),
                permissions: [CourseBlueprintLogic.globalAdminPermission]
            )
        )
    }

    func testBlueprintRole() {
        var master = sampleCourse(orgId: "org-1")
        master.isBlueprint = true
        XCTAssertEqual(CourseBlueprintLogic.blueprintRole(for: master), .master)

        var child = sampleCourse(orgId: "org-1")
        child.blueprintParentCourseCode = "BP-001"
        if case let .child(code) = CourseBlueprintLogic.blueprintRole(for: child) {
            XCTAssertEqual(code, "BP-001")
        } else {
            XCTFail("expected child role")
        }

        XCTAssertEqual(CourseBlueprintLogic.blueprintRole(for: sampleCourse(orgId: "org-1")), .none)
    }

    func testShouldLoadBlueprintDetails() {
        var master = sampleCourse(orgId: "org-1")
        master.isBlueprint = true
        XCTAssertTrue(
            CourseBlueprintLogic.shouldLoadBlueprintDetails(
                course: master,
                canManage: true
            )
        )
        XCTAssertFalse(
            CourseBlueprintLogic.shouldLoadBlueprintDetails(
                course: master,
                canManage: false
            )
        )
    }

    func testPushDisabledWhenOffline() {
        XCTAssertNotNil(CourseBlueprintLogic.pushDisabledReason(isOnline: false, childCount: 2))
    }

    func testPushDisabledWithoutChildren() {
        XCTAssertNotNil(CourseBlueprintLogic.pushDisabledReason(isOnline: true, childCount: 0))
    }

    func testPushEnabledWhenOnlineWithChildren() {
        XCTAssertNil(CourseBlueprintLogic.pushDisabledReason(isOnline: true, childCount: 1))
    }

    func testMutationsDisabledWhenOffline() {
        XCTAssertNotNil(CourseBlueprintLogic.mutationsDisabledReason(isOnline: false))
        XCTAssertNil(CourseBlueprintLogic.mutationsDisabledReason(isOnline: true))
    }

    func testCacheKey() {
        XCTAssertEqual(
            CourseBlueprintLogic.cacheKeyBlueprintData(courseCode: "C-1"),
            "course:C-1:blueprint"
        )
    }

    private func sampleCourse(orgId: String?) -> CourseSummary {
        CourseSummary(
            id: "1",
            courseCode: "C-1",
            title: "Intro",
            description: "Desc",
            orgId: orgId
        )
    }
}
