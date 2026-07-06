import Foundation

enum BehaviorAwardMode: String, CaseIterable {
    case award
    case referral
}

struct HallPassCountdown: Equatable {
    var remainingSeconds: Int
    var isExpired: Bool
    var isOverdue: Bool
}

enum BehaviorLogic {
    static let studentRoles: Set<String> = ["student", "learner"]
    static let hallPassDestinations = ["bathroom", "office", "library", "nurse", "other"]
    static let defaultPassMinutes = 5

    static func studentRoster(from enrollments: [CourseEnrollment]) -> [CourseEnrollment] {
        enrollments.filter { studentRoles.contains($0.role.lowercased()) }
    }

    static func studentLabel(_ enrollment: CourseEnrollment) -> String {
        let name = enrollment.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return "Student"
    }

    static func activeCategories(_ categories: [BehaviorCategory]) -> [BehaviorCategory] {
        categories.filter(\.active)
    }

    static func positiveCategories(_ categories: [BehaviorCategory]) -> [BehaviorCategory] {
        activeCategories(categories).filter(\.isPositive)
    }

    static func negativeCategories(_ categories: [BehaviorCategory]) -> [BehaviorCategory] {
        activeCategories(categories).filter(\.isNegative)
    }

    static func awardPayload(
        studentIds: Set<String>,
        categoryId: String,
        note: String?
    ) -> [PBISAwardInput] {
        let trimmedNote = note?.trimmingCharacters(in: .whitespacesAndNewlines)
        return studentIds.map { studentId in
            PBISAwardInput(
                studentId: studentId,
                categoryId: categoryId,
                points: 1,
                note: trimmedNote?.isEmpty == false ? trimmedNote : nil
            )
        }
    }

    static func isActiveHallPass(_ pass: HallPass) -> Bool {
        let status = pass.status.lowercased()
        return status == "requested" || status == "approved"
    }

    static func hallPassCountdown(
        pass: HallPass,
        now: Date = Date()
    ) -> HallPassCountdown? {
        guard pass.status.lowercased() == "approved" else { return nil }
        guard let approvedAt = parseDate(pass.approvedAt) else { return nil }
        let estimated = pass.estimatedMins ?? defaultPassMinutes
        let deadline = approvedAt.addingTimeInterval(TimeInterval(estimated * 60))
        let remaining = Int(deadline.timeIntervalSince(now))
        let overdue = pass.overdue == true || remaining <= 0
        return HallPassCountdown(
            remainingSeconds: max(remaining, 0),
            isExpired: remaining <= 0,
            isOverdue: overdue
        )
    }

    static func formatCountdown(_ countdown: HallPassCountdown) -> String {
        let total = countdown.remainingSeconds
        let minutes = total / 60
        let seconds = total % 60
        return String(format: "%d:%02d", minutes, seconds)
    }

    static func destinationLabel(_ destination: String) -> String {
        switch destination.lowercased() {
        case "bathroom": return L.text("mobile.hallpass.destination.bathroom")
        case "office": return L.text("mobile.hallpass.destination.office")
        case "library": return L.text("mobile.hallpass.destination.library")
        case "nurse": return L.text("mobile.hallpass.destination.nurse")
        default: return L.text("mobile.hallpass.destination.other")
        }
    }

    static func statusLabel(_ status: String) -> String {
        switch status.lowercased() {
        case "requested": return L.text("mobile.hallpass.status.requested")
        case "approved": return L.text("mobile.hallpass.status.approved")
        case "denied": return L.text("mobile.hallpass.status.denied")
        case "returned": return L.text("mobile.hallpass.status.returned")
        default: return status
        }
    }

    static func storedPassKey(sectionId: String) -> String {
        "hallPass:\(sectionId)"
    }

    private static func parseDate(_ value: String?) -> Date? {
        guard let value, !value.isEmpty else { return nil }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: value) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: value)
    }
}
