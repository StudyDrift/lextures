import Foundation

/// Endpoints behind the redesigned shell: profile, notifications, announcements,
/// grades, syllabus, submissions, grading, and attendance.
/// Shapes are documented in `docs/MOBILE_PLAN.md` §2.
extension LMSAPI {
    // MARK: - Profile

    static func fetchMe(accessToken: String) async throws -> MeProfile {
        let (data, _) = try await client.request(
            path: "/api/v1/me",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(MeProfile.self, from: data)
    }

    // MARK: - Notifications

    static func fetchNotifications(accessToken: String) async throws -> NotificationsPage {
        let (data, _) = try await client.request(
            path: "/api/v1/me/notifications",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotificationsPage.self, from: data)
    }

    static func markNotificationRead(id: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/notifications/\(encodePath(id))/read",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func markAllNotificationsRead(accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/notifications/read-all",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
    }

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

    // MARK: - My grades (student)

    static func fetchMyGrades(courseCode: String, accessToken: String) async throws -> MyGradesResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/my-grades",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(MyGradesResponse.self, from: data)
    }

    // MARK: - Syllabus

    static func fetchSyllabus(courseCode: String, accessToken: String) async throws -> SyllabusPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/syllabus",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SyllabusPayload.self, from: data)
    }

    // MARK: - Assignment submissions

    static func fetchMySubmission(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> AssignmentSubmission? {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/mine",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(MySubmissionResponse.self, from: data).submission
    }

    static func fetchSubmissions(
        courseCode: String,
        itemId: String,
        graded: String?, // "graded" | "ungraded" | nil for all
        accessToken: String
    ) async throws -> [AssignmentSubmission] {
        var path = "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions"
        if let graded, !graded.isEmpty {
            path += "?graded=\(graded)"
        }
        let (data, _) = try await client.request(path: path, authorized: true, accessToken: accessToken)
        return try decode(SubmissionsListResponse.self, from: data).submissions
    }

    static func fetchSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String
    ) async throws -> SubmissionGrade {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/\(encodePath(submissionId))/grade",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SubmissionGrade.self, from: data)
    }

    struct SubmissionGradePut: Encodable {
        var pointsEarned: Double?
        var instructorComment: String?
        var clearGrade: Bool?
    }

    static func putSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        body: SubmissionGradePut,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/\(encodePath(submissionId))/grade",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
    }

    // MARK: - Grading backlog (staff)

    static func fetchGradingBacklog(courseCode: String, accessToken: String) async throws -> [GradingBacklogItem] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/grading-backlog",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(GradingBacklogResponse.self, from: data).items
    }

    // MARK: - Attendance

    static func fetchAttendanceSessions(courseCode: String, accessToken: String) async throws -> [AttendanceSession] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(AttendanceSessionsResponse.self, from: data).sessions
    }

    static func fetchAttendanceSessionDetail(
        courseCode: String,
        sessionId: String,
        accessToken: String
    ) async throws -> AttendanceSessionDetail {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions/\(encodePath(sessionId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(AttendanceSessionDetail.self, from: data)
    }

    struct SelfReportBody: Encodable {
        var status: String
    }

    static func selfReportAttendance(
        courseCode: String,
        sessionId: String,
        status: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions/\(encodePath(sessionId))/self-report",
            method: "POST",
            body: SelfReportBody(status: status),
            authorized: true,
            accessToken: accessToken
        )
    }
}
