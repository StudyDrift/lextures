import Foundation

/// Student academic advising: notes, degree progress, appointment config (M7.8).
extension LMSAPI {
    static func fetchAdvisingNotes(accessToken: String) async throws -> [AdvisingNote] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/advising-notes",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.advising.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdvisingNotesResponse.self, from: data).notes ?? []
    }

    static func fetchDegreeProgress(accessToken: String) async throws -> DegreeProgress {
        let (data, response) = try await client.request(
            path: "/api/v1/me/degree-progress",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.advising.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DegreeProgress.self, from: data)
    }

    static func fetchMyAdvisingConfig(accessToken: String) async throws -> MyAdvisingConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/me/advising/config",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.advising.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MyAdvisingConfig.self, from: data)
    }
}
