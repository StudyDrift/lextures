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

    static func fetchNotificationPreferences(accessToken: String) async throws -> [NotificationPreference] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/notification-preferences",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotificationPreferencesResponse.self, from: data).preferences
    }

    static func updateNotificationPreferences(
        _ preferences: [NotificationPreference],
        accessToken: String
    ) async throws -> [NotificationPreference] {
        let body = NotificationPreferencesUpdate(
            preferences: preferences.map {
                NotificationPreferencePatch(
                    eventType: $0.eventType,
                    emailEnabled: $0.emailEnabled,
                    pushEnabled: $0.pushEnabled,
                    smsEnabled: $0.smsEnabled,
                    digestMode: $0.digestMode
                )
            }
        )
        let (data, _) = try await client.request(
            path: "/api/v1/me/notification-preferences",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(NotificationPreferencesResponse.self, from: data).preferences
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

    static func fetchPlatformFeatures(accessToken: String) async throws -> PlatformFeatures {
        let (data, _) = try await client.request(
            path: "/api/v1/platform/features",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(PlatformFeatures.self, from: data)
    }

    static func fetchSubmissionAnnotations(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String
    ) async throws -> [SubmissionAnnotation] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/\(encodePath(submissionId))/annotations",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SubmissionAnnotationsResponse.self, from: data).annotations
    }

    static func fetchSubmissionFeedbackMedia(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String
    ) async throws -> [SubmissionFeedbackMedia] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/\(encodePath(submissionId))/feedback-media",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SubmissionFeedbackMediaResponse.self, from: data).items
    }

    static func fetchFeedbackPlaybackInfo(
        courseCode: String,
        itemId: String,
        submissionId: String,
        mediaId: String,
        accessToken: String
    ) async throws -> FeedbackPlaybackInfo {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/\(encodePath(submissionId))/feedback-media/\(encodePath(mediaId))/url",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(FeedbackPlaybackInfo.self, from: data)
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

    static func submitAssignmentText(
        courseCode: String,
        itemId: String,
        text: String,
        accessToken: String
    ) async throws -> AssignmentSubmission {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/text",
            method: "POST",
            body: SubmitAssignmentTextRequest(text: text),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SubmitAssignmentResponse.self, from: data).submission
    }

    static func uploadAssignmentFile(
        courseCode: String,
        itemId: String,
        fileData: Data,
        fileName: String,
        mimeType: String,
        accessToken: String,
        onProgress: ((Double) -> Void)? = nil
    ) async throws -> AssignmentSubmission {
        let (data, _) = try await client.uploadMultipart(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(itemId))/submissions/upload",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: fileData,
            accessToken: accessToken,
            onProgress: onProgress
        )
        return try decode(SubmitAssignmentResponse.self, from: data).submission
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

    // MARK: - Quiz delivery (M4.1)

    static func fetchModuleQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String?,
        accessToken: String
    ) async throws -> ModuleQuizPayload {
        var path = "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))"
        if let attemptId, !attemptId.isEmpty {
            let encoded = attemptId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? attemptId
            path += "?attemptId=\(encoded)"
        }
        let (data, _) = try await client.request(path: path, authorized: true, accessToken: accessToken)
        return try decode(ModuleQuizPayload.self, from: data)
    }

    static func startQuiz(
        courseCode: String,
        itemId: String,
        accessCode: String?,
        accessToken: String
    ) async throws -> QuizStartResponse {
        let trimmed = accessCode?.trimmingCharacters(in: .whitespacesAndNewlines)
        let body = QuizStartRequest(quizAccessCode: (trimmed?.isEmpty == false) ? trimmed : nil)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/start",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(QuizStartResponse.self, from: data)
    }

    static func fetchQuizCurrentQuestion(
        courseCode: String,
        itemId: String,
        attemptId: String,
        accessToken: String
    ) async throws -> QuizCurrentQuestionResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/attempts/\(encodePath(attemptId))/current-question",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(QuizCurrentQuestionResponse.self, from: data)
    }

    static func advanceQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String,
        responseItem: QuizQuestionResponseItem,
        accessToken: String
    ) async throws -> QuizAdvanceResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/attempts/\(encodePath(attemptId))/advance",
            method: "POST",
            body: responseItem,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(QuizAdvanceResponse.self, from: data)
    }

    static func submitQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String,
        responses: [QuizQuestionResponseItem]?,
        accessToken: String
    ) async throws -> QuizSubmitResponse {
        let body = QuizSubmitRequest(attemptId: attemptId, responses: responses)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/submit",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(QuizSubmitResponse.self, from: data)
    }

    static func fetchQuizResults(
        courseCode: String,
        itemId: String,
        attemptId: String,
        accessToken: String
    ) async throws -> QuizResultsResponse {
        let encoded = attemptId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? attemptId
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/results?attemptId=\(encoded)",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(QuizResultsResponse.self, from: data)
    }

    static func postQuizFocusLoss(
        courseCode: String,
        itemId: String,
        attemptId: String,
        eventType: String,
        accessToken: String
    ) async {
        let body = QuizFocusLossRequest(eventType: eventType, durationMs: nil)
        _ = try? await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/attempts/\(encodePath(attemptId))/focus-loss",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
    }

    static func postQuizQuestionRun(
        courseCode: String,
        itemId: String,
        attemptId: String,
        questionId: String,
        code: String,
        languageId: Int?,
        accessToken: String
    ) async throws -> QuizCodeRunResponse {
        let body = QuizCodeRunRequest(code: code, languageId: languageId)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/attempts/\(encodePath(attemptId))/questions/\(encodePath(questionId))/run",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(QuizCodeRunResponse.self, from: data)
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

    // MARK: - Planner (todos + calendar, M2.1)

    static func fetchCalendarTokenInfo(accessToken: String) async throws -> CalendarTokenInfo {
        let (data, _) = try await client.request(
            path: "/api/v1/me/calendar-token",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CalendarTokenInfo.self, from: data)
    }

    static func createCalendarToken(accessToken: String) async throws -> CalendarTokenCreated {
        let (data, _) = try await client.request(
            path: "/api/v1/me/calendar-token",
            method: "POST",
            body: EmptyBody(),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CalendarTokenCreated.self, from: data)
    }

    static func fetchAcademicCalendarEvents(
        orgId: String,
        termId: String?,
        accessToken: String
    ) async throws -> [AcademicCalendarEvent] {
        var path = "/api/v1/orgs/\(encodePath(orgId))/calendar/events"
        if let termId, !termId.isEmpty {
            let encoded = termId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? termId
            path += "?term_id=\(encoded)"
        }
        let (data, response) = try await client.request(path: path, authorized: true, accessToken: accessToken)
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AcademicCalendarEventsResponse.self, from: data).events
    }

    // MARK: - Module progress & completion (M3.1)

    /// Returns nil when conditional release is disabled (HTTP 404).
    static func fetchModulesProgress(courseCode: String, accessToken: String) async throws -> ModulesProgressSnapshot? {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/modules/progress",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ModulesProgressSnapshot.self, from: data)
    }

    static func markItemComplete(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> MarkItemCompleteResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/items/\(encodePath(itemId))/complete",
            method: "POST",
            body: EmptyBody(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return (try? decode(MarkItemCompleteResponse.self, from: data)) ?? MarkItemCompleteResponse()
    }

    private struct EmptyBody: Encodable {}

    // MARK: - Course files (M3.2)

    static func fetchCourseFilesRoot(courseCode: String, accessToken: String) async throws -> CourseFileFolderContents {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/files",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseFileFolderContents.self, from: data)
    }

    static func fetchCourseFilesFolder(
        courseCode: String,
        folderId: String,
        accessToken: String
    ) async throws -> CourseFileFolderContents {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/files/folders/\(encodePath(folderId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseFileFolderContents.self, from: data)
    }

    // MARK: - Interactive content (M3.3)

    static func fetchModuleH5P(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> ModuleH5PPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/h5p-items/\(encodePath(itemId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ModuleH5PPayload.self, from: data)
    }

    static func fetchModuleScorm(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> ModuleScormPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/scorm-items/\(encodePath(itemId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ModuleScormPayload.self, from: data)
    }

    static func launchScorm(
        courseCode: String,
        scoId: String,
        accessToken: String
    ) async throws -> ScormLaunchResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/scorm/\(encodePath(scoId))/launch",
            method: "POST",
            body: EmptyBody(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ScormLaunchResponse.self, from: data)
    }

    static func fetchModuleLtiLink(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> ModuleLtiLinkPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/lti-links/\(encodePath(itemId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ModuleLtiLinkPayload.self, from: data)
    }

    static func postLtiEmbedTicket(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> LtiEmbedTicketResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/lti-links/\(encodePath(itemId))/embed-ticket",
            method: "POST",
            body: EmptyBody(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LtiEmbedTicketResponse.self, from: data)
    }

    static func fetchModuleVibeActivity(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> ModuleVibeActivityPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/vibe-activities/\(encodePath(itemId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ModuleVibeActivityPayload.self, from: data)
    }

    static func postXAPIStatement(
        courseCode: String,
        packageId: String,
        statement: [String: Any],
        accessToken: String
    ) async throws {
        let payload: [String: Any] = [
            "courseCode": courseCode,
            "packageId": packageId,
            "statement": statement,
        ]
        let bodyData = try JSONSerialization.data(withJSONObject: payload)
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/xapi/statements",
            method: "POST",
            bodyData: bodyData,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) || response.statusCode == 204 else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    // MARK: - Office hours (M7.3)

    static func fetchOfficeHoursAvailability(
        courseCode: String,
        accessToken: String
    ) async throws -> OfficeHoursAvailability {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/availability",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let raw = try decode(OfficeHoursAvailabilityResponse.self, from: data)
        return OfficeHoursAvailability(
            windows: raw.windows ?? [],
            slots: raw.slots ?? []
        )
    }

    static func bookOfficeHoursSlot(
        slotId: String,
        note: String?,
        accessToken: String
    ) async throws -> AppointmentSlot {
        let trimmed = note?.trimmingCharacters(in: .whitespacesAndNewlines)
        let body = BookOfficeHoursSlotBody(note: (trimmed?.isEmpty == false) ? trimmed : nil)
        let (data, response) = try await client.request(
            path: "/api/v1/slots/\(encodePath(slotId))/book",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 409 {
            throw APIError.httpStatus(409, message: L.text("mobile.officeHours.conflict"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AppointmentSlot.self, from: data)
    }

    static func cancelOfficeHoursBooking(slotId: String, accessToken: String) async throws -> AppointmentSlot {
        let (data, response) = try await client.request(
            path: "/api/v1/slots/\(encodePath(slotId))/book",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AppointmentSlot.self, from: data)
    }

    static func fetchMyAppointments(accessToken: String) async throws -> [AppointmentSlot] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/appointments",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MyAppointmentsResponse.self, from: data).appointments ?? []
    }

    static func fetchMeetingJoinURL(meetingId: String, accessToken: String) async throws -> String? {
        let info = try await fetchMeetingJoinInfo(meetingId: meetingId, accessToken: accessToken)
        return info?.joinUrl
    }

    // MARK: - Live meetings (M7.5)

    static func fetchCourseMeetings(courseCode: String, accessToken: String) async throws -> [VirtualMeeting] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/meetings",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseMeetingsResponse.self, from: data).meetings ?? []
    }

    static func fetchMeetingJoinInfo(meetingId: String, accessToken: String) async throws -> MeetingJoinInfo? {
        let (data, response) = try await client.request(
            path: "/api/v1/meetings/\(encodePath(meetingId))/join",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else { return nil }
        let raw = try decode(MeetingJoinResponse.self, from: data)
        guard let join = raw.joinUrl?.trimmingCharacters(in: .whitespacesAndNewlines), !join.isEmpty else {
            return nil
        }
        let host = raw.hostUrl?.trimmingCharacters(in: .whitespacesAndNewlines)
        return MeetingJoinInfo(
            joinUrl: join,
            hostUrl: (host?.isEmpty == false) ? host : nil,
            meetingId: raw.meetingId ?? meetingId,
            status: raw.status ?? "scheduled"
        )
    }

    static func patchMeeting(meetingId: String, status: String, accessToken: String) async throws -> VirtualMeeting {
        let (data, response) = try await client.request(
            path: "/api/v1/meetings/\(encodePath(meetingId))",
            method: "PATCH",
            body: PatchMeetingBody(status: status),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(VirtualMeeting.self, from: data)
    }

    static func fetchMeetingAttendance(meetingId: String, accessToken: String) async throws -> [MeetingAttendanceRecord] {
        let (data, response) = try await client.request(
            path: "/api/v1/meetings/\(encodePath(meetingId))/attendance",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MeetingAttendanceResponse.self, from: data).attendance ?? []
    }

    static func fetchCourseWhiteboards(courseCode: String, accessToken: String) async throws -> [CourseWhiteboard] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/whiteboards",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseWhiteboardsResponse.self, from: data).whiteboards ?? []
    }

    static func fetchCourseWhiteboard(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws -> CourseWhiteboard {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/whiteboards/\(encodePath(boardId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseWhiteboard.self, from: data)
    }

}
