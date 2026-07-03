import Foundation

/// Gamification profile, badges, streak freeze, and course leaderboard (M9.3).
extension LMSAPI {
    static func fetchGamificationProfile(accessToken: String) async throws -> GamificationProfile {
        let (data, response) = try await client.request(
            path: "/api/v1/me/gamification",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.gamification.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GamificationProfile.self, from: data)
    }

    static func fetchMyGamificationBadges(accessToken: String) async throws -> [GamificationBadge] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/badges",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GamificationBadgesListResponse.self, from: data).badges ?? []
    }

    static func freezeGamificationStreak(accessToken: String) async throws -> GamificationProfile {
        let (data, response) = try await client.request(
            path: "/api/v1/me/gamification/freeze-streak",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GamificationProfile.self, from: data)
    }

    static func fetchCourseLeaderboard(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseLeaderboardResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/leaderboard",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.gamification.leaderboardUnavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseLeaderboardResponse.self, from: data)
    }
}