import Foundation

/// Course feed & channels endpoints (M7.6). Group-channel variants reuse the same
/// `FeedMessage`/`FeedChannel` shapes so a future group-spaces screen (M7.4) can plug in.
extension LMSAPI {
    static func fetchFeedChannels(courseCode: String, accessToken: String) async throws -> [FeedChannel] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/channels",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedChannelsResponse.self, from: data).channels
    }

    static func createFeedChannel(courseCode: String, name: String, accessToken: String) async throws -> FeedChannel {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/channels",
            method: "POST",
            body: CreateFeedChannelRequest(name: name),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedChannel.self, from: data)
    }

    static func fetchFeedMessages(
        courseCode: String,
        channelId: String,
        accessToken: String
    ) async throws -> [FeedMessage] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/channels/\(encodePath(channelId))/messages",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedMessagesResponse.self, from: data).messages
    }

    static func postFeedMessage(
        courseCode: String,
        channelId: String,
        body: String,
        parentMessageId: String? = nil,
        accessToken: String,
        idempotencyKey: String? = nil
    ) async throws -> String {
        let request = PostFeedMessageRequest(
            body: body,
            parentMessageId: parentMessageId,
            mentionUserIds: [],
            mentionsEveryone: false
        )
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/channels/\(encodePath(channelId))/messages",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PostFeedMessageResponse.self, from: data).id
    }

    static func patchFeedMessage(
        courseCode: String,
        messageId: String,
        body: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/messages/\(encodePath(messageId))",
            method: "PATCH",
            body: PatchFeedMessageRequest(body: body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func deleteFeedMessage(courseCode: String, messageId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/messages/\(encodePath(messageId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard response.statusCode == 204 || (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func pinFeedMessage(
        courseCode: String,
        messageId: String,
        pinned: Bool,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/messages/\(encodePath(messageId))/pin",
            method: "PATCH",
            body: PinFeedMessageRequest(pinned: pinned),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func likeFeedMessage(courseCode: String, messageId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/messages/\(encodePath(messageId))/like",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func unlikeFeedMessage(courseCode: String, messageId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/messages/\(encodePath(messageId))/like",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchFeedRoster(courseCode: String, accessToken: String) async throws -> [FeedRosterPerson] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/roster",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedRosterResponse.self, from: data).people
    }

    static func uploadFeedImage(
        courseCode: String,
        imageData: Data,
        fileName: String,
        mimeType: String,
        accessToken: String,
        onProgress: ((Double) -> Void)? = nil
    ) async throws -> FeedImageUpload {
        let (data, response) = try await client.uploadMultipart(
            path: "/api/v1/courses/\(encodePath(courseCode))/feed/upload-image",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: imageData,
            accessToken: accessToken,
            onProgress: onProgress
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedImageUpload.self, from: data)
    }

    // MARK: Group feed (reused by a future group-spaces screen, M7.4)

    static func fetchGroupFeedChannels(
        courseCode: String,
        groupId: String,
        accessToken: String
    ) async throws -> [FeedChannel] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/groups/\(encodePath(groupId))/feed/channels",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedChannelsResponse.self, from: data).channels
    }

    static func fetchGroupFeedMessages(
        courseCode: String,
        groupId: String,
        channelId: String,
        accessToken: String
    ) async throws -> [FeedMessage] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/groups/\(encodePath(groupId))"
                + "/feed/channels/\(encodePath(channelId))/messages",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(FeedMessagesResponse.self, from: data).messages
    }

    static func postGroupFeedMessage(
        courseCode: String,
        groupId: String,
        channelId: String,
        body: String,
        accessToken: String,
        idempotencyKey: String? = nil
    ) async throws -> String {
        let request = PostFeedMessageRequest(
            body: body,
            parentMessageId: nil,
            mentionUserIds: [],
            mentionsEveryone: false
        )
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/groups/\(encodePath(groupId))"
                + "/feed/channels/\(encodePath(channelId))/messages",
            method: "POST",
            body: request,
            authorized: true,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PostFeedMessageResponse.self, from: data).id
    }
}
