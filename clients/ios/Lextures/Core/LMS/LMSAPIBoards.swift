import Foundation

/// Visual collaboration boards REST (VC.M1). Mirrors web `boards-api`.
extension LMSAPI {
    static func fetchBoards(
        courseCode: String,
        includeArchived: Bool = false,
        accessToken: String
    ) async throws -> [Board] {
        var path = "/api/v1/courses/\(encodePath(courseCode))/boards"
        if includeArchived {
            path += "?includeArchived=true"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardsListResponse.self, from: data).boards ?? []
    }

    static func createBoard(
        courseCode: String,
        title: String,
        description: String = "",
        accessToken: String
    ) async throws -> Board {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards",
            method: "POST",
            body: CreateBoardRequest(title: title, description: description),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func fetchBoard(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> Board {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func patchBoard(
        courseCode: String,
        boardId: String,
        title: String? = nil,
        description: String? = nil,
        archived: Bool? = nil,
        layout: String? = nil,
        layoutLocked: Bool? = nil,
        accessToken: String
    ) async throws -> Board {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))",
            method: "PATCH",
            body: PatchBoardRequest(
                title: title,
                description: description,
                archived: archived,
                layout: layout,
                layoutLocked: layoutLocked
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func deleteBoard(
        courseCode: String,
        boardId: String,
        hard: Bool = false,
        accessToken: String
    ) async throws {
        var path = "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))"
        if hard {
            path += "?hard=true"
        }
        let (data, response) = try await client.request(
            path: path,
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }
}
