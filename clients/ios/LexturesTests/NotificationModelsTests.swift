import XCTest
@testable import Lextures

final class NotificationModelsTests: XCTestCase {
    func testCategoryMapping() {
        XCTAssertEqual(NotificationLogic.category(for: "grade_posted"), .grades)
        XCTAssertEqual(NotificationLogic.category(for: "discussion_reply"), .discussions)
        XCTAssertEqual(NotificationLogic.category(for: "inbox_message"), .messages)
        XCTAssertEqual(NotificationLogic.category(for: "unknown_event"), .other)
    }

    func testFilterUnreadAndGrades() {
        let notifications = [
            makeNotification(id: "1", eventType: "grade_posted", isRead: false),
            makeNotification(id: "2", eventType: "discussion_reply", isRead: true),
            makeNotification(id: "3", eventType: "grade_posted", isRead: true),
        ]
        XCTAssertEqual(NotificationLogic.filter(notifications, by: .unread).map(\.id), ["1"])
        XCTAssertEqual(NotificationLogic.filter(notifications, by: .grades).map(\.id), ["1", "3"])
        XCTAssertEqual(NotificationLogic.filter(notifications, by: .discussions).map(\.id), ["2"])
    }

    func testPushPreferenceGating() {
        let preferences = [
            NotificationPreference(eventType: "discussion_reply", pushEnabled: false),
            NotificationPreference(eventType: "grade_posted", pushEnabled: true),
        ]
        XCTAssertFalse(NotificationLogic.isPushEnabled(eventType: "discussion_reply", preferences: preferences))
        XCTAssertTrue(NotificationLogic.isPushEnabled(eventType: "grade_posted", preferences: preferences))
        XCTAssertTrue(NotificationLogic.isPushEnabled(eventType: "missing", preferences: preferences))
    }

    func testGroupedPreferencesSortsWithinCategory() {
        let preferences = [
            NotificationPreference(eventType: "discussion_reply"),
            NotificationPreference(eventType: "grade_posted"),
            NotificationPreference(eventType: "assignment_created"),
        ]
        let grouped = NotificationLogic.groupedPreferences(preferences)
        XCTAssertEqual(grouped.map(\.0), [.grades, .assignments, .discussions])
        XCTAssertEqual(grouped.first?.1.map(\.eventType), ["grade_posted"])
    }

    private func makeNotification(id: String, eventType: String, isRead: Bool) -> AppNotification {
        AppNotification(
            id: id,
            eventType: eventType,
            title: "Title",
            body: "Body",
            actionUrl: nil,
            isRead: isRead,
            createdAt: "2026-06-30T12:00:00Z"
        )
    }
}
