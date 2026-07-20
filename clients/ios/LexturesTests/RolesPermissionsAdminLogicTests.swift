import XCTest
@testable import Lextures

final class RolesPermissionsAdminLogicTests: XCTestCase {
    func testAdminSettingsEnabledRequiresFlag() {
        var features = MobilePlatformFeatures()
        features.ffMobileAdminConsole = false
        XCTAssertFalse(RolesPermissionsAdminLogic.adminSettingsEnabled(features))
        features.ffMobileAdminSettings = true
        XCTAssertTrue(RolesPermissionsAdminLogic.adminSettingsEnabled(features))
    }

    func testCanManageRolesRequiresRbacManage() {
        XCTAssertFalse(RolesPermissionsAdminLogic.canManageRoles(permissions: []))
        XCTAssertTrue(
            RolesPermissionsAdminLogic.canManageRoles(
                permissions: [RolesPermissionsAdminLogic.rbacManagePermission]
            )
        )
    }

    func testShouldShowEntryRequiresFlagAndPermission() {
        var features = MobilePlatformFeatures()
        features.ffMobileAdminConsole = false
        features.ffMobileAdminSettings = true
        XCTAssertFalse(
            RolesPermissionsAdminLogic.shouldShowEntry(features: features, permissions: [])
        )
        XCTAssertTrue(
            RolesPermissionsAdminLogic.shouldShowEntry(
                features: features,
                permissions: [RolesPermissionsAdminLogic.rbacManagePermission]
            )
        )
    }

    func testFilterRolesMatchesNameAndPermission() {
        let roles = [
            RoleWithPermissions(
                id: "1",
                name: "Teacher",
                permissions: [
                    RBACPermission(id: "p1", permissionString: "course:foo:item:read", description: "Read items")
                ]
            ),
            RoleWithPermissions(
                id: "2",
                name: "Admin",
                permissions: [
                    RBACPermission(id: "p2", permissionString: "global:app:rbac:manage", description: "Manage RBAC")
                ]
            ),
        ]
        XCTAssertEqual(
            RolesPermissionsAdminLogic.filterRoles(roles, query: "teacher").map(\.id),
            ["1"]
        )
        XCTAssertEqual(
            RolesPermissionsAdminLogic.filterRoles(roles, query: "rbac").map(\.id),
            ["2"]
        )
    }

    func testBlocksSelfElevationForRbacManageRole() {
        let role = RoleWithPermissions(
            id: "admin",
            name: "Global Admin",
            permissions: [
                RBACPermission(
                    id: "p1",
                    permissionString: RolesPermissionsAdminLogic.rbacManagePermission,
                    description: ""
                )
            ]
        )
        XCTAssertTrue(
            RolesPermissionsAdminLogic.blocksSelfElevation(
                role: role,
                targetUserId: "me",
                currentUserId: "me"
            )
        )
        XCTAssertFalse(
            RolesPermissionsAdminLogic.blocksSelfElevation(
                role: role,
                targetUserId: "other",
                currentUserId: "me"
            )
        )
    }

    func testAddRoleUserRequestPayload() {
        let request = RolesPermissionsAdminLogic.addRoleUserRequest(userId: "user-123")
        XCTAssertEqual(request.userId, "user-123")
    }

    func testUserDisplayLabelPrefersName() {
        let user = RBACUserBrief(id: "1", email: "a@example.com", displayName: "Alex Admin")
        XCTAssertEqual(RolesPermissionsAdminLogic.userDisplayLabel(user), "Alex Admin")
    }
}
