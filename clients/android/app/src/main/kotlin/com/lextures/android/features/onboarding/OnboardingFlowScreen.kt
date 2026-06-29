package com.lextures.android.features.onboarding

import android.Manifest
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.AuthTextField
import com.lextures.android.core.design.AuthScreenContainer
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.PublicAuthBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.DiagnosticQuestion
import com.lextures.android.core.lms.LearnerGoals
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OnboardingStep
import com.lextures.android.core.lms.PriorKnowledgeLevel
import com.lextures.android.core.lms.onboardingTopics
import com.lextures.android.core.push.PushManager
import com.lextures.android.core.routing.CourseDeepLinkSection
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.features.home.HomeScreen
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun AuthenticatedRootScreen(session: AuthSession, modifier: Modifier = Modifier) {
    val accessToken by session.accessToken.collectAsState()
    var gate by remember { mutableStateOf(OnboardingGate.Loading) }
    var pendingDeepLink by remember { mutableStateOf<DeepLinkDestination?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        gate = OnboardingGate.Loading
        gate = runCatching { LmsApi.fetchOnboardingStatus(token) }
            .getOrNull()
            ?.let { if (it.completed) OnboardingGate.Done else OnboardingGate.ShowFlow }
            ?: OnboardingGate.Done
    }

    when (gate) {
        OnboardingGate.Loading -> Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = LexturesColors.Primary)
        }
        OnboardingGate.ShowFlow -> OnboardingFlowScreen(
            session = session,
            modifier = modifier,
            onFinished = { destination ->
                pendingDeepLink = destination
                gate = OnboardingGate.Done
            },
        )
        OnboardingGate.Done -> HomeScreen(
            session = session,
            initialDeepLink = pendingDeepLink,
            modifier = modifier,
        )
    }
}

private enum class OnboardingGate { Loading, ShowFlow, Done }

