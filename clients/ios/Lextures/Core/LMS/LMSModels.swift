import Foundation

/// Subset of web `CoursePublic` (camelCase JSON) used by the mobile app.
struct CourseSummary: Decodable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var title: String
    var description: String
    var heroImageUrl: String?
    var startsAt: String?
    var endsAt: String?
    var published: Bool?
    var catalogNickname: String?
    var notebookEnabled: Bool?
    var viewerEnrollmentRoles: [String]?

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
struct CourseStructureItem: Decodable, Identifiable, Hashable {
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
    private static let isoFractional: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    private static let iso: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    static func parse(_ raw: String?) -> Date? {
        guard let raw, !raw.isEmpty else { return nil }
        return isoFractional.date(from: raw) ?? iso.date(from: raw)
    }

    static func shortDateTime(_ raw: String?) -> String {
        guard let date = parse(raw) else { return "" }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    static func relative(_ raw: String?) -> String {
        guard let date = parse(raw) else { return "" }
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        return formatter.localizedString(for: date, relativeTo: Date())
    }
}
