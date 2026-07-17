import Foundation

/// Board layout, sections, and arrange REST (VC.M3). Mirrors web `boards-api`.
extension LMSAPI {
    static func fetchBoardSections(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> [BoardSection] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/sections",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardSectionsListResponse.self, from: data).sections ?? []
    }

    static func createBoardSection(
        courseCode: String,
        boardId: String,
        title: String,
        sortIndex: Double? = nil,
        accessToken: String
    ) async throws -> BoardSection {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/sections",
            method: "POST",
            body: CreateBoardSectionRequest(title: title, sortIndex: sortIndex),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardSection.self, from: data)
    }

    static func patchBoardSection(
        courseCode: String,
        boardId: String,
        sectionId: String,
        title: String? = nil,
        sortIndex: Double? = nil,
        accessToken: String
    ) async throws -> BoardSection {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/sections/\(encodePath(sectionId))",
            method: "PATCH",
            body: PatchBoardSectionRequest(title: title, sortIndex: sortIndex),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardSection.self, from: data)
    }

    static func deleteBoardSection(
        courseCode: String,
        boardId: String,
        sectionId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/sections/\(encodePath(sectionId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func arrangeBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        input: ArrangeBoardPostInput,
        accessToken: String
    ) async throws -> BoardPost {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/arrange",
            method: "PATCH",
            body: input,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardPost.self, from: data)
    }
}
