import Foundation

/// District blueprint course API (M13.11).
extension LMSAPI {
    static func patchCourseBlueprint(
        courseCode: String,
        isBlueprint: Bool,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint",
            method: "PATCH",
            body: BlueprintPatchRequest(isBlueprint: isBlueprint),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func fetchBlueprintChildren(
        courseCode: String,
        accessToken: String
    ) async throws -> [BlueprintChildRow] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint/children",
            authorized: true,
            accessToken: accessToken
        )
        let response = try decode(BlueprintChildrenResponse.self, from: data)
        return response.children ?? []
    }

    static func postBlueprintChildLink(
        courseCode: String,
        childCourseCode: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint/children",
            method: "POST",
            body: BlueprintLinkChildRequest(childCourseCode: childCourseCode),
            authorized: true,
            accessToken: accessToken
        )
    }

    static func deleteBlueprintChildLink(
        courseCode: String,
        childCourseCode: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint/children/\(encodePath(childCourseCode))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func postBlueprintPush(
        courseCode: String,
        accessToken: String
    ) async throws -> BlueprintPushResult {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint/push",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(BlueprintPushResult.self, from: data)
    }

    static func fetchBlueprintSyncLogs(
        courseCode: String,
        accessToken: String
    ) async throws -> [BlueprintSyncLogRow] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/blueprint/sync-logs",
            authorized: true,
            accessToken: accessToken
        )
        let response = try decode(BlueprintSyncLogsResponse.self, from: data)
        return response.logs ?? []
    }

    static func fetchBlueprintPayload(
        courseCode: String,
        accessToken: String
    ) async throws -> BlueprintCachedPayload {
        async let children = fetchBlueprintChildren(courseCode: courseCode, accessToken: accessToken)
        async let logs = fetchBlueprintSyncLogs(courseCode: courseCode, accessToken: accessToken)
        return try await BlueprintCachedPayload(children: children, syncLogs: logs)
    }
}
