import XCTest
@testable import Lextures

final class OfficeHoursLogicTests: XCTestCase {
    func testUpcomingAvailableSlotsFiltersPastAndBooked() {
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let slots = [
            makeSlot(id: "past", start: "2023-01-01T15:00:00Z", status: "available"),
            makeSlot(id: "open", start: "2099-06-01T15:00:00Z", status: "available"),
            makeSlot(id: "taken", start: "2099-06-02T15:00:00Z", status: "booked"),
        ]
        let upcoming = OfficeHoursLogic.upcomingAvailableSlots(slots, now: now)
        XCTAssertEqual(upcoming.map(\.id), ["open"])
    }

    func testMyBookedSlotsRequiresStudentId() {
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let slots = [
            makeSlot(id: "mine", start: "2099-06-01T15:00:00Z", status: "booked", studentId: "u1"),
            makeSlot(id: "other", start: "2099-06-02T15:00:00Z", status: "booked"),
        ]
        let mine = OfficeHoursLogic.myBookedSlots(slots, now: now)
        XCTAssertEqual(mine.map(\.id), ["mine"])
    }

    func testCalendarEventsFromBookings() {
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
            officeHoursEnabled: true,
            orgId: nil,
            termId: nil,
            viewerEnrollmentRoles: ["student"]
        )
        let window = AvailabilityWindow(
            id: "w1",
            instructorId: "i1",
            courseId: course.id,
            dayOfWeek: 2,
            windowDate: nil,
            startTime: "15:00",
            endTime: "16:00",
            slotDurationMinutes: 15,
            location: "Room 101",
            isVirtual: false,
            status: "active",
            createdAt: nil
        )
        let slot = makeSlot(id: "s1", start: "2099-06-01T15:00:00Z", status: "booked", studentId: "u1")
        let events = OfficeHoursLogic.collectCalendarEvents(
            studentCourses: [course],
            availabilityByCourseCode: [
                course.courseCode: OfficeHoursAvailability(windows: [window], slots: [slot]),
            ]
        )
        XCTAssertEqual(events.count, 1)
        XCTAssertEqual(events[0].kind, .officeHours)
        XCTAssertEqual(events[0].officeHoursSlotId, "s1")
    }

    private func makeSlot(
        id: String,
        start: String,
        status: String,
        studentId: String? = nil
    ) -> AppointmentSlot {
        AppointmentSlot(
            id: id,
            windowId: "w1",
            slotStart: start,
            slotEnd: "2099-06-01T15:15:00Z",
            studentId: studentId,
            studentNote: nil,
            meetingId: nil,
            status: status,
            bookedAt: nil
        )
    }
}
