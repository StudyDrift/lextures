package com.lextures.android.features.paths

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PathCourseProgress
import com.lextures.android.core.lms.PathProgress
import com.lextures.android.core.lms.PathsLogic
import kotlinx.serialization.serializer
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader

@Composable
fun PathRunnerScreen(
    session: AuthSession,
    initialPath: PathProgress,
    onOpenCourse: (CourseSummary) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val offline = remember { OfflineService.get(context) }

    var progress by remember { mutableStateOf(initialPath) }
    var coursesByCode by remember { mutableStateOf<Map<String, CourseSummary>>(emptyMap()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken, progress.pathId) {
        val token = accessToken ?: return@LaunchedEffect
        try {
            progress = offline.cachedFetch(
                key = OfflineCacheKey.pathProgress(progress.pathId),
                accessToken = token,
                serializer = serializer<PathProgress>(),
            ) { LmsApi.fetchPathProgress(progress.pathId, token) }.first
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_paths_error_load)
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        TextButtonBack(onBack = onBack, label = L.text(context, localePrefs, R.string.mobile_ia_close))
        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                LinearProgressIndicator(progress = { progress.percent / 100f }, modifier = Modifier.fillMaxWidth())
                Text(progress.progressLabel, fontSize = 14.sp, color = textSecondary())
                if (progress.justCompleted) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_paths_completedBanner),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = LexturesColors.Amber,
                    )
                }
            }
        }
        errorMessage?.let { LmsErrorBanner(it) }
        LmsSectionHeader(L.text(context, localePrefs, R.string.mobile_paths_steps))
        LazyColumn(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            items(PathsLogic.sortedCourses(progress.courses), key = { it.courseId }) { course ->
                StepRow(
                    course = course,
                    isNext = PathsLogic.nextCourse(progress)?.courseId == course.courseId,
                    locked = PathsLogic.isLocked(course),
                    onOpen = {
                        val token = accessToken ?: return@StepRow
                        scope.launch {
                            val cached = coursesByCode[course.courseCode]
                            if (cached != null) {
                                onOpenCourse(cached)
                                return@launch
                            }
                            runCatching {
                                val summary = LmsApi.fetchCourse(course.courseCode, token)
                                coursesByCode = coursesByCode + (course.courseCode to summary)
                                onOpenCourse(summary)
                            }.onFailure {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_paths_error_openCourse)
                            }
                        }
                    },
                )
            }
        }
    }
}

@Composable
private fun StepRow(
    course: PathCourseProgress,
    isNext: Boolean,
    locked: Boolean,
    onOpen: () -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LmsCard(accent = if (isNext) accentColor() else null) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = when {
                    course.isCompleted -> Icons.Default.CheckCircle
                    locked -> Icons.Default.Lock
                    else -> Icons.Default.RadioButtonUnchecked
                },
                contentDescription = null,
                tint = when {
                    course.isCompleted -> accentColor()
                    locked -> textSecondary()
                    else -> accentColor()
                },
            )
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(course.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                Text(course.courseCode.uppercase(), fontSize = 11.sp, color = textSecondary())
                when {
                    isNext -> Text(
                        L.text(context, localePrefs, R.string.mobile_paths_nextStep),
                        fontSize = 11.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                    )
                    locked -> Text(
                        L.text(context, localePrefs, R.string.mobile_paths_locked),
                        fontSize = 11.sp,
                        color = textSecondary(),
                    )
                }
            }
            if (!locked) {
                Button(onClick = onOpen) {
                    Text(
                        L.text(
                            context,
                            localePrefs,
                            if (isNext) R.string.mobile_paths_continue else R.string.mobile_paths_openCourse,
                        ),
                    )
                }
            }
        }
    }
}

@Composable
private fun TextButtonBack(onBack: () -> Unit, label: String) {
    androidx.compose.material3.TextButton(onClick = onBack) {
        Text(label)
    }
}