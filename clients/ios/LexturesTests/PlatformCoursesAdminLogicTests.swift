import XCTest
@testable import Lextures

final class PlatformCoursesAdminLogicTests: XCTestCase {
    func testToggleFilter() {
        XCTAssertEqual(PlatformCoursesAdminLogic.toggleFilter(current: nil, tapped: .draft), .draft)
        XCTAssertNil(PlatformCoursesAdminLogic.toggleFilter(current: .draft, tapped: .draft))
    }

    func testValueMaps() {
        let stats = CoursesDashboardStats(createdLast7Days: 1, activeCourses: 2, draftCourses: 3, totalCourses: 10, archivedCourses: 4)
        XCTAssertEqual(PlatformCoursesAdminLogic.value(for: .created7d, in: stats), 1)
        XCTAssertEqual(PlatformCoursesAdminLogic.value(for: .active, in: stats), 2)
        XCTAssertEqual(PlatformCoursesAdminLogic.value(for: .draft, in: stats), 3)
        XCTAssertEqual(PlatformCoursesAdminLogic.value(for: .total, in: stats), 10)
        XCTAssertEqual(PlatformCoursesAdminLogic.value(for: .archived, in: stats), 4)
    }

    func testFilterRawValues() {
        XCTAssertEqual(CoursesListFilter.created7d.rawValue, "created_7d")
    }
}
