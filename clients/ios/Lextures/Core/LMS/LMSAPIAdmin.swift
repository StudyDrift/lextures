import Foundation

/// Admin settings API (M14.10) — global archived courses.
extension LMSAPI {
    static func fetchArchivedCourses(accessToken: String) async throws -> [ArchivedCourseRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ArchivedCoursesListResponse.self, from: data).courses
    }

    static func restoreArchivedCourse(courseCode: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses/\(encodePath(courseCode))/restore",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func deleteArchivedCoursePermanently(courseCode: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses/\(encodePath(courseCode))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }
}