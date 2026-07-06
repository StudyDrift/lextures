import Foundation

/// Announcement and district broadcast endpoints (M11.2).
extension LMSAPI {
    // MARK: - Announcements (org broadcasts)

    static func fetchMyBroadcasts(accessToken: String) async throws -> [Broadcast] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/broadcasts",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(BroadcastsResponse.self, from: data).broadcasts
    }

    static func acknowledgeBroadcast(id: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/broadcasts/\(encodePath(id))/acknowledge",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func createBroadcast(
        orgId: String,
        type: String,
        subject: String,
        body: String,
        accessToken: String
    ) async throws -> Broadcast {
        let (data, response) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/broadcasts",
            method: "POST",
            body: CreateBroadcastRequest(type: type, subject: subject, body: body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CreateBroadcastResponse.self, from: data).broadcast
    }

    static func createCourseAnnouncement(
        courseCode: String,
        channelId: String,
        title: String,
        body: String,
        sectionName: String?,
        mentionsEveryone: Bool,
        accessToken: String
    ) async throws -> String {
        let text = AnnouncementLogic.formatAnnouncementBody(
            title: title,
            body: body,
            sectionName: sectionName,
            mentionsEveryone: mentionsEveryone
        )
        return try await postFeedMessage(
            courseCode: courseCode,
            channelId: channelId,
            body: text,
            accessToken: accessToken
        )
    }
}