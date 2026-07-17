import Foundation

/// Board sharing, members, and public link resolve (VC.M6). Mirrors web `boards-api`.
extension LMSAPI {
    static func patchBoardAccess(
        courseCode: String,
        boardId: String,
        visibility: String? = nil,
        visibilityTarget: String? = nil,
        attribution: String? = nil,
        canPost: Bool? = nil,
        canInteract: Bool? = nil,
        canArrange: Bool? = nil,
        accessToken: String
    ) async throws -> Board {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))",
            method: "PATCH",
            body: PatchBoardRequest(
                visibility: visibility,
                visibilityTarget: visibilityTarget,
                attribution: attribution,
                canPost: canPost,
                canInteract: canInteract,
                canArrange: canArrange
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func fetchBoardMembers(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> [BoardMember] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/members",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardMembersListResponse.self, from: data).members ?? []
    }

    static func upsertBoardMember(
        courseCode: String,
        boardId: String,
        userId: String,
        role: String,
        accessToken: String
    ) async throws -> BoardMember {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/members",
            method: "POST",
            body: UpsertBoardMemberRequest(userId: userId, role: role),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardMember.self, from: data)
    }

    static func removeBoardMember(
        courseCode: String,
        boardId: String,
        userId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/members/\(encodePath(userId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchBoardShares(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> [BoardShare] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/shares",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardSharesListResponse.self, from: data).shares ?? []
    }

    static func createBoardShare(
        courseCode: String,
        boardId: String,
        capability: String,
        password: String? = nil,
        expiresAt: String? = nil,
        accessToken: String
    ) async throws -> BoardShare {
        let trimmedPassword = password?.trimmingCharacters(in: .whitespacesAndNewlines)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/shares",
            method: "POST",
            body: CreateBoardShareRequest(
                capability: capability,
                password: (trimmedPassword?.isEmpty == false) ? trimmedPassword : nil,
                expiresAt: expiresAt
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardShare.self, from: data)
    }

    static func revokeBoardShare(
        courseCode: String,
        boardId: String,
        shareId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/shares/\(encodePath(shareId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    /// Public resolve — no auth token; password via header only (never logged).
    static func resolveBoardLink(token: String, password: String? = nil) async throws -> BoardLinkResolve {
        var headers: [String: String] = [:]
        if let password, !password.isEmpty {
            headers["X-Board-Share-Password"] = password
        }
        let (data, response) = try await client.request(
            path: "/api/v1/board-links/\(encodePath(token))",
            authorized: false,
            extraHeaders: headers
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardLinkResolve.self, from: data)
    }

    static func createBoardLinkPost(
        token: String,
        displayName: String,
        text: String,
        password: String? = nil
    ) async throws -> BoardPost {
        var headers: [String: String] = [:]
        if let password, !password.isEmpty {
            headers["X-Board-Share-Password"] = password
        }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        let (data, response) = try await client.request(
            path: "/api/v1/board-links/\(encodePath(token))/posts",
            method: "POST",
            body: CreateBoardLinkPostRequest(
                displayName: displayName.trimmingCharacters(in: .whitespacesAndNewlines),
                contentType: "text",
                title: "",
                body: BoardPostBody(html: trimmed, text: trimmed)
            ),
            authorized: false,
            extraHeaders: headers
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardPost.self, from: data)
    }
}
