package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

class LiveQuizLogicTest {
    @Before
    fun setUp() {
        LiveQuizPlayerSessionStore.resetForTests()
        LiveQuizObservability.resetForTests()
    }

    @Test
    fun normalizeAndValidateJoinCode() {
        assertEquals("AB12", LiveQuizLogic.normalizeJoinCode("  ab12  "))
        assertTrue(LiveQuizLogic.isValidJoinCode("AB12"))
        assertFalse(LiveQuizLogic.isValidJoinCode(""))
        assertFalse(LiveQuizLogic.isValidJoinCode("bad code"))
    }

    @Test
    fun validateNickname() {
        val ok = LiveQuizLogic.validateNickname("  Ada  ")
        assertTrue(ok is LiveQuizLogic.NicknameValidation.Ok)
        assertEquals("Ada", (ok as LiveQuizLogic.NicknameValidation.Ok).nickname)

        val empty = LiveQuizLogic.validateNickname("")
        assertTrue(empty is LiveQuizLogic.NicknameValidation.Invalid)
        assertEquals(
            LiveQuizLogic.NicknameReason.Empty,
            (empty as LiveQuizLogic.NicknameValidation.Invalid).reason,
        )

        val tooLong = LiveQuizLogic.validateNickname("x".repeat(25))
        assertEquals(
            LiveQuizLogic.NicknameReason.TooLong,
            (tooLong as LiveQuizLogic.NicknameValidation.Invalid).reason,
        )

        val charset = LiveQuizLogic.validateNickname("bad@name")
        assertEquals(
            LiveQuizLogic.NicknameReason.Charset,
            (charset as LiveQuizLogic.NicknameValidation.Invalid).reason,
        )
    }

    @Test
    fun joinErrorMapping() {
        assertEquals(LiveQuizLogic.JoinErrorReason.NotFound, LiveQuizLogic.joinErrorReason(404, ""))
        assertEquals(LiveQuizLogic.JoinErrorReason.RateLimited, LiveQuizLogic.joinErrorReason(429, ""))
        assertEquals(
            LiveQuizLogic.JoinErrorReason.NicknameTaken,
            LiveQuizLogic.joinErrorReason(409, "nickname taken"),
        )
        assertEquals(
            LiveQuizLogic.JoinErrorReason.OneSession,
            LiveQuizLogic.joinErrorReason(409, "already connected"),
        )
        assertEquals(
            LiveQuizLogic.JoinErrorReason.LobbyLocked,
            LiveQuizLogic.joinErrorReason(403, "lobby locked"),
        )
        assertEquals(
            LiveQuizLogic.JoinErrorReason.GameEnded,
            LiveQuizLogic.joinErrorReason(400, "game ended"),
        )
        assertEquals(
            LiveQuizLogic.JoinErrorReason.NicknameDenied,
            LiveQuizLogic.joinErrorReason(400, "isn't allowed"),
        )
    }

    @Test
    fun workspaceSectionGate() {
        val course = CourseSummary(
            id = "1",
            courseCode = "C1",
            title = "Course",
            description = "",
            interactiveQuizzesEnabled = true,
        )
        assertFalse(
            LiveQuizLogic.shouldShowWorkspaceSection(
                course,
                MobilePlatformFeatures(ffMobileLiveQuiz = false),
            ),
        )
        assertTrue(
            LiveQuizLogic.shouldShowWorkspaceSection(
                course,
                MobilePlatformFeatures(ffMobileLiveQuiz = true),
            ),
        )
        assertFalse(
            LiveQuizLogic.shouldShowWorkspaceSection(
                course.copy(interactiveQuizzesEnabled = false),
                MobilePlatformFeatures(ffMobileLiveQuiz = true),
            ),
        )
    }

    @Test
    fun playerSessionStoreRoundTrip() {
        val session = LiveQuizPlayerSession(
            gameId = "g1",
            courseCode = "C1",
            playerId = "p1",
            playerToken = "tok",
            nickname = "Ada",
            joinCode = "AB12",
        )
        LiveQuizPlayerSessionStore.save(session)
        assertEquals(session, LiveQuizPlayerSessionStore.load("g1"))
        LiveQuizPlayerSessionStore.clear("g1")
        assertNull(LiveQuizPlayerSessionStore.load("g1"))
    }
}

