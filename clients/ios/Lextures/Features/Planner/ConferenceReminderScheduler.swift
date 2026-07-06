import Foundation
import UserNotifications

/// Schedules local reminders for booked parent–teacher conferences (M10.2 / M0.1).
enum ConferenceReminderScheduler {
    static func notificationIdentifier(for slotId: String) -> String {
        "conference-reminder:\(slotId)"
    }

    static func scheduleReminder(
        slot: ConferenceSlot,
        teacherName: String,
        childName: String
    ) async {
        let lead = DueReminderScheduler.selectedLeadTime
        await cancelReminder(for: slot.id)
        guard lead != .none, let start = LMSDates.parse(slot.startAt) else { return }
        let fireDate = start.addingTimeInterval(-Double(lead.rawValue * 60))
        guard fireDate > Date() else { return }

        let content = UNMutableNotificationContent()
        content.title = L.text("mobile.parent.conferences.reminder.title")
        content.body = L.format(
            "mobile.parent.conferences.reminder.body",
            teacherName,
            childName
        )
        content.sound = .default
        content.userInfo = ["action_url": "/parent/conferences"]

        let components = Calendar.current.dateComponents(
            [.year, .month, .day, .hour, .minute],
            from: fireDate
        )
        let trigger = UNCalendarNotificationTrigger(dateMatching: components, repeats: false)
        let request = UNNotificationRequest(
            identifier: notificationIdentifier(for: slot.id),
            content: content,
            trigger: trigger
        )
        try? await UNUserNotificationCenter.current().add(request)
    }

    static func cancelReminder(for slotId: String) async {
        UNUserNotificationCenter.current()
            .removePendingNotificationRequests(withIdentifiers: [notificationIdentifier(for: slotId)])
    }
}
