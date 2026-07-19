import Foundation

/// Interactive live quiz API (MOB.5 Phase 1).
extension LMSAPI {
    struct LiveQuizJoinError: LocalizedError, Equatable {
        let status: Int
        let reason: LiveQuizLogic.JoinErrorReason

        var errorDescription: String? {
            L.text(String.LocalizationValue(LiveQuizLogic.joinErrorLocalizationKey(reason)))
        }
    }

    private struct NicknameBody: Encodable {
        var nickname: String
    }

    static func lookupJoinCode(_ code: String) async throws -> LiveQuizJoinLookup {
        let normalized = LiveQuizLogic.normalizeJoinCode(code)
        do {
            let (data, _) = try await client.request(
                path: "/api/v1/live-quizzes/join/\(encodePath(normalized))",
                authorized: false
            )
            return try decode(LiveQuizJoinLookup.self, from: data)
        } catch let APIError.httpStatus(status, message) {
            throw LiveQuizJoinError(
                status: status,
                reason: LiveQuizLogic.joinErrorReason(status: status, body: message ?? "")
            )
        }
    }

    static func joinLiveGame(
        courseCode: String,
        gameId: String,
        nickname: String,
        accessToken: String
    ) async throws -> LiveQuizJoinPlayerResult {
        do {
            let (data, _) = try await client.request(
                path: "/api/v1/courses/\(encodePath(courseCode))/live-quizzes/games/\(encodePath(gameId))/players",
                method: "POST",
                body: NicknameBody(nickname: nickname),
                authorized: true,
                accessToken: accessToken
            )
            return try decode(LiveQuizJoinPlayerResult.self, from: data)
        } catch let APIError.httpStatus(status, message) {
            throw LiveQuizJoinError(
                status: status,
                reason: LiveQuizLogic.joinErrorReason(status: status, body: message ?? "")
            )
        }
    }

    static func joinLiveGameAsGuest(
        code: String,
        nickname: String
    ) async throws -> LiveQuizJoinPlayerResult {
        let normalized = LiveQuizLogic.normalizeJoinCode(code)
        do {
            let (data, _) = try await client.request(
                path: "/api/v1/live-quizzes/join/\(encodePath(normalized))/players",
                method: "POST",
                body: NicknameBody(nickname: nickname),
                authorized: false
            )
            return try decode(LiveQuizJoinPlayerResult.self, from: data)
        } catch let APIError.httpStatus(status, message) {
            throw LiveQuizJoinError(
                status: status,
                reason: LiveQuizLogic.joinErrorReason(status: status, body: message ?? "")
            )
        }
    }

    static func fetchMyGameResults(
        courseCode: String,
        gameId: String,
        accessToken: String
    ) async throws -> LiveQuizMyResults {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/live-quizzes/games/\(encodePath(gameId))/my-results",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(LiveQuizMyResults.self, from: data)
    }

    /// Course kits list (Phase 1 hub — browse + join entry). Host start is Phase 2.
    static func listQuizKits(
        courseCode: String,
        accessToken: String,
        page: Int = 1,
        pageSize: Int = 50
    ) async throws -> LiveQuizKitsListResult {
        var components = URLComponents()
        components.queryItems = [
            URLQueryItem(name: "page", value: String(page)),
            URLQueryItem(name: "pageSize", value: String(pageSize)),
        ]
        let qs = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/live-quizzes/kits\(qs)",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(LiveQuizKitsListResult.self, from: data)
    }
}

struct LiveQuizKitSummary: Codable, Equatable, Identifiable {
    var id: String
    var title: String
    var description: String?
    var questionCount: Int?
    var status: String?
    var archived: Bool?
}

struct LiveQuizKitsListResult: Codable, Equatable {
    var kits: [LiveQuizKitSummary]
    var total: Int?
    var page: Int?
    var pageSize: Int?
    var totalPages: Int?

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        kits = try c.decodeIfPresent([LiveQuizKitSummary].self, forKey: .kits) ?? []
        total = try c.decodeIfPresent(Int.self, forKey: .total)
        page = try c.decodeIfPresent(Int.self, forKey: .page)
        pageSize = try c.decodeIfPresent(Int.self, forKey: .pageSize)
        totalPages = try c.decodeIfPresent(Int.self, forKey: .totalPages)
    }

    private enum CodingKeys: String, CodingKey {
        case kits, total, page, pageSize, totalPages
    }
}
