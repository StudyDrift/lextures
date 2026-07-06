import XCTest
@testable import Lextures

final class AnnouncementLogicTests: XCTestCase {
    private func course(staff: Bool, feed: Bool = true) -> CourseSummary {
        CourseSummary(
            id: "c1",
            courseCode: "BIO101",
            title: "Biology",
            description: "",
            orgId: "org-1",
            viewerEnrollmentRoles: staff ? ["teacher"] : ["student"],
            feedEnabled: feed
        )
    }

    func testCanComposeCourseAnnouncementRequiresStaffAndFeed() {
        XCTAssertTrue(AnnouncementLogic.canComposeCourseAnnouncement(course: course(staff: true)))
        XCTAssertFalse(AnnouncementLogic.canComposeCourseAnnouncement(course: course(staff: false)))
        XCTAssertFalse(AnnouncementLogic.canComposeCourseAnnouncement(course: course(staff: true, feed: false)))
    }

    func testCanComposeBroadcastRequiresFeatureAndPermission() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(AnnouncementLogic.canComposeBroadcast(permissions: [], features: features))

        features.ffBroadcasts = true
        XCTAssertFalse(AnnouncementLogic.canComposeBroadcast(permissions: [], features: features))

        XCTAssertTrue(
            AnnouncementLogic.canComposeBroadcast(
                permissions: [AnnouncementLogic.orgBroadcastManagePermission],
                features: features
            )
        )
        XCTAssertTrue(
            AnnouncementLogic.canComposeBroadcast(
                permissions: [AnnouncementLogic.globalAdminPermission],
                features: features
            )
        )
    }

    func testAnnouncementsChannelId() {
        let channels = [
            FeedChannel(id: "1", name: "general", sortOrder: 0, createdAt: ""),
            FeedChannel(id: "2", name: "Announcements", sortOrder: 1, createdAt: ""),
        ]
        XCTAssertEqual(AnnouncementLogic.announcementsChannelId(channels: channels), "2")
    }

    func testFormatAnnouncementBody() {
        let body = AnnouncementLogic.formatAnnouncementBody(
            title: "Snow day",
            body: "No class tomorrow.",
            sectionName: "Period 2",
            mentionsEveryone: true
        )
        XCTAssertTrue(body.contains("**Snow day**"))
        XCTAssertTrue(body.contains("No class tomorrow."))
        XCTAssertTrue(body.contains("Period 2"))
        XCTAssertTrue(body.contains("@everyone"))
    }

    func testAudienceLabel() {
        let whole = AnnouncementLogic.audienceLabel(
            course: course(staff: true),
            audience: .wholeCourse,
            sectionName: nil
        )
        XCTAssertEqual(whole, "Biology")

        let section = AnnouncementLogic.audienceLabel(
            course: course(staff: true),
            audience: .section,
            sectionName: "Period 2"
        )
        XCTAssertEqual(section, "Biology · Period 2")
    }
}