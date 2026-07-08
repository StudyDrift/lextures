import Foundation

/// Course outcomes settings API (M13.5).
extension LMSAPI {
    static func fetchCourseOutcomes(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseOutcomesListResponse {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseOutcomesListResponse.self, from: data)
    }

    static func createCourseOutcome(
        courseCode: String,
        body: CreateCourseOutcomeBody,
        accessToken: String
    ) async throws -> CourseOutcome {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes",
            method: "POST",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseOutcome.self, from: data)
    }

    static func patchCourseOutcome(
        courseCode: String,
        outcomeId: String,
        body: PatchCourseOutcomeBody,
        accessToken: String
    ) async throws -> CourseOutcome {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes/\(encodePath(outcomeId))",
            method: "PATCH",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseOutcome.self, from: data)
    }

    static func deleteCourseOutcome(
        courseCode: String,
        outcomeId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes/\(encodePath(outcomeId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func addCourseOutcomeLink(
        courseCode: String,
        outcomeId: String,
        body: AddCourseOutcomeLinkBody,
        accessToken: String
    ) async throws -> CourseOutcomeLink {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes/\(encodePath(outcomeId))/links",
            method: "POST",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseOutcomeLink.self, from: data)
    }

    static func deleteCourseOutcomeLink(
        courseCode: String,
        outcomeId: String,
        linkId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes/\(encodePath(outcomeId))/links/\(encodePath(linkId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func createCourseOutcomeSubOutcome(
        courseCode: String,
        outcomeId: String,
        body: CreateCourseOutcomeSubOutcomeBody,
        accessToken: String
    ) async throws -> CourseOutcomeSubOutcome {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/outcomes/\(encodePath(outcomeId))/sub-outcomes",
            method: "POST",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseOutcomeSubOutcome.self, from: data)
    }
}