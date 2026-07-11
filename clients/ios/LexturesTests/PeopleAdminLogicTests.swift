import XCTest
@testable import Lextures

final class PeopleAdminLogicTests: XCTestCase {
    func testAdminSettingsEnabledRequiresFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(PeopleAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(PeopleAdminLogic.adminSettingsEnabled(features))
    }

    func testCanManagePeopleRequiresRbacManage() {
        XCTAssertFalse(PeopleAdminLogic.canManagePeople(permissions: []))
        XCTAssertTrue(
            PeopleAdminLogic.canManagePeople(
                permissions: [PeopleAdminLogic.rbacManagePermission]
            )
        )
    }

    func testShouldShowEntryRequiresFlagAndPermission() {
        var features = MobilePlatformFeatures()
        features.ffMobileAdminSettings = true
        XCTAssertFalse(
            PeopleAdminLogic.shouldShowEntry(features: features, permissions: [])
        )
        XCTAssertTrue(
            PeopleAdminLogic.shouldShowEntry(
                features: features,
                permissions: [PeopleAdminLogic.rbacManagePermission]
            )
        )
    }

    func testPersonDisplayNamePrefersDisplayName() {
        let row = PersonRow(
            id: "1",
            email: "a@example.com",
            firstName: "Alex",
            lastName: "Admin",
            displayName: "Alex Admin",
            orgId: "org",
            orgName: "Org",
            role: "teacher",
            active: true,
            createdAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertEqual(PeopleAdminLogic.personDisplayName(row), "Alex Admin")
    }

    func testPersonDisplayNameFallsBackToEmail() {
        let row = PersonRow(
            id: "1",
            email: "a@example.com",
            firstName: nil,
            lastName: nil,
            displayName: nil,
            orgId: "org",
            orgName: "Org",
            role: "teacher",
            active: true,
            createdAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertEqual(PeopleAdminLogic.personDisplayName(row), "a@example.com")
    }

    func testBlocksSelfSuspend() {
        XCTAssertTrue(PeopleAdminLogic.blocksSelfSuspend(targetUserId: "me", currentUserId: "me"))
        XCTAssertFalse(PeopleAdminLogic.blocksSelfSuspend(targetUserId: "other", currentUserId: "me"))
    }

    func testIsErasedDetectsErasedEmail() {
        XCTAssertTrue(PeopleAdminLogic.isErased(email: "user@erased.invalid"))
        XCTAssertFalse(PeopleAdminLogic.isErased(email: "user@example.com"))
    }

    func testInvitePersonRequestTrimsFields() {
        let request = PeopleAdminLogic.invitePersonRequest(
            email: "  teacher@school.edu ",
            firstName: " Pat ",
            lastName: " "
        )
        XCTAssertEqual(request.email, "teacher@school.edu")
        XCTAssertEqual(request.firstName, "Pat")
        XCTAssertNil(request.lastName)
    }

    func testPatchPersonRequestPayload() {
        XCTAssertTrue(PeopleAdminLogic.patchPersonRequest(active: false).active == false)
        XCTAssertTrue(PeopleAdminLogic.patchPersonRequest(active: true).active == true)
    }

    func testResendInviteRequestTrimsEmail() {
        XCTAssertEqual(
            PeopleAdminLogic.resendInviteRequest(email: "  a@b.com ").email,
            "a@b.com"
        )
    }

    func testRoleMatchesReportByName() {
        let report = PersonReport(
            id: "1",
            email: "a@example.com",
            firstName: nil,
            lastName: nil,
            displayName: nil,
            orgId: "org",
            orgName: "Org",
            role: "Teacher",
            active: true,
            createdAt: "2026-01-01T00:00:00Z",
            lastActivityAt: nil,
            enrollmentCount: 0,
            enrollments: [],
            recentActivity: []
        )
        let role = RoleWithPermissions(id: "r1", name: "teacher", permissions: [])
        XCTAssertTrue(PeopleAdminLogic.roleMatchesReport(role, report: report))
    }
}
