package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
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
        decode<SubmissionsListResponse>(body).submissions
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
}
