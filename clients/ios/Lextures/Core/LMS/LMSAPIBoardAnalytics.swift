import Foundation

/// Board analytics & org governance REST (MOB.8 / VC.10). Mirrors web `boards-api`.
extension LMSAPI {
    static func fetchBoardAnalytics(
        courseCode: String,
        boardId: String,
        days: Int = 14,
        accessToken: String
    ) async throws -> BoardAnalyticsSummary {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/analytics?days=\(days)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardAnalyticsSummary.self, from: data)
    }

    static func fetchAdminBoardPolicies(
        orgId: String? = nil,
        accessToken: String
    ) async throws -> BoardOrgPolicies {
        var path = "/api/v1/admin/boards/policies"
        if let orgId, !orgId.isEmpty {
            path += "?orgId=\(encodePath(orgId))"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardOrgPolicies.self, from: data)
    }

    static func patchAdminBoardPolicies(
        externalSharing: Bool? = nil,
        minorModerationFloor: Bool? = nil,
        defaultAttribution: String? = nil,
        boardCapPerCourse: Int? = nil,
        clearBoardCap: Bool? = nil,
        orgId: String? = nil,
        accessToken: String
    ) async throws -> BoardOrgPolicies {
        var path = "/api/v1/admin/boards/policies"
        if let orgId, !orgId.isEmpty {
            path += "?orgId=\(encodePath(orgId))"
        }
        let (data, response) = try await client.request(
            path: path,
            method: "PATCH",
            body: PatchBoardOrgPoliciesRequest(
                externalSharing: externalSharing,
                minorModerationFloor: minorModerationFloor,
                defaultAttribution: defaultAttribution,
                boardCapPerCourse: boardCapPerCourse,
                clearBoardCap: clearBoardCap
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardOrgPolicies.self, from: data)
    }

    static func fetchAdminBoardsOverview(
        orgId: String? = nil,
        activeDays: Int = 30,
        accessToken: String
    ) async throws -> BoardAdminOverview {
        var items = ["activeDays=\(activeDays)"]
        if let orgId, !orgId.isEmpty {
            items.append("orgId=\(encodePath(orgId))")
        }
        let path = "/api/v1/admin/boards/overview?\(items.joined(separator: "&"))"
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardAdminOverview.self, from: data)
    }
}
