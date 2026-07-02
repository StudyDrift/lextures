package com.lextures.android.features.insights

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Slider
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CoachingTip
import com.lextures.android.core.lms.CourseProgressSummary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.InsightsLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PostReflectionJournalBody
import com.lextures.android.core.lms.PutStudyGoalBody
import com.lextures.android.core.lms.ReflectionJournalEntry
import com.lextures.android.core.lms.StudyStats
import com.lextures.android.core.lms.StudyTimeAllocationRow
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer

@Composable
fun InsightsScreen(
    session: AuthSession,
    onOpenCourse: (CourseSummary) -> Unit,
    onOpenReview: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var stats by remember { mutableStateOf<StudyStats?>(null) }
    var journal by remember { mutableStateOf<List<ReflectionJournalEntry>>(emptyList()) }
    var tips by remember { mutableStateOf<List<CoachingTip>>(emptyList()) }
    var courseProgress by remember { mutableStateOf<List<CourseProgressSummary>>(emptyList()) }
    var studentCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var optedIn by remember { mutableStateOf(false) }
    var weeklyHours by remember { mutableFloatStateOf(10f) }
    var remindersEnabled by remember { mutableStateOf(false) }
    var journalDraft by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            val goal = runCatching { LmsApi.fetchStudyGoal(token) }.getOrNull()
            if (goal != null) {
                optedIn = goal.optedIn
                if (goal.weeklyHours > 0f) weeklyHours = goal.weeklyHours
            } else {
                runCatching { LmsApi.fetchStudyStats(token) }.getOrNull()?.let { probe ->
                    optedIn = probe.optedIn
                    probe.weeklyGoalHours?.takeIf { it > 0f }?.let { weeklyHours = it }
                }
            }

            if (!optedIn) {
                stats = null
                journal = emptyList()
                tips = emptyList()
                courseProgress = emptyList()
                return
            }

            val statsResult = offline.cachedFetch(
                key = OfflineCacheKey.studyStats(),
                accessToken = token,
                serializer = serializer<StudyStats>(),
            ) { LmsApi.fetchStudyStats(token) }
            val journalResult = offline.cachedFetch(
                key = OfflineCacheKey.reflectionJournal(),
                accessToken = token,
                serializer = serializer<List<ReflectionJournalEntry>>(),
            ) { LmsApi.fetchReflectionJournal(token) }
            val tipsResult = offline.cachedFetch(
                key = OfflineCacheKey.coachingTips(),
                accessToken = token,
                serializer = serializer<com.lextures.android.core.lms.CoachingTipsResponse>(),
            ) { LmsApi.fetchCoachingTips(token) }

            stats = statsResult.first
            journal = journalResult.first
            tips = tipsResult.first.history
            stats?.weeklyGoalHours?.takeIf { it > 0f }?.let { weeklyHours = it }

            val courses = LmsApi.fetchCourses(token)
            studentCourses = courses.filter { it.viewerIsStudent }
            courseProgress = studentCourses.take(8).mapNotNull { course ->
                val snapshot = runCatching {
                    LmsApi.fetchModulesProgress(course.courseCode, token)
                }.getOrNull() ?: return@mapNotNull null
                CourseProgressSummary(
                    courseCode = course.courseCode,
                    title = course.displayTitle,
                    percentComplete = InsightsLogic.moduleCompletionPercent(snapshot),
                )
            }.sortedByDescending { it.percentComplete }

            runCatching { LmsApi.fetchReminderConfig(token) }.getOrNull()?.let {
                remindersEnabled = it.enabled
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_insights_error_load)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        if (loading && stats == null && optedIn) {
            CircularProgressIndicator(modifier = Modifier.padding(24.dp))
        } else {
            errorMessage?.let { LmsErrorBanner(message = it) }

            LmsCard {
                Text(
                    text = L.text(R.string.mobile_insights_goalsTitle),
                    fontSize = 17.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = L.text(R.string.mobile_insights_optIn),
                        fontSize = 14.sp,
                        color = textPrimary(),
                    )
                    Switch(checked = optedIn, onCheckedChange = { optedIn = it })
                }
                if (optedIn) {
                    Text(
                        text = L.format(R.string.mobile_insights_weeklyGoal, weeklyHours.toDouble()),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    Slider(
                        value = weeklyHours,
                        onValueChange = { weeklyHours = it },
                        valueRange = 0f..40f,
                        steps = 79,
                    )
                }
                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        scope.launch {
                            saving = true
                            try {
                                LmsApi.putStudyGoal(
                                    PutStudyGoalBody(weeklyHours = weeklyHours, optedIn = optedIn),
                                    token,
                                )
                                load(token)
                            } catch (_: Exception) {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_insights_error_saveGoals)
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                ) {
                    Text(
                        if (saving) {
                            L.text(R.string.mobile_insights_saving)
                        } else {
                            L.text(R.string.mobile_insights_saveGoals)
                        },
                    )
                }
            }

            if (optedIn) {
                stats?.let { weekStats ->
                    WeekStatsCard(weekStats)
                    if (courseProgress.isNotEmpty()) {
                        CourseProgressCard(
                            rows = courseProgress,
                            onOpen = { code ->
                                studentCourses.firstOrNull { it.courseCode == code }?.let(onOpenCourse)
                            },
                        )
                    }
                    if (weekStats.timeAllocation.isNotEmpty()) {
                        AllocationCard(weekStats.timeAllocation)
                    }
                }

                JournalCard(
                    draft = journalDraft,
                    onDraftChange = { journalDraft = it },
                    entries = journal,
                    onAdd = {
                        val token = accessToken ?: return@JournalCard
                        if (!InsightsLogic.journalEntryValid(journalDraft)) return@JournalCard
                        val text = journalDraft.trim()
                        journalDraft = ""
                        scope.launch {
                            try {
                                if (isOnline) {
                                    LmsApi.createReflectionJournalEntry(
                                        PostReflectionJournalBody(entryText = text),
                                        token,
                                    )
                                } else {
                                    offline.enqueueMutation(
                                        method = "POST",
                                        path = "/api/v1/me/reflection-journal",
                                        bodyJson = Json.encodeToString(
                                            PostReflectionJournalBody.serializer(),
                                            PostReflectionJournalBody(entryText = text),
                                        ),
                                        label = L.text(context, localePrefs, R.string.mobile_insights_journal_addLabel),
                                        accessToken = token,
                                    )
                                }
                                load(token)
                            } catch (_: Exception) {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_insights_error_journal)
                            }
                        }
                    },
                    onDelete = { id ->
                        val token = accessToken ?: return@JournalCard
                        scope.launch {
                            runCatching { LmsApi.deleteReflectionJournalEntry(id, token) }
                            load(token)
                        }
                    },
                )

                if (tips.isNotEmpty()) {
                    TipsCard(
                        tips = tips,
                        onRate = { id, rating ->
                            val token = accessToken ?: return@TipsCard
                            scope.launch {
                                runCatching { LmsApi.rateCoachingTip(id, rating, token) }
                                load(token)
                            }
                        },
                    )
                }

                LmsCard {
                    Text(
                        text = L.text(R.string.mobile_insights_remindersTitle),
                        fontSize = 17.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(
                            text = L.text(R.string.mobile_insights_remindersToggle),
                            fontSize = 14.sp,
                            color = textPrimary(),
                        )
                        Switch(
                            checked = remindersEnabled,
                            onCheckedChange = { enabled ->
                                val token = accessToken ?: return@Switch
                                scope.launch {
                                    runCatching { LmsApi.patchReminderConfig(enabled, token) }
                                        .onSuccess { remindersEnabled = it.enabled }
                                        .onFailure {
                                            errorMessage = L.text(
                                                context,
                                                localePrefs,
                                                R.string.mobile_insights_error_reminders,
                                            )
                                        }
                                }
                            },
                        )
                    }
                }

                LmsCard {
                    Text(
                        text = L.text(R.string.mobile_insights_actionsTitle),
                        fontSize = 17.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    TextButton(onClick = onOpenReview) {
                        Text(L.text(R.string.mobile_insights_openReview))
                    }
                }
            }
        }
    }
}

