package com.lextures.android.features.paths

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Button
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
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
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LearnerRecommendationItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PathProgress
import com.lextures.android.core.lms.PathsLogic
import com.lextures.android.core.lms.RecommendationEventBody
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsSectionHeader

data class DashboardWhatsNext(
    val course: CourseSummary,
    val primary: LearnerRecommendationItem?,
    val chips: List<LearnerRecommendationItem>,
    val degraded: Boolean,
)

@Composable
fun DashboardStudySection(
    session: AuthSession,
    studentCourses: List<CourseSummary>,
    learningPathsEnabled: Boolean,
    onOpenReview: () -> Unit,
    onOpenRecommendation: (CourseSummary, CourseStructureItem) -> Unit,
    onOpenPaths: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()

    var myPaths by remember { mutableStateOf<List<PathProgress>>(emptyList()) }
    var whatsNext by remember { mutableStateOf<DashboardWhatsNext?>(null) }

    LaunchedEffect(accessToken, studentCourses.map { it.id }.joinToString(), learningPathsEnabled) {
        val token = accessToken ?: return@LaunchedEffect
        if (learningPathsEnabled) {
            myPaths = runCatching { LmsApi.fetchMyPaths(token) }.getOrDefault(emptyList())
        } else {
            myPaths = emptyList()
        }

        val learnerId = NotebookStore.jwtSubject(token) ?: return@LaunchedEffect
        val course = studentCourses.firstOrNull { it.viewerIsStudent } ?: studentCourses.firstOrNull()
            ?: return@LaunchedEffect
        try {
            val responses = coroutineScope {
                PathsLogic.recommendationSurfaces.map { surface ->
                    async {
                        LmsApi.fetchLearnerRecommendations(learnerId, course.id, surface, token, limit = 4)
                    }
                }.awaitAll()
            }
            val merged = PathsLogic.mergeRecommendations(responses)
            whatsNext = DashboardWhatsNext(course, merged.primary, merged.chips, merged.degraded)
            merged.primary?.let { primary ->
                runCatching {
                    LmsApi.postRecommendationEvent(
                        RecommendationEventBody(
                            courseId = course.id,
                            itemId = primary.itemId,
                            surface = primary.surface,
                            eventType = "impression",
                            rank = 0,
                        ),
                        token,
                    )
                }
            }
        } catch (_: Exception) {
            whatsNext = null
        }
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
        if (myPaths.isNotEmpty()) {
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        LmsSectionHeader(L.text(context, localePrefs, R.string.mobile_paths_dashboardTitle))
                        TextButton(onClick = onOpenPaths) {
                            Text(L.text(context, localePrefs, R.string.mobile_paths_viewAll))
                        }
                    }
                    myPaths.take(3).forEach { path ->
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text(path.pathTitle, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                Text(path.progressLabel, fontSize = 11.sp, color = textSecondary())
                            }
                            LinearProgressIndicator(progress = { path.percent / 100f }, modifier = Modifier.fillMaxWidth())
                        }
                    }
                }
            }
        }

        whatsNext?.let { bundle ->
            LmsCard(accent = accentColor()) {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(bundle.course.displayTitle, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textSecondary())
                    bundle.primary?.let { primary ->
                        Text(primary.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(primary.reason, fontSize = 12.sp, color = textSecondary())
                        if (bundle.degraded) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_paths_recommendationsDegraded),
                                fontSize = 11.sp,
                                color = LexturesColors.Amber,
                            )
                        }
                        Button(
                            onClick = {
                                if (primary.itemType == "review_card") {
                                    onOpenReview()
                                } else {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        val structure = runCatching {
                                            LmsApi.fetchCourseStructure(bundle.course.courseCode, token)
                                        }.getOrDefault(emptyList())
                                        PathsLogic.structureItem(primary, structure)?.let {
                                            onOpenRecommendation(bundle.course, it)
                                        }
                                    }
                                }
                                scope.launch {
                                    val token = accessToken ?: return@launch
                                    runCatching {
                                        LmsApi.postRecommendationEvent(
                                            RecommendationEventBody(
                                                courseId = bundle.course.id,
                                                itemId = primary.itemId,
                                                surface = primary.surface,
                                                eventType = "click",
                                                rank = 0,
                                            ),
                                            token,
                                        )
                                    }
                                }
                            },
                        ) {
                            Text(L.text(context, localePrefs, R.string.mobile_paths_go))
                        }
                        if (bundle.chips.isNotEmpty()) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .horizontalScroll(rememberScrollState()),
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                bundle.chips.forEachIndexed { index, chip ->
                                    TextButton(
                                        onClick = {
                                            if (chip.itemType == "review_card") {
                                                onOpenReview()
                                            } else {
                                                scope.launch {
                                                    val token = accessToken ?: return@launch
                                                    val structure = runCatching {
                                                        LmsApi.fetchCourseStructure(bundle.course.courseCode, token)
                                                    }.getOrDefault(emptyList())
                                                    PathsLogic.structureItem(chip, structure)?.let {
                                                        onOpenRecommendation(bundle.course, it)
                                                    }
                                                }
                                            }
                                            scope.launch {
                                                val token = accessToken ?: return@launch
                                                runCatching {
                                                    LmsApi.postRecommendationEvent(
                                                        RecommendationEventBody(
                                                            courseId = bundle.course.id,
                                                            itemId = chip.itemId,
                                                            surface = chip.surface,
                                                            eventType = "click",
                                                            rank = index + 1,
                                                        ),
                                                        token,
                                                    )
                                                }
                                            }
                                        },
                                    ) {
                                        Text(chip.title, fontSize = 11.sp)
                                    }
                                }
                            }
                        }
                    } ?: Text(
                        context.getString(R.string.mobile_paths_caughtUpInCourse, bundle.course.displayTitle),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}