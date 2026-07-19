import XCTest
@testable import Lextures

final class CoursePeopleLogicTests: XCTestCase {
    private func enrollment(
        id: String,
        name: String? = nil,
        role: String,
        sectionId: String? = nil,
        invited: Bool = false,
        state: String? = nil
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
            state: state,
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

    func testCanAddEnrollmentsRequiresFlagPermissionAndOnline() {
        var features = MobilePlatformFeatures()
        let perms = ["course:BIO101:enrollments:update"]
        XCTAssertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        features.ffMobileEnrollmentAdd = true
        XCTAssertTrue(
            CoursePeopleLogic.canAddEnrollments(
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        XCTAssertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: false
            )
        )
        XCTAssertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode: "BIO101",
                permissions: ["course:BIO101:enrollments:read"],
                features: features,
                isOnline: true
            )
        )
    }

    func testParseAndValidateEmails() {
        let emails = CoursePeopleLogic.parseEmails("Alex@School.edu, blair@school.edu; casey@school.edu\nnate@school.edu")
        XCTAssertEqual(emails, ["alex@school.edu", "blair@school.edu", "casey@school.edu", "nate@school.edu"])

        XCTAssertEqual(
            CoursePeopleLogic.validateEmailsForAdd(""),
            .emailsRequired
        )
        XCTAssertEqual(
            CoursePeopleLogic.validateEmailsForAdd("").errorKey,
            "mobile.people.add.error.emailsRequired"
        )
        XCTAssertEqual(
            CoursePeopleLogic.validateEmailsForAdd("not-an-email"),
            .invalidEmail
        )
        XCTAssertEqual(
            CoursePeopleLogic.validateEmailsForAdd("not-an-email").errorKey,
            "mobile.people.add.error.invalidEmail"
        )
        XCTAssertEqual(
            CoursePeopleLogic.validateEmailsForAdd("ok@school.edu"),
            .ok(["ok@school.edu"])
        )
    }

    func testBuildAddRequestAndSummarize() {
        let request = CoursePeopleLogic.buildAddRequest(
            emails: ["a@school.edu", "b@school.edu"],
            courseRole: "Teacher"
        )
        XCTAssertEqual(request.emails, "a@school.edu\nb@school.edu")
        XCTAssertEqual(request.courseRole, "instructor")
        XCTAssertTrue(CoursePeopleLogic.isAssignableRole("ta"))

        let summary = CoursePeopleLogic.summarizeAddResponse(
            AddCourseEnrollmentsResponse(
                added: ["a@school.edu"],
                alreadyEnrolled: ["b@school.edu"],
                notFound: ["c@school.edu"]
            )
        )
        XCTAssertTrue(summary.didAdd)
        XCTAssertTrue(summary.hasConflicts)
        XCTAssertEqual(summary.alreadyEnrolled, ["b@school.edu"])
    }

    func testStateHelpersAndChangeGate() {
        XCTAssertTrue(CoursePeopleLogic.isInactiveState("dropped"))
        XCTAssertFalse(CoursePeopleLogic.isInactiveState("active"))
        XCTAssertEqual(CoursePeopleLogic.deactivateState(for: "active"), "dropped")
        XCTAssertEqual(CoursePeopleLogic.deactivateState(for: "dropped"), "active")
        XCTAssertEqual(CoursePeopleLogic.stateLabelKey("waitlist"), "mobile.people.state.waitlist")

        var features = MobilePlatformFeatures(ffEnrollmentStateMachine: true)
        let student = enrollment(id: "1", role: "student", state: "active")
        let teacher = enrollment(id: "2", role: "teacher")
        let perms = ["course:BIO101:enrollments:update"]
        XCTAssertTrue(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment: student,
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        XCTAssertFalse(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment: teacher,
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
        features.ffEnrollmentStateMachine = false
        XCTAssertFalse(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment: student,
                courseCode: "BIO101",
                permissions: perms,
                features: features,
                isOnline: true
            )
        )
    }
}
