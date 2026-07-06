import XCTest
@testable import Lextures

final class ConferenceLogicTests: XCTestCase {
    func testUpcomingAvailableSlotsFiltersBookedAndPast() {
        let future = ISO8601DateFormatter().string(from: Date().addingTimeInterval(3600))
        let past = ISO8601DateFormatter().string(from: Date().addingTimeInterval(-3600))
        let slots = [
            ConferenceSlot(id: "1", availabilityId: "a", startAt: future, endAt: future, status: "open"),
            ConferenceSlot(id: "2", availabilityId: "a", startAt: past, endAt: past, status: "open"),
            ConferenceSlot(id: "3", availabilityId: "a", startAt: future, endAt: future, status: "booked"),
        ]
        XCTAssertEqual(ConferenceLogic.upcomingAvailableSlots(slots).map(\.id), ["1"])
    }

    func testMyBookedSlotsMatchesChild() {
        let future = ISO8601DateFormatter().string(from: Date().addingTimeInterval(3600))
        let slots = [
            ConferenceSlot(
                id: "1",
                availabilityId: "a",
                startAt: future,
                endAt: future,
                status: "booked",
                bookedByParent: "p1",
                bookedForChild: "child1"
            ),
            ConferenceSlot(
                id: "2",
                availabilityId: "a",
                startAt: future,
                endAt: future,
                status: "booked",
                bookedByParent: "p1",
                bookedForChild: "child2"
            ),
        ]
        XCTAssertEqual(ConferenceLogic.myBookedSlots(slots, parentId: "p1", studentId: "child1").map(\.id), ["1"])
    }

    func testCalendarEventsFromBookings() {
        let future = ISO8601DateFormatter().string(from: Date().addingTimeInterval(3600))
        let end = ISO8601DateFormatter().string(from: Date().addingTimeInterval(5400))
        let booking = ParentConferenceBooking(
            slot: ConferenceSlot(
                id: "s1",
                availabilityId: "a",
                startAt: future,
                endAt: end,
                status: "booked",
                bookedForChild: "child1"
            ),
            teacher: ConferenceTeacher(teacherId: "t1", displayName: "Ms. Lee"),
            studentId: "child1",
            childName: "Alex",
            availability: ConferenceAvailability(
                id: "a",
                location: "Room 12",
                videoLink: "https://meet.example.com/abc"
            )
        )
        let events = ConferenceLogic.calendarEvents(from: [booking])
        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].kind, .conference)
        XCTAssertEqual(events[0].conferenceSlotId, "s1")
        XCTAssertEqual(events[0].videoLink, "https://meet.example.com/abc")
        XCTAssertEqual(events[0].courseTitle, "Alex")
    }

    func testLocationLabelVirtual() {
        let availability = ConferenceAvailability(id: "a", location: "Room 5", videoLink: "https://meet.example.com")
        XCTAssertNotNil(ConferenceLogic.locationLabel(availability: availability))
    }
}
