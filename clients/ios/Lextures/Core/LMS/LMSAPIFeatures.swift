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

    // MARK: - Account settings (editable profile)

    static func fetchAccountProfile(accessToken: String) async throws -> AccountProfile {
        let (data, _) = try await client.request(
            path: "/api/v1/settings/account",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(AccountProfile.self, from: data)
    }

    static func updateAccountProfile(
        _ patch: AccountProfilePatch,
        accessToken: String
    ) async throws -> AccountProfile {
        let (data, _) = try await client.request(
            path: "/api/v1/settings/account",
            method: "PATCH",
            body: patch,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(AccountProfile.self, from: data)
    }

    // MARK: - My accommodations

    static func fetchMyAccommodations(accessToken: String) async throws -> [MyAccommodation] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/accommodations",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(MyAccommodationsResponse.self, from: data).accommodations
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

    // MARK: - Device push tokens (APNs / FCM)

    static func registerDeviceToken(
        token: String,
        platform: String,
        accessToken: String
    ) async throws -> DeviceTokenResponse {
        let bundleId = Bundle.main.bundleIdentifier
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String
        let body = DeviceTokenRegistration(
            token: token,
            platform: platform,
            appBundleId: bundleId,
            appVersion: version
        )
        let (data, _) = try await client.request(
            path: "/api/v1/me/device-tokens",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(DeviceTokenResponse.self, from: data)
    }

    static func deregisterDeviceToken(id: String, accessToken: String) async throws {
        _ = try await client.request(
            path: "/api/v1/me/device-tokens/\(encodePath(id))",
            method: "DELETE",
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

    static func fetchQuizAttempts(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> [QuizAttemptSummary] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/attempts",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(QuizAttemptsListResponse.self, from: data).attempts
    }

    static func fetchGradingSubmissions(
        courseCode: String,
        backlogItem: GradingBacklogItem,
        graded: String?,
        accessToken: String
    ) async throws -> [AssignmentSubmission] {
        if backlogItem.isQuiz {
            let attempts = try await fetchQuizAttempts(
                courseCode: courseCode,
                itemId: backlogItem.resolvedItemId,
                accessToken: accessToken
            )
            let submissions = GradingSubmissionMapper.quizAttemptsToSubmissions(attempts)
            return GradingSubmissionMapper.filterSubmissions(submissions, graded: graded ?? "all")
        }
        return try await fetchSubmissions(
            courseCode: courseCode,
            itemId: backlogItem.resolvedItemId,
            graded: graded,
            accessToken: accessToken
        )
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

    // MARK: - Onboarding (plan 15.11 / M1.3)

    /// Returns nil when the onboarding feature flag is off (HTTP 404).
    static func fetchOnboardingStatus(accessToken: String) async throws -> OnboardingStatus? {
        let (data, response) = try await client.request(
            path: "/api/v1/me/onboarding-status",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OnboardingStatus.self, from: data)
    }

    static func postOnboarding(body: [String: Any], accessToken: String) async throws -> LearnerGoals {
        let bodyData = try JSONSerialization.data(withJSONObject: body)
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/me/onboarding",
            method: "POST",
            bodyData: bodyData,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GoalsEnvelope.self, from: data).goals
    }

    static func fetchDiagnosticQuestions(topic: String, accessToken: String) async throws -> [DiagnosticQuestion] {
        let encoded = topic.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? topic
        let (data, _) = try await client.request(
            path: "/api/v1/me/onboarding/diagnostic-questions?topic=\(encoded)",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(DiagnosticQuestionsResponse.self, from: data).questions
    }

    static func saveStudyReminderPrefs(optIn: Bool, reminderTime: String, accessToken: String) async {
        guard optIn else { return }
        struct Preference: Encodable {
            var eventType = "study_reminder"
            var emailEnabled = true
            var pushEnabled = true
            var digestMode = "instant"
        }
        struct Body: Encodable {
            var preferences: [Preference]
        }
        _ = try? await client.request(
            path: "/api/v1/me/notification-preferences",
            method: "PUT",
            body: Body(preferences: [Preference()]),
            authorized: true,
            accessToken: accessToken
        )
        _ = reminderTime
    }
}
