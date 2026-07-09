import XCTest
@testable import Lextures

final class ArchivedCoursesAdminLogicTests: XCTestCase {
    func testAdminSettingsEnabledRequiresFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(ArchivedCoursesAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(ArchivedCoursesAdminLogic.adminSettingsEnabled(features))
    }

    func testCanManageArchivedCoursesRequiresRbacManage() {
        XCTAssertFalse(ArchivedCoursesAdminLogic.canManageArchivedCourses(permissions: []))
        XCTAssertTrue(
            ArchivedCoursesAdminLogic.canManageArchivedCourses(
                permissions: [ArchivedCoursesAdminLogic.rbacManagePermission]
            )
        )
    }

    func testShouldShowEntryRequiresFlagAndPermission() {
        var features = MobilePlatformFeatures()
        features.ffMobileAdminSettings = true
        XCTAssertFalse(
            ArchivedCoursesAdminLogic.shouldShowEntry(features: features, permissions: [])
        )
        XCTAssertTrue(
            ArchivedCoursesAdminLogic.shouldShowEntry(
                features: features,
                permissions: [ArchivedCoursesAdminLogic.rbacManagePermission]
            )
        )
    }

    func testFilterRowsMatchesTitleAndCode() {
        let rows = [
            ArchivedCourseRow(
                id: "1",
                courseCode: "C-ALG101",
                title: "Algebra I",
                archivedAt: "2026-01-01T00:00:00Z"
            ),
            ArchivedCourseRow(
                id: "2",
                courseCode: "C-BIO201",
                title: "Biology",
                archivedAt: "2026-01-02T00:00:00Z"
            ),
        ]
        XCTAssertEqual(
            ArchivedCoursesAdminLogic.filterRows(rows, query: "alg").map(\.courseCode),
            ["C-ALG101"]
        )
        XCTAssertEqual(
            ArchivedCoursesAdminLogic.filterRows(rows, query: "bio201").map(\.courseCode),
            ["C-BIO201"]
        )
    }

    func testDeleteConfirmMatchesCourseCode() {
        let row = ArchivedCourseRow(
            id: "1",
            courseCode: "C-DEL01",
            title: "Delete me",
            archivedAt: nil
        )
        XCTAssertFalse(ArchivedCoursesAdminLogic.deleteConfirmMatches(typed: "wrong", row: row))
        XCTAssertTrue(ArchivedCoursesAdminLogic.deleteConfirmMatches(typed: "c-del01", row: row))
    }

    func testArchivedByLabelPrefersNameThenEmail() {
        let withName = ArchivedCourseRow(
            id: "1",
            courseCode: "C-1",
            title: "T",
            archivedAt: nil,
            archivedByName: "  Pat  ",
            archivedByEmail: "pat@example.com"
        )
        XCTAssertEqual(ArchivedCoursesAdminLogic.archivedByLabel(withName), "Pat")

        let withEmail = ArchivedCourseRow(
            id: "2",
            courseCode: "C-2",
            title: "T",
            archivedAt: nil,
            archivedByEmail: "admin@example.com"
        )
        XCTAssertEqual(ArchivedCoursesAdminLogic.archivedByLabel(withEmail), "admin@example.com")
    }
}