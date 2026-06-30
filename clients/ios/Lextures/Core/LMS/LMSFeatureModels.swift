import Foundation

// MARK: - Profile

/// GET `/api/v1/me`.
struct MeProfile: Decodable {
    var id: String
    var email: String
    var displayName: String?

    /// First name for greetings; falls back to the email local part.
    var firstName: String {
        let name = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty {
            return String(name.split(separator: " ").first ?? Substring(name))
        }
        return String(email.split(separator: "@").first ?? "there")
    }

    /// Two-letter initials for the avatar chip.
    var initials: String {
        let name = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let source = name.isEmpty ? String(email.split(separator: "@").first ?? "?") : name
        let parts = source.split(separator: " ")
        if parts.count >= 2,
           let firstInitial = parts.first?.first,
           let lastInitial = parts.last?.first {
            return String([firstInitial, lastInitial]).uppercased()
        }
        return String(source.prefix(2)).uppercased()
    }
}

// MARK: - Notifications

/// Row from GET `/api/v1/me/notifications`.
struct AppNotification: Codable, Identifiable, Hashable {
    var id: String
    var eventType: String
    var title: String
    var body: String
    var actionUrl: String?
    var isRead: Bool
    var createdAt: String

    init(
        id: String,
        eventType: String,
        title: String,
        body: String,
        actionUrl: String? = nil,
        isRead: Bool = false,
        createdAt: String = ""
    ) {
        self.id = id
        self.eventType = eventType
        self.title = title
        self.body = body
        self.actionUrl = actionUrl
        self.isRead = isRead
        self.createdAt = createdAt
    }
}

struct NotificationsPage: Codable {
    var notifications: [AppNotification]
    var unreadCount: Int

    enum CodingKeys: String, CodingKey { case notifications, unreadCount }

    init(notifications: [AppNotification] = [], unreadCount: Int = 0) {
        self.notifications = notifications
        self.unreadCount = unreadCount
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        notifications = try container.decodeIfPresent([AppNotification].self, forKey: .notifications) ?? []
        unreadCount = try container.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
    }
}

/// Row from GET `/api/v1/me/notification-preferences`.
struct NotificationPreference: Codable, Identifiable, Hashable {
    var id: String { eventType }
    var eventType: String
    var emailEnabled: Bool
    var pushEnabled: Bool
    var smsEnabled: Bool
    var digestMode: String

    enum CodingKeys: String, CodingKey {
        case eventType, emailEnabled, pushEnabled, smsEnabled, digestMode
    }

    init(
        eventType: String,
        emailEnabled: Bool = true,
        pushEnabled: Bool = true,
        smsEnabled: Bool = false,
        digestMode: String = "instant"
    ) {
        self.eventType = eventType
        self.emailEnabled = emailEnabled
        self.pushEnabled = pushEnabled
        self.smsEnabled = smsEnabled
        self.digestMode = digestMode
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        eventType = try container.decode(String.self, forKey: .eventType)
        emailEnabled = try container.decodeIfPresent(Bool.self, forKey: .emailEnabled) ?? true
        pushEnabled = try container.decodeIfPresent(Bool.self, forKey: .pushEnabled) ?? true
        smsEnabled = try container.decodeIfPresent(Bool.self, forKey: .smsEnabled) ?? false
        digestMode = try container.decodeIfPresent(String.self, forKey: .digestMode) ?? "instant"
    }
}

struct NotificationPreferencesResponse: Decodable {
    var preferences: [NotificationPreference]

    enum CodingKeys: String, CodingKey { case preferences }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        preferences = try container.decodeIfPresent([NotificationPreference].self, forKey: .preferences) ?? []
    }
}

struct NotificationPreferencePatch: Encodable {
    var eventType: String
    var emailEnabled: Bool?
    var pushEnabled: Bool?
    var smsEnabled: Bool?
    var digestMode: String?
}

struct NotificationPreferencesUpdate: Encodable {
    var preferences: [NotificationPreferencePatch]
}

struct DeviceTokenRegistration: Encodable {
    var token: String
    var platform: String
    var appBundleId: String?
    var appVersion: String?
}

