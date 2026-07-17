import Foundation

/// Board moderation, reports, and safety queue REST (VC.M7). Mirrors web `boards-api`.
extension LMSAPI {
    static func fetchBoardModerationQueue(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> BoardModerationQueue {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/moderation/queue",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardModerationQueue.self, from: data)
    }

    static func patchBoardModeration(
        courseCode: String,
        boardId: String,
        moderationMode: String? = nil,
        filterAction: String? = nil,
        locked: Bool? = nil,
        frozenUntil: String? = nil,
        freezeMinutes: Int? = nil,
        accessToken: String
    ) async throws -> Board {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))",
            method: "PATCH",
            body: PatchBoardRequest(
                moderationMode: moderationMode,
                filterAction: filterAction,
                locked: locked,
                frozenUntil: frozenUntil,
                freezeMinutes: freezeMinutes
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func approveBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        try await postModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            action: "approve",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func rejectBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        try await postModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            action: "reject",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func hideBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        try await postModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            action: "hide",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func removeBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        try await postModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            action: "remove",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func hideBoardComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardComment {
        try await commentModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            commentId: commentId,
            action: "hide",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func removeBoardComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardComment {
        try await commentModerationAction(
            courseCode: courseCode,
            boardId: boardId,
            postId: postId,
            commentId: commentId,
            action: "remove",
            reason: reason,
            accessToken: accessToken
        )
    }

    static func reportBoardContent(
        courseCode: String,
        boardId: String,
        postId: String? = nil,
        commentId: String? = nil,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardReport {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/reports",
            method: "POST",
            body: CreateBoardReportRequest(postId: postId, commentId: commentId, reason: reason),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardReport.self, from: data)
    }

    static func resolveBoardReport(
        courseCode: String,
        boardId: String,
        reportId: String,
        action: String,
        reason: String? = nil,
        accessToken: String
    ) async throws -> BoardReport {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/reports/\(encodePath(reportId))/resolve",
            method: "POST",
            body: ResolveBoardReportRequest(action: action, reason: reason),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardReport.self, from: data)
    }

    private static func postModerationAction(
        courseCode: String,
        boardId: String,
        postId: String,
        action: String,
        reason: String?,
        accessToken: String
    ) async throws -> BoardPost {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/\(action)",
            method: "POST",
            body: BoardModerationActionRequest(reason: reason ?? ""),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardPost.self, from: data)
    }

    private static func commentModerationAction(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        action: String,
        reason: String?,
        accessToken: String
    ) async throws -> BoardComment {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))/comments/\(encodePath(commentId))/\(action)",
            method: "POST",
            body: BoardModerationActionRequest(reason: reason ?? ""),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardComment.self, from: data)
    }
}
