package com.lextures.android.features.gamification

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.EmojiEvents
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
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
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CredentialsLogic
import com.lextures.android.core.lms.GamificationLogic
import com.lextures.android.core.lms.GamificationProfile
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GamificationScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val offline = remember { OfflineService.get(context) }
    val accessToken = session.accessToken.value

    var profile by remember { mutableStateOf<GamificationProfile?>(null) }
    var studentCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var selectedCourseCode by remember { mutableStateOf<String?>(null) }
    var leaderboard by remember { mutableStateOf<com.lextures.android.core.lms.CourseLeaderboardResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var leaderboardLoading by remember { mutableStateOf(false) }
    var freezing by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var courseMenuExpanded by remember { mutableStateOf(false) }

    suspend fun loadLeaderboard(courseCode: String) {
        val token = accessToken ?: return
        leaderboardLoading = true
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.gamificationLeaderboard(courseCode),
                accessToken = token,
                serializer = serializer<com.lextures.android.core.lms.CourseLeaderboardResponse>(),
            ) {
                LmsApi.fetchCourseLeaderboard(courseCode, token)
            }
            leaderboard = result.first
        } catch (_: Exception) {
            leaderboard = null
        } finally {
            leaderboardLoading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = profile == null
        errorMessage = null
        try {
            val profileResult = offline.cachedFetch(
                key = OfflineCacheKey.gamificationProfile(),
                accessToken = token,
                serializer = serializer<GamificationProfile>(),
            ) {
                LmsApi.fetchGamificationProfile(token)
            }
            profile = profileResult.first
            studentCourses = LmsApi.fetchCourses(token).filter { it.viewerIsStudent }
            if (selectedCourseCode == null) {
                selectedCourseCode = studentCourses.firstOrNull()?.courseCode
            }
            selectedCourseCode?.let { loadLeaderboard(it) }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_gamification_loadError)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(selectedCourseCode, accessToken) {
        selectedCourseCode?.let { loadLeaderboard(it) }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        when {
            loading -> LmsSkeletonList(count = 4)
            errorMessage != null && profile == null -> LmsEmptyState(
                icon = Icons.Default.EmojiEvents,
                title = L.text(context, localePrefs, R.string.mobile_gamification_errorTitle),
                message = errorMessage!!,
            )
            profile != null -> {
                val current = profile!!
                LmsSectionHeader(
                    title = L.text(context, localePrefs, R.string.mobile_gamification_statsTitle),
                    modifier = Modifier.padding(bottom = 8.dp),
                )
                LmsCard {
                    statRow(
                        context.getString(R.string.mobile_gamification_currentStreak),
                        context.getString(R.string.mobile_gamification_days, current.currentStreak),
                    )
                    statRow(
                        context.getString(R.string.mobile_gamification_longestStreak),
                        context.getString(R.string.mobile_gamification_days, current.longestStreak),
                    )
                    statRow(context.getString(R.string.mobile_gamification_totalXp), current.xpTotal.toString())
                    statRow(context.getString(R.string.mobile_gamification_level), current.level.toString())
                    LinearProgressIndicator(
                        progress = { (current.levelProgressPct / 100.0).toFloat().coerceIn(0f, 1f) },
                        modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                    )
                    if (GamificationLogic.canUseStreakFreeze(current)) {
                        Button(
                            onClick = {
                                val token = accessToken ?: return@Button
                                freezing = true
                                scope.launch {
                                    try {
                                        profile = LmsApi.freezeGamificationStreak(token)
                                    } catch (_: Exception) {
                                        errorMessage = L.text(context, localePrefs, R.string.mobile_gamification_freezeError)
                                    } finally {
                                        freezing = false
                                    }
                                }
                            },
                            enabled = !freezing,
                        ) {
                            Text(L.text(context, localePrefs, R.string.mobile_gamification_useFreeze))
                        }
                    }
                }

                LmsSectionHeader(
                    title = L.text(context, localePrefs, R.string.mobile_gamification_badgesTitle),
                    modifier = Modifier.padding(top = 16.dp, bottom = 8.dp),
                )
                LmsCard {
                    val badges = current.badges.orEmpty()
                    if (badges.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_gamification_noBadges),
                            color = textSecondary(),
                        )
                    } else {
                        badges.forEach { badge ->
                            Text(
                                GamificationLogic.badgeLabel(badge.badgeType),
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            Text(
                                CredentialsLogic.issuedDateLabel(badge.awardedAt),
                                fontSize = 12.sp,
                                color = textSecondary(),
                                modifier = Modifier.padding(bottom = 8.dp),
                            )
                        }
                    }
                }

                LmsSectionHeader(
                    title = L.text(context, localePrefs, R.string.mobile_gamification_leaderboardTitle),
                    modifier = Modifier.padding(top = 16.dp, bottom = 8.dp),
                )
                LmsCard {
                    if (!GamificationLogic.shouldShowLeaderboard(current)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_gamification_leaderboardOptOut),
                            color = textSecondary(),
                        )
                    } else if (studentCourses.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_gamification_noCoursesForLeaderboard),
                            color = textSecondary(),
                        )
                    } else {
                        val selectedCourse = studentCourses.firstOrNull { it.courseCode == selectedCourseCode }
                            ?: studentCourses.first()
                        ExposedDropdownMenuBox(
                            expanded = courseMenuExpanded,
                            onExpandedChange = { courseMenuExpanded = it },
                        ) {
                            OutlinedTextField(
                                value = selectedCourse.displayTitle,
                                onValueChange = {},
                                readOnly = true,
                                modifier = Modifier.menuAnchor().fillMaxWidth(),
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = courseMenuExpanded) },
                            )
                            ExposedDropdownMenu(
                                expanded = courseMenuExpanded,
                                onDismissRequest = { courseMenuExpanded = false },
                            ) {
                                studentCourses.forEach { course ->
                                    DropdownMenuItem(
                                        text = { Text(course.displayTitle) },
                                        onClick = {
                                            selectedCourseCode = course.courseCode
                                            courseMenuExpanded = false
                                        },
                                    )
                                }
                            }
                        }
                        if (leaderboardLoading) {
                            Text(L.text(context, localePrefs, R.string.mobile_gamification_loadingLeaderboard))
                        } else {
                            leaderboard?.topEntries.orEmpty().forEach { entry ->
                                Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                                    Text("#${entry.rank}", modifier = Modifier.padding(end = 8.dp))
                                    Text(entry.displayName, modifier = Modifier.weight(1f))
                                    Text(context.getString(R.string.mobile_gamification_xpEarned, entry.xpEarned))
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun statRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Text(label, color = textSecondary(), modifier = Modifier.weight(1f))
        Text(value, fontWeight = FontWeight.SemiBold, color = textPrimary())
    }
}