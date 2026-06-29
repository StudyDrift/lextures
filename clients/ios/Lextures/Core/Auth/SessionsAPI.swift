import Foundation

/// Device session list/revoke endpoints (`GET/DELETE /api/v1/me/sessions`).
enum SessionsAPI {
    static let client = APIClient()

    struct ActiveSession: Decodable, Identifiable, Equatable {
        var id: String
        var createdAt: String
        var lastUsedAt: String
        var deviceLabel: String
        var location: String
        var authMethod: String
        var isCurrent: Bool
    }

    private struct SessionsResponse: Decodable {
        var sessions: [ActiveSession]
    }

    static func fetchSessions(accessToken: String) async throws -> [ActiveSession] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/sessions",
            authorized: true,
            accessToken: accessToken
        )
        return try LMSAPI.decode(SessionsResponse.self, from: data).sessions
    }

    static func revokeSession(id: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/sessions/\(encodePath(id))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func revokeOtherSessions(accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/sessions",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    private static func encodePath(_ value: String) -> String {
        value.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? value
    }
}
