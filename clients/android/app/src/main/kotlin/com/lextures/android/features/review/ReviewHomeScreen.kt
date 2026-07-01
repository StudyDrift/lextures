package com.lextures.android.features.review

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
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
import com.lextures.android.core.lms.LearnerRecommendationItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ReviewLogic
import com.lextures.android.core.lms.ReviewQueueResponse
import com.lextures.android.core.lms.ReviewStats
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReviewHomeScreen(
    session: AuthSession,
    shell: HomeShellState?,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }

    var stats by remember { mutableStateOf<ReviewStats?>(null) }
    var queue by remember { mutableStateOf<ReviewQueueResponse?>(null) }
    var recommendations by remember { mutableStateOf<List<LearnerRecommendationItem>>(emptyList()) }
    var selectedCourseCode by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var showSession by remember { mutableStateOf(false) }
    var reminderEnabled by remember {
        mutableStateOf(ReviewReminderScheduler.isEnabled(context))
    }

    val userId = remember(shell?.profile?.id, accessToken) {
        shell?.profile?.id ?: NotebookStore.jwtSubject(accessToken)
    }

    val filteredItems = remember(queue, selectedCourseCode) {
        ReviewLogic.filterQueue(queue?.items.orEmpty(), selectedCourseCode)
    }
    val dueCount = remember(stats, queue, selectedCourseCode, filteredItems) {
        if (!selectedCourseCode.isNullOrEmpty()) filteredItems.size
        else stats?.dueToday ?: queue?.totalDue ?: filteredItems.size
    }
    val courseFilters = remember(queue) {
        ReviewLogic.courseFilters(queue?.items.orEmpty())
    }

    suspend fun load(token: String, learnerId: String) {
        loading = true
        errorMessage = null
        try {
            coroutineScope {
                val statsDeferred = async {
                    offline.cachedFetch(
                        key = OfflineCacheKey.reviewStats(),
                        accessToken = token,
                        serializer = serializer<ReviewStats>(),
                    ) { LmsApi.fetchLearnerReviewStats(learnerId, token) }.first
                }
                val queueDeferred = async {
                    offline.cachedFetch(
                        key = OfflineCacheKey.reviewQueue(),
                        accessToken = token,
                        serializer = serializer<ReviewQueueResponse>(),
                    ) {
                        LmsApi.fetchLearnerReviewQueue(learnerId, token, ReviewLogic.PREFETCH_LIMIT)
                    }.first
                }
                stats = statsDeferred.await()
                queue = queueDeferred.await()
            }
            ReviewReminderScheduler.reschedule(context, localePrefs, stats?.dueToday ?: queue?.totalDue ?: 0)
            val courseId = queue?.items?.firstOrNull()?.courseId
            recommendations = if (courseId != null) {
                runCatching {
                    LmsApi.fetchLearnerRecommendations(learnerId, courseId, "review", token).recommendations
                }.getOrDefault(emptyList())
            } else {
                emptyList()
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_review_error_load)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, userId) {
        val token = accessToken ?: return@LaunchedEffect
        val learnerId = userId ?: return@LaunchedEffect
        load(token, learnerId)
    }

    if (showSession) {
        ReviewSessionScreen(
            session = session,
            shell = shell,
            initialQueue = filteredItems,
            totalDue = dueCount,
            initialStreak = stats?.streak ?: 0,
            onBack = { showSession = false },
            onFinished = {
                showSession = false
                val token = accessToken
                val learnerId = userId
                if (token != null && learnerId != null) {
                    scope.launch { load(token, learnerId) }
                }
            },
            modifier = modifier,
        )
        return
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        errorMessage?.let { LmsErrorBanner(message = it) }

        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (loading && queue == null) {
                    CircularProgressIndicator()
                } else {
                    Text(
                        text = context.resources.getQuantityString(
                            R.plurals.mobile_review_dueCount,
                            dueCount,
                            dueCount,
                        ),
                        fontSize = 28.sp,
                        fontWeight = FontWeight.Bold,
                        color = textPrimary(),
                    )
                    val streak = stats?.streak ?: 0
                    if (streak > 0) {
                        Text(
                            text = context.resources.getQuantityString(
                                R.plurals.mobile_review_streak,
                                streak,
                                streak,
                            ),
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textSecondary(),
                        )
                    } else {
                        Text(
                            text = L.text(context, localePrefs, R.string.mobile_review_subtitle),
                            fontSize = 14.sp,
                            color = textSecondary(),
                        )
                    }
                }
            }
        }

        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = L.text(context, localePrefs, R.string.mobile_review_reminder_label),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            text = L.text(context, localePrefs, R.string.mobile_review_reminder_hint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                    Switch(
                        checked = reminderEnabled,
                        onCheckedChange = { enabled ->
                            reminderEnabled = enabled
                            ReviewReminderScheduler.setEnabled(context, enabled)
                            ReviewReminderScheduler.reschedule(context, localePrefs, dueCount)
                        },
                    )
                }
                if (reminderEnabled) {
                    val pickerState = rememberTimePickerState(
                        initialHour = ReviewReminderScheduler.reminderHour(context),
                        initialMinute = ReviewReminderScheduler.reminderMinute(context),
                    )
                    TimePicker(state = pickerState)
                    LaunchedEffect(pickerState.hour, pickerState.minute) {
                        ReviewReminderScheduler.setReminderTime(context, pickerState.hour, pickerState.minute)
                        ReviewReminderScheduler.reschedule(context, localePrefs, dueCount)
                    }
                }
            }
        }

        if (courseFilters.isNotEmpty()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                FilterChip(
                    selected = selectedCourseCode == null,
                    onClick = { selectedCourseCode = null },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_review_filter_all)) },
                )
                courseFilters.forEach { filter ->
                    FilterChip(
                        selected = selectedCourseCode == filter.courseCode,
                        onClick = { selectedCourseCode = filter.courseCode },
                        label = { Text(filter.courseTitle) },
                    )
                }
            }
        }

        if (recommendations.isNotEmpty()) {
            LmsSectionHeader(title = L.text(context, localePrefs, R.string.mobile_review_recommendations))
            recommendations.take(3).forEach { item ->
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(item.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(item.reason, fontSize = 12.sp, color = textSecondary())
                    }
                }
            }
        }

        if (filteredItems.isEmpty() && !loading) {
            LmsEmptyState(
                icon = Icons.Default.CheckCircle,
                title = L.text(context, localePrefs, R.string.mobile_review_caughtUpTitle),
                message = L.text(context, localePrefs, R.string.mobile_review_caughtUpMessage),
            )
        } else {
            Button(
                onClick = { showSession = true },
                enabled = filteredItems.isNotEmpty() && !loading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    L.text(
                        context,
                        localePrefs,
                        if (dueCount > 0) R.string.mobile_review_start else R.string.mobile_review_open,
                    ),
                )
            }
        }
    }
}
