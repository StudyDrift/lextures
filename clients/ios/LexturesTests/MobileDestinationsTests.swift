import XCTest
@testable import Lextures

final class MobileDestinationsTests: XCTestCase {
    func testInstructorShellTabsForegroundTeach() {
        let tabs = MobileDestinations.shellTabs(context: .teaching)
        XCTAssertEqual(tabs, [.home, .courses, .teach, .inbox, .profile])
    }

    func testParentShellTabsScopeChildren() {
        let tabs = MobileDestinations.shellTabs(context: .parent)
        XCTAssertEqual(tabs, [.home, .children, .calendar, .inbox, .profile])
        XCTAssertFalse(tabs.contains(.notebooks))
    }

    func testStudentShellTabsUnchangedShape() {
        let tabs = MobileDestinations.shellTabs(context: .learning)
        XCTAssertEqual(tabs, [.home, .courses, .notebooks, .inbox, .profile])
    }

    func testCourseWorkspaceHidesAttendanceForStaffWhenDisabled() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["teacher"],
            attendanceEnabled: false
        )
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course, hasAttendanceSessions: true)
        )
        XCTAssertFalse(sections.contains(.attendance))
    }

    func testCourseWorkspaceHidesDisabledFeatures() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"],
            feedEnabled: true,
            discussionsEnabled: false,
            liveSessionsEnabled: true,
            filesEnabled: true,
            attendanceEnabled: false
        )
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course, hasAttendanceSessions: false)
        )
        XCTAssertTrue(sections.contains(.feed))
        XCTAssertTrue(sections.contains(.live))
        XCTAssertFalse(sections.contains(.discussions))
        XCTAssertFalse(sections.contains(.library))
    }

    func testCourseWorkspaceShowsLibraryWhenResourcesPresent() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"],
            feedEnabled: false,
            discussionsEnabled: false,
            liveSessionsEnabled: false,
            filesEnabled: false,
            attendanceEnabled: false
        )
        var features = MobilePlatformFeatures()
        features.ffLibrary = true
        features.ffMobileLibraryEreserves = true
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(
                course: course,
                hasLibraryResources: true,
                platformFeatures: features
            )
        )
        XCTAssertTrue(sections.contains(.library))
    }

    func testRoleSnapshotMultiContextResolution() {
        let snapshot = RoleSnapshot(
            hasStudentEnrollment: true,
            hasStaffEnrollment: true,
            hasParentDashboard: false,
            hasSelfPacedEnrollment: false
        )
        XCTAssertEqual(snapshot.availableContexts, [.teaching, .learning])
        XCTAssertEqual(snapshot.resolvedContext(stored: .teaching), .teaching)
        XCTAssertEqual(snapshot.resolvedContext(stored: .parent), .teaching)
    }

    func testChipOverflowSplit() {
        let sections: [CourseWorkspaceSection] = [
            .overview, .modules, .files, .grades, .discussions, .feed, .live,
        ]
        let split = MobileDestinations.splitCourseChips(sections)
        XCTAssertEqual(split.visible.count, 6)
        XCTAssertEqual(split.overflow, [.live])
    }

    func testDeepLinkMapsToWorkspaceSection() {
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .feed), .feed)
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .attendance), .attendance)
    }
}