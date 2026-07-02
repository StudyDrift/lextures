import XCTest
@testable import Lextures

final class CoursePeopleLogicTests: XCTestCase {
    private func enrollment(
        id: String,
        name: String? = nil,
        role: String,
        sectionId: String? = nil,
        invited: Bool = false
    ) -> CourseEnrollment {
        CourseEnrollment(
            id: id,
            userId: "user-\(id)",
            displayName: name,
            avatarUrl: nil,
            role: role,
            roleDisplay: nil,
            lastCourseAccessAt: nil,
            sectionId: sectionId,
            sectionCode: nil,
            sectionName: nil,
            state: nil,
            invitationPending: invited
        )
    }

    func testEnrollmentRoleRankOrdersStaffBeforeStudents() {
        XCTAssertLessThan(CoursePeopleLogic.enrollmentRoleRank("teacher"), CoursePeopleLogic.enrollmentRoleRank("student"))
        XCTAssertLessThan(CoursePeopleLogic.enrollmentRoleRank("ta"), CoursePeopleLogic.enrollmentRoleRank("student"))
    }

    func testFilterByRoleAndSection() {
        let rows = [
            enrollment(id: "1", name: "Alex", role: "student", sectionId: "sec-a"),
            enrollment(id: "2", name: "Blair", role: "teacher"),
            enrollment(id: "3", name: "Casey", role: "student", sectionId: "sec-b"),
        ]
        let staffOnly = CoursePeopleLogic.filter(
            enrollments: rows,
            search: "",
            roleFilter: .staff,
            sectionId: nil
        )
        XCTAssertEqual(staffOnly.map(\.id), ["2"])

        let sectionA = CoursePeopleLogic.filter(
            enrollments: rows,
            search: "",
            roleFilter: .all,
            sectionId: "sec-a"
        )
        XCTAssertEqual(sectionA.map(\.id), ["1"])
    }

    func testSearchMatchesNameRoleAndSection() {
        let rows = [
            enrollment(id: "1", name: "Alex Rivera", role: "student", sectionId: "sec-a"),
            enrollment(id: "2", name: "Blair", role: "ta"),
        ]
        let matches = CoursePeopleLogic.filter(
            enrollments: rows,
            search: "rivera",
            roleFilter: .all,
            sectionId: nil
        )
        XCTAssertEqual(matches.map(\.id), ["1"])
    }

    func testGroupedSectionsOrdersTeachersThenStudents() {
        let rows = [
            enrollment(id: "1", name: "Zoe", role: "student"),
            enrollment(id: "2", name: "Alex", role: "teacher"),
            enrollment(id: "3", name: "Blair", role: "ta"),
        ]
        let groups = CoursePeopleLogic.groupedSections(from: rows)
        XCTAssertEqual(groups.map(\.kind), [.teachers, .tas, .students])
        XCTAssertEqual(groups[0].enrollments.map(\.id), ["2"])
        XCTAssertEqual(groups[2].enrollments.map(\.id), ["1"])
    }

    func testCanUpdateEnrollmentsPermission() {
        XCTAssertTrue(
            CoursePeopleLogic.canUpdateEnrollments(
                courseCode: "BIO101",
                permissions: ["course:BIO101:enrollments:update"]
            )
        )
        XCTAssertFalse(
            CoursePeopleLogic.canUpdateEnrollments(
                courseCode: "BIO101",
                permissions: ["course:BIO101:enrollments:read"]
            )
        )
    }
}