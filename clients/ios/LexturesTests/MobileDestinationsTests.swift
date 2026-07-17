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

    func testCourseWorkspaceShowsGroupsAndCollabDocsWhenEnabled() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"],
            groupSpacesEnabled: true,
            collabDocsEnabled: true
        )
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course)
        )
        XCTAssertTrue(sections.contains(.groups))
        XCTAssertTrue(sections.contains(.collabDocs))
    }

    func testCourseWorkspaceShowsBoardsWhenVisualBoardsEnabled() {
        let on = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"],
            visualBoardsEnabled: true
        )
        let off = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"],
            visualBoardsEnabled: false
        )
        XCTAssertTrue(
            MobileDestinations.courseWorkspaceSections(CourseWorkspaceContext(course: on)).contains(.boards)
        )
        XCTAssertFalse(
            MobileDestinations.courseWorkspaceSections(CourseWorkspaceContext(course: off)).contains(.boards)
        )
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .boards), .boards)
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
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .behavior), .behavior)
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .hallPass), .hallPass)
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .insights), .instructorInsights)
        XCTAssertEqual(CourseWorkspaceSection.from(deepLink: .boards), .boards)
        guard case let .course(_, section, itemId) = DeepLinkRouter.resolve("/courses/cs101/boards/board-1") else {
            return XCTFail("expected course deep link")
        }
        XCTAssertEqual(section, .boards)
        XCTAssertEqual(itemId, "board-1")
    }

    func testCourseWorkspaceShowsInsightsForStaffWhenEnabled() {
        var features = MobilePlatformFeatures()
        features.atRiskAlertsEnabled = true
        let course = CourseSummary(
            id: "1", courseCode: "demo", title: "Demo", description: "",
            viewerEnrollmentRoles: ["teacher"]
        )
        let ctx = CourseWorkspaceContext(course: course, platformFeatures: features)
        let sections = MobileDestinations.courseWorkspaceSections(ctx)
        XCTAssertTrue(sections.contains(.instructorInsights))
    }

    func testCourseWorkspaceShowsBehaviorForStaffWhenClassroomSignalsOn() {
        var features = MobilePlatformFeatures()
        features.ffClassroomSignals = true
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["teacher"],
            sectionsEnabled: true
        )
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course, platformFeatures: features)
        )
        XCTAssertTrue(sections.contains(.behavior))
        XCTAssertTrue(sections.contains(.hallPass))
    }
}