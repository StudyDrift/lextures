import Foundation

/// AI tutor endpoints (M7.2).
extension LMSAPI {
    static func fetchTutorConversation(courseCode: String, accessToken: String) async throws -> TutorConversationResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/conversation",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(TutorConversationResponse.self, from: data)
    }

    static func resetTutorConversation(courseCode: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/conversation",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func fetchTutorSessions(courseCode: String, accessToken: String) async throws -> [TutorSessionSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/sessions",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        return try decode([TutorSessionSummary].self, from: data)
    }

    static func createTutorSession(
        courseCode: String,
        accessToken: String,
        title: String? = nil
    ) async throws -> TutorSessionSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/sessions",
            method: "POST",
            body: CreateTutorSessionRequest(title: title),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(TutorSessionSummary.self, from: data)
    }

    static func fetchTutorSession(
        courseCode: String,
        sessionId: String,
        accessToken: String
    ) async throws -> TutorSessionDetailResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/sessions/\(encodePath(sessionId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(TutorSessionDetailResponse.self, from: data)
    }

    static func deleteTutorSession(courseCode: String, sessionId: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/sessions/\(encodePath(sessionId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func fetchTokenBudget(accessToken: String) async throws -> TutorTokenBudgetResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/me/token-budget",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(TutorTokenBudgetResponse.self, from: data)
    }

    static func queryNotebooks(
        _ request: NotebookRagQueryRequest,
        accessToken: String
    ) async throws -> NotebookRagQueryResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/me/notebooks/query",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotebookRagQueryResponse.self, from: data)
    }

    static func tutorMessageStream(
        courseCode: String,
        message: String,
        accessToken: String
    ) -> AsyncThrowingStream<TutorStreamEvent, Error> {
        TutorStreamClient().stream(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/message",
            body: TutorMessageRequest(message: message),
            accessToken: accessToken
        )
    }

    static func tutorSessionMessageStream(
        courseCode: String,
        sessionId: String,
        content: String,
        accessToken: String
    ) -> AsyncThrowingStream<TutorStreamEvent, Error> {
        TutorStreamClient().stream(
            path: "/api/v1/courses/\(encodePath(courseCode))/tutor/sessions/\(encodePath(sessionId))/messages",
            body: TutorSessionMessageRequest(content: content),
            accessToken: accessToken
        )
    }

    static func studyBuddyMessageStream(
        courseCode: String,
        message: String,
        sessionId: String?,
        accessToken: String
    ) -> AsyncThrowingStream<TutorStreamEvent, Error> {
        TutorStreamClient().stream(
            path: "/api/v1/courses/\(encodePath(courseCode))/study-buddy/message",
            body: StudyBuddyMessageRequest(message: message, sessionId: sessionId ?? ""),
            accessToken: accessToken
        )
    }
}