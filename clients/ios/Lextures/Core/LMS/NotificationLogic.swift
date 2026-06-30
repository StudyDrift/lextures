import Foundation

// MARK: - Categories & filters

enum NotificationCategory: String, CaseIterable, Identifiable, Hashable {
    case grades
    case assignments
    case discussions
    case announcements
    case messages
    case reminders
    case account
    case courses
    case other

    var id: String { rawValue }
}

enum NotificationFilter: String, CaseIterable, Identifiable, Hashable {
    case all
    case unread
    case grades
    case assignments
    case discussions
    case announcements
    case messages
    case reminders

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .all: return "mobile.notifications.filter.all"
        case .unread: return "mobile.notifications.filter.unread"
        case .grades: return "mobile.notifications.category.grades"
        case .assignments: return "mobile.notifications.category.assignments"
        case .discussions: return "mobile.notifications.category.discussions"
        case .announcements: return "mobile.notifications.category.announcements"
        case .messages: return "mobile.notifications.category.messages"
        case .reminders: return "mobile.notifications.category.reminders"
        }
    }

    var localizedLabel: String {
        switch self {
        case .all: return L.text("mobile.notifications.filter.all")
        case .unread: return L.text("mobile.notifications.filter.unread")
        case .grades: return L.text("mobile.notifications.category.grades")
        case .assignments: return L.text("mobile.notifications.category.assignments")
        case .discussions: return L.text("mobile.notifications.category.discussions")
        case .announcements: return L.text("mobile.notifications.category.announcements")
        case .messages: return L.text("mobile.notifications.category.messages")
        case .reminders: return L.text("mobile.notifications.category.reminders")
        }
    }
}

enum NotificationLogic {
    static func category(for eventType: String) -> NotificationCategory {
        switch eventType {
        case "grade_posted":
            return .grades
        case "assignment_created", "assignment_due_reminder", "submission_received",
             "incomplete_granted", "incomplete_reminder":
            return .assignments
        case "discussion_reply":
            return .discussions
        case "course_announcement", "meeting_reminder", "conference_confirmed",
             "conference_reminder", "coaching_tip_weekly":
            return .announcements
        case "inbox_message":
            return .messages
        case "study_reminder_daily", "study_reminder_streak_at_risk", "study_reminder_weekly_summary":
            return .reminders
        case "password_reset", "welcome_invite", "payment_failed", "ceu_awarded", "certificate_issued":
            return .account
        case "canvas_course_imported", "course_copy_imported", "course_copy_import_failed":
            return .courses
        default:
            return .other
        }
    }

    static func matchesFilter(_ notification: AppNotification, filter: NotificationFilter) -> Bool {
        switch filter {
        case .all:
            return true
        case .unread:
            return !notification.isRead
        case .grades:
            return category(for: notification.eventType) == .grades
        case .assignments:
            return category(for: notification.eventType) == .assignments
        case .discussions:
            return category(for: notification.eventType) == .discussions
        case .announcements:
            return category(for: notification.eventType) == .announcements
        case .messages:
            return category(for: notification.eventType) == .messages
        case .reminders:
            return category(for: notification.eventType) == .reminders
        }
    }

    static func filter(_ notifications: [AppNotification], by filter: NotificationFilter) -> [AppNotification] {
        notifications.filter { matchesFilter($0, filter: filter) }
    }

    static func eventLabelKey(for eventType: String) -> String {
        "mobile.notifications.event.\(eventType)"
    }

    static func eventLabel(for eventType: String) -> String {
        let key = eventLabelKey(for: eventType)
        let localized = String(
            localized: String.LocalizationValue(stringLiteral: key),
            locale: LocalePreferences.effectiveLocaleValue()
        )
        return localized == key ? eventType.replacingOccurrences(of: "_", with: " ") : localized
    }

    static func categoryLabel(for category: NotificationCategory) -> String {
        switch category {
        case .grades: return L.text("mobile.notifications.category.grades")
        case .assignments: return L.text("mobile.notifications.category.assignments")
        case .discussions: return L.text("mobile.notifications.category.discussions")
        case .announcements: return L.text("mobile.notifications.category.announcements")
        case .messages: return L.text("mobile.notifications.category.messages")
        case .reminders: return L.text("mobile.notifications.category.reminders")
        case .account: return L.text("mobile.notifications.category.account")
        case .courses: return L.text("mobile.notifications.category.courses")
        case .other: return L.text("mobile.notifications.category.other")
        }
    }

    static func groupedPreferences(_ preferences: [NotificationPreference]) -> [(NotificationCategory, [NotificationPreference])] {
        let grouped = Dictionary(grouping: preferences) { category(for: $0.eventType) }
        return NotificationCategory.allCases.compactMap { category in
            guard let rows = grouped[category], !rows.isEmpty else { return nil }
            return (category, rows.sorted { $0.eventType < $1.eventType })
        }
    }

    static func isPushEnabled(eventType: String, preferences: [NotificationPreference]) -> Bool {
        preferences.first(where: { $0.eventType == eventType })?.pushEnabled ?? true
    }

    static func isEmailEnabled(eventType: String, preferences: [NotificationPreference]) -> Bool {
        preferences.first(where: { $0.eventType == eventType })?.emailEnabled ?? true
    }
}

// MARK: - Preferences cache (push gating + offline reads)

enum NotificationPreferencesCache {
    private static let storageKey = "notification_preferences_cache"

    static func save(_ preferences: [NotificationPreference], ownerKey: String) {
        guard let data = try? JSONEncoder().encode(preferences) else { return }
        UserDefaults.standard.set(data, forKey: key(for: ownerKey))
    }

    static func load(ownerKey: String) -> [NotificationPreference] {
        guard let data = UserDefaults.standard.data(forKey: key(for: ownerKey)),
              let decoded = try? JSONDecoder().decode([NotificationPreference].self, from: data) else {
            return []
        }
        return decoded
    }

    static func clear(ownerKey: String) {
        UserDefaults.standard.removeObject(forKey: key(for: ownerKey))
    }

    private static func key(for ownerKey: String) -> String {
        "\(storageKey).\(ownerKey)"
    }
}
