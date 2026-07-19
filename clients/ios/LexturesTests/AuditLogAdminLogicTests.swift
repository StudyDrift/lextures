import XCTest
@testable import Lextures

final class AuditLogAdminLogicTests: XCTestCase {
    func testGateRequiresConsoleFlagsAndRbac() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            adminConsoleEnabled: true,
            adminAuditLogEnabled: true
        )
        XCTAssertFalse(AuditLogAdminLogic.shouldShowInMenu(features: features, permissions: []))
        XCTAssertTrue(
            AuditLogAdminLogic.shouldShowInMenu(
                features: features,
                permissions: [AuditLogAdminLogic.rbacManagePermission]
            )
        )

        features.adminAuditLogEnabled = false
        XCTAssertFalse(
            AuditLogAdminLogic.shouldShowInMenu(
                features: features,
                permissions: [AuditLogAdminLogic.rbacManagePermission]
            )
        )
    }

    func testNormalizedActionFilter() {
        XCTAssertNil(AuditLogAdminLogic.normalizedActionFilter("  "))
        XCTAssertEqual(AuditLogAdminLogic.normalizedActionFilter(" user_deactivate "), "user_deactivate")
    }

    func testTargetLabel() {
        XCTAssertEqual(AuditLogAdminLogic.targetLabel(type: nil, id: nil), "—")
        XCTAssertEqual(AuditLogAdminLogic.targetLabel(type: "user", id: nil), "user")
        XCTAssertEqual(AuditLogAdminLogic.targetLabel(type: nil, id: "abc"), "abc")
        XCTAssertEqual(AuditLogAdminLogic.targetLabel(type: "user", id: "abc"), "user / abc")
    }

    func testWebPath() {
        XCTAssertEqual(AuditLogAdminLogic.webPath(), "/org-admin/audit-log")
    }
}
