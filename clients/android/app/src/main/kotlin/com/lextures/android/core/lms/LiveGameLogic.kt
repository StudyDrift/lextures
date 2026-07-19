package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import org.json.JSONObject

/** Live-game state machine helpers (MOB.5). Shape mirrors web `live-quiz-realtime.ts`. */
object LiveGameLogic {
    enum class Role(val wire: String) {
        Host("host"),
        Projector("projector"),
        Player("player"),
    }

    enum class Phase(val wire: String) {
        Lobby("lobby"),
        QuestionIntro("question_intro"),
        QuestionOpen("question_open"),
        QuestionLocked("question_locked"),
        QuestionReveal("question_reveal"),
        Leaderboard("leaderboard"),
        Podium("podium"),
        Ended("ended"),
        WaitingForHost("waiting_for_host"),
        ;

        companion object {
            fun parse(raw: String?): Phase? = entries.firstOrNull { it.wire == raw }
        }
    }

    enum class ConnStatus {
        Connecting,
        Connected,
        Reconnecting,
        Ended,
        Kicked,
        Disconnected,
    }

    enum class QuestionType(val wire: String) {
        McSingle("mc_single"),
        McMultiple("mc_multiple"),
        TrueFalse("true_false"),
        TypeAnswer("type_answer"),
        Numeric("numeric"),
        Poll("poll"),
        Ordering("ordering"),
        WordCloud("word_cloud"),
        ;

        companion object {
            fun parse(raw: String?): QuestionType? = entries.firstOrNull { it.wire == raw }
        }
    }

    enum class PointsStyle(val wire: String) {
        Standard("standard"),
        Double("double"),
        NoPoints("no_points"),
        ;

        companion object {
            fun parse(raw: String?): PointsStyle =
                entries.firstOrNull { it.wire == raw } ?: Standard
        }
    }

    enum class PlaySurface {
        Lobby,
        WaitingForHost,
        Question,
        Leaderboard,
        Podium,
        Ended,
        Kicked,
        Connecting,
    }

    sealed class AnswerPayload {
        data class OptionId(val id: String) : AnswerPayload()
        data class OptionIds(val ids: List<String>) : AnswerPayload()
        data class Text(val text: String) : AnswerPayload()
        data class Value(val value: Double) : AnswerPayload()
        data class Order(val order: List<String>) : AnswerPayload()

        fun toJsonObject(): JSONObject = when (this) {
            is OptionId -> JSONObject().put("optionId", id)
            is OptionIds -> JSONObject().put("optionIds", org.json.JSONArray(ids))
            is Text -> JSONObject().put("text", text)
            is Value -> JSONObject().put("value", value)
            is Order -> JSONObject().put("order", org.json.JSONArray(order))
        }
    }

    fun playSurface(phase: Phase?, conn: ConnStatus): PlaySurface {
        if (conn == ConnStatus.Kicked) return PlaySurface.Kicked
        if (conn == ConnStatus.Connecting || conn == ConnStatus.Reconnecting) {
            return PlaySurface.Connecting
        }
        return when (phase) {
            null -> PlaySurface.Connecting
            Phase.Lobby, Phase.QuestionIntro -> PlaySurface.Lobby
            Phase.WaitingForHost -> PlaySurface.WaitingForHost
            Phase.QuestionOpen, Phase.QuestionLocked, Phase.QuestionReveal -> PlaySurface.Question
            Phase.Leaderboard -> PlaySurface.Leaderboard
            Phase.Podium -> PlaySurface.Podium
            Phase.Ended -> PlaySurface.Ended
        }
    }

    fun canSubmitAnswer(phase: Phase?, hasAnswered: Boolean, conn: ConnStatus): Boolean {
        if (phase != Phase.QuestionOpen) return false
        if (hasAnswered) return false
        return conn == ConnStatus.Connected
    }

    fun buildAnswer(
        questionType: QuestionType,
        selectedOptionId: String?,
        selectedOptionIds: List<String>,
        text: String?,
        numeric: Double?,
        order: List<String>?,
    ): AnswerPayload? = when (questionType) {
        QuestionType.McSingle, QuestionType.TrueFalse ->
            selectedOptionId?.takeIf { it.isNotEmpty() }?.let { AnswerPayload.OptionId(it) }
        QuestionType.McMultiple, QuestionType.Poll ->
            selectedOptionIds.takeIf { it.isNotEmpty() }?.let { AnswerPayload.OptionIds(it) }
        QuestionType.TypeAnswer, QuestionType.WordCloud ->
            text?.trim()?.takeIf { it.isNotEmpty() }?.let { AnswerPayload.Text(it) }
        QuestionType.Numeric ->
            numeric?.let { AnswerPayload.Value(it) }
        QuestionType.Ordering ->
            order?.takeIf { it.isNotEmpty() }?.let { AnswerPayload.Order(it) }
    }

    fun answerMessage(
        questionIndex: Int,
        answer: AnswerPayload,
        clientSentAt: String,
        powerUp: String? = null,
    ): JSONObject {
        val msg = JSONObject()
            .put("type", "answer")
            .put("questionIndex", questionIndex)
            .put("answer", answer.toJsonObject())
            .put("clientSentAt", clientSentAt)
        if (powerUp != null) msg.put("powerUp", powerUp)
        return msg
    }

    fun authHandshake(authToken: String?, role: Role, playerToken: String?): JSONObject {
        val msg = JSONObject()
            .put("authToken", authToken.orEmpty())
            .put("role", role.wire)
        if (!playerToken.isNullOrEmpty()) msg.put("playerToken", playerToken)
        return msg
    }

