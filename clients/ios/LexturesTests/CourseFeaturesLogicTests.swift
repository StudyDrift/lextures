import XCTest
@testable import Lextures

final class CourseFeaturesLogicTests: XCTestCase {
    func testIsEnabledDefaults() {
        let course = sampleCourse()
        XCTAssertTrue(CourseFeaturesLogic.isEnabled(.notebook, course: course))
        XCTAssertTrue(CourseFeaturesLogic.isEnabled(.feed, course: course))
        XCTAssertTrue(CourseFeaturesLogic.isEnabled(.calendar, course: course))
        XCTAssertFalse(CourseFeaturesLogic.isEnabled(.discussions, course: course))
    }

    func testApplyToggleUpdatesCourse() {
        let course = sampleCourse()
        let updated = CourseFeaturesLogic.applyToggle(course: course, tool: .discussions, enabled: true)
        XCTAssertTrue(CourseFeaturesLogic.isEnabled(.discussions, course: updated))
    }

    func testBuildFeaturesPatchReflectsCourse() {
        var course = sampleCourse()
        course.discussionsEnabled = true
        course.sectionsEnabled = true
        let patch = CourseFeaturesLogic.buildFeaturesPatch(from: course)
        XCTAssertTrue(patch.discussionsEnabled)
        XCTAssertTrue(patch.sectionsEnabled)
        XCTAssertTrue(patch.notebookEnabled)
    }

    func testShouldConfirmDisableOnlyWhenEnabled() {
        XCTAssertTrue(CourseFeaturesLogic.shouldConfirmDisable(.sections, currentlyEnabled: true))
        XCTAssertFalse(CourseFeaturesLogic.shouldConfirmDisable(.sections, currentlyEnabled: false))
    }

    func testConsortiumGating() {
        var features = MobilePlatformFeatures()
        features.ffConsortiumSharing = false
        XCTAssertFalse(CourseFeaturesLogic.consortiumSectionEnabled(features))
        features.ffConsortiumSharing = true
        XCTAssertTrue(CourseFeaturesLogic.consortiumSectionEnabled(features))
    }

    func testVideoCaptionsGating() {
        var features = MobilePlatformFeatures()
        features.videoCaptionsEnabled = false
        XCTAssertFalse(CourseFeaturesLogic.videoCaptionsSectionEnabled(features))
        features.videoCaptionsEnabled = true
        XCTAssertTrue(CourseFeaturesLogic.videoCaptionsSectionEnabled(features))
    }

    func testFilterToolsEmptyQueryReturnsAll() {
        XCTAssertEqual(
            CourseFeaturesLogic.filterTools(CourseFeaturesLogic.allToolRows, query: "").count,
            CourseFeaturesLogic.allToolRows.count
        )
    }

    private func sampleCourse() -> CourseSummary {
        CourseSummary(
            id: "1",
            courseCode: "C-1",
            title: "Intro",
            description: "Desc",
            notebookEnabled: true,
            calendarEnabled: true,
            feedEnabled: true,
            discussionsEnabled: false
        )
    }
}
