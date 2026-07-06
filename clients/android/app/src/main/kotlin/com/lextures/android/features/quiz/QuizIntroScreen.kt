package com.lextures.android.features.quiz

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.activity.ComponentActivity
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ModuleItemDetail
import com.lextures.android.core.lms.ModuleQuizPayload
import com.lextures.android.core.lms.QuizAnswerState
import com.lextures.android.core.lms.QuizAttemptSummary
import com.lextures.android.core.lms.QuizLogic
import com.lextures.android.core.lms.QuizQuestion
import com.lextures.android.core.lms.QuizResultsResponse
import com.lextures.android.core.lms.QuizSaveState
import com.lextures.android.core.lms.QuizStartResponse
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.courses.RowHeader
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

@Composable
fun QuizIntroScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    onProgressChanged: suspend () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    var detail by remember { mutableStateOf<ModuleItemDetail?>(null) }
    var quiz by remember { mutableStateOf<ModuleQuizPayload?>(null) }
    var attempts by remember { mutableStateOf<List<QuizAttemptSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var starting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var startResponse by remember { mutableStateOf<QuizStartResponse?>(null) }
    var showTaker by remember { mutableStateOf(false) }
    var showPreview by remember { mutableStateOf(false) }
    var showLockdownConsent by remember { mutableStateOf(false) }
    var proctoringRequired by remember { mutableStateOf(false) }
    val isStaff = course.viewerIsStaff

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        error = null
        try {
            detail = LmsApi.fetchItemDetail(course.courseCode, item, token)
            quiz = LmsApi.fetchModuleQuiz(course.courseCode, item.id, null, token)
            attempts = LmsApi.fetchQuizAttempts(course.courseCode, item.id, token)
            proctoringRequired = LmsApi.fetchQuizProctoringConfig(course.courseCode, item.id, token)?.required == true
        } catch (e: Exception) {
            error = session.mapError(e)
        } finally {
            loading = false
        }
    }

    val scope = rememberCoroutineScope()

    val previewQuiz = quiz
    if (showPreview && previewQuiz != null) {
        QuizPreviewScreen(
            title = item.title,
            quiz = previewQuiz,
            onBack = { showPreview = false },
            modifier = modifier,
        )
        return
    }

    if (showTaker && startResponse != null) {
        QuizTakerScreen(
            session = session,
            course = course,
            item = item,
            quiz = quiz,
            start = startResponse!!,
            onBack = {
                showTaker = false
            },
            onFinished = {
                showTaker = false
                scope.launch { onProgressChanged() }
            },
            modifier = modifier,
        )
        return
    }

    Column(modifier = modifier.fillMaxSize()) {
        RowHeader(title = item.title, onBack = onBack)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            error?.let { LmsErrorBanner(it) }
            if (loading) {
                Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = LexturesColors.Primary)
                }
            } else {
                Text(
                    text = detail?.title ?: quiz?.title ?: item.title,
                    fontWeight = FontWeight.Bold,
                )
                val markdown = detail?.markdown ?: quiz?.markdown
                if (!markdown.isNullOrBlank()) {
                    LmsCard {
                        Text(markdown)
                    }
                }
                val timeLimit = detail?.timeLimitMinutes ?: quiz?.timeLimitMinutes
                val maxAttempts = detail?.maxAttempts ?: quiz?.maxAttempts
                if (timeLimit != null || maxAttempts != null) {
                    LmsCard {
                        timeLimit?.let { Text("Time limit: $it min") }
                        if (detail?.unlimitedAttempts != true && quiz?.unlimitedAttempts != true) {
                            maxAttempts?.let { Text("Max attempts: $it") }
                        } else {
                            Text("Unlimited attempts")
                        }
                    }
                }
                if (attempts.isNotEmpty()) {
                    LmsCard {
                        Text(quizPreviousAttemptsLabel(), fontWeight = FontWeight.SemiBold)
                        attempts.forEach { attempt ->
                            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(quizAttemptNumberLabel(attempt.attemptNumber))
                                attempt.scorePercent?.let { Text("${it.toInt()}%") }
                            }
                        }
                    }
                }
                val unlimited = detail?.unlimitedAttempts == true || quiz?.unlimitedAttempts == true
                val canStart = quiz?.isAdaptive != true && (unlimited || attempts.size < (detail?.maxAttempts ?: quiz?.maxAttempts ?: 1))
                if (quiz?.isAdaptive == true) {
                    LmsCard { Text(quizAdaptiveWebOnlyLabel()) }
                } else if (isStaff) {
                    AuthPrimaryButton(
                        text = quizPreviewLabel(),
                        onClick = { showPreview = true },
                        modifier = Modifier.fillMaxWidth(),
                    )
                } else if (!canStart) {
                    LmsCard { Text(quizNoAttemptsLabel()) }
                } else {
                    val scope = rememberCoroutineScope()
                    val lockdownMode = detail?.lockdownMode ?: quiz?.lockdownMode
                    val needsConsent = QuizLogic.needsLockdownConsent(lockdownMode) ||
                        QuizLogic.requiresDeviceLockdown(lockdownMode, proctoringRequired)
                    AuthPrimaryButton(
                        text = if (starting) "…" else quizStartLabel(),
                        onClick = {
                            if (needsConsent) {
                                showLockdownConsent = true
                            } else {
                                val token = accessToken ?: return@AuthPrimaryButton
                                scope.launch {
                                    starting = true
                                    try {
                                        val start = LmsApi.startQuiz(course.courseCode, item.id, null, token)
                                        quiz = LmsApi.fetchModuleQuiz(course.courseCode, item.id, start.attemptId, token)
                                        startResponse = start
                                        showTaker = true
                                    } catch (e: Exception) {
                                        error = session.mapError(e)
                                    } finally {
                                        starting = false
                                    }
                                }
                            }
                        },
                        enabled = !starting,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    if (showLockdownConsent) {
                        LockdownConsentDialog(
                            lockdownMode = lockdownMode,
                            onConfirm = {
                                showLockdownConsent = false
                                val token = accessToken ?: return@LockdownConsentDialog
                                scope.launch {
                                    starting = true
                                    try {
                                        val start = LmsApi.startQuiz(course.courseCode, item.id, null, token)
                                        quiz = LmsApi.fetchModuleQuiz(course.courseCode, item.id, start.attemptId, token)
                                        startResponse = start
                                        showTaker = true
                                    } catch (e: Exception) {
                                        error = session.mapError(e)
                                    } finally {
                                        starting = false
                                    }
                                }
                            },
                            onDismiss = { showLockdownConsent = false },
                        )
                    }
                }
            }
        }
    }
}

