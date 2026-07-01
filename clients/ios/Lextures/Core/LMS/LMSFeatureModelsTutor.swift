import Foundation

// MARK: - AI tutor (M7.2)

struct TutorCitation: Codable, Equatable, Hashable {
    var sourceId: String
    var chunkId: String
    var excerpt: String
    var title: String?
}

struct TutorMessage: Codable, Equatable, Hashable {
    var role: String
    var content: String
    var citations: [TutorCitation]?
    var id: String?
}

struct TutorConversationResponse: Decodable {
    var conversationId: String
    var messages: [TutorMessage]
    var tokensUsed: Int
    var tokenLimit: Int
    var periodMonth: String
}

struct TutorSessionSummary: Codable, Identifiable, Equatable, Hashable {
    var id: String
    var title: String?
    var createdAt: String
    var lastActive: String
}

struct TutorSessionDetailResponse: Decodable {
    var id: String
    var title: String?
    var createdAt: String
    var lastActive: String
    var messages: [TutorMessage]
}

struct TutorTokenBudgetResponse: Decodable {
    var tokensUsed: Int
    var tokenLimit: Int
    var periodMonth: String
}

struct TutorMessageRequest: Encodable {
    var message: String
}

struct TutorSessionMessageRequest: Encodable {
    var content: String
}

struct CreateTutorSessionRequest: Encodable {
    var title: String?
}

struct StudyBuddyMessageRequest: Encodable {
    var message: String
    var sessionId: String
}

struct NotebookRagNotebookInput: Encodable {
    var courseCode: String
    var courseTitle: String
    var markdown: String
}

struct NotebookRagQueryRequest: Encodable {
    var question: String
    var notebooks: [NotebookRagNotebookInput]
}

struct NotebookRagSource: Decodable, Equatable {
    var courseCode: String
    var courseTitle: String
    var excerpt: String
}

struct NotebookRagQueryResponse: Decodable {
    var answerMarkdown: String
    var sources: [NotebookRagSource]?
}

struct AskAiRoute: Hashable {}