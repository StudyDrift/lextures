import Foundation

/// Study insights, goals, reflection journal, and coaching tips (M8.3).
extension LMSAPI {
    static func fetchStudyStats(accessToken: String) async throws -> StudyStats {
        let (data, response) = try await client.request(
            path: "/api/v1/me/study-stats",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudyStats.self, from: data)
    }

    static func fetchStudyGoal(accessToken: String) async throws -> StudyGoal {
        let (data, response) = try await client.request(
            path: "/api/v1/me/study-goal",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudyGoal.self, from: data)
    }

    static func putStudyGoal(body: PutStudyGoalBody, accessToken: String) async throws -> StudyGoal {
        let (data, response) = try await client.request(
            path: "/api/v1/me/study-goal",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudyGoal.self, from: data)
    }

    static func fetchReflectionJournal(accessToken: String) async throws -> [ReflectionJournalEntry] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reflection-journal",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReflectionJournalListResponse.self, from: data).entries ?? []
    }

    static func createReflectionJournalEntry(
        body: PostReflectionJournalBody,
        accessToken: String
    ) async throws -> String {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reflection-journal",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PostReflectionJournalResponse.self, from: data).id
    }

    static func deleteReflectionJournalEntry(id: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reflection-journal/\(encodePath(id))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchCoachingTips(accessToken: String) async throws -> CoachingTipsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/coaching-tips",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CoachingTipsResponse.self, from: data)
    }

    static func rateCoachingTip(id: String, rating: Int, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/me/coaching-tips/\(encodePath(id))/rating",
            method: "POST",
            body: RateCoachingTipBody(rating: rating),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchReminderConfig(accessToken: String) async throws -> ReminderConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reminder-config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReminderConfig.self, from: data)
    }

    static func patchReminderConfig(enabled: Bool, accessToken: String) async throws -> ReminderConfig {
        struct Body: Encodable { var enabled: Bool }
        let (data, response) = try await client.request(
            path: "/api/v1/me/reminder-config",
            method: "PATCH",
            body: Body(enabled: enabled),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReminderConfig.self, from: data)
    }
}