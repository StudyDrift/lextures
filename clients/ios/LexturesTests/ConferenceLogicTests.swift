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
}
