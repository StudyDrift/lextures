import Foundation
import UserNotifications

/// Daily local reminder to complete spaced-repetition reviews (M8.1 / M0.1 hook).
enum ReviewReminderScheduler {
    private static let enabledKey = "lextures.review.reminder.enabled"
    private static let hourKey = "lextures.review.reminder.hour"
    private static let minuteKey = "lextures.review.reminder.minute"
    private static let notificationId = "review-daily-reminder"

    static var isEnabled: Bool {
        get { UserDefaults.standard.bool(forKey: enabledKey) }
        set { UserDefaults.standard.set(newValue, forKey: enabledKey) }
    }

    static var reminderHour: Int {
        get {
            let stored = UserDefaults.standard.object(forKey: hourKey) as? Int
            return stored ?? 18
        }
        set { UserDefaults.standard.set(newValue, forKey: hourKey) }
    }

    static var reminderMinute: Int {
        get {
            let stored = UserDefaults.standard.object(forKey: minuteKey) as? Int
            return stored ?? 0
        }
        set { UserDefaults.standard.set(newValue, forKey: minuteKey) }
    }

    static func formattedTime() -> String {
        var components = DateComponents()
        components.hour = reminderHour
        components.minute = reminderMinute
        let date = Calendar.current.date(from: components) ?? Date()
        return date.formatted(date: .omitted, time: .shortened)
    }

    static func reschedule(dueCount: Int) async {
        await cancel()
        guard isEnabled, dueCount > 0 else { return }

        let content = UNMutableNotificationContent()
        content.title = L.text("mobile.review.reminder.title")
        content.body = dueCount > 0
            ? L.plural("mobile.review.reminder.body", count: dueCount)
            : L.text("mobile.review.reminder.bodyDefault")
        content.sound = .default
        content.userInfo = ["action_url": "/review"]

        var components = DateComponents()
        components.hour = reminderHour
        components.minute = reminderMinute
        let trigger = UNCalendarNotificationTrigger(dateMatching: components, repeats: true)
        let request = UNNotificationRequest(identifier: notificationId, content: content, trigger: trigger)
        try? await UNUserNotificationCenter.current().add(request)
    }

    static func cancel() async {
        UNUserNotificationCenter.current()
            .removePendingNotificationRequests(withIdentifiers: [notificationId])
    }
}
