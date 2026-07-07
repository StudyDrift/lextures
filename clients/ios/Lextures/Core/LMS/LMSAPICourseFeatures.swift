import Foundation

/// Course features API (M13.2).
extension LMSAPI {
    static func patchCourseFeatures(
        courseCode: String,
        body: CourseFeaturesPatch,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/features",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func patchCourseCaptionPolicy(
        courseCode: String,
        requireCaptions: Bool,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/caption-policy",
            method: "PATCH",
            body: CourseCaptionPolicyPatch(requireCaptions: requireCaptions),
            authorized: true,
            accessToken: accessToken
        )
    }

    static func fetchCourseConsortiumSettings(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseConsortiumSettings? {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/consortium-settings",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseConsortiumSettings.self, from: data)
    }

    static func patchCourseConsortiumSettings(
        courseCode: String,
        consortiumShareable: Bool,
        accessToken: String
    ) async throws -> CourseConsortiumSettings {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/consortium-settings",
            method: "PATCH",
            bodyData: try JSONEncoder().encode(
                CourseConsortiumSettingsPatch(consortiumShareable: consortiumShareable)
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseConsortiumSettings.self, from: data)
    }
}
