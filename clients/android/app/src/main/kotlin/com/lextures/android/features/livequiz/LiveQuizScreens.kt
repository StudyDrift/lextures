package com.lextures.android.features.livequiz

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LiveGameLogic
import com.lextures.android.core.lms.LiveGameStateFrame
import com.lextures.android.core.lms.LiveQuizApi
import com.lextures.android.core.lms.LiveQuizJoinLookup
import com.lextures.android.core.lms.LiveQuizKitSummary
import com.lextures.android.core.lms.LiveQuizLogic
import com.lextures.android.core.lms.LiveQuizMyResults
import com.lextures.android.core.lms.LiveQuizObservability
import com.lextures.android.core.lms.LiveQuizPlayerSession
import com.lextures.android.core.lms.LiveQuizPlayerSessionStore
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

@Composable
fun LiveQuizHubScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    var kits by remember { mutableStateOf<List<LiveQuizKitSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showJoin by remember { mutableStateOf(false) }

    LaunchedEffect(course.courseCode, accessToken) {
        loading = true
        errorMessage = null
        val token = accessToken
        if (token.isNullOrBlank()) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_liveQuiz_error_authRequired)
            loading = false
            return@LaunchedEffect
        }
        runCatching {
            LiveQuizApi.listQuizKits(course.courseCode, token)
        }.onSuccess { result ->
            kits = result.kits.filter { it.archived != true }
        }.onFailure {
            errorMessage = L.text(context, localePrefs, R.string.mobile_liveQuiz_error_generic)
        }
        loading = false
    }

    Column(
        modifier = modifier.fillMaxWidth().padding(vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = L.text(R.string.mobile_liveQuiz_hub_title),
                color = textPrimary(),
                fontWeight = FontWeight.SemiBold,
            )
            Button(
                onClick = { showJoin = true },
                modifier = Modifier
                    .testTag("liveQuiz.join.button")
                    .semantics { contentDescription = L.text(R.string.mobile_liveQuiz_join_button) },
            ) {
                Text(L.text(R.string.mobile_liveQuiz_join_button))
            }
        }
        Text(
            text = L.text(R.string.mobile_liveQuiz_hub_subtitle),
            color = textSecondary(),
        )
        when {
            loading -> CircularProgressIndicator(modifier = Modifier.padding(top = 16.dp))
            errorMessage != null -> Text(text = errorMessage!!, color = androidx.compose.ui.graphics.Color.Red)
            kits.isEmpty() -> Text(text = L.text(R.string.mobile_liveQuiz_hub_empty), color = textSecondary())
            else -> kits.forEach { kit ->
                LmsCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier = Modifier.padding(12.dp)) {
                        Text(kit.title, color = textPrimary(), fontWeight = FontWeight.Medium)
                        kit.questionCount?.let { count ->
                            Text(
                                text = L.format(R.string.mobile_liveQuiz_hub_questionCount, count),
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }
        }
    }

    if (showJoin) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showJoin = false }) {
            LiveQuizPlayScreen(
                session = session,
                initialCode = null,
                onClose = { showJoin = false },
            )
        }
    }
}

