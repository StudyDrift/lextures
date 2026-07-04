import XCTest
@testable import Lextures

final class LiveMeetingsLogicTests: XCTestCase {
    func testGroupMeetingsPartitionsByStatus() {
        let meetings = [
            makeMeeting(id: "1", status: "live"),
            makeMeeting(id: "2", status: "scheduled"),
            makeMeeting(id: "3", status: "ended"),
        ]
        let grouped = LiveMeetingsLogic.groupMeetings(meetings)
        XCTAssertEqual(grouped.live.map(\.id), ["1"])
        XCTAssertEqual(grouped.upcoming.map(\.id), ["2"])
        XCTAssertEqual(grouped.past.map(\.id), ["3"])
    }

    func testIsLiveOrSoonWithinThirtyMinutes() {
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let soon = makeMeeting(
            id: "soon",
            status: "scheduled",
            start: "2099-06-01T15:10:00Z"
        )
        let later = makeMeeting(
            id: "later",
            status: "scheduled",
            start: "2099-06-01T17:00:00Z"
        )
        XCTAssertTrue(LiveMeetingsLogic.isLiveOrSoon(soon, now: now))
        XCTAssertFalse(LiveMeetingsLogic.isLiveOrSoon(later, now: now))
    }

    func testCalendarEventsFromScheduledMeetings() {
        let course = CourseSummary(
            id: "1",
            courseCode: "BIO101",
            title: "Biology",
            description: "",
            heroImageUrl: nil,
            startsAt: nil,
            endsAt: nil,
            published: true,
            catalogNickname: nil,
            notebookEnabled: true,
            calendarEnabled: true,
            officeHoursEnabled: false,
            orgId: nil,
            termId: nil,
            viewerEnrollmentRoles: ["student"],
            liveSessionsEnabled: true
        )
        let meeting = makeMeeting(
            id: "m1",
            status: "scheduled",
            start: "2099-06-01T15:00:00Z",
            end: "2099-06-01T16:00:00Z"
        )
        let events = LiveMeetingsLogic.collectCalendarEvents(
            studentCourses: [course],
            meetingsByCourseCode: [course.courseCode: [meeting]]
        )
        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].kind, .liveMeeting)
        XCTAssertEqual(events[0].meetingId, "m1")
    }

    func testCollectLiveAndUpcomingPrioritizesLive() {
        let course = CourseSummary(
            id: "1",
            courseCode: "BIO101",
            title: "Biology",
            description: "",
            heroImageUrl: nil,
            startsAt: nil,
            endsAt: nil,
            published: true,
            catalogNickname: nil,
            notebookEnabled: true,
            calendarEnabled: true,
            officeHoursEnabled: false,
            orgId: nil,
            termId: nil,
            viewerEnrollmentRoles: ["student"],
            liveSessionsEnabled: true
        )
        let live = makeMeeting(id: "live", status: "live", start: "2099-06-02T15:00:00Z")
        let soon = makeMeeting(id: "soon", status: "scheduled", start: "2099-06-01T15:05:00Z")
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let items = LiveMeetingsLogic.collectLiveAndUpcoming(
            courses: [course],
            meetingsByCourseCode: [course.courseCode: [live, soon]],
            now: now
        )
        XCTAssertEqual(items.count, 2)
        XCTAssertEqual(items[0].meeting.id, "live")
    }

    private func makeMeeting(
        id: String,
        status: String,
        start: String = "2099-06-01T15:00:00Z",
        end: String? = "2099-06-01T16:00:00Z"
    ) -> VirtualMeeting {
        VirtualMeeting(
            id: id,
            courseId: "c1",
            sectionId: nil,
            provider: "jitsi",
            title: "Class",
            scheduledStart: start,
            scheduledEnd: end,
            joinUrl: "https://example.com/join",
            hostUrl: nil,
            externalMeetingId: nil,
            status: status,
            createdBy: "u1",
            createdAt: "2099-01-01T00:00:00Z"
        )
    }
}