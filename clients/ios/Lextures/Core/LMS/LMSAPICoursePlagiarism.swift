import Foundation

/// Course plagiarism settings API (M13.7).
extension LMSAPI {
    static func fetchCoursePlagiarismSettings(
        courseCode: String,
        accessToken: String
    ) async throws -> CoursePlagiarismSettings {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/plagiarism-settings",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CoursePlagiarismSettings.self, from: data)
    }

    static func patchCoursePlagiarismSettings(
        courseCode: String,
        body: PatchCoursePlagiarismBody,
        accessToken: String
    ) async throws -> CoursePlagiarismSettings {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/plagiarism-settings",
            method: "PATCH",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CoursePlagiarismSettings.self, from: data)
    }
}
