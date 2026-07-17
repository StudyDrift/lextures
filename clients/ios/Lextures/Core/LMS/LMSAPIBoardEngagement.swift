import Foundation

/// Board reactions, comments, and grade-sync REST (VC.M5). Mirrors web `boards-api`.
extension LMSAPI {
    static func putBoardPostReaction(
        courseCode: String,
        boardId: String,
        postId: String,
        kind: String? = nil,
        value: Double? = nil,
        accessToken: String
    ) async throws -> BoardReactionResult {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/reaction",
            method: "PUT",
            body: PutBoardReactionRequest(kind: kind, value: value),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardReactionResult.self, from: data)
    }

    static func deleteBoardPostReaction(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/reaction",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchBoardPostComments(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String
    ) async throws -> [BoardComment] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/comments",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardCommentsListResponse.self, from: data).comments ?? []
    }

    static func createBoardPostComment(
        courseCode: String,
        boardId: String,
        postId: String,
        body: BoardPostBody,
        parentId: String? = nil,
        accessToken: String
    ) async throws -> BoardComment {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/comments",
            method: "POST",
            body: CreateBoardCommentRequest(body: body, parentId: parentId),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardComment.self, from: data)
    }

    static func patchBoardPostComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        body: BoardPostBody? = nil,
        hidden: Bool? = nil,
        accessToken: String
    ) async throws -> BoardComment {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/comments/\(encodePath(commentId))",
            method: "PATCH",
            body: PatchBoardCommentRequest(body: body, hidden: hidden),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardComment.self, from: data)
    }

    static func deleteBoardPostComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/comments/\(encodePath(commentId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func syncBoardPostGrade(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String
    ) async throws -> BoardGradeSyncResult {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/grade-sync",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardGradeSyncResult.self, from: data)
    }
}
