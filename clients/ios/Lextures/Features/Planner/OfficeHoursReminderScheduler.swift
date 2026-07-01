import Foundation
import UserNotifications

/// Schedules local reminders for booked office-hours appointments (M7.3 / M0.1).
enum OfficeHoursReminderScheduler {
    static func notificationIdentifier(for slotId: String) -> String {
        "office-hours-reminder:\(slotId)"
    }

    static func scheduleReminder(
        slot: AppointmentSlot,
        courseCode: String,
        courseTitle: String
    ) async {
        let lead = DueReminderScheduler.selectedLeadTime
        await cancelReminder(for: slot.id)
        guard lead != .none, let start = LMSDates.parse(slot.slotStart) else { return }
        let fireDate = start.addingTimeInterval(-Double(lead.rawValue * 60))
        guard fireDate > Date() else { return }

        let content = UNMutableNotificationContent()
        content.title = L.text("mobile.officeHours.reminder.title")
        content.body = "\(L.text("mobile.officeHours.calendarTitle")) · \(courseTitle)"
        content.sound = .default
        content.userInfo = ["action_url": "/courses/\(courseCode)/office-hours"]

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
