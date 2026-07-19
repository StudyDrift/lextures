import XCTest
@testable import Lextures

final class SettingsMenuLogicTests: XCTestCase {
    private let rbac = SettingsMenuLogic.rbacManagePermission

    func testHubHiddenWhenFlagOff() {
        var features = MobilePlatformFeatures(ffMobileAdminConsole: false, adminConsoleEnabled: true)
        XCTAssertFalse(SettingsMenuLogic.shouldShowHubEntry(features: features, permissions: [rbac]))
    }

    func testHubRequiresRbacOrAuditVisibility() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            adminConsoleEnabled: true,
            adminAuditLogEnabled: true
        )
        XCTAssertFalse(SettingsMenuLogic.shouldShowHubEntry(features: features, permissions: []))
        XCTAssertTrue(SettingsMenuLogic.shouldShowHubEntry(features: features, permissions: [rbac]))
    }

    func testVisibleGroupsHideAdminItemsWithoutRbac() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            ffMobileAdminSettings: true,
            adminConsoleEnabled: true,
            adminAuditLogEnabled: true
        )
        let groups = SettingsMenuLogic.visibleGroups(features: features, permissions: [])
        XCTAssertTrue(groups.isEmpty)
    }

    func testVisibleGroupsIncludeShippedAndAuditLog() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            ffMobileAdminSettings: true,
            adminConsoleEnabled: true,
            adminAuditLogEnabled: true
        )
        let groups = SettingsMenuLogic.visibleGroups(features: features, permissions: [rbac])
        let ids = Set(groups.flatMap { $0.items.map(\.id) })
        XCTAssertTrue(ids.contains(.platformSettings))
        XCTAssertTrue(ids.contains(.people))
        XCTAssertTrue(ids.contains(.auditLog))
        XCTAssertTrue(groups.contains { $0.id == .platform })
        XCTAssertTrue(groups.contains { $0.id == .compliance })
    }

    func testAuditLogHiddenWhenAdminConsoleFlagOff() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            adminConsoleEnabled: false,
            adminAuditLogEnabled: true
        )
        XCTAssertFalse(
            SettingsMenuLogic.isItemVisible(.auditLog, features: features, permissions: [rbac])
        )
    }

    func testPhase1RegistryGroupsMatchWebInventoryLabels() {
        let labels = Dictionary(uniqueKeysWithValues: SettingsMenuLogic.phase1Registry.map { ($0.group, $0.items) })
        XCTAssertEqual(labels[.platform]?.contains(.platformSettings), true)
        XCTAssertEqual(labels[.compliance]?.contains(.auditLog), true)
        XCTAssertEqual(labels[.integrations]?.contains(.integrations), true)
        XCTAssertEqual(labels[.studentRecords]?.contains(.transcriptsAdvising), true)
        XCTAssertTrue(labels[.schoolOperations]?.isEmpty == true)
    }

    func testSearchFiltersItems() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole: true,
            ffMobileAdminSettings: true,
            adminConsoleEnabled: true,
            adminAuditLogEnabled: true
        )
        let groups = SettingsMenuLogic.visibleGroups(
            features: features,
            permissions: [rbac],
            query: "audit"
        )
        let ids = groups.flatMap { $0.items.map(\.id) }
        XCTAssertEqual(ids, [.auditLog])
    }
}