/**
 * Read-only teacher preview of a quiz. Renders every question with the shared
 * question renderer using ephemeral local answers; nothing is saved and no
 * attempt is started. Mirrors the web "Student preview" modal.
 */
@Composable
fun QuizPreviewScreen(
    title: String,
    quiz: ModuleQuizPayload,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val answers = remember { mutableStateMapOf<String, QuizAnswerState>() }
    Column(modifier = modifier.fillMaxSize()) {
        RowHeader(title = title, onBack = onBack)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            LmsCard { Text(quizPreviewNoteLabel()) }
            val questions = quiz.questions.orEmpty()
            if (questions.isEmpty()) {
                LmsCard { Text(quizPreviewEmptyLabel()) }
            } else {
                questions.forEachIndexed { index, question ->
                    Text(quizQuestionNumberLabel(index + 1), fontWeight = FontWeight.SemiBold)
                    QuizQuestionContent(
                        question = question,
                        answer = answers[question.id] ?: QuizAnswerState(),
                        saveState = QuizSaveState.Idle,
                        onChange = { answers[question.id] = it },
                    )
                }
            }
        }
    }
}

@Composable
fun QuizTakerScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    quiz: ModuleQuizPayload?,
    start: QuizStartResponse,
    onBack: () -> Unit,
    onFinished: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val activity = context as? ComponentActivity
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()
    val deviceLockdownRequired = remember(start.lockdownMode) {
        QuizLogic.requiresDeviceLockdown(start.lockdownMode)
    }
    val lockdownController = remember(activity) {
        activity?.let { LockdownController(it) }
    }

    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var questions by remember { mutableStateOf<List<QuizQuestion>>(emptyList()) }
    var currentIndex by remember { mutableIntStateOf(start.currentQuestionIndex) }
    var totalQuestions by remember { mutableIntStateOf(0) }
    var serverLockdown by remember { mutableStateOf(QuizLogic.isServerLockdown(start.lockdownMode)) }
    var serverQuestion by remember { mutableStateOf<QuizQuestion?>(null) }
    var serverCompleted by remember { mutableStateOf(false) }
    var timerSeconds by remember { mutableStateOf<Int?>(QuizLogic.secondsRemaining(start.deadlineAt)) }
    var advancing by remember { mutableStateOf(false) }
    var submitting by remember { mutableStateOf(false) }
    var showResults by remember { mutableStateOf(false) }
    var results by remember { mutableStateOf<QuizResultsResponse?>(null) }
    val answers = remember { mutableStateMapOf<String, QuizAnswerState>() }
    val saveStates = remember { mutableStateMapOf<String, QuizSaveState>() }
    val flagged = remember { mutableStateMapOf<String, Boolean>() }

    val lifecycleOwner = LocalLifecycleOwner.current
    androidx.compose.runtime.DisposableEffect(lifecycleOwner, start.attemptId, accessToken, lockdownController) {
        val controller = lockdownController
        if (controller != null && deviceLockdownRequired) {
            controller.activate { eventType ->
                val token = accessToken ?: return@activate
                scope.launch {
                    LmsApi.postQuizFocusLoss(
                        course.courseCode, item.id, start.attemptId, eventType, token,
                    )
                }
            }
            lifecycleOwner.lifecycle.addObserver(controller)
            onDispose {
                controller.deactivate()
                lifecycleOwner.lifecycle.removeObserver(controller)
            }
        } else {
            val observer = LifecycleEventObserver { _, event ->
                if (event == Lifecycle.Event.ON_STOP) {
                    val token = accessToken ?: return@LifecycleEventObserver
                    scope.launch {
                        LmsApi.postQuizFocusLoss(
                            course.courseCode, item.id, start.attemptId, "app_background", token,
                        )
                    }
                }
            }
            lifecycleOwner.lifecycle.addObserver(observer)
            onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            if (serverLockdown) {
                val cur = LmsApi.fetchQuizCurrentQuestion(course.courseCode, item.id, start.attemptId, token)
                serverQuestion = cur.question
                serverCompleted = cur.completed
                currentIndex = cur.questionIndex
                totalQuestions = cur.totalQuestions
                cur.question?.let { questions = listOf(it) }
            } else {
                val payload = quiz ?: LmsApi.fetchModuleQuiz(course.courseCode, item.id, start.attemptId, token)
                questions = payload.questions.orEmpty()
                totalQuestions = questions.size
            }
        } catch (e: Exception) {
            error = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(start.deadlineAt) {
        while (isActive) {
            timerSeconds = QuizLogic.secondsRemaining(start.deadlineAt)
            if (timerSeconds == 0) {
                val token = accessToken ?: break
                submitting = true
                try {
                    val responses = if (serverLockdown) null else questions.map {
                        QuizLogic.buildResponseItem(it, answers[it.id])
                    }
                    LmsApi.submitQuiz(course.courseCode, item.id, start.attemptId, responses, token)
                    results = LmsApi.fetchQuizResults(course.courseCode, item.id, start.attemptId, token)
                    showResults = true
                } catch (e: Exception) {
                    error = session.mapError(e)
                } finally {
                    submitting = false
                }
                break
            }
            delay(1000)
        }
    }

    if (showResults && results != null) {
        QuizResultsScreen(
            title = item.title,
            results = results!!,
            onDone = onFinished,
            modifier = modifier,
        )
        return
    }

    Column(modifier = modifier.fillMaxSize()) {
        RowHeader(title = item.title, onBack = onBack)
        timerSeconds?.let { seconds ->
            Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
                Text(quizTimerLabel(QuizLogic.formatTimer(seconds)), fontWeight = FontWeight.SemiBold)
            }
        }
        if (deviceLockdownRequired) {
            Text(
                quizLockdownKioskBanner(),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                fontWeight = FontWeight.SemiBold,
            )
        }
        lockdownController?.platformWarning?.let { warning ->
            Text(
                warning,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
        lockdownController?.focusLossBanner?.let { banner ->
            Text(
                banner,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                fontWeight = FontWeight.SemiBold,
            )
        }
        error?.let { LmsErrorBanner(it, Modifier.padding(horizontal = 16.dp)) }
        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            serverCompleted && serverQuestion == null -> {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    AuthPrimaryButton(
                        text = quizSubmitLabel(),
                        onClick = {
                            val token = accessToken ?: return@AuthPrimaryButton
                            scope.launch {
                                submitting = true
                                try {
                                    LmsApi.submitQuiz(course.courseCode, item.id, start.attemptId, null, token)
                                    results = LmsApi.fetchQuizResults(course.courseCode, item.id, start.attemptId, token)
                                    showResults = true
                                } catch (e: Exception) {
                                    error = session.mapError(e)
                                } finally {
                                    submitting = false
                                }
                            }
                        },
                        enabled = !submitting,
                    )
                }
            }
            else -> {
                val question = if (serverLockdown) serverQuestion else questions.getOrNull(currentIndex)
                if (question == null) {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        AuthPrimaryButton(
                            text = quizSubmitLabel(),
                            onClick = {
                                val token = accessToken ?: return@AuthPrimaryButton
                                scope.launch {
                                    submitting = true
                                    try {
                                        val responses = questions.map {
                                            QuizLogic.buildResponseItem(it, answers[it.id])
                                        }
                                        LmsApi.submitQuiz(course.courseCode, item.id, start.attemptId, responses, token)
                                        results = LmsApi.fetchQuizResults(course.courseCode, item.id, start.attemptId, token)
                                        showResults = true
                                    } catch (e: Exception) {
                                        error = session.mapError(e)
                                    } finally {
                                        submitting = false
                                    }
                                }
                            },
                            enabled = !submitting,
                        )
                    }
                } else {
                    Column(
                        Modifier
                            .weight(1f)
                            .verticalScroll(rememberScrollState())
                            .padding(16.dp),
                    ) {
                        Text(quizProgressLabel(currentIndex + 1, maxOf(totalQuestions, questions.size, 1)))
                        QuizQuestionContent(
                            question = question,
                            answer = answers[question.id] ?: QuizAnswerState(),
                            saveState = saveStates[question.id] ?: QuizSaveState.Idle,
                            onChange = { answers[question.id] = it; saveStates[question.id] = QuizSaveState.Saved },
                            codeRunContext = accessToken?.let { token ->
                                CodeQuestionRunContext(
                                    courseCode = course.courseCode,
                                    itemId = item.id,
                                    attemptId = start.attemptId,
                                    accessToken = token,
                                )
                            },
                        )
                    }
                    Row(
                        Modifier.fillMaxWidth().padding(16.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        if (!serverLockdown && currentIndex > 0 && start.backNavigationAllowed) {
                            AuthPrimaryButton(
                                text = quizPreviousLabel(),
                                onClick = { currentIndex -= 1 },
                            )
                        }
                        AuthPrimaryButton(
                            text = when {
                                serverLockdown && serverCompleted -> quizSubmitLabel()
                                !serverLockdown && currentIndex >= questions.lastIndex -> quizSubmitLabel()
                                else -> quizNextLabel()
                            },
                            onClick = {
                                val token = accessToken ?: return@AuthPrimaryButton
                                scope.launch {
                                    if (serverLockdown) {
                                        advancing = true
                                        try {
                                            val body = QuizLogic.buildResponseItem(question, answers[question.id])
                                            if (isOnline) {
                                                val res = LmsApi.advanceQuiz(
                                                    course.courseCode, item.id, start.attemptId, body, token,
                                                )
                                                serverCompleted = res.completed
                                                if (res.completed) {
                                                    serverQuestion = null
                                                    return@launch
                                                }
                                                val cur = LmsApi.fetchQuizCurrentQuestion(
                                                    course.courseCode, item.id, start.attemptId, token,
                                                )
                                                serverQuestion = cur.question
                                                currentIndex = cur.questionIndex
                                                totalQuestions = cur.totalQuestions
                                            } else {
                                                error = context.getString(R.string.mobile_quiz_notYetSaved)
                                                saveStates[question.id] = QuizSaveState.Queued
                                            }
                                        } catch (e: Exception) {
                                            error = session.mapError(e)
                                            saveStates[question.id] = QuizSaveState.Failed
                                        } finally {
                                            advancing = false
                                        }
                                    } else if (currentIndex >= questions.lastIndex) {
                                        submitting = true
                                        try {
                                            val responses = questions.map {
                                                QuizLogic.buildResponseItem(it, answers[it.id])
                                            }
                                            LmsApi.submitQuiz(course.courseCode, item.id, start.attemptId, responses, token)
                                            results = LmsApi.fetchQuizResults(course.courseCode, item.id, start.attemptId, token)
                                            showResults = true
                                        } catch (e: Exception) {
                                            error = session.mapError(e)
                                        } finally {
                                            submitting = false
                                        }
                                    } else {
                                        currentIndex += 1
                                    }
                                }
                            },
                            enabled = !advancing && !submitting,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LockdownConsentDialog(
    lockdownMode: String?,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    val isKiosk = QuizLogic.isKioskMode(lockdownMode)
    val title = if (isKiosk) quizLockdownKioskTitle() else quizLockdownOneAtATimeTitle()
    val bullets = if (isKiosk) {
        listOf(
            quizLockdownKioskBulletBack(),
            quizLockdownKioskBulletHints(),
            quizLockdownKioskBulletFocus(),
        )
    } else {
        listOf(
            quizLockdownOneAtATimeBulletBack(),
            quizLockdownOneAtATimeBulletHints(),
        )
    }

    androidx.compose.ui.window.Dialog(onDismissRequest = onDismiss) {
        LmsCard {
            Column(
                modifier = Modifier.padding(8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(title, fontWeight = FontWeight.Bold)
                bullets.forEach { bullet ->
                    Text("• $bullet")
                }
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    AuthPrimaryButton(
                        text = quizLockdownConfirmLabel(),
                        onClick = onConfirm,
                    )
                    TextButton(onClick = onDismiss) {
                        Text(quizLockdownCancelLabel())
                    }
                }
            }
        }
    }
}
