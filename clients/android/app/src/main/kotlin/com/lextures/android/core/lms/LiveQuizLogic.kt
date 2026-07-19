package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.Serializable
import java.util.concurrent.ConcurrentHashMap

/** Pure helpers for interactive live quizzes (MOB.5 Phase 1 — student play). */
object LiveQuizLogic {
    const val NICKNAME_MAX_LENGTH = 24
    private val nicknameAllowed = Regex("""^[\p{L}\p{N} _.\-'!]+$""")

    enum class JoinStep { Code, Nickname, Play }

    enum class NicknameReason { Empty, TooLong, Charset }

    sealed class NicknameValidation {
        data class Ok(val nickname: String) : NicknameValidation()
        data class Invalid(val reason: NicknameReason) : NicknameValidation()
    }

    enum class JoinErrorReason {
        NotFound,
        RateLimited,
        NicknameTaken,
        NicknameInvalid,
        NicknameDenied,
        LobbyLocked,
        Banned,
        OneSession,
        GameEnded,
        AuthRequired,
        JoinFailed,
        Unknown,
    }

    fun normalizeJoinCode(raw: String): String =
        raw.trim().uppercase()

    fun isValidJoinCode(raw: String): Boolean {
        val code = normalizeJoinCode(raw)
        if (code.isEmpty() || code.length > 12) return false
        return code.all { it.isLetterOrDigit() }
    }

    fun normalizeNickname(raw: String): String = raw.trim()

    fun validateNickname(raw: String): NicknameValidation {
        val nickname = normalizeNickname(raw)
        if (nickname.isEmpty()) return NicknameValidation.Invalid(NicknameReason.Empty)
        if (nickname.length > NICKNAME_MAX_LENGTH) {
            return NicknameValidation.Invalid(NicknameReason.TooLong)
        }
        if (!nicknameAllowed.matches(nickname)) {
            return NicknameValidation.Invalid(NicknameReason.Charset)
        }
        return NicknameValidation.Ok(nickname)
    }

    /** Dual gate: per-course interactive quizzes + mobile rollout kill-switch. */
    fun shouldShowWorkspaceSection(
        course: CourseSummary,
        features: MobilePlatformFeatures,
    ): Boolean = course.isInteractiveQuizzesEnabled && features.ffMobileLiveQuiz

    fun liveQuizEntryEnabled(
        courseEnabled: Boolean,
        features: MobilePlatformFeatures,
    ): Boolean = courseEnabled && features.ffMobileLiveQuiz

    fun joinErrorReason(status: Int, body: String): JoinErrorReason {
        val lower = body.lowercase()
        return when (status) {
            404 -> JoinErrorReason.NotFound
            429 -> JoinErrorReason.RateLimited
            409 -> if (lower.contains("already connected")) {
                JoinErrorReason.OneSession
            } else {
                JoinErrorReason.NicknameTaken
            }
            401, 403 -> when {
                lower.contains("lobby") -> JoinErrorReason.LobbyLocked
                lower.contains("rejoin") || lower.contains("cannot") -> JoinErrorReason.Banned
                else -> JoinErrorReason.AuthRequired
            }
            400 -> when {
                lower.contains("ended") -> JoinErrorReason.GameEnded
                lower.contains("isn't allowed") || lower.contains("isn’t allowed") ||
                    lower.contains("not allowed") -> JoinErrorReason.NicknameDenied
                lower.contains("nickname") -> JoinErrorReason.NicknameInvalid
                else -> JoinErrorReason.JoinFailed
            }
            else -> JoinErrorReason.Unknown
        }
    }

    fun joinErrorLocalizationKey(reason: JoinErrorReason): String = when (reason) {
        JoinErrorReason.NotFound -> "mobile.liveQuiz.error.notFound"
        JoinErrorReason.RateLimited -> "mobile.liveQuiz.error.rateLimited"
        JoinErrorReason.NicknameTaken -> "mobile.liveQuiz.error.nicknameTaken"
        JoinErrorReason.NicknameInvalid -> "mobile.liveQuiz.error.nicknameInvalid"
        JoinErrorReason.NicknameDenied -> "mobile.liveQuiz.error.nicknameDenied"
        JoinErrorReason.LobbyLocked -> "mobile.liveQuiz.error.lobbyLocked"
        JoinErrorReason.Banned -> "mobile.liveQuiz.error.banned"
        JoinErrorReason.OneSession -> "mobile.liveQuiz.error.oneSession"
        JoinErrorReason.GameEnded -> "mobile.liveQuiz.error.gameEnded"
        JoinErrorReason.AuthRequired -> "mobile.liveQuiz.error.authRequired"
        JoinErrorReason.JoinFailed -> "mobile.liveQuiz.error.joinFailed"
        JoinErrorReason.Unknown -> "mobile.liveQuiz.error.generic"
    }

    fun nicknameReasonLocalizationKey(reason: NicknameReason): String = when (reason) {
        NicknameReason.Empty -> "mobile.liveQuiz.nickname.error.empty"
        NicknameReason.TooLong -> "mobile.liveQuiz.nickname.error.tooLong"
        NicknameReason.Charset -> "mobile.liveQuiz.nickname.error.charset"
    }

    fun webSocketPath(courseCode: String, gameId: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/live-quizzes/games/${encodePath(gameId)}/ws"

    private fun encodePath(value: String): String =
        java.net.URLEncoder.encode(value, "UTF-8").replace("+", "%20")
}

@Serializable
data class LiveQuizPlayerSession(
    val gameId: String,
    val courseCode: String,
    val playerId: String,
    val playerToken: String,
    val nickname: String,
    val joinCode: String,
)

/** In-memory session store for unit tests; UI layer should persist via SharedPreferences. */
object LiveQuizPlayerSessionStore {
    private val sessions = ConcurrentHashMap<String, LiveQuizPlayerSession>()

    fun save(session: LiveQuizPlayerSession) {
        sessions[session.gameId] = session
    }

    fun load(gameId: String): LiveQuizPlayerSession? = sessions[gameId]

    fun clear(gameId: String) {
        sessions.remove(gameId)
    }

    fun resetForTests() {
        sessions.clear()
    }
}

object LiveQuizObservability {
    private val counters = ConcurrentHashMap<String, Int>()

    fun record(event: String, attributes: Map<String, String> = emptyMap()) {
        val key = if (attributes.isEmpty()) {
            event
        } else {
            event + "|" + attributes.toSortedMap().entries.joinToString(",") { "${it.key}=${it.value}" }
        }
        counters.merge(key, 1) { a, b -> a + b }
    }

    fun count(event: String): Int =
        counters.entries.filter { it.key == event || it.key.startsWith("$event|") }
            .sumOf { it.value }

    fun resetForTests() {
        counters.clear()
    }
}
