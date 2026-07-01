package com.lextures.android.core.realtime

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Owns the app-wide realtime sockets (mailbox/courses/enrollments + notifications) for the
 * lifetime of an authenticated session. Feature screens observe the revision counters and
 * re-run their existing REST fetches when they bump — mirrors
 * `clients/web/src/context/inbox-unread-provider.tsx`, which does the same with local state
 * instead of a shared cache.
 */
object RealtimeManager {
    private val json = Json { ignoreUnknownKeys = true }

    private val _mailboxRevision = MutableStateFlow(0)
    val mailboxRevision: StateFlow<Int> = _mailboxRevision.asStateFlow()

    private val _coursesRevision = MutableStateFlow(0)
    val coursesRevision: StateFlow<Int> = _coursesRevision.asStateFlow()

    private val _enrollmentsRevision = MutableStateFlow(0)
    val enrollmentsRevision: StateFlow<Int> = _enrollmentsRevision.asStateFlow()

    private val _lastEnrollmentCourseCode = MutableStateFlow<String?>(null)
    val lastEnrollmentCourseCode: StateFlow<String?> = _lastEnrollmentCourseCode.asStateFlow()

    private val _notificationsRevision = MutableStateFlow(0)
    val notificationsRevision: StateFlow<Int> = _notificationsRevision.asStateFlow()

    private var commSocket: WebSocketClient? = null
    private var notificationsSocket: WebSocketClient? = null

    /** Connects (or reconnects, if the token changed) the app-wide sockets. */
    fun configure(accessToken: () -> String?) {
        if (commSocket == null) {
            commSocket = WebSocketClient(
                path = "/api/v1/communication/ws",
                accessTokenProvider = accessToken,
                onMessage = ::handleCommMessage,
            )
        }
        if (notificationsSocket == null) {
            notificationsSocket = WebSocketClient(
                path = "/api/v1/ws/notifications",
                accessTokenProvider = accessToken,
                onMessage = ::handleNotificationsMessage,
            )
        }
        commSocket?.connect()
        notificationsSocket?.connect()
    }

    /** Called on sign-out to stop reconnect attempts and close both sockets. */
    fun disconnect() {
        commSocket?.disconnect()
        notificationsSocket?.disconnect()
    }

    private fun handleCommMessage(text: String) {
        val event = runCatching { json.decodeFromString<CommWsEvent>(text) }.getOrNull() ?: return
        when (event.type) {
            "mailbox_updated" -> _mailboxRevision.value += 1
            "courses_updated" -> _coursesRevision.value += 1
            "enrollments_updated" -> {
                _lastEnrollmentCourseCode.value = event.courseCode
                _enrollmentsRevision.value += 1
            }
        }
    }

    private fun handleNotificationsMessage(text: String) {
        val event = runCatching { json.decodeFromString<CommWsEvent>(text) }.getOrNull() ?: return
        if (event.type == "notification_updated") {
            _notificationsRevision.value += 1
        }
    }

    @Serializable
    private data class CommWsEvent(val type: String, val courseCode: String? = null)
}