/** First-run onboarding wizard (parity with web `onboarding-page`). */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun OnboardingFlowScreen(
    session: AuthSession,
    onFinished: (DeepLinkDestination?) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val pushManager = remember { PushManager.getInstance(context) }

    var step by rememberSaveable { mutableStateOf(OnboardingStep.Welcome.name) }
    var loading by rememberSaveable { mutableStateOf(true) }
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var goals by remember { mutableStateOf<LearnerGoals?>(null) }

    var topic by rememberSaveable { mutableStateOf("") }
    var goalText by rememberSaveable { mutableStateOf("") }
    var hasTargetDate by rememberSaveable { mutableStateOf(false) }
    var targetDateMillis by rememberSaveable { mutableStateOf(System.currentTimeMillis()) }
    var priorLevel by rememberSaveable { mutableStateOf(PriorKnowledgeLevel.Beginner.wire) }
    var dailyMinutes by rememberSaveable { mutableStateOf("20") }
    var reminderOptIn by rememberSaveable { mutableStateOf(false) }
    var reminderHour by rememberSaveable { mutableIntStateOf(9) }
    var reminderMinute by rememberSaveable { mutableIntStateOf(0) }
    var termsAccepted by rememberSaveable { mutableStateOf(false) }

    var questions by remember { mutableStateOf<List<DiagnosticQuestion>>(emptyList()) }
    var questionIndex by rememberSaveable { mutableIntStateOf(0) }
    var answers by remember { mutableStateOf<Map<String, Int>>(emptyMap()) }

    var showDatePicker by remember { mutableStateOf(false) }
    var showTimePicker by remember { mutableStateOf(false) }

    val notificationPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) pushManager.requestTokenSync()
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        runCatching { LmsApi.fetchOnboardingStatus(token) }
            .onSuccess { status ->
                when {
                    status == null || status.completed -> onFinished(null)
                    status.step > 0 -> {
                        step = OnboardingStep.entries
                            .firstOrNull { it.value == status.step.coerceAtMost(6) }
                            ?.name
                            ?: OnboardingStep.Welcome.name
                    }
                }
            }
            .onFailure { errorMessage = context.getString(R.string.mobile_onboarding_error_load) }
        loading = false
    }

    LaunchedEffect(step, topic, accessToken) {
        if (step != OnboardingStep.Diagnostic.name || topic.isBlank()) return@LaunchedEffect
        val token = accessToken ?: return@LaunchedEffect
        questions = runCatching { LmsApi.fetchDiagnosticQuestions(topic, token) }.getOrDefault(emptyList())
        questionIndex = 0
        answers = emptyMap()
    }

    suspend fun persistStep(next: OnboardingStep, body: Map<String, Any?>): Boolean {
        val token = accessToken ?: return false
        submitting = true
        errorMessage = null
        val payload = body.toMutableMap()
        payload["step"] = next.value
        val result = runCatching { LmsApi.postOnboarding(payload, token) }
        submitting = false
        return result.fold(
            onSuccess = {
                goals = it
                step = next.name
                true
            },
            onFailure = {
                errorMessage = context.getString(R.string.mobile_onboarding_error_save)
                false
            },
        )
    }

    PublicAuthBackground(modifier = modifier.fillMaxSize()) {
        Box(modifier = Modifier.fillMaxSize()) {
            if (loading) {
                Column(
                    modifier = Modifier.align(Alignment.Center),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    CircularProgressIndicator(color = LexturesColors.Primary)
                    Text(L.text(R.string.mobile_onboarding_loading), color = textSecondary())
                }
            } else {
                AuthScreenContainer {
                    val currentStep = OnboardingStep.valueOf(step)
                    OnboardingShell(
                        step = currentStep,
                        title = onboardingTitle(currentStep),
                        onBack = onboardingBack(currentStep)?.let { back -> { step = back.name } },
                    ) {
                        errorMessage?.let {
                            Text(text = it, color = LexturesColors.Error, fontSize = 14.sp, modifier = Modifier.padding(bottom = 12.dp))
                        }
                        when (currentStep) {
                            OnboardingStep.Welcome -> {
                                Text(L.text(R.string.mobile_onboarding_welcome_subtitle), color = textSecondary(), fontSize = 14.sp)
                                AuthPrimaryButton(
                                    text = L.text(R.string.mobile_onboarding_getStarted),
                                    enabled = !submitting,
                                    onClick = { scope.launch { persistStep(OnboardingStep.Topic, emptyMap()) } },
                                    modifier = Modifier.padding(top = 12.dp),
                                )
                            }
                            OnboardingStep.Topic -> TopicStepContent(
                                topic, goalText, hasTargetDate, targetDateMillis, submitting,
                                onTopic = { topic = it },
                                onGoalText = { goalText = it },
                                onToggleTargetDate = { hasTargetDate = it },
                                onPickDate = { showDatePicker = true },
                                onContinue = {
                                    scope.launch {
                                        persistStep(
                                            OnboardingStep.Experience,
                                            mapOf(
                                                "topic" to topic,
                                                "goalText" to goalText,
                                                "targetDate" to if (hasTargetDate) isoDate(targetDateMillis) else null,
                                            ),
                                        )
                                    }
                                },
                            )
                            OnboardingStep.Experience -> ExperienceStepContent(
                                priorLevel, submitting,
                                onSelect = { priorLevel = it },
                                onContinue = {
                                    scope.launch {
                                        persistStep(OnboardingStep.Diagnostic, mapOf("priorKnowledgeLevel" to priorLevel))
                                    }
                                },
                            )
                            OnboardingStep.Diagnostic -> DiagnosticScreen(
                                questions = questions,
                                questionIndex = questionIndex,
                                answers = answers,
                                submitting = submitting,
                                onSelectAnswer = { id, index -> answers = answers + (id to index) },
                                onNextQuestion = { questionIndex += 1 },
                                onSubmitAnswers = {
                                    scope.launch { persistStep(OnboardingStep.Habits, mapOf("diagnosticAnswers" to answers)) }
                                },
                                onSkip = {
                                    scope.launch { persistStep(OnboardingStep.Habits, mapOf("skipDiagnostic" to true)) }
                                },
                            )
                            OnboardingStep.Habits -> HabitsStepContent(
                                dailyMinutes, reminderOptIn, reminderHour, reminderMinute, submitting,
                                onDailyMinutes = { dailyMinutes = it },
                                onReminderOptIn = { enabled ->
                                    reminderOptIn = enabled
                                    if (enabled) {
                                        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
                                            !pushManager.hasNotificationPermission()
                                        ) {
                                            notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
                                        } else {
                                            pushManager.requestTokenSync()
                                        }
                                    }
                                },
                                onPickTime = { showTimePicker = true },
                                onContinue = {
                                    scope.launch {
                                        persistStep(
                                            OnboardingStep.Consent,
                                            mapOf(
                                                "dailyMinutes" to (dailyMinutes.toIntOrNull() ?: 20),
                                                "reminderOptIn" to reminderOptIn,
                                                "reminderTime" to reminderTime(reminderHour, reminderMinute),
                                            ),
                                        )
                                    }
                                },
                            )
                            OnboardingStep.Consent -> ConsentStepContent(
                                termsAccepted, submitting,
                                onTerms = { termsAccepted = it },
                                onFinish = {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        submitting = true
                                        errorMessage = null
                                        runCatching {
                                            LmsApi.saveStudyReminderPrefs(
                                                reminderOptIn,
                                                reminderTime(reminderHour, reminderMinute),
                                                token,
                                            )
                                            goals = LmsApi.postOnboarding(
                                                mapOf(
                                                    "step" to OnboardingStep.Complete.value,
                                                    "complete" to true,
                                                    "termsAccepted" to true,
                                                    "reminderOptIn" to reminderOptIn,
                                                    "reminderTime" to reminderTime(reminderHour, reminderMinute),
                                                ),
                                                token,
                                            )
                                        }.onSuccess {
                                            step = OnboardingStep.Complete.name
                                        }.onFailure {
                                            errorMessage = context.getString(R.string.mobile_onboarding_error_complete)
                                        }
                                        submitting = false
                                    }
                                },
                            )
                            OnboardingStep.Complete -> CompleteStepContent(
                                goals = goals,
                                onOpenCourse = { code ->
                                    onFinished(DeepLinkDestination.Course(code, CourseDeepLinkSection.Overview))
                                },
                                onDashboard = { onFinished(null) },
                            )
                        }
                    }
                }
            }

            if (!loading && step != OnboardingStep.Complete.name) {
                TextButton(
                    onClick = {
                        scope.launch {
                            val token = accessToken ?: return@launch
                            submitting = true
                            runCatching { LmsApi.postOnboarding(mapOf("skipAll" to true), token) }
                                .onSuccess { onFinished(null) }
                                .onFailure { errorMessage = context.getString(R.string.mobile_onboarding_error_skip) }
                            submitting = false
                        }
                    },
                    enabled = !submitting,
                    modifier = Modifier.align(Alignment.TopEnd).padding(12.dp),
                ) {
                    Text(L.text(R.string.mobile_onboarding_skipForNow), color = textSecondary())
                }
            }
        }
    }

    if (showDatePicker) {
        val state = rememberDatePickerState(initialSelectedDateMillis = targetDateMillis)
        DatePickerDialog(
            onDismissRequest = { showDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    state.selectedDateMillis?.let { targetDateMillis = it }
                    showDatePicker = false
                }) { Text(L.text(R.string.mobile_onboarding_continue)) }
            },
        ) { DatePicker(state = state) }
    }

    if (showTimePicker) {
        val state = rememberTimePickerState(initialHour = reminderHour, initialMinute = reminderMinute)
        DatePickerDialog(
            onDismissRequest = { showTimePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    reminderHour = state.hour
                    reminderMinute = state.minute
                    showTimePicker = false
                }) { Text(L.text(R.string.mobile_onboarding_continue)) }
            },
        ) { TimePicker(state = state) }
    }
}

