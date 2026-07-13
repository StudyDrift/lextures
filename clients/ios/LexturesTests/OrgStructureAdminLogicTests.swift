import XCTest
@testable import Lextures

final class OrgStructureAdminLogicTests: XCTestCase {
    func testAdminSettingsEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(OrgStructureAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(OrgStructureAdminLogic.adminSettingsEnabled(features))
    }

    func testCanManageOrganizations() {
        XCTAssertFalse(OrgStructureAdminLogic.canManageOrganizations(permissions: []))
        XCTAssertTrue(
            OrgStructureAdminLogic.canManageOrganizations(
                permissions: [OrgStructureAdminLogic.rbacManagePermission]
            )
        )
    }

    func testCanManageOrgUnitsAndTerms() {
        XCTAssertFalse(OrgStructureAdminLogic.canManageOrgUnitsAndTerms(permissions: []))
        XCTAssertTrue(
            OrgStructureAdminLogic.canManageOrgUnitsAndTerms(
                permissions: [OrgStructureAdminLogic.orgUnitsAdminPermission]
            )
        )
        XCTAssertTrue(
            OrgStructureAdminLogic.canManageOrgUnitsAndTerms(
                permissions: [OrgStructureAdminLogic.rbacManagePermission]
            )
        )
    }

    func testShouldShowEntry() {
        let features = MobilePlatformFeatures(ffMobileAdminSettings: true)
        XCTAssertFalse(
            OrgStructureAdminLogic.shouldShowEntry(features: features, permissions: [])
        )
        XCTAssertTrue(
            OrgStructureAdminLogic.shouldShowEntry(
                features: features,
                permissions: [OrgStructureAdminLogic.orgUnitsAdminPermission]
            )
        )
    }

    func testCreateTermRequest() {
        let request = OrgStructureAdminLogic.createTermRequest(
            name: "  Spring 2026 ",
            termType: "",
            startDate: "2026-01-10",
            endDate: "2026-05-15"
        )
        XCTAssertEqual(request.name, "Spring 2026")
        XCTAssertEqual(request.termType, OrgStructureAdminLogic.defaultTermType)
        XCTAssertEqual(request.startDate, "2026-01-10")
        XCTAssertEqual(request.endDate, "2026-05-15")
    }

    func testPatchTermDatesRequest() {
        let request = OrgStructureAdminLogic.patchTermDatesRequest(
            startDate: "2026-01-10",
            endDate: "2026-05-15"
        )
        XCTAssertEqual(request.startDate, "2026-01-10")
        XCTAssertEqual(request.endDate, "2026-05-15")
        XCTAssertNil(request.name)
    }

    func testDateRangeValidation() {
        XCTAssertTrue(OrgStructureAdminLogic.isValidDateRange(start: "2026-01-01", end: "2026-06-01"))
        XCTAssertFalse(OrgStructureAdminLogic.isValidDateRange(start: "2026-06-01", end: "2026-01-01"))
    }

    func testFlattenTree() {
        let child = OrgUnitTreeNode(
            id: "child",
            name: "Child",
            unitType: "department",
            status: "active",
            childCourseCount: 0,
            children: []
        )
        let root = OrgUnitTreeNode(
            id: "root",
            name: "Root",
            unitType: "school",
            status: "active",
            childCourseCount: 1,
            children: [child]
        )
        XCTAssertEqual(OrgStructureAdminLogic.flattenTree([root]).map(\.id), ["root", "child"])
    }
}