struct DeviceTokenResponse: Decodable {
    var id: String
}

struct DeviceTokensPage: Decodable {
    var tokens: [DevicePushToken]
}

struct DevicePushToken: Decodable, Identifiable {
    var id: String
    var platform: String
    var appBundleId: String?
    var appVersion: String?
    var isActive: Bool
    var createdAt: String
}

// MARK: - Announcements (org broadcasts)

/// Row from GET `/api/v1/me/broadcasts`.
struct Broadcast: Decodable, Identifiable, Hashable {
    var id: String
    var type: String // "announcement" | "emergency"
    var subject: String
    var body: String
    var sentAt: String?
    var createdAt: String

    var isEmergency: Bool { type == "emergency" }
}

struct BroadcastsResponse: Decodable {
    var broadcasts: [Broadcast]
}

// MARK: - My grades

/// Column from `/my-grades` / gradebook grid (subset used on mobile).
struct GradeColumn: Codable, Identifiable, Hashable {
    var id: String
    var kind: String
    var title: String
    var maxPoints: Double?
    var dueAt: String?
    var assignmentGroupId: String?
}

struct AssignmentGroup: Codable, Hashable {
    var id: String
    var name: String
    var weightPercent: Double

    enum CodingKeys: String, CodingKey { case id, name, weightPercent }

    init(id: String, name: String, weightPercent: Double) {
        self.id = id
        self.name = name
        self.weightPercent = weightPercent
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        weightPercent = try container.decodeIfPresent(Double.self, forKey: .weightPercent) ?? 0
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(weightPercent, forKey: .weightPercent)
    }
}

/// GET `/courses/{code}/my-grades` (student only).
struct MyGradesResponse: Codable {
    var columns: [GradeColumn]
    var grades: [String: String]
    var displayGrades: [String: String]
    var assignmentGroups: [AssignmentGroup]
    var heldGradeItemIds: [String]
    var droppedGrades: [String: Bool]
    var gradeStatuses: [String: String]

    enum CodingKeys: String, CodingKey {
        case columns, grades, displayGrades, assignmentGroups
        case heldGradeItemIds, droppedGrades, gradeStatuses
    }

    init(
        columns: [GradeColumn] = [],
        grades: [String: String] = [:],
        displayGrades: [String: String] = [:],
        assignmentGroups: [AssignmentGroup] = [],
        heldGradeItemIds: [String] = [],
        droppedGrades: [String: Bool] = [:],
        gradeStatuses: [String: String] = [:]
    ) {
        self.columns = columns
        self.grades = grades
        self.displayGrades = displayGrades
        self.assignmentGroups = assignmentGroups
        self.heldGradeItemIds = heldGradeItemIds
        self.droppedGrades = droppedGrades
        self.gradeStatuses = gradeStatuses
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        columns = try container.decodeIfPresent([GradeColumn].self, forKey: .columns) ?? []
        grades = try container.decodeIfPresent([String: String].self, forKey: .grades) ?? [:]
        displayGrades = try container.decodeIfPresent([String: String].self, forKey: .displayGrades) ?? [:]
        assignmentGroups = (try? container.decodeIfPresent([AssignmentGroup].self, forKey: .assignmentGroups)) ?? []
        heldGradeItemIds = try container.decodeIfPresent([String].self, forKey: .heldGradeItemIds) ?? []
        droppedGrades = try container.decodeIfPresent([String: Bool].self, forKey: .droppedGrades) ?? [:]
        gradeStatuses = try container.decodeIfPresent([String: String].self, forKey: .gradeStatuses) ?? [:]
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(columns, forKey: .columns)
        try container.encode(grades, forKey: .grades)
        try container.encode(displayGrades, forKey: .displayGrades)
        try container.encode(assignmentGroups, forKey: .assignmentGroups)
        try container.encode(heldGradeItemIds, forKey: .heldGradeItemIds)
        try container.encode(droppedGrades, forKey: .droppedGrades)
        try container.encode(gradeStatuses, forKey: .gradeStatuses)
    }
}

// MARK: - Syllabus

