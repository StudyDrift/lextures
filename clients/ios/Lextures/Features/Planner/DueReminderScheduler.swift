import Foundation
import UserNotifications

/// Schedules optional local due-date reminders (M2.1 / M0.1 hook).
enum DueReminderScheduler {
    private static let defaultsKey = "lextures.planner.reminderLeadMinutes"

    static var selectedLeadTime: DueReminderLeadTime {
        get {
            let raw = UserDefaults.standard.integer(forKey: defaultsKey)
            return DueReminderLeadTime(rawValue: raw) ?? .none
        }
        set {
            UserDefaults.standard.set(newValue.rawValue, forKey: defaultsKey)
        }
    }

    static func notificationIdentifier(for itemKey: String) -> String {
        "due-reminder:\(itemKey)"
    }

    static func scheduleReminder(for item: StudentTodoItem) async {
        let lead = selectedLeadTime
        await cancelReminder(for: item.key)
        guard lead != .none, let due = item.dueAt else { return }
        let fireDate = due.addingTimeInterval(-Double(lead.rawValue * 60))
        guard fireDate > Date() else { return }

        let content = UNMutableNotificationContent()
        content.title = L.text("mobile.planner.reminder.title")
        content.body = "\(item.title) · \(item.courseTitle)"
        content.sound = .default
        if let structureId = item.structureItemId, item.kind == .dueItem {
            content.userInfo = [
                "action_url": "/courses/\(item.courseCode)/modules/\(item.structureKind ?? "content")/\(structureId)",
            ]
        }

        let components = Calendar.current.dateComponents(
            [.year, .month, .day, .hour, .minute],
            from: fireDate
        )
        let trigger = UNCalendarNotificationTrigger(dateMatching: components, repeats: false)
        let request = UNNotificationRequest(
            identifier: notificationIdentifier(for: item.key),
            content: content,
            trigger: trigger
        )
        try? await UNUserNotificationCenter.current().add(request)
    }

    static func cancelReminder(for itemKey: String) async {
        UNUserNotificationCenter.current()
            .removePendingNotificationRequests(withIdentifiers: [notificationIdentifier(for: itemKey)])
    }
}
