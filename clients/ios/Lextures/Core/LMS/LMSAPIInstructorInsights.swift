import Foundation

/// Staff course health, at-risk alerts, and student progress (M11.3).
extension LMSAPI {
    static func fetchCourseAtRisk(courseCode: String, accessToken: String) async throws -> AtRiskListResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/at-risk",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AtRiskListResponse.self, from: data)
    }

    static func fetchInstructorInsights(courseCode: String, accessToken: String) async throws -> InstructorInsightsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/analytics/insights",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(InstructorInsightsResponse.self, from: data)
    }

    static func fetchStudentProgress(
        courseCode: String,
        enrollmentId: String,
        accessToken: String
    ) async throws -> StudentProgressResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/enrollments/\(encodePath(enrollmentId))/progress",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudentProgressResponse.self, from: data)
    }

    static func fetchStudentProgressActivity(
        courseCode: String,
        enrollmentId: String,
        cursor: String?,
        accessToken: String
    ) async throws -> StudentProgressActivityResponse {
        var path = "/api/v1/courses/\(encodePath(courseCode))/enrollments/\(encodePath(enrollmentId))/progress/activity"
        if let cursor, !cursor.isEmpty {
            path += "?cursor=\(encodePath(cursor))"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudentProgressActivityResponse.self, from: data)
    }
}
