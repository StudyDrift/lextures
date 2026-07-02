import Foundation

/// Subset of web `CoursePublic` (camelCase JSON) used by the mobile app.
struct CourseSummary: Codable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var title: String
    var description: String
    var heroImageUrl: String?
    var startsAt: String?
    var endsAt: String?
    var published: Bool?
    var catalogNickname: String?
    var catalogPinned: Bool?
    var notebookEnabled: Bool?
    var calendarEnabled: Bool?
    var officeHoursEnabled: Bool?
    var orgId: String?
    var termId: String?
    var viewerEnrollmentRoles: [String]?
    var feedEnabled: Bool?
    var discussionsEnabled: Bool?
    var liveSessionsEnabled: Bool?
    var filesEnabled: Bool?
    var attendanceEnabled: Bool?
    var sectionsEnabled: Bool?
    var resubmissionWorkflowEnabled: Bool?
    var aiTutorEnabled: Bool?
    var viewerStudentEnrollmentId: String?
    var standardsAlignmentEnabled: Bool?
    var reportCardsEnabled: Bool?

    var isCalendarEnabled: Bool { calendarEnabled != false }

    var isMasteryEnabled: Bool { standardsAlignmentEnabled == true }

    var isAiTutorEnabled: Bool { aiTutorEnabled == true }

    var isOfficeHoursEnabled: Bool { officeHoursEnabled == true }

    var isFeedEnabled: Bool { feedEnabled != false }

    var isDiscussionsEnabled: Bool { discussionsEnabled == true }

    var isLiveSessionsEnabled: Bool { liveSessionsEnabled == true }

    var isFilesEnabled: Bool { filesEnabled != false }

    var isAttendanceEnabled: Bool { attendanceEnabled != false }

    var isSectionsEnabled: Bool { sectionsEnabled != false }

    var isPinned: Bool { catalogPinned == true }

    var displayTitle: String {
        let nick = catalogNickname?.trimmingCharacters(in: .whitespacesAndNewlines)
        if let nick, !nick.isEmpty { return nick }
        return title
    }

    var viewerIsStudent: Bool {
        viewerEnrollmentRoles?.contains { $0.lowercased() == "student" } ?? false
    }

    var viewerIsStaff: Bool {
        let staff: Set<String> = ["teacher", "ta", "designer", "grader"]
        return viewerEnrollmentRoles?.contains { staff.contains($0.lowercased()) } ?? false
    }
}

struct CoursesResponse: Decodable {
    var courses: [CourseSummary]
}

/// Mirrors web `CourseStructureItem` (subset).
struct CourseStructureItem: Codable, Identifiable, Hashable {
    var id: String
    var sortOrder: Int
    var kind: String
    var title: String
    var parentId: String?
    var published: Bool
    var dueAt: String?
    var pointsWorth: Double?
    var pointsPossible: Double?

    var isModule: Bool { kind == "module" }

    var isGradable: Bool {
        kind == "assignment" || kind == "quiz" || kind == "content_page"
    }
}

struct CourseStructureResponse: Decodable {
    var items: [CourseStructureItem]
}

/// Tolerant union of the per-kind item GET responses
/// (`/content-pages/{id}`, `/assignments/{id}`, `/quizzes/{id}`, `/external-links/{id}`).
struct ModuleItemDetail: Codable {
    var title: String?
    var markdown: String?
    var dueAt: String?
    var availableFrom: String?
    var availableUntil: String?
    var updatedAt: String?
    var pointsWorth: Int?

    // Quiz settings (the web "preview box")
    var unlimitedAttempts: Bool?
    var maxAttempts: Int?
    var gradeAttemptPolicy: String?
    var oneQuestionAtATime: Bool?
    var shuffleQuestions: Bool?
    var lockdownMode: String?
    var adaptiveDeliveryMode: String?
    var timeLimitMinutes: Int?
    var passingScorePercent: Int?
    var requiresQuizAccessCode: Bool?
    var questions: [QuestionStub]?

    // Assignment submission settings
    var submissionAllowText: Bool?
    var submissionAllowFileUpload: Bool?
    var submissionAllowUrl: Bool?
    var lateSubmissionPolicy: String?
    var latePenaltyPercent: Int?

    // External link
    var url: String?
    var provider: String?

    struct QuestionStub: Codable {
        var id: String?
    }

    var questionCount: Int { questions?.count ?? 0 }
}

/// Mirrors web `MailboxMessage` (snake_case JSON from the communication API).
struct MailboxParty: Decodable, Hashable {
    var name: String
    var email: String
}

struct MailboxMessage: Decodable, Identifiable, Hashable {
    var id: String
    var from: MailboxParty
    var to: String
    var subject: String
    var snippet: String
    var body: String
    var sentAt: String
    var read: Bool
    var starred: Bool
    var folder: String
    var hasAttachment: Bool

    enum CodingKeys: String, CodingKey {
        case id, from, to, subject, snippet, body, read, starred, folder
        case sentAtSnake = "sent_at"
        case sentAtCamel = "sentAt"
        case hasAttachmentSnake = "has_attachment"
        case hasAttachmentCamel = "hasAttachment"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        from = try container.decode(MailboxParty.self, forKey: .from)
        to = try container.decodeIfPresent(String.self, forKey: .to) ?? ""
        subject = try container.decodeIfPresent(String.self, forKey: .subject) ?? ""
        snippet = try container.decodeIfPresent(String.self, forKey: .snippet) ?? ""
        body = try container.decodeIfPresent(String.self, forKey: .body) ?? ""
        read = try container.decodeIfPresent(Bool.self, forKey: .read) ?? false
        starred = try container.decodeIfPresent(Bool.self, forKey: .starred) ?? false
        folder = try container.decodeIfPresent(String.self, forKey: .folder) ?? "inbox"
        sentAt = try container.decodeIfPresent(String.self, forKey: .sentAtSnake)
            ?? container.decodeIfPresent(String.self, forKey: .sentAtCamel) ?? ""
        hasAttachment = try container.decodeIfPresent(Bool.self, forKey: .hasAttachmentSnake)
            ?? container.decodeIfPresent(Bool.self, forKey: .hasAttachmentCamel) ?? false
    }
}

struct MailboxMessagesResponse: Decodable {
    var messages: [MailboxMessage]
}

enum MailboxFolder: String, CaseIterable, Identifiable {
    case inbox, starred, sent, drafts, trash

    var id: String { rawValue }

    var label: String {
        switch self {
        case .inbox: return "Inbox"
        case .starred: return "Starred"
        case .sent: return "Sent"
        case .drafts: return "Drafts"
        case .trash: return "Trash"
        }
    }

    var systemImage: String {
        switch self {
        case .inbox: return "tray"
        case .starred: return "star"
        case .sent: return "paperplane"
        case .drafts: return "doc.text"
        case .trash: return "trash"
        }
    }
}

enum LMSDates {
    static func parse(_ raw: String?) -> Date? {
        DateFormatting.parse(raw)
    }

    static func shortDateTime(_ raw: String?) -> String {
        DateFormatting.formatDateTime(raw)
    }

    static func relative(_ raw: String?) -> String {
        DateFormatting.formatRelative(raw)
    }
}
