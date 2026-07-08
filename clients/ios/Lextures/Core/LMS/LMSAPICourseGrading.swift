import Foundation

/// Course grading settings API (M13.4).
extension LMSAPI {
    static func fetchCourseGradingSettings(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseGradingSettings {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grading",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseGradingSettings.self, from: data)
    }

    static func putCourseGradingSettings(
        courseCode: String,
        body: PutCourseGradingSettingsBody,
        accessToken: String
    ) async throws -> CourseGradingSettings {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grading",
            method: "PUT",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseGradingSettings.self, from: data)
    }

    static func fetchCourseGradingScheme(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseGradingSchemeRecord? {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grading-scheme",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let envelope = try decode(CourseGradingSchemeEnvelope.self, from: data)
        return envelope.scheme
    }

    static func putCourseGradingScheme(
        courseCode: String,
        body: PutCourseGradingSchemeBody,
        accessToken: String
    ) async throws -> CourseGradingSchemeRecord? {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grading-scheme",
            method: "PUT",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let envelope = try decode(CourseGradingSchemeEnvelope.self, from: data)
        return envelope.scheme
    }

    static func patchCourseStructureItemAssignmentGroup(
        courseCode: String,
        itemId: String,
        assignmentGroupId: String?,
        accessToken: String
    ) async throws -> CourseStructureItem {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/items/\(encodePath(itemId))/assignment-group",
            method: "PATCH",
            bodyData: try JSONEncoder().encode(PatchItemAssignmentGroupBody(assignmentGroupId: assignmentGroupId)),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseStructureItem.self, from: data)
    }
}