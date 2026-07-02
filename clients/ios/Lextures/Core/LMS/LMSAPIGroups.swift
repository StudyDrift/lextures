import Foundation

/// Group spaces and collaborative documents (M7.4).
extension LMSAPI {
    static func fetchMyGroups(courseCode: String, accessToken: String) async throws -> [GroupPublic] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/my-groups",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GroupsListResponse.self, from: data).groups ?? []
    }

    static func fetchAllGroups(courseCode: String, accessToken: String) async throws -> [GroupPublic] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/groups",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GroupsListResponse.self, from: data).groups ?? []
    }

    static func fetchCollabDocs(courseCode: String, accessToken: String) async throws -> [CollabDoc] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/collab-docs",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CollabDocsListResponse.self, from: data).docs ?? []
    }

    static func fetchCollabDoc(
        courseCode: String,
        docId: String,
        accessToken: String
    ) async throws -> CollabDoc {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/collab-docs/\(encodePath(docId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CollabDoc.self, from: data)
    }
}