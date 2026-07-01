import Foundation

/// Discussions endpoints (M7.1).
extension LMSAPI {
    static func fetchDiscussionForums(courseCode: String, accessToken: String) async throws -> [DiscussionForum] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/forums",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionForumsResponse.self, from: data).forums ?? []
    }

    static func fetchDiscussionThreads(
        courseCode: String,
        forumId: String,
        accessToken: String
    ) async throws -> [DiscussionThreadSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/forums/\(encodePath(forumId))/threads",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionThreadsResponse.self, from: data).threads ?? []
    }

    static func fetchDiscussionThread(
        courseCode: String,
        threadId: String,
        accessToken: String
    ) async throws -> DiscussionThreadDetail {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/discussion-threads/\(encodePath(threadId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionThreadDetail.self, from: data)
    }

    static func fetchDiscussionPosts(
        courseCode: String,
        threadId: String,
        accessToken: String
    ) async throws -> DiscussionPostsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/discussion-threads/\(encodePath(threadId))/posts",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionPostsResponse.self, from: data)
    }

    static func createDiscussionThread(
        courseCode: String,
        forumId: String,
        title: String,
        body: Data,
        accessToken: String
    ) async throws -> DiscussionThreadDetail {
        let request = CreateDiscussionThreadRequest(title: title, body: body)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/forums/\(encodePath(forumId))/threads",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionThreadDetail.self, from: data)
    }

    static func createDiscussionPost(
        courseCode: String,
        threadId: String,
        parentPostId: String?,
        body: Data,
        accessToken: String,
        idempotencyKey: String? = nil
    ) async throws -> DiscussionPost {
        let request = CreateDiscussionPostRequest(
            parentPostId: parentPostId,
            body: body,
            idempotencyKey: idempotencyKey
        )
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/discussion-threads/\(encodePath(threadId))/posts",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionPost.self, from: data)
    }

    static func deleteDiscussionPost(
        courseCode: String,
        postId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/discussion-posts/\(encodePath(postId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard response.statusCode == 204 || (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func upvoteDiscussionPost(
        courseCode: String,
        postId: String,
        accessToken: String
    ) async throws -> DiscussionUpvoteResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/discussion-posts/\(encodePath(postId))/upvote",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(DiscussionUpvoteResponse.self, from: data)
    }
}