@Composable
private fun OnboardingShell(
    step: OnboardingStep,
    title: String,
    onBack: (() -> Unit)?,
    content: @Composable () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        val progress = step.value / 6f
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(6.dp)
                .clip(RoundedCornerShape(999.dp))
                .background(LexturesColors.FieldBorder.copy(alpha = 0.6f)),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth(progress.coerceIn(0f, 1f))
                    .height(6.dp)
                    .clip(RoundedCornerShape(999.dp))
                    .background(LexturesColors.Primary),
            )
        }
        onBack?.let { TextButton(onClick = it) { Text(L.text(R.string.mobile_onboarding_back)) } }
        Text(text = title, style = LexturesType.display(26), color = textPrimary())
        content()
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun TopicStepContent(
    topic: String,
    goalText: String,
    hasTargetDate: Boolean,
    targetDateMillis: Long,
    submitting: Boolean,
    onTopic: (String) -> Unit,
    onGoalText: (String) -> Unit,
    onToggleTargetDate: (Boolean) -> Unit,
    onPickDate: () -> Unit,
    onContinue: () -> Unit,
) {
    Text(L.text(R.string.mobile_onboarding_topic_subtitle), color = textSecondary(), fontSize = 14.sp)
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        onboardingTopics.forEach { item ->
            val selected = topic == item.id
            Text(
                text = L.text(item.labelRes),
                modifier = Modifier
                    .clip(RoundedCornerShape(999.dp))
                    .background(if (selected) LexturesColors.BrandTeal.copy(alpha = 0.18f) else LexturesColors.CardBackground)
                    .border(1.dp, if (selected) LexturesColors.Primary else LexturesColors.FieldBorder, RoundedCornerShape(999.dp))
                    .clickable { onTopic(item.id) }
                    .padding(horizontal = 14.dp, vertical = 8.dp),
                color = textPrimary(),
                fontSize = 14.sp,
            )
        }
    }
    AuthTextField(
        title = L.text(R.string.mobile_onboarding_goal_label),
        value = goalText,
        onValueChange = onGoalText,
        placeholder = L.text(R.string.mobile_onboarding_goal_placeholder),
    )
    ToggleRow(L.text(R.string.mobile_onboarding_targetDate_toggle), hasTargetDate, onToggleTargetDate)
    if (hasTargetDate) {
        TextButton(onClick = onPickDate) { Text(isoDate(targetDateMillis), color = textPrimary()) }
    }
    AuthPrimaryButton(
        text = L.text(R.string.mobile_onboarding_continue),
        enabled = topic.isNotBlank() && !submitting,
        onClick = onContinue,
    )
}

@Composable
private fun ExperienceStepContent(
    priorLevel: String,
    submitting: Boolean,
    onSelect: (String) -> Unit,
    onContinue: () -> Unit,
) {
    listOf(
        PriorKnowledgeLevel.Beginner to Pair(R.string.mobile_onboarding_experience_beginner, R.string.mobile_onboarding_experience_beginnerHint),
        PriorKnowledgeLevel.Intermediate to Pair(R.string.mobile_onboarding_experience_intermediate, R.string.mobile_onboarding_experience_intermediateHint),
        PriorKnowledgeLevel.Advanced to Pair(R.string.mobile_onboarding_experience_advanced, R.string.mobile_onboarding_experience_advancedHint),
    ).forEach { (level, labels) ->
        val selected = priorLevel == level.wire
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(14.dp))
                .border(1.dp, if (selected) LexturesColors.Primary else LexturesColors.FieldBorder, RoundedCornerShape(14.dp))
                .clickable { onSelect(level.wire) }
                .padding(14.dp),
        ) {
            Text(L.text(labels.first), fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(L.text(labels.second), fontSize = 12.sp, color = textSecondary())
        }
    }
    AuthPrimaryButton(text = L.text(R.string.mobile_onboarding_continue), enabled = !submitting, onClick = onContinue)
}

@Composable
private fun HabitsStepContent(
    dailyMinutes: String,
    reminderOptIn: Boolean,
    reminderHour: Int,
    reminderMinute: Int,
    submitting: Boolean,
    onDailyMinutes: (String) -> Unit,
    onReminderOptIn: (Boolean) -> Unit,
    onPickTime: () -> Unit,
    onContinue: () -> Unit,
) {
    AuthTextField(
        title = L.text(R.string.mobile_onboarding_habits_dailyMinutes),
        value = dailyMinutes,
        onValueChange = onDailyMinutes,
        placeholder = "20",
        keyboardType = KeyboardType.Number,
    )
    ToggleRow(
        label = L.text(R.string.mobile_onboarding_habits_reminder),
        checked = reminderOptIn,
        onChecked = onReminderOptIn,
        description = L.text(R.string.mobile_onboarding_habits_reminderHint),
    )
    if (reminderOptIn) {
        TextButton(onClick = onPickTime) {
            Text(reminderTime(reminderHour, reminderMinute), color = textPrimary())
        }
    }
    AuthPrimaryButton(text = L.text(R.string.mobile_onboarding_continue), enabled = !submitting, onClick = onContinue)
}

