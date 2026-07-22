import XCTest
@testable import Lextures

final class PeopleAdminMetricsLogicTests: XCTestCase {
    func testToggleFilter() {
        XCTAssertEqual(PeopleAdminLogic.toggleFilter(current: nil, tapped: .active), .active)
        XCTAssertNil(PeopleAdminLogic.toggleFilter(current: .active, tapped: .active))
    }

    func testValueMaps() {
        let stats = PeopleDashboardStats(signupsLast7Days: 1, activeAccounts: 2, totalAccounts: 5, recentlyActive30Days: 3, suspendedAccounts: 4)
        XCTAssertEqual(PeopleAdminLogic.value(for: .signups7d, in: stats), 1)
        XCTAssertEqual(PeopleAdminLogic.value(for: .active, in: stats), 2)
        XCTAssertEqual(PeopleAdminLogic.value(for: .recent30d, in: stats), 3)
        XCTAssertEqual(PeopleAdminLogic.value(for: .total, in: stats), 5)
        XCTAssertEqual(PeopleAdminLogic.value(for: .suspended, in: stats), 4)
    }

    func testFilterRawValues() {
        XCTAssertEqual(PeopleListFilter.signups7d.rawValue, "signups_7d")
        XCTAssertEqual(PeopleListFilter.recent30d.rawValue, "recent_30d")
    }
}
