import Foundation

// MARK: - AI tutor (M7.2)

enum TutorStreamEvent: Equatable {
    case content(String)
    case error(String)
    case done(
        conversationId: String? = nil,
        messageId: String? = nil,
        sessionId: String? = nil,
        citations: [TutorCitation] = []
    )
}

enum TutorChatMode: Equatable {
    case course(course: CourseSummary, item: CourseStructureItem?)
    case askAi
}

struct TutorDisplayMessage: Identifiable, Equatable {
    let id: String
    let role: String
    var content: String
    var citations: [TutorCitation]
    var isStreaming: Bool

    init(
        id: String = UUID().uuidString.lowercased(),
        role: String,
        content: String,
        citations: [TutorCitation] = [],
        isStreaming: Bool = false
    ) {
        self.id = id
        self.role = role
        self.content = content
        self.citations = citations
        self.isStreaming = isStreaming
    }
}

enum TutorLogic {
    static let maxMessageLength = 2000

    static func shouldShowFab(course: CourseSummary) -> Bool {
        course.isAiTutorEnabled
    }

    static func askAiEnabled(platform: MobilePlatformFeatures) -> Bool {
        platform.ragNotebookEnabled || platform.aiStudyBuddyEnabled
    }

    static func disclosureStorageKey(courseCode: String?) -> String {
        if let courseCode, !courseCode.isEmpty {
            return "tutor-disclosure-\(courseCode)"
        }
        return "tutor-disclosure-ask-ai"
    }

    static func hasAcceptedDisclosure(courseCode: String?) -> Bool {
        UserDefaults.standard.bool(forKey: disclosureStorageKey(courseCode: courseCode))
    }

    static func acceptDisclosure(courseCode: String?) {
        UserDefaults.standard.set(true, forKey: disclosureStorageKey(courseCode: courseCode))
    }

    static func contextPrefix(itemTitle: String?, itemKind: String?) -> String? {
        let title = itemTitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !title.isEmpty else { return nil }
        if let kind = itemKind?.trimmingCharacters(in: .whitespacesAndNewlines), !kind.isEmpty {
            return "[Context: viewing \(kind.replacingOccurrences(of: "_", with: " ")) \"\(title)\"]"
        }
        return "[Context: viewing \"\(title)\"]"
    }

    static func messageWithContext(
        _ text: String,
        itemTitle: String?,
        itemKind: String?,
        includeContext: Bool
    ) -> String {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard includeContext, let prefix = contextPrefix(itemTitle: itemTitle, itemKind: itemKind) else {
            return trimmed
        }
        return "\(prefix)\n\n\(trimmed)"
    }

    static func parseStreamEvent(_ jsonLine: String) -> TutorStreamEvent? {
        guard let data = jsonLine.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let type = object["type"] as? String else {
            return nil
        }
        switch type {
        case "content":
            guard let text = object["text"] as? String else { return nil }
            return .content(text)
        case "error":
            let message = (object["message"] as? String) ?? L.text("mobile.tutor.streamError")
            return .error(message)
        case "done":
            let citations = decodeCitations(object["citations"])
            return .done(
                conversationId: object["conversationId"] as? String,
                messageId: object["messageId"] as? String,
                sessionId: object["sessionId"] as? String,
                citations: citations
            )
        default:
            return nil
        }
    }

    static func parseSSELine(_ line: String) -> TutorStreamEvent? {
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.hasPrefix("data: ") else { return nil }
        return parseStreamEvent(String(trimmed.dropFirst(6)))
    }

    static func budgetLabel(used: Int, limit: Int) -> String {
        L.format("mobile.tutor.tokenBudget", used, limit)
    }

    static func gracefulHttpMessage(statusCode: Int, body: String?) -> String {
        if statusCode == 402 || body?.contains("BUDGET_EXCEEDED") == true {
            return L.text("mobile.tutor.budgetExceeded")
        }
        if statusCode == 403 {
            return body?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty
                ?? L.text("mobile.tutor.disabled")
        }
        if statusCode == 503 {
            return L.text("mobile.tutor.unavailable")
        }
        return body?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty
            ?? L.text("mobile.tutor.sendError")
    }

    private static func decodeCitations(_ value: Any?) -> [TutorCitation] {
        guard let array = value as? [[String: Any]] else { return [] }
        return array.compactMap { row in
            guard let sourceId = row["sourceId"] as? String,
                  let chunkId = row["chunkId"] as? String,
                  let excerpt = row["excerpt"] as? String else {
                return nil
            }
            return TutorCitation(
                sourceId: sourceId,
                chunkId: chunkId,
                excerpt: excerpt,
                title: row["title"] as? String
            )
        }
    }
}

private extension String {
    var nonEmpty: String? {
        isEmpty ? nil : self
    }
}