@Composable
private fun ConsentStepContent(
    termsAccepted: Boolean,
    submitting: Boolean,
    onTerms: (Boolean) -> Unit,
    onFinish: () -> Unit,
) {
    ToggleRow(L.text(R.string.mobile_onboarding_consent_terms), termsAccepted, onTerms)
    AuthPrimaryButton(
        text = L.text(R.string.mobile_onboarding_finish),
        enabled = termsAccepted && !submitting,
        onClick = onFinish,
    )
}

@Composable
private fun CompleteStepContent(
    goals: LearnerGoals?,
    onOpenCourse: (String) -> Unit,
    onDashboard: () -> Unit,
) {
    Text(L.text(R.string.mobile_onboarding_complete_subtitle), color = textSecondary(), fontSize = 14.sp)
    val recommended = goals?.recommendedCourseTitle ?: goals?.recommendedCourseCode
    if (recommended != null) {
        LmsCard {
            Text(L.text(R.string.mobile_onboarding_complete_startHere), fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = LexturesColors.PrimaryMuted)
            Text(recommended, fontWeight = FontWeight.SemiBold, color = textPrimary(), modifier = Modifier.padding(top = 4.dp))
            goals?.recommendedCourseCode?.let { code ->
                AuthPrimaryButton(
                    text = L.text(R.string.mobile_onboarding_complete_openCourse),
                    onClick = { onOpenCourse(code) },
                    modifier = Modifier.padding(top = 8.dp),
                )
            }
        }
    } else {
        Text(L.text(R.string.mobile_onboarding_complete_browseCatalog), color = textSecondary(), fontSize = 14.sp)
    }
    AuthPrimaryButton(text = L.text(R.string.mobile_onboarding_complete_goToDashboard), onClick = onDashboard)
}

@Composable
private fun ToggleRow(
    label: String,
    checked: Boolean,
    onChecked: (Boolean) -> Unit,
    description: String? = null,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(label, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary())
            description?.let { Text(it, fontSize = 12.sp, color = textSecondary()) }
        }
        Switch(checked = checked, onCheckedChange = onChecked)
    }
}

@Composable
private fun onboardingTitle(step: OnboardingStep): String = when (step) {
    OnboardingStep.Welcome -> L.text(R.string.mobile_onboarding_welcome_title)
    OnboardingStep.Topic -> L.text(R.string.mobile_onboarding_topic_title)
    OnboardingStep.Experience -> L.text(R.string.mobile_onboarding_experience_title)
    OnboardingStep.Diagnostic -> L.text(R.string.mobile_onboarding_diagnostic_title)
    OnboardingStep.Habits -> L.text(R.string.mobile_onboarding_habits_title)
    OnboardingStep.Consent -> L.text(R.string.mobile_onboarding_consent_title)
    OnboardingStep.Complete -> L.text(R.string.mobile_onboarding_complete_title)
}

private fun onboardingBack(step: OnboardingStep): OnboardingStep? = when (step) {
    OnboardingStep.Topic -> OnboardingStep.Welcome
    OnboardingStep.Experience -> OnboardingStep.Topic
    OnboardingStep.Diagnostic -> OnboardingStep.Experience
    OnboardingStep.Habits -> OnboardingStep.Diagnostic
    OnboardingStep.Consent -> OnboardingStep.Habits
    else -> null
}

private fun isoDate(millis: Long): String =
    Instant.ofEpochMilli(millis).atZone(ZoneId.systemDefault()).toLocalDate()
        .format(DateTimeFormatter.ISO_LOCAL_DATE)

private fun reminderTime(hour: Int, minute: Int): String =
    LocalTime.of(hour, minute).format(DateTimeFormatter.ofPattern("HH:mm"))
