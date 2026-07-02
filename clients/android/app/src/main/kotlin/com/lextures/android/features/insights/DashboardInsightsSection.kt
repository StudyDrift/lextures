package com.lextures.android.features.insights

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
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
import com.lextures.android.core.lms.CoachingTipsResponse
import com.lextures.android.core.lms.InsightsLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.StudyStats
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import kotlinx.serialization.serializer

@Composable
fun DashboardInsightsSection(
    session: AuthSession,
    onOpenInsights: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }

    var stats by remember { mutableStateOf<StudyStats?>(null) }
    var latestTip by remember { mutableStateOf<CoachingTip?>(null) }
    var tipDismissed by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        try {
            val loadedStats = offline.cachedFetch(
                key = OfflineCacheKey.studyStats(),
                accessToken = token,
                serializer = serializer<StudyStats>(),
            ) { LmsApi.fetchStudyStats(token) }.first
            stats = loadedStats.takeIf { it.optedIn }
            latestTip = offline.cachedFetch(
                key = OfflineCacheKey.coachingTips(),
                accessToken = token,
                serializer = serializer<CoachingTipsResponse>(),
            ) { LmsApi.fetchCoachingTips(token) }.first.let { InsightsLogic.latestCoachingTip(it) }
        } catch (_: Exception) {
            stats = null
            latestTip = null
        }
    }

    val weekStats = stats ?: return
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (latestTip != null && !tipDismissed) {
            LmsCard {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(
                            text = L.text(R.string.mobile_insights_dashboardCoachingTitle),
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            text = latestTip!!.tipText,
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        TextButton(onClick = onOpenInsights) {
                            Text(L.text(R.string.mobile_insights_dashboardOpen))
                        }
                    }
                    TextButton(onClick = { tipDismissed = true }) {
                        Text("×")
                    }
                }
            }
        }

        LmsCard {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(
                    text = L.text(R.string.mobile_insights_dashboardTitle),
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                TextButton(onClick = onOpenInsights) {
                    Text(L.text(R.string.mobile_insights_dashboardOpen))
                }
            }
            if (weekStats.loginStreakDays > 0) {
                Text(
                    text = L.plural(R.plurals.mobile_insights_streak, weekStats.loginStreakDays),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
            }
            val hours = InsightsLogic.hoursFromSeconds(weekStats.timeOnTaskSecondsThisWeek)
            Text(
                text = L.format(R.string.mobile_insights_timeOnTask, InsightsLogic.formatHours(hours)),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            InsightsLogic.goalProgressPercent(weekStats.goalProgressHours, weekStats.weeklyGoalHours)?.let { pct ->
                LinearProgressIndicator(progress = { pct / 100f }, modifier = Modifier.fillMaxWidth())
            }
        }
    }
}