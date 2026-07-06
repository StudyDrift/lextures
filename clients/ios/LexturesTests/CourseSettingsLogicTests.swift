import XCTest
@testable import Lextures

final class CourseSettingsLogicTests: XCTestCase {
    func testCourseItemCreatePermission() {
        XCTAssertEqual(
            CourseSettingsLogic.courseItemCreatePermission(courseCode: "C-ABC123"),
            "course:C-ABC123:item:create"
        )
    }

    func testCanManageCourse() {
        let perms = ["course:C-1:item:create", "other"]
        XCTAssertTrue(CourseSettingsLogic.canManageCourse(courseCode: "C-1", permissions: perms))
        XCTAssertFalse(CourseSettingsLogic.canManageCourse(courseCode: "C-2", permissions: perms))
    }

    func testValidateTitleRequired() {
        let error = CourseSettingsLogic.validateGeneralForm(
            title: "  ",
            courseHomeLanding: .data,
            courseHomeContentItemId: ""
        )
        XCTAssertNotNil(error?.title)
    }

    func testValidateContentPageRequired() {
        let error = CourseSettingsLogic.validateGeneralForm(
            title: "Course",
            courseHomeLanding: .content_page,
            courseHomeContentItemId: ""
        )
        XCTAssertNotNil(error?.courseHome)
    }

    func testIsoDurationRoundTrip() {
        let parts = CourseSettingsLogic.isoDurationToParts(iso: "P3M")
        XCTAssertEqual(parts.amount, "3")
        XCTAssertEqual(parts.unit, .M)
        XCTAssertEqual(CourseSettingsLogic.partsToIsoDuration(amount: "3", unit: .M), "P3M")
    }

    func testHeroPositionFormatCenterIsNil() {
        XCTAssertNil(CourseSettingsLogic.formatHeroObjectPosition(x: 50, y: 50))
        XCTAssertEqual(CourseSettingsLogic.formatHeroObjectPosition(x: 30, y: 70), "30% 70%")
    }

    func testDirtyDetectionTitleChange() {
        var course = sampleCourse()
        var form = CourseSettingsLogic.applyCourseToForm(course)
        XCTAssertFalse(CourseSettingsLogic.isGeneralFormDirty(form: form, course: course))
        form.title = "Renamed"
        XCTAssertTrue(CourseSettingsLogic.isGeneralFormDirty(form: form, course: course))
    }

    private func sampleCourse() -> CourseSummary {
        CourseSummary(
            id: "1",
            courseCode: "C-1",
            title: "Intro",
            description: "Desc",
            published: false
        )
    }
}