@Composable
fun LiveQuizPlayScreen(
    session: AuthSession,
    initialCode: String?,
    onClose: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    var step by remember { mutableStateOf(LiveQuizLogic.JoinStep.Code) }
    var code by remember { mutableStateOf(initialCode.orEmpty()) }
    var nickname by remember { mutableStateOf("") }
    var lookup by remember { mutableStateOf<LiveQuizJoinLookup?>(null) }
    var playerSession by remember { mutableStateOf<LiveQuizPlayerSession?>(null) }
    var busy by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var gameState by remember { mutableStateOf<LiveGameStateFrame?>(null) }
    var conn by remember { mutableStateOf(LiveGameLogic.ConnStatus.Connecting) }
    var answeredIndex by remember { mutableStateOf<Int?>(null) }
    var selectedOptionId by remember { mutableStateOf<String?>(null) }
    var selectedOptionIds by remember { mutableStateOf(setOf<String>()) }
    var answerText by remember { mutableStateOf("") }
    var answerNumeric by remember { mutableStateOf("") }
    var orderIds by remember { mutableStateOf(listOf<String>()) }
    var myResults by remember { mutableStateOf<LiveQuizMyResults?>(null) }
    var socket by remember { mutableStateOf<LiveGameSocket?>(null) }

    fun teardown() {
        socket?.disconnect()
        socket = null
    }

    fun startSocket(sess: LiveQuizPlayerSession) {
        teardown()
        val s = LiveGameSocket(
            courseCode = sess.courseCode,
            gameId = sess.gameId,
            role = LiveGameLogic.Role.Player,
            playerToken = sess.playerToken,
            accessTokenProvider = { accessToken },
            onState = { frame ->
                val prev = gameState?.questionIndex
                val nextPhase = LiveGameLogic.Phase.parse(frame.phase) ?: LiveGameLogic.Phase.Lobby
                if (LiveGameLogic.shouldClearAnsweredIndex(prev, frame.questionIndex, nextPhase)) {
                    answeredIndex = null
                    selectedOptionId = null
                    selectedOptionIds = emptySet()
                    answerText = ""
                    answerNumeric = ""
                    orderIds = frame.question?.options?.map { it.id }.orEmpty()
                }
                gameState = frame
                if (frame.phase == LiveGameLogic.Phase.Ended.wire) {
                    scope.launch {
                        val token = accessToken ?: return@launch
                        runCatching {
                            LiveQuizApi.fetchMyGameResults(sess.courseCode, sess.gameId, token)
                        }.onSuccess {
                            myResults = it
                            LiveQuizObservability.record("live_quiz_end")
                        }
                    }
                }
            },
            onAck = { ack ->
                if (ack.ok) answeredIndex = ack.questionIndex
                LiveQuizObservability.record(
                    "live_quiz_answer",
                    mapOf(
                        "ok" to if (ack.ok) "1" else "0",
                        "type" to (gameState?.question?.questionType.orEmpty()),
                    ),
                )
            },
            onKicked = {
                LiveQuizPlayerSessionStore.clear(sess.gameId)
            },
            onConn = { conn = it },
        )
        socket = s
        s.connect()
    }

    DisposableEffect(Unit) {
        onDispose { teardown() }
    }

    LaunchedEffect(initialCode) {
        if (!initialCode.isNullOrBlank()) {
            code = LiveQuizLogic.normalizeJoinCode(initialCode)
            busy = true
            errorMessage = null
            runCatching { LiveQuizApi.lookupJoinCode(code) }
                .onSuccess { result ->
                    lookup = result
                    val existing = LiveQuizPlayerSessionStore.load(result.gameId)
                    if (existing != null) {
                        playerSession = existing
                        step = LiveQuizLogic.JoinStep.Play
                        startSocket(existing)
                    } else {
                        step = LiveQuizLogic.JoinStep.Nickname
                    }
                }
                .onFailure { e ->
                    errorMessage = if (e is LiveQuizApi.LiveQuizJoinError) {
                        L.text(context, localePrefs, joinErrorRes(e.reason))
                    } else {
                        L.text(context, localePrefs, R.string.mobile_liveQuiz_error_generic)
                    }
                }
            busy = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                text = L.text(R.string.mobile_liveQuiz_join_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            TextButton(onClick = {
                teardown()
                onClose()
            }) {
                Text(L.text(R.string.mobile_common_close))
            }
        }

        when (step) {
            LiveQuizLogic.JoinStep.Code -> {
                Text(L.text(R.string.mobile_liveQuiz_join_codePrompt), color = textPrimary())
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }
                OutlinedTextField(
                    value = code,
                    onValueChange = { code = it },
                    label = { Text(L.text(R.string.mobile_liveQuiz_join_code)) },
                    modifier = Modifier.fillMaxWidth().testTag("liveQuiz.join.code"),
                    singleLine = true,
                )
                Button(
                    onClick = {
                        scope.launch {
                            busy = true
                            errorMessage = null
                            runCatching { LiveQuizApi.lookupJoinCode(code) }
                                .onSuccess { result ->
                                    lookup = result
                                    val existing = LiveQuizPlayerSessionStore.load(result.gameId)
                                    if (existing != null) {
                                        playerSession = existing
                                        step = LiveQuizLogic.JoinStep.Play
                                        startSocket(existing)
                                    } else {
                                        step = LiveQuizLogic.JoinStep.Nickname
                                    }
                                    LiveQuizObservability.record("live_quiz_join", mapOf("step" to "lookup"))
                                }
                                .onFailure { e ->
                                    errorMessage = if (e is LiveQuizApi.LiveQuizJoinError) {
                                        L.text(context, localePrefs, joinErrorRes(e.reason))
                                    } else {
                                        L.text(context, localePrefs, R.string.mobile_liveQuiz_error_generic)
                                    }
                                }
                            busy = false
                        }
                    },
                    enabled = !busy && LiveQuizLogic.isValidJoinCode(code),
                ) {
                    Text(L.text(R.string.mobile_liveQuiz_join_continue))
                }
            }
            LiveQuizLogic.JoinStep.Nickname -> {
                lookup?.kitTitle?.takeIf { it.isNotBlank() }?.let {
                    Text(it, fontWeight = FontWeight.Medium, color = textPrimary())
                }
                Text(L.text(R.string.mobile_liveQuiz_join_nicknamePrompt), color = textPrimary())
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }
                OutlinedTextField(
                    value = nickname,
                    onValueChange = { nickname = it },
                    label = { Text(L.text(R.string.mobile_liveQuiz_join_nickname)) },
                    modifier = Modifier.fillMaxWidth().testTag("liveQuiz.join.nickname"),
                    singleLine = true,
                )
                Button(
                    onClick = {
                        scope.launch {
                            busy = true
                            errorMessage = null
                            when (val validation = LiveQuizLogic.validateNickname(nickname)) {
                                is LiveQuizLogic.NicknameValidation.Invalid -> {
                                    errorMessage = L.text(context, localePrefs, nicknameErrorRes(validation.reason))
                                }
                                is LiveQuizLogic.NicknameValidation.Ok -> {
                                    val lu = lookup
                                    if (lu == null) {
                                        busy = false
                                        return@launch
                                    }
                                    runCatching {
                                        val token = accessToken
                                        when {
                                            !token.isNullOrBlank() -> LiveQuizApi.joinLiveGame(
                                                lu.courseCode,
                                                lu.gameId,
                                                validation.nickname,
                                                token,
                                            )
                                            lu.allowsGuests -> LiveQuizApi.joinLiveGameAsGuest(
                                                code,
                                                validation.nickname,
                                            )
                                            else -> throw LiveQuizApi.LiveQuizJoinError(
                                                401,
                                                LiveQuizLogic.JoinErrorReason.AuthRequired,
                                            )
                                        }
                                    }.onSuccess { joined ->
                                        val sess = LiveQuizPlayerSession(
                                            gameId = lu.gameId,
                                            courseCode = lu.courseCode,
                                            playerId = joined.playerId,
                                            playerToken = joined.playerToken,
                                            nickname = joined.nickname,
                                            joinCode = LiveQuizLogic.normalizeJoinCode(code),
                                        )
                                        LiveQuizPlayerSessionStore.save(sess)
                                        playerSession = sess
                                        step = LiveQuizLogic.JoinStep.Play
                                        LiveQuizObservability.record(
                                            "live_quiz_join",
                                            mapOf("rejoined" to if (joined.rejoined == true) "1" else "0"),
                                        )
                                        startSocket(sess)
                                    }.onFailure { e ->
                                        errorMessage = if (e is LiveQuizApi.LiveQuizJoinError) {
                                            L.text(context, localePrefs, joinErrorRes(e.reason))
                                        } else {
                                            L.text(context, localePrefs, R.string.mobile_liveQuiz_error_generic)
                                        }
                                    }
                                }
                            }
                            busy = false
                        }
                    },
                    enabled = !busy,
                ) {
                    Text(L.text(R.string.mobile_liveQuiz_join_submit))
                }
                TextButton(onClick = {
                    step = LiveQuizLogic.JoinStep.Code
                    errorMessage = null
                }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            }
            LiveQuizLogic.JoinStep.Play -> {
                val phase = LiveGameLogic.Phase.parse(gameState?.phase)
                val surface = LiveGameLogic.playSurface(phase, conn)
                Text(connLabel(conn), color = textSecondary())
                when (surface) {
                    LiveGameLogic.PlaySurface.Connecting -> {
                        CircularProgressIndicator()
                        Text(L.text(R.string.mobile_liveQuiz_play_connecting), color = textSecondary())
                    }
                    LiveGameLogic.PlaySurface.Lobby, LiveGameLogic.PlaySurface.WaitingForHost -> {
                        Text(
                            gameState?.kitTitle ?: lookup?.kitTitle.orEmpty(),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(L.text(R.string.mobile_liveQuiz_play_lobbyWaiting), color = textSecondary())
                        gameState?.players?.let {
                            Text(L.format(R.string.mobile_liveQuiz_play_playerCount, it.size), color = textSecondary())
                        }
                    }
                    LiveGameLogic.PlaySurface.Question -> {
                        val q = gameState?.question
                        if (q != null) {
                            val qType = LiveGameLogic.QuestionType.parse(q.questionType)
                                ?: LiveGameLogic.QuestionType.McSingle
                            val hasAnswered = answeredIndex == gameState?.questionIndex
                            Text(q.prompt, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            when (qType) {
                                LiveGameLogic.QuestionType.McSingle, LiveGameLogic.QuestionType.TrueFalse -> {
                                    q.options.forEachIndexed { index, opt ->
                                        Row(
                                            modifier = Modifier
                                                .fillMaxWidth()
                                                .clickable(enabled = !hasAnswered) { selectedOptionId = opt.id }
                                                .padding(8.dp),
                                        ) {
                                            Text("${LiveGameLogic.answerShapeLabel(index)} ${opt.text}")
                                        }
                                    }
                                }
                                LiveGameLogic.QuestionType.McMultiple, LiveGameLogic.QuestionType.Poll -> {
                                    q.options.forEachIndexed { index, opt ->
                                        Row(
                                            modifier = Modifier
                                                .fillMaxWidth()
                                                .clickable(enabled = !hasAnswered) {
                                                    selectedOptionIds = if (opt.id in selectedOptionIds) {
                                                        selectedOptionIds - opt.id
                                                    } else {
                                                        selectedOptionIds + opt.id
                                                    }
                                                }
                                                .padding(8.dp),
                                        ) {
                                            Text("${LiveGameLogic.answerShapeLabel(index)} ${opt.text}")
                                        }
                                    }
                                }
                                LiveGameLogic.QuestionType.TypeAnswer, LiveGameLogic.QuestionType.WordCloud -> {
                                    OutlinedTextField(
                                        value = answerText,
                                        onValueChange = { answerText = it },
                                        enabled = !hasAnswered,
                                        modifier = Modifier.fillMaxWidth(),
                                        label = { Text(L.text(R.string.mobile_liveQuiz_play_typeAnswer)) },
                                    )
                                }
                                LiveGameLogic.QuestionType.Numeric -> {
                                    OutlinedTextField(
                                        value = answerNumeric,
                                        onValueChange = { answerNumeric = it },
                                        enabled = !hasAnswered,
                                        modifier = Modifier.fillMaxWidth(),
                                        label = { Text(L.text(R.string.mobile_liveQuiz_play_numeric)) },
                                    )
                                }
                                LiveGameLogic.QuestionType.Ordering -> {
                                    val ids = orderIds.ifEmpty { q.options.map { it.id } }
                                    if (orderIds.isEmpty()) orderIds = ids
                                    ids.forEach { id ->
                                        val text = q.options.firstOrNull { it.id == id }?.text ?: id
                                        Text(text, modifier = Modifier.padding(8.dp), color = textPrimary())
                                    }
                                }
                            }
                            if (LiveGameLogic.canSubmitAnswer(phase, hasAnswered, conn)) {
                                Button(
                                    onClick = {
                                        val index = gameState?.questionIndex ?: return@Button
                                        val payload = LiveGameLogic.buildAnswer(
                                            questionType = qType,
                                            selectedOptionId = selectedOptionId,
                                            selectedOptionIds = selectedOptionIds.toList(),
                                            text = answerText,
                                            numeric = answerNumeric.trim().toDoubleOrNull(),
                                            order = orderIds.ifEmpty { q.options.map { it.id } },
                                        ) ?: return@Button
                                        answeredIndex = index
                                        socket?.submitAnswer(index, payload)
                                    },
                                    modifier = Modifier.testTag("liveQuiz.play.submit"),
                                ) {
                                    Text(L.text(R.string.mobile_liveQuiz_play_submit))
                                }
                            }
                        }
                    }
                    LiveGameLogic.PlaySurface.Leaderboard, LiveGameLogic.PlaySurface.Podium -> {
                        Text(L.text(R.string.mobile_liveQuiz_play_leaderboard), fontWeight = FontWeight.SemiBold)
                        gameState?.you?.let {
                            Text(L.format(R.string.mobile_liveQuiz_play_yourRank, it.rank, it.totalScore))
                        }
                        (gameState?.leaderboard ?: gameState?.podium).orEmpty().forEach { entry ->
                            Text("#${entry.rank} ${entry.nickname} — ${entry.totalScore}")
                        }
                    }
                    LiveGameLogic.PlaySurface.Ended -> {
                        Text(L.text(R.string.mobile_liveQuiz_play_ended), fontWeight = FontWeight.SemiBold)
                        myResults?.let {
                            Text(
                                L.format(
                                    R.string.mobile_liveQuiz_results_summary,
                                    it.rank,
                                    it.totalScore,
                                    it.correct,
                                    it.answered,
                                ),
                            )
                        }
                    }
                    LiveGameLogic.PlaySurface.Kicked -> {
                        Text(L.text(R.string.mobile_liveQuiz_play_kicked), color = androidx.compose.ui.graphics.Color.Red)
                    }
                }
            }
        }
        Spacer(modifier = Modifier.height(24.dp))
    }
}

@Composable
private fun connLabel(conn: LiveGameLogic.ConnStatus): String = when (conn) {
    LiveGameLogic.ConnStatus.Connecting -> L.text(R.string.mobile_liveQuiz_conn_connecting)
    LiveGameLogic.ConnStatus.Connected -> L.text(R.string.mobile_liveQuiz_conn_connected)
    LiveGameLogic.ConnStatus.Reconnecting -> L.text(R.string.mobile_liveQuiz_conn_reconnecting)
    LiveGameLogic.ConnStatus.Ended -> L.text(R.string.mobile_liveQuiz_conn_ended)
    LiveGameLogic.ConnStatus.Kicked -> L.text(R.string.mobile_liveQuiz_conn_kicked)
    LiveGameLogic.ConnStatus.Disconnected -> L.text(R.string.mobile_liveQuiz_conn_disconnected)
}

private fun joinErrorRes(reason: LiveQuizLogic.JoinErrorReason): Int = when (reason) {
    LiveQuizLogic.JoinErrorReason.NotFound -> R.string.mobile_liveQuiz_error_notFound
    LiveQuizLogic.JoinErrorReason.RateLimited -> R.string.mobile_liveQuiz_error_rateLimited
    LiveQuizLogic.JoinErrorReason.NicknameTaken -> R.string.mobile_liveQuiz_error_nicknameTaken
    LiveQuizLogic.JoinErrorReason.NicknameInvalid -> R.string.mobile_liveQuiz_error_nicknameInvalid
    LiveQuizLogic.JoinErrorReason.NicknameDenied -> R.string.mobile_liveQuiz_error_nicknameDenied
    LiveQuizLogic.JoinErrorReason.LobbyLocked -> R.string.mobile_liveQuiz_error_lobbyLocked
    LiveQuizLogic.JoinErrorReason.Banned -> R.string.mobile_liveQuiz_error_banned
    LiveQuizLogic.JoinErrorReason.OneSession -> R.string.mobile_liveQuiz_error_oneSession
    LiveQuizLogic.JoinErrorReason.GameEnded -> R.string.mobile_liveQuiz_error_gameEnded
    LiveQuizLogic.JoinErrorReason.AuthRequired -> R.string.mobile_liveQuiz_error_authRequired
    LiveQuizLogic.JoinErrorReason.JoinFailed -> R.string.mobile_liveQuiz_error_joinFailed
    LiveQuizLogic.JoinErrorReason.Unknown -> R.string.mobile_liveQuiz_error_generic
}

private fun nicknameErrorRes(reason: LiveQuizLogic.NicknameReason): Int = when (reason) {
    LiveQuizLogic.NicknameReason.Empty -> R.string.mobile_liveQuiz_nickname_error_empty
    LiveQuizLogic.NicknameReason.TooLong -> R.string.mobile_liveQuiz_nickname_error_tooLong
    LiveQuizLogic.NicknameReason.Charset -> R.string.mobile_liveQuiz_nickname_error_charset
}
