import Foundation

// MARK: - Intro course (IC07)

extension LMSAPI {
    static func fetchIntroCourseProgress(accessToken: String) async throws -> IntroCourseProgress {
        let (data, response) = try await client.request(
            path: "/api/v1/me/intro-course",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(IntroCourseProgress.self, from: data)
    }

    static func markIntroCelebrationSeen(accessToken: String) async throws {
        let (_, response) = try await client.request(
            path: "/api/v1/me/intro-course/celebration-seen",
            method: "PUT",
            authorized: true,
            accessToken: accessToken
        )
        guard response.statusCode == 204 || (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: "Could not save celebration state.")
        }
    }
}