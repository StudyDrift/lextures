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
        if parts.count >= 2, let a = parts.first?.first, let b = parts.last?.first {
            return String([a, b]).uppercased()
        }
        return String(source.prefix(2)).uppercased()
    }
}

// MARK: - Notifications

/// Row from GET `/api/v1/me/notifications`.
struct AppNotification: Decodable, Identifiable, Hashable {
    var id: String
    var eventType: String
    var title: String
    var body: String
    var actionUrl: String?
    var isRead: Bool
    var createdAt: String
}

struct NotificationsPage: Decodable {
    var notifications: [AppNotification]
    var unreadCount: Int

    enum CodingKeys: String, CodingKey { case notifications, unreadCount }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        notifications = try container.decodeIfPresent([AppNotification].self, forKey: .notifications) ?? []
        unreadCount = try container.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
    }
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
struct GradeColumn: Decodable, Identifiable, Hashable {
    var id: String
    var kind: String
    var title: String
    var maxPoints: Double?
    var dueAt: String?
    var assignmentGroupId: String?
}

struct AssignmentGroup: Decodable, Hashable {
    var id: String
    var name: String
    var weightPercent: Double

    enum CodingKeys: String, CodingKey { case id, name, weightPercent }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        weightPercent = try container.decodeIfPresent(Double.self, forKey: .weightPercent) ?? 0
    }
}

/// GET `/courses/{code}/my-grades` (student only).
struct MyGradesResponse: Decodable {
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
    var submittedAt: String
    var updatedAt: String?
    var versionNumber: Int?
    var resubmissionRequested: Bool?
    var revisionDueAt: String?
    var revisionFeedback: String?

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

// MARK: - Grading backlog (staff)

/// Row from GET `/courses/{code}/grading-backlog`.
struct GradingBacklogItem: Decodable, Identifiable, Hashable {
    var assignmentId: String
    var assignmentTitle: String
    var ungradedCount: Int

    var id: String { assignmentId }
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