struct SyllabusSection: Decodable, Identifiable, Hashable {
    var id: String
    var heading: String
    var markdown: String
}

/// GET `/courses/{code}/syllabus`.
struct SyllabusPayload: Decodable {
    var sections: [SyllabusSection]
    var updatedAt: String?
    var requireSyllabusAcceptance: Bool?
    var syllabusAcceptancePending: Bool?

    var hasContent: Bool {
        sections.contains { !$0.markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
    }
}

// MARK: - Assignment submissions

/// Row from `/assignments/{item}/submissions` and `/submissions/mine`.
struct AssignmentSubmission: Decodable, Identifiable, Hashable {
    var id: String
    var submittedBy: String?
    var submittedByDisplayName: String?
    var blindLabel: String?
    var attachmentFilename: String?
    var attachmentMimeType: String?
    var attachmentContentPath: String?
    var bodyText: String?
    var submittedAt: String
    var updatedAt: String?
    var versionNumber: Int?
    var resubmissionRequested: Bool?
    var revisionDueAt: String?
    var revisionFeedback: String?
    var isGraded: Bool?

    /// Name shown in staff lists; respects blind grading.
    var displayName: String {
        if let blind = blindLabel, !blind.isEmpty { return blind }
        if let name = submittedByDisplayName, !name.isEmpty { return name }
        return "Student"
    }
}

struct MySubmissionResponse: Decodable {
    var submission: AssignmentSubmission?
}

struct SubmissionsListResponse: Decodable {
    var submissions: [AssignmentSubmission]

    enum CodingKeys: String, CodingKey { case submissions }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        // The list endpoint returns a roster row for every enrolled student, including
        // non-submitters that lack `id`/`submittedAt`. Decode leniently and keep only
        // real submissions so one placeholder doesn't fail the whole list.
        let rows = try container.decodeIfPresent([LossyDecodable<AssignmentSubmission>].self, forKey: .submissions) ?? []
        submissions = rows.compactMap(\.value)
    }
}

/// Decodes `T` if possible, otherwise yields `nil` without throwing — used to skip
/// malformed/placeholder elements inside an array without failing the whole decode.
struct LossyDecodable<T: Decodable>: Decodable {
    let value: T?

    init(from decoder: Decoder) throws {
        value = try? T(from: decoder)
    }
}

/// GET/PUT `.../submissions/{id}/grade`.
struct SubmissionGrade: Decodable {
    var submissionId: String?
    var pointsEarned: Double?
    var maxPoints: Double?
    var instructorComment: String?
    var posted: Bool?
    var excused: Bool?
}

// MARK: - Quiz attempts (staff)

/// Row from GET `/quizzes/{item}/attempts`.
struct QuizAttemptSummary: Decodable, Identifiable, Hashable {
    var id: String
    var studentUserId: String?
    var attemptNumber: Int
    var submittedAt: String
    var scorePercent: Double?
    var pointsEarned: Double
    var pointsPossible: Double
    var studentName: String?
    var needsManualGrading: Bool?
}

struct QuizAttemptsListResponse: Decodable {
    var attempts: [QuizAttemptSummary]
}

// MARK: - Grading backlog (staff)

/// Row from GET `/courses/{code}/grading-backlog`.
struct GradingBacklogItem: Decodable, Identifiable, Hashable {
    var itemId: String?
    var itemType: String? // "assignment" | "quiz"
    var assignmentId: String
    var assignmentTitle: String
    var ungradedCount: Int

    var resolvedItemId: String { itemId ?? assignmentId }
    var isQuiz: Bool { itemType == "quiz" }
    var id: String { "\(itemType ?? "assignment")-\(resolvedItemId)" }
}

