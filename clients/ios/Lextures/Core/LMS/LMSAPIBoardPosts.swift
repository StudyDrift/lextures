import Foundation

/// Board posts, attachments, and link preview REST (VC.M2). Mirrors web `boards-api`.
extension LMSAPI {
    static func fetchBoardPosts(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> [BoardPost] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let posts = try decode(BoardPostsListResponse.self, from: data).posts ?? []
        return posts.map(normalizePost)
    }

    static func createBoardPost(
        courseCode: String,
        boardId: String,
        contentType: String,
        title: String? = nil,
        body: BoardPostBody? = nil,
        linkUrl: String? = nil,
        attachmentId: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts",
            method: "POST",
            body: CreateBoardPostRequest(
                contentType: contentType,
                title: title,
                body: body,
                linkUrl: linkUrl,
                attachmentId: attachmentId
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return normalizePost(try decode(BoardPost.self, from: data))
    }

    static func patchBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        title: String? = nil,
        body: BoardPostBody? = nil,
        linkUrl: String? = nil,
        accessToken: String
    ) async throws -> BoardPost {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))",
            method: "PATCH",
            body: PatchBoardPostRequest(title: title, body: body, linkUrl: linkUrl),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return normalizePost(try decode(BoardPost.self, from: data))
    }

    static func deleteBoardPost(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/posts/\(encodePath(postId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func uploadBoardAttachment(
        courseCode: String,
        boardId: String,
        fileName: String,
        mimeType: String,
        fileData: Data,
        altText: String? = nil,
        contentType: String? = nil,
        accessToken: String,
        onProgress: ((Double) -> Void)? = nil
    ) async throws -> BoardAttachment {
        var extra: [String: String] = [:]
        if let altText, !altText.isEmpty { extra["altText"] = altText }
        if let contentType, !contentType.isEmpty { extra["contentType"] = contentType }
        let (data, _) = try await client.uploadMultipart(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/attachments",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: fileData,
            extraFields: extra,
            accessToken: accessToken,
            onProgress: onProgress
        )
        return normalizeAttachment(try decode(BoardAttachment.self, from: data))
    }

    static func fetchBoardLinkPreview(
        courseCode: String,
        boardId: String,
        url: String,
        accessToken: String
    ) async throws -> BoardLinkPreview {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/link-preview",
            method: "POST",
            body: BoardLinkPreviewRequest(url: url),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardLinkPreview.self, from: data)
    }

    private static func normalizePost(_ post: BoardPost) -> BoardPost {
        var next = post
        if let att = post.attachment {
            next.attachment = normalizeAttachment(att)
        }
        return next
    }

    private static func normalizeAttachment(_ att: BoardAttachment) -> BoardAttachment {
        var next = att
        if let url = att.url, let absolute = BoardsLogic.absoluteURL(url) {
            next.url = absolute.absoluteString
        }
        return next
    }
}