@RunWith(RobolectricTestRunner::class)
class LiveGameLogicTest {
    @Test
    fun answerPayloadShapes() {
        assertEquals("b", LiveGameLogic.AnswerPayload.OptionId("b").toJsonObject().getString("optionId"))
        assertEquals(
            listOf("a", "c"),
            (0 until LiveGameLogic.AnswerPayload.OptionIds(listOf("a", "c")).toJsonObject().getJSONArray("optionIds").length())
                .map { LiveGameLogic.AnswerPayload.OptionIds(listOf("a", "c")).toJsonObject().getJSONArray("optionIds").getString(it) },
        )
        assertEquals("paris", LiveGameLogic.AnswerPayload.Text("paris").toJsonObject().getString("text"))
        assertEquals(42.0, LiveGameLogic.AnswerPayload.Value(42.0).toJsonObject().getDouble("value"), 0.0)
    }

    @Test
    fun buildAnswerPerType() {
        assertEquals(
            LiveGameLogic.AnswerPayload.OptionId("a"),
            LiveGameLogic.buildAnswer(
                LiveGameLogic.QuestionType.McSingle,
                "a",
                emptyList(),
                null,
                null,
                null,
            ),
        )
        assertNull(
            LiveGameLogic.buildAnswer(
                LiveGameLogic.QuestionType.McMultiple,
                null,
                emptyList(),
                null,
                null,
                null,
            ),
        )
        assertEquals(
            LiveGameLogic.AnswerPayload.Text("hi"),
            LiveGameLogic.buildAnswer(
                LiveGameLogic.QuestionType.TypeAnswer,
                null,
                emptyList(),
                " hi ",
                null,
                null,
            ),
        )
    }

    @Test
    fun playSurfaceAndSubmitGate() {
        assertEquals(
            LiveGameLogic.PlaySurface.Lobby,
            LiveGameLogic.playSurface(LiveGameLogic.Phase.Lobby, LiveGameLogic.ConnStatus.Connected),
        )
        assertEquals(
            LiveGameLogic.PlaySurface.Question,
            LiveGameLogic.playSurface(LiveGameLogic.Phase.QuestionOpen, LiveGameLogic.ConnStatus.Connected),
        )
        assertEquals(
            LiveGameLogic.PlaySurface.Kicked,
            LiveGameLogic.playSurface(LiveGameLogic.Phase.QuestionOpen, LiveGameLogic.ConnStatus.Kicked),
        )
        assertTrue(
            LiveGameLogic.canSubmitAnswer(
                LiveGameLogic.Phase.QuestionOpen,
                hasAnswered = false,
                LiveGameLogic.ConnStatus.Connected,
            ),
        )
        assertFalse(
            LiveGameLogic.canSubmitAnswer(
                LiveGameLogic.Phase.QuestionLocked,
                hasAnswered = false,
                LiveGameLogic.ConnStatus.Connected,
            ),
        )
    }

    @Test
    fun authHandshakeAndReconnectDelay() {
        val guest = LiveGameLogic.authHandshake(null, LiveGameLogic.Role.Player, "pt")
        assertEquals("", guest.getString("authToken"))
        assertEquals("player", guest.getString("role"))
        assertEquals("pt", guest.getString("playerToken"))
        assertEquals(500, LiveGameLogic.reconnectDelayMs(0))
        assertEquals(8000, LiveGameLogic.reconnectDelayMs(4))
        assertEquals(8000, LiveGameLogic.reconnectDelayMs(10))
    }

    @Test
    fun parseInboundMessages() {
        val state = LiveGameLogic.parseInbound(
            """{"type":"state","seq":2,"gameId":"g1","phase":"question_open","status":"running","questionIndex":0,"joinCode":"AB12","kitTitle":"Demo","pacing":"manual","players":[],"questionCount":1}""",
        )
        assertTrue(state is LiveGameInboundMessage.State)
        assertEquals(2, (state as LiveGameInboundMessage.State).frame.seq)

        val ack = LiveGameLogic.parseInbound(
            """{"type":"answer_ack","ok":true,"questionIndex":0,"isCorrect":true,"points":100}""",
        )
        assertTrue(ack is LiveGameInboundMessage.AnswerAck)
        assertEquals(100, (ack as LiveGameInboundMessage.AnswerAck).ack.points)

        assertTrue(LiveGameLogic.parseInbound("""{"type":"kicked"}""") is LiveGameInboundMessage.Kicked)
    }

    @Test
    fun answerShapesAreStable() {
        assertEquals("triangle", LiveGameLogic.answerShapeName(0))
        assertEquals("triangle", LiveGameLogic.answerShapeName(8))
        assertTrue(LiveGameLogic.answerShapeLabel(1).isNotEmpty())
    }
}
