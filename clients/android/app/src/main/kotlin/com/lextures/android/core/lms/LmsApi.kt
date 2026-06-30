package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import java.net.URLEncoder

/** LMS endpoints used by the post-auth tabs (parity with web `courses-api` / `communication-api`). */
object LmsApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodeQuery(value: String): String = URLEncoder.encode(value, "UTF-8")

    private fun encodePath(value: String): String = encodeQuery(value).replace("+", "%20")

    // Courses

    suspend fun fetchCourses(accessToken: String): List<CourseSummary> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses", accessToken = accessToken)
        decode<CoursesResponse>(body).courses
    }

    /** Single-course GET includes `viewerEnrollmentRoles` (list GET does not). */
    suspend fun fetchCourse(courseCode: String, accessToken: String): CourseSummary = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses/${encodePath(courseCode)}", accessToken = accessToken)
        decode<CourseSummary>(body)
    }

    suspend fun fetchCourseStructure(courseCode: String, accessToken: String): List<CourseStructureItem> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/structure",
                accessToken = accessToken,
            )
            decode<CourseStructureResponse>(body).items
        }

    /** Per-kind detail GET for a structure item; null when the kind has no detail endpoint. */
    suspend fun fetchItemDetail(
        courseCode: String,
        item: CourseStructureItem,
        accessToken: String,
    ): ModuleItemDetail? = withContext(Dispatchers.IO) {
        val resource = when (item.kind) {
            "content_page" -> "content-pages"
            "assignment" -> "assignments"
            "quiz" -> "quizzes"
            "external_link" -> "external-links"
            else -> return@withContext null
        }
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/$resource/${encodePath(item.id)}",
            accessToken = accessToken,
        )
        decode<ModuleItemDetail>(body)
    }

    // Inbox (communication)

    suspend fun fetchMailboxMessages(
        folder: MailboxFolder,
        query: String,
        accessToken: String,
    ): List<MailboxMessage> = withContext(Dispatchers.IO) {
        val params = "folder=${folder.wire}&q=${encodeQuery(query.trim())}"
        val (body, _) = client.request("/api/v1/communication/messages?$params", accessToken = accessToken)
        decode<MailboxMessagesResponse>(body).messages
    }

    suspend fun fetchUnreadInboxCount(accessToken: String): Int = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/communication/unread-count", accessToken = accessToken)
        decode<UnreadInboxResponse>(body).unreadInbox ?: 0
    }

    suspend fun patchMailbox(messageId: String, patch: MailboxPatchRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages/${encodePath(messageId)}",
                method = "PATCH",
                body = client.encodeBody(patch, MailboxPatchRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }

    suspend fun sendMessage(request: SendMessageRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages",
                method = "POST",
                body = client.encodeBody(request, SendMessageRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }

    // Profile

    suspend fun fetchMe(accessToken: String): MeProfile = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me", accessToken = accessToken)
        decode<MeProfile>(body)
    }

    // Account settings (editable profile)

    suspend fun fetchAccountProfile(accessToken: String): AccountProfile = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/settings/account", accessToken = accessToken)
        decode<AccountProfile>(body)
    }

    suspend fun updateAccountProfile(patch: AccountProfilePatch, accessToken: String): AccountProfile =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                path = "/api/v1/settings/account",
                method = "PATCH",
                body = client.encodeBody(patch, AccountProfilePatch.serializer()),
                accessToken = accessToken,
            )
            decode<AccountProfile>(body)
        }

    // My accommodations

    suspend fun fetchMyAccommodations(accessToken: String): List<MyAccommodation> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/accommodations", accessToken = accessToken)
        decode<MyAccommodationsResponse>(body).accommodations
    }

    // Notifications

    suspend fun fetchNotifications(accessToken: String): NotificationsPage = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/notifications", accessToken = accessToken)
        decode<NotificationsPage>(body)
    }

    suspend fun markNotificationRead(id: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/me/notifications/${encodePath(id)}/read",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun markAllNotificationsRead(accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/me/notifications/read-all",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun fetchNotificationPreferences(accessToken: String): List<NotificationPreference> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request("/api/v1/me/notification-preferences", accessToken = accessToken)
            decode<NotificationPreferencesResponse>(body).preferences
        }

    suspend fun updateNotificationPreferences(
        preferences: List<NotificationPreference>,
        accessToken: String,
    ): List<NotificationPreference> = withContext(Dispatchers.IO) {
        val update = NotificationPreferencesUpdate(
            preferences = preferences.map {
                NotificationPreferencePatch(
                    eventType = it.eventType,
                    emailEnabled = it.emailEnabled,
                    pushEnabled = it.pushEnabled,
                    smsEnabled = it.smsEnabled,
                    digestMode = it.digestMode,
                )
            },
        )
        val (body, _) = client.request(
            path = "/api/v1/me/notification-preferences",
            method = "PUT",
            body = json.encodeToString(NotificationPreferencesUpdate.serializer(), update),
            accessToken = accessToken,
        )
        decode<NotificationPreferencesResponse>(body).preferences
    }

    // Announcements (org broadcasts)

    suspend fun fetchMyBroadcasts(accessToken: String): List<Broadcast> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/broadcasts", accessToken = accessToken)
        decode<BroadcastsResponse>(body).broadcasts
    }

    suspend fun acknowledgeBroadcast(id: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/broadcasts/${encodePath(id)}/acknowledge",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    // My grades (student)

    suspend fun fetchMyGrades(courseCode: String, accessToken: String): MyGradesResponse =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/my-grades",
                accessToken = accessToken,
            )
            decode<MyGradesResponse>(body)
        }

    // Syllabus

    suspend fun fetchSyllabus(courseCode: String, accessToken: String): SyllabusPayload =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/syllabus",
                accessToken = accessToken,
            )
            decode<SyllabusPayload>(body)
        }

    // Assignment submissions

    suspend fun fetchMySubmission(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): AssignmentSubmission? = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/mine",
            accessToken = accessToken,
        )
        decode<MySubmissionResponse>(body).submission
    }

    suspend fun fetchSubmissions(
        courseCode: String,
        itemId: String,
        graded: String?, // "graded" | "ungraded" | null for all
        accessToken: String,
    ): List<AssignmentSubmission> = withContext(Dispatchers.IO) {
        var path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions"
        if (!graded.isNullOrEmpty()) path += "?graded=$graded"
        val (body, _) = client.request(path, accessToken = accessToken)
        // Drop roster placeholders (enrolled students with no submission) — no id to grade.
        decode<SubmissionsListResponse>(body).submissions.filter { it.id.isNotBlank() }
    }

    suspend fun fetchQuizAttempts(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): List<QuizAttemptSummary> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts",
            accessToken = accessToken,
        )
        decode<QuizAttemptsListResponse>(body).attempts
    }

    suspend fun fetchGradingSubmissions(
        courseCode: String,
        backlogItem: GradingBacklogItem,
        graded: String?,
        accessToken: String,
    ): List<AssignmentSubmission> = withContext(Dispatchers.IO) {
        if (backlogItem.isQuiz) {
            val attempts = fetchQuizAttempts(courseCode, backlogItem.resolvedItemId, accessToken)
            val submissions = GradingSubmissionMapper.quizAttemptsToSubmissions(attempts)
            GradingSubmissionMapper.filterSubmissions(submissions, graded ?: "all")
        } else {
            fetchSubmissions(courseCode, backlogItem.resolvedItemId, graded, accessToken)
        }
    }

    suspend fun fetchSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String,
    ): SubmissionGrade = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}" +
                "/submissions/${encodePath(submissionId)}/grade",
            accessToken = accessToken,
        )
        decode<SubmissionGrade>(body)
    }

    suspend fun putSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        gradeBody: SubmissionGradePut,
        accessToken: String,
    ) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}" +
                    "/submissions/${encodePath(submissionId)}/grade",
                method = "PUT",
                body = client.encodeBody(gradeBody, SubmissionGradePut.serializer()),
                accessToken = accessToken,
            )
        }
    }

    // Grading backlog (staff)

    suspend fun fetchGradingBacklog(courseCode: String, accessToken: String): List<GradingBacklogItem> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/grading-backlog",
                accessToken = accessToken,
            )
            decode<GradingBacklogResponse>(body).items
        }

    suspend fun registerDeviceToken(token: String, platform: String, accessToken: String): DeviceTokenResponse =
        withContext(Dispatchers.IO) {
            val body = client.encodeBody(
                DeviceTokenRegistration(
                    token = token,
                    platform = platform,
                    appBundleId = "com.lextures.android",
                    appVersion = com.lextures.android.BuildConfig.VERSION_NAME,
                ),
                DeviceTokenRegistration.serializer(),
            )
            val (response, _) = client.request(
                path = "/api/v1/me/device-tokens",
                method = "POST",
                body = body,
                accessToken = accessToken,
            )
            decode(response)
        }

    suspend fun deregisterDeviceToken(id: String, accessToken: String) = withContext(Dispatchers.IO) {
        client.request(
            path = "/api/v1/me/device-tokens/${encodePath(id)}",
            method = "DELETE",
            accessToken = accessToken,
        )
    }

    // Attendance

    suspend fun fetchAttendanceSessions(courseCode: String, accessToken: String): List<AttendanceSession> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions",
                accessToken = accessToken,
            )
            decode<AttendanceSessionsResponse>(body).sessions
        }

    suspend fun fetchAttendanceSessionDetail(
        courseCode: String,
        sessionId: String,
        accessToken: String,
    ): AttendanceSessionDetail = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}",
            accessToken = accessToken,
        )
        decode<AttendanceSessionDetail>(body)
    }

    suspend fun selfReportAttendance(
        courseCode: String,
        sessionId: String,
        status: String,
        accessToken: String,
    ) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}/self-report",
                method = "POST",
                body = client.encodeBody(SelfReportBody(status), SelfReportBody.serializer()),
                accessToken = accessToken,
            )
        }
    }

    // Onboarding (plan 15.11 / M1.3)

    /** Returns null when the onboarding feature flag is off (HTTP 404). */
    suspend fun fetchOnboardingStatus(accessToken: String): OnboardingStatus? = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/me/onboarding-status", accessToken = accessToken)
        when (code) {
            404 -> null
            in 200..299 -> decode<OnboardingStatus>(body)
            else -> throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }
    }

    suspend fun postOnboarding(payload: Map<String, Any?>, accessToken: String): LearnerGoals =
        withContext(Dispatchers.IO) {
            val body = buildJsonObject {
                payload.forEach { (key, value) ->
                    when (value) {
                        null -> Unit
                        is Boolean -> put(key, JsonPrimitive(value))
                        is Int -> put(key, JsonPrimitive(value))
                        is Double -> put(key, JsonPrimitive(value))
                        is String -> put(key, JsonPrimitive(value))
                        is Map<*, *> -> {
                            @Suppress("UNCHECKED_CAST")
                            val answers = value as Map<String, Int>
                            put(
                                key,
                                buildJsonObject {
                                    answers.forEach { (answerKey, answerValue) ->
                                        put(answerKey, JsonPrimitive(answerValue))
                                    }
                                },
                            )
                        }
                        else -> Unit
                    }
                }
            }.toString()
            val (response, code) = client.requestRaw(
                path = "/api/v1/me/onboarding",
                method = "POST",
                body = body,
                accessToken = accessToken,
            )
            if (code !in 200..299) {
                throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
            }
            decode<GoalsEnvelope>(response).goals
        }

    suspend fun fetchDiagnosticQuestions(topic: String, accessToken: String): List<DiagnosticQuestion> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/me/onboarding/diagnostic-questions?topic=${encodeQuery(topic)}",
                accessToken = accessToken,
            )
            decode<DiagnosticQuestionsResponse>(body).questions
        }

    suspend fun saveStudyReminderPrefs(optIn: Boolean, reminderTime: String, accessToken: String) {
        if (!optIn) return
        withContext(Dispatchers.IO) {
            runCatching {
                val body = """
                    {"preferences":[{"eventType":"study_reminder","emailEnabled":true,"pushEnabled":true,"digestMode":"instant"}]}
                """.trimIndent()
                client.request(
                    path = "/api/v1/me/notification-preferences",
                    method = "PUT",
                    body = body,
                    accessToken = accessToken,
                )
            }
        }
        reminderTime
    }

    // Planner (todos + calendar, M2.1)

    suspend fun fetchCalendarTokenInfo(accessToken: String): CalendarTokenInfo = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/calendar-token", accessToken = accessToken)
        decode(body)
    }

    suspend fun createCalendarToken(accessToken: String): CalendarTokenCreated = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/me/calendar-token",
            method = "POST",
            body = "{}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchAcademicCalendarEvents(
        orgId: String,
        termId: String?,
        accessToken: String,
    ): List<AcademicCalendarEvent> = withContext(Dispatchers.IO) {
        var path = "/api/v1/orgs/${encodePath(orgId)}/calendar/events"
        if (!termId.isNullOrEmpty()) {
            path += "?term_id=${encodeQuery(termId)}"
        }
        val (body, code) = client.requestRaw(path, accessToken = accessToken)
        when (code) {
            404 -> emptyList()
            in 200..299 -> decode<AcademicCalendarEventsResponse>(body).events
            else -> throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }
    }
}