@Composable
private fun WeekStatsCard(stats: StudyStats) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_insights_thisWeek),
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        if (stats.loginStreakDays > 0) {
            Text(
                text = L.plural(R.plurals.mobile_insights_streak, stats.loginStreakDays),
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        } else {
            Text(
                text = L.text(R.string.mobile_insights_streakEmpty),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
        val hours = InsightsLogic.hoursFromSeconds(stats.timeOnTaskSecondsThisWeek)
        Text(
            text = L.format(R.string.mobile_insights_timeOnTask, InsightsLogic.formatHours(hours)),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        InsightsLogic.goalProgressPercent(stats.goalProgressHours, stats.weeklyGoalHours)?.let { pct ->
            LinearProgressIndicator(progress = { pct / 100f }, modifier = Modifier.fillMaxWidth())
        }
        if (stats.lowStudyEfficiency) {
            Text(
                text = L.text(R.string.mobile_insights_lowEfficiency),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun CourseProgressCard(
    rows: List<CourseProgressSummary>,
    onOpen: (String) -> Unit,
) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_insights_courseProgress),
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        rows.forEach { row ->
            Column(modifier = Modifier.padding(vertical = 4.dp)) {
                Row(modifier = Modifier.fillMaxWidth()) {
                    Text(
                        text = row.title,
                        fontSize = 13.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = "${row.percentComplete}%",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                LinearProgressIndicator(
                    progress = { row.percentComplete / 100f },
                    modifier = Modifier.fillMaxWidth(),
                )
                TextButton(onClick = { onOpen(row.courseCode) }) {
                    Text(L.text(R.string.mobile_insights_openCourse))
                }
            }
        }
    }
}

@Composable
private fun AllocationCard(rows: List<StudyTimeAllocationRow>) {
    val maxMinutes = InsightsLogic.maxAllocationMinutes(rows)
    LmsCard {
        Text(
            text = L.text(R.string.mobile_insights_timeAllocation),
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        rows.forEach { row ->
            Column(modifier = Modifier.padding(vertical = 4.dp)) {
                Row(modifier = Modifier.fillMaxWidth()) {
                    Text(
                        text = row.moduleTitle,
                        fontSize = 12.sp,
                        color = textPrimary(),
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = L.format(R.string.mobile_insights_minutes, row.minutes.toInt()),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                LinearProgressIndicator(
                    progress = {
                        (InsightsLogic.barWidthPercent(row.minutes, maxMinutes) / 100f).toFloat()
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}

@Composable
private fun JournalCard(
    draft: String,
    onDraftChange: (String) -> Unit,
    entries: List<ReflectionJournalEntry>,
    onAdd: () -> Unit,
    onDelete: (String) -> Unit,
) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_insights_journalTitle),
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        Text(
            text = L.text(R.string.mobile_insights_journalHint),
            fontSize = 11.sp,
            color = textSecondary(),
        )
        OutlinedTextField(
            value = draft,
            onValueChange = onDraftChange,
            modifier = Modifier.fillMaxWidth(),
            placeholder = {
                Text(L.text(R.string.mobile_insights_journalPlaceholder))
            },
            minLines = 3,
        )
        Button(onClick = onAdd, enabled = InsightsLogic.journalEntryValid(draft)) {
            Text(L.text(R.string.mobile_insights_journalAdd))
        }
        entries.forEach { entry ->
            Text(
                text = entry.createdAt.take(10),
                fontSize = 11.sp,
                color = textSecondary(),
            )
            Text(text = entry.entryText, fontSize = 14.sp, color = textPrimary())
            TextButton(onClick = { onDelete(entry.id) }) {
                Text(L.text(R.string.mobile_insights_journalDelete))
            }
        }
    }
}

@Composable
private fun TipsCard(
    tips: List<CoachingTip>,
    onRate: (String, Int) -> Unit,
) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_insights_coachingTitle),
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        tips.forEach { tip ->
            Text(
                text = L.format(R.string.mobile_insights_coachingWeek, tip.weekOf),
                fontSize = 11.sp,
                color = textSecondary(),
            )
            Text(text = tip.tipText, fontSize = 14.sp, color = textPrimary())
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                TextButton(onClick = { onRate(tip.id, 1) }) {
                    Text(L.text(R.string.mobile_insights_coachingHelpful))
                }
                TextButton(onClick = { onRate(tip.id, -1) }) {
                    Text(L.text(R.string.mobile_insights_coachingNotHelpful))
                }
            }
        }
    }
}