    fun helloMessage(resumeSeq: Int = 0): JSONObject =
        JSONObject().put("type", "hello").put("resumeSeq", resumeSeq)

    fun catchupMessage(afterSeq: Int): JSONObject =
        JSONObject().put("type", "catchup").put("afterSeq", afterSeq)

    fun reconnectDelayMs(retry: Int): Int =
        minOf(8000, 500 * (1 shl minOf(retry, 4)))

    fun shouldClearAnsweredIndex(
        previousQuestionIndex: Int?,
        nextQuestionIndex: Int,
        nextPhase: Phase,
    ): Boolean {
        if (previousQuestionIndex == null) return false
        return previousQuestionIndex != nextQuestionIndex && nextPhase == Phase.QuestionOpen
    }

    fun answerShapeLabel(index: Int): String {
        val shapes = listOf("▲", "◆", "●", "■", "★", "✚", "⬡", "▼")
        if (index < 0) return shapes[0]
        return shapes[index % shapes.size]
    }

    fun answerShapeName(index: Int): String {
        val names = listOf(
            "triangle", "diamond", "circle", "square",
            "star", "cross", "hexagon", "invertedTriangle",
        )
        if (index < 0) return names[0]
        return names[index % names.size]
    }

    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    fun parseInbound(raw: String): LiveGameInboundMessage {
        return try {
            val type = JSONObject(raw).optString("type")
            when (type) {
                "kicked" -> LiveGameInboundMessage.Kicked
                "answer_ack" -> LiveGameInboundMessage.AnswerAck(
                    json.decodeFromString(LiveGameAnswerAck.serializer(), raw),
                )
                "state" -> LiveGameInboundMessage.State(
                    json.decodeFromString(LiveGameStateFrame.serializer(), raw),
                )
                else -> LiveGameInboundMessage.Unknown
            }
        } catch (_: Exception) {
            LiveGameInboundMessage.Unknown
        }
    }
}

@Serializable
data class LiveQuizJoinLookup(
    val gameId: String,
    val courseCode: String = "",
    val kitTitle: String = "",
    val requiresAuth: Boolean = true,
    val allowsGuests: Boolean = false,
    val phase: String = "",
    val status: String = "",
)

@Serializable
data class LiveQuizJoinPlayerResult(
    val playerId: String,
    val nickname: String,
    val playerToken: String,
    val totalScore: Int = 0,
    val streak: Int? = null,
    val rejoined: Boolean? = null,
)

@Serializable
data class LiveQuizMyResults(
    val sessionId: String = "",
    val nickname: String = "",
    val totalScore: Int = 0,
    val rank: Int = 0,
    val playerCount: Int = 0,
    val answered: Int = 0,
    val correct: Int = 0,
)

@Serializable
data class LiveGameQuestionOption(
    val id: String,
    val text: String = "",
)

@Serializable
data class LiveGameQuestion(
    val index: Int = 0,
    val questionType: String = "",
    val prompt: String = "",
    val options: List<LiveGameQuestionOption> = emptyList(),
    val timeLimitSeconds: Int = 0,
    val pointsStyle: String = "standard",
    val correctOptionIds: List<String>? = null,
    val explanation: String? = null,
)

@Serializable
data class LiveGameLeaderboardEntry(
    val rank: Int = 0,
    val playerId: String = "",
    val nickname: String = "",
    val totalScore: Int = 0,
)

@Serializable
data class LiveGameYou(
    val rank: Int = 0,
    val totalScore: Int = 0,
    val streak: Int = 0,
)

@Serializable
data class LiveGamePlayer(
    val id: String,
    val nickname: String = "",
    val totalScore: Int = 0,
    val streak: Int = 0,
    val connected: Boolean = false,
    val renamedByHost: Boolean? = null,
    val isGuest: Boolean? = null,
)

@Serializable
data class LiveGameStateFrame(
    val type: String = "state",
    val seq: Int = 0,
    val gameId: String = "",
    val phase: String = "",
    val status: String = "",
    val questionIndex: Int = 0,
    val joinCode: String = "",
    val kitTitle: String = "",
    val pacing: String = "",
    val players: List<LiveGamePlayer> = emptyList(),
    val questionCount: Int = 0,
    val namesMuted: Boolean? = null,
    val lobbyLocked: Boolean? = null,
    val allowGuests: Boolean? = null,
    val deadline: String? = null,
    val answerCount: Int? = null,
    val leaderboard: List<LiveGameLeaderboardEntry>? = null,
    val podium: List<LiveGameLeaderboardEntry>? = null,
    val you: LiveGameYou? = null,
    val scoringProfile: String? = null,
    val powerUpsEnabled: Boolean? = null,
    val question: LiveGameQuestion? = null,
)

@Serializable
data class LiveGameAnswerAck(
    val type: String = "answer_ack",
    val ok: Boolean = false,
    val questionIndex: Int? = null,
    val isCorrect: Boolean? = null,
    val points: Int? = null,
    val streak: Int? = null,
    val totalScore: Int? = null,
    val rank: Int? = null,
    val duplicate: Boolean? = null,
    val late: Boolean? = null,
    val alreadyAnswered: Boolean? = null,
    val error: String? = null,
)

sealed class LiveGameInboundMessage {
    data class State(val frame: LiveGameStateFrame) : LiveGameInboundMessage()
    data class AnswerAck(val ack: LiveGameAnswerAck) : LiveGameInboundMessage()
    data object Kicked : LiveGameInboundMessage()
    data object Unknown : LiveGameInboundMessage()
}
