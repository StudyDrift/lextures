import Foundation

/// LMS endpoints used by the post-auth tabs (parity with web `courses-api` / `communication-api`).
enum LMSAPI {
    static let client = APIClient()

    static func decode<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
        do {
            return try JSONDecoder().decode(type, from: data)
        } catch {
            throw APIError.decoding(error)
        }
    }

    // MARK: - Courses

    static func fetchCourses(accessToken: String) async throws -> [CourseSummary] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CoursesResponse.self, from: data).courses
    }

    /// Single-course GET includes `viewerEnrollmentRoles` (list GET does not).
    static func fetchCourse(courseCode: String, accessToken: String) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func fetchCourseStructure(courseCode: String, accessToken: String) async throws -> [CourseStructureItem] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureResponse.self, from: data).items
    }

    /// Per-kind detail GET for a structure item; nil when the kind has no detail endpoint.
    static func fetchItemDetail(
        courseCode: String,
        item: CourseStructureItem,
        accessToken: String
    ) async throws -> ModuleItemDetail? {
        let resource: String?
        switch item.kind {
        case "content_page": resource = "content-pages"
        case "assignment": resource = "assignments"
        case "quiz": resource = "quizzes"
        case "external_link": resource = "external-links"
        default: resource = nil
        }
        guard let resource else { return nil }
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/\(resource)/\(encodePath(item.id))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ModuleItemDetail.self, from: data)
    }

    // MARK: - Inbox (communication)

    static func fetchMailboxMessages(
        folder: MailboxFolder,
        query: String,
        accessToken: String
    ) async throws -> [MailboxMessage] {
        var components = URLComponents()
        components.queryItems = [
            URLQueryItem(name: "folder", value: folder.rawValue),
            URLQueryItem(name: "q", value: query.trimmingCharacters(in: .whitespacesAndNewlines)),
        ]
        let queryString = components.percentEncodedQuery ?? ""
        let (data, _) = try await client.request(
            path: "/api/v1/communication/messages?\(queryString)",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(MailboxMessagesResponse.self, from: data).messages
    }

    static func fetchUnreadInboxCount(accessToken: String) async throws -> Int {
        let (data, _) = try await client.request(
            path: "/api/v1/communication/unread-count",
            authorized: true,
            accessToken: accessToken
        )
        struct UnreadResponse: Decodable {
            var unreadInbox: Int?
            enum CodingKeys: String, CodingKey {
                case snake = "unread_inbox"
                case camel = "unreadInbox"
            }
            init(from decoder: Decoder) throws {
                let container = try decoder.container(keyedBy: CodingKeys.self)
                unreadInbox = try container.decodeIfPresent(Int.self, forKey: .snake)
                    ?? container.decodeIfPresent(Int.self, forKey: .camel)
            }
        }
        return try decode(UnreadResponse.self, from: data).unreadInbox ?? 0
    }

    struct MailboxPatch: Encodable {
        var read: Bool?
        var starred: Bool?
        var folder: String?
    }

    static func patchMailbox(messageId: String, patch: MailboxPatch, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/communication/messages/\(encodePath(messageId))",
            method: "PATCH",
            body: patch,
            authorized: true,
            accessToken: accessToken
        )
    }

    struct SendMessageRequest: Encodable {
        var toEmail: String?
        var subject: String
        var body: String
        var draft: Bool?

        enum CodingKeys: String, CodingKey {
            case toEmail = "to_email"
            case subject, body, draft
        }
    }

    static func sendMessage(_ request: SendMessageRequest, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/communication/messages",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken
        )
    }

    // MARK: - Notebook tasks (dashboard sync, parity with web `notebook-tasks-api`)

    struct NotebookTaskUpsert: Encodable {
        var id: String
        var courseCode: String
        var notebookPageId: String
        var taskText: String
        var completed: Bool
        var dueAt: String?

        enum CodingKeys: String, CodingKey {
            case id, courseCode, notebookPageId, taskText, completed, dueAt
        }

        // Explicit `dueAt: null` (web parity) — synthesized encoding would omit the key,
        // which leaves a stale due date on the server.
        func encode(to encoder: Encoder) throws {
            var container = encoder.container(keyedBy: CodingKeys.self)
            try container.encode(id, forKey: .id)
            try container.encode(courseCode, forKey: .courseCode)
            try container.encode(notebookPageId, forKey: .notebookPageId)
            try container.encode(taskText, forKey: .taskText)
            try container.encode(completed, forKey: .completed)
            try container.encode(dueAt, forKey: .dueAt)
        }
    }

    static func upsertNotebookTask(_ task: NotebookTaskUpsert, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/notebook-tasks",
            method: "POST",
            body: task,
            authorized: true,
            accessToken: accessToken
        )
    }

    static func fetchNotebookTasks(accessToken: String) async throws -> [NotebookTask] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/notebook-tasks",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotebookTasksResponse.self, from: data).tasks
    }

    // MARK: - Notebooks (server sync, parity with web `student-notebook-sync`)

    struct NotebookEntry: Decodable {
        var courseCode: String
        var updatedAt: String
        var data: CourseNotebook?

        enum CodingKeys: String, CodingKey { case courseCode, updatedAt, data }

        init(from decoder: Decoder) throws {
            let container = try decoder.container(keyedBy: CodingKeys.self)
            courseCode = try container.decode(String.self, forKey: .courseCode)
            updatedAt = try container.decode(String.self, forKey: .updatedAt)
            // Lenient: one malformed document must not break the whole list.
            data = try? container.decode(CourseNotebook.self, forKey: .data)
        }
    }

    private struct NotebooksResponse: Decodable {
        var notebooks: [NotebookEntry]
    }

    static func fetchNotebooks(accessToken: String) async throws -> [NotebookEntry] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/notebooks",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotebooksResponse.self, from: data).notebooks
    }

    static func putNotebook(courseCode: String, notebook: CourseNotebook, accessToken: String) async throws {
        let code = courseCode.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? courseCode
        _ = try await client.request(
            path: "/api/v1/me/notebooks?courseCode=\(code)",
            method: "PUT",
            body: notebook,
            authorized: true,
            accessToken: accessToken
        )
    }

    static func encodePath(_ component: String) -> String {
        component.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? component
    }
}