enum GradingSubmissionMapper {
    static func quizAttemptsToSubmissions(_ attempts: [QuizAttemptSummary]) -> [AssignmentSubmission] {
        var byStudent: [String: QuizAttemptSummary] = [:]
        for attempt in attempts {
            let key = attempt.studentUserId?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty ?? attempt.id
            if let existing = byStudent[key] {
                if attempt.attemptNumber >= existing.attemptNumber {
                    byStudent[key] = attempt
                }
            } else {
                byStudent[key] = attempt
            }
        }
        return byStudent.values
            .sorted { ($0.studentName ?? "") < ($1.studentName ?? "") }
            .map { attempt in
                AssignmentSubmission(
                    id: attempt.id,
                    submittedBy: attempt.studentUserId,
                    submittedByDisplayName: attempt.studentName,
                    blindLabel: nil,
                    attachmentFilename: nil,
                    submittedAt: attempt.submittedAt,
                    updatedAt: nil,
                    versionNumber: attempt.attemptNumber > 1 ? attempt.attemptNumber : nil,
                    resubmissionRequested: nil,
                    revisionDueAt: nil,
                    revisionFeedback: nil,
                    isGraded: attempt.needsManualGrading == false
                )
            }
    }

    static func filterSubmissions(_ submissions: [AssignmentSubmission], graded: String?) -> [AssignmentSubmission] {
        guard let graded, graded != "all" else { return submissions }
        if graded == "graded" {
            return submissions.filter { $0.isGraded == true }
        }
        return submissions.filter { $0.isGraded != true }
    }
}

private extension String {
    var nonEmpty: String? {
        isEmpty ? nil : self
    }
}

struct GradingBacklogResponse: Decodable {
    var items: [GradingBacklogItem]
}

// MARK: - Attendance

/// Session from GET `/courses/{code}/attendance/sessions`.
struct AttendanceSession: Decodable, Identifiable, Hashable {
    var id: String
    var title: String?
    var collectionMethod: String // "roll_call" | "self_report"
    var sessionDate: String?
    var status: String // "open" | "closed"

    var isOpen: Bool { status == "open" }
    var isSelfReport: Bool { collectionMethod == "self_report" }

    var displayTitle: String {
        if let title, !title.isEmpty { return title }
        return "Attendance"
    }
}

struct AttendanceRecord: Decodable, Hashable {
    var studentUserId: String
    var displayName: String?
    var status: String
    var recordedAt: String?
}

/// GET `.../attendance/sessions/{id}` — session plus viewer-specific fields.
struct AttendanceSessionDetail: Decodable {
    var id: String
    var title: String?
    var collectionMethod: String
    var sessionDate: String?
    var status: String
    var records: [AttendanceRecord]?
    var myRecord: AttendanceRecord?
    var canSelfReport: Bool?
}

struct AttendanceSessionsResponse: Decodable {
    var sessions: [AttendanceSession]
}

// MARK: - Planner (todos + calendar, M2.1)

/// Row from GET `/api/v1/me/notebook-tasks`.
struct NotebookTask: Decodable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var notebookPageId: String
    var taskText: String
    var completed: Bool
    var dueAt: String?
    var createdAt: String?
    var updatedAt: String?
}

struct NotebookTasksResponse: Decodable {
    var tasks: [NotebookTask]
}

struct CalendarCourseFeed: Decodable, Hashable {
    var courseId: String
    var courseCode: String
    var title: String
    var feedUrl: String
}

struct CalendarTokenStatus: Decodable {
    var hasToken: Bool?
}

struct CalendarTokenInfo: Decodable {
    var hasToken: Bool?
    var personalFeedUrl: String?
    var expiresAt: String?
    var courseFeeds: [CalendarCourseFeed]?
}

struct CalendarTokenCreated: Decodable {
    var token: String
    var feedUrl: String?
    var expiresAt: String?
}

struct AcademicCalendarEvent: Decodable, Identifiable, Hashable {
    var id: String
    var orgId: String
    var termId: String?
    var eventType: String
    var eventName: String
    var startDate: String
    var endDate: String?
    var allDay: Bool
    var notes: String?
}

struct AcademicCalendarEventsResponse: Decodable {
    var events: [AcademicCalendarEvent]
}

/// Human label + tint key for an attendance status string.
enum AttendanceStatusInfo {
    static func label(_ status: String) -> String {
        switch status {
        case "present": return "Present"
        case "absent": return "Absent"
        case "tardy": return "Tardy"
        case "excused": return "Excused"
        case "not_recorded": return "Not recorded"
        default: return status.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }
}
