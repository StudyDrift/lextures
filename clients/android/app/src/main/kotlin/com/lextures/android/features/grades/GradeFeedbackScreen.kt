package com.lextures.android.features.grades

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.Description
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import androidx.compose.foundation.background
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.GradeColumn
import com.lextures.android.core.lms.GradeComment
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.RubricCriterion
import com.lextures.android.core.lms.RubricDefinition
import com.lextures.android.core.lms.RubricLevel
import com.lextures.android.core.lms.SubmissionAnnotation
import com.lextures.android.core.lms.SubmissionFeedbackMedia
import com.lextures.android.core.lms.SubmissionGrade
import com.lextures.android.features.files.AnnotatedFilePreviewScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState

/** Full student feedback: rubric, comments, annotated file, a/v playback (M6.1). */
@Composable
fun GradeFeedbackScreen(
    session: AuthSession,
    course: CourseSummary,
    column: GradeColumn,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    var grade by remember { mutableStateOf<SubmissionGrade?>(null) }
    var submission by remember { mutableStateOf<AssignmentSubmission?>(null) }
    var annotations by remember { mutableStateOf<List<SubmissionAnnotation>>(emptyList()) }
    var feedbackMedia by remember { mutableStateOf<List<SubmissionFeedbackMedia>>(emptyList()) }
    var playbackPaths by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var feedbackMediaEnabled by remember { mutableStateOf(true) }
    var openPreview by remember { mutableStateOf<FilePreviewTarget?>(null) }

    LaunchedEffect(accessToken, column.id) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            feedbackMediaEnabled = runCatching {
                LmsApi.fetchPlatformFeatures(token).feedbackMediaEnabled
            }.getOrNull() != false

            if (column.kind != "assignment") {
                loading = false
                return@LaunchedEffect
            }

            submission = LmsApi.fetchMySubmission(course.courseCode, column.id, token)
            val sub = submission ?: run {
                loading = false
                return@LaunchedEffect
            }

            grade = LmsApi.fetchSubmissionGrade(course.courseCode, column.id, sub.id, token)
            if (grade?.posted == false) {
                errorMessage = "This grade has not been released yet."
                grade = null
                loading = false
                return@LaunchedEffect
            }

            annotations = runCatching {
                LmsApi.fetchSubmissionAnnotations(course.courseCode, column.id, sub.id, token)
            }.getOrDefault(emptyList())

            if (feedbackMediaEnabled) {
                feedbackMedia = runCatching {
                    LmsApi.fetchSubmissionFeedbackMedia(course.courseCode, column.id, sub.id, token)
                }.getOrDefault(emptyList())
                playbackPaths = feedbackMedia.associate { media ->
                    val path = runCatching {
                        LmsApi.fetchFeedbackPlaybackInfo(
                            course.courseCode, column.id, sub.id, media.id, token,
                        ).contentPath
                    }.getOrNull().orEmpty()
                    media.id to path
                }.filterValues { it.isNotEmpty() }
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    openPreview?.let { target ->
        AnnotatedFilePreviewScreen(
            session = session,
            target = target,
            annotations = annotations,
            onBack = { openPreview = null },
            modifier = modifier,
        )
        return
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = column.title,
                style = LexturesType.display(18, FontWeight.Bold),
                color = textPrimary(),
                modifier = Modifier.weight(1f),
            )
        }

        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            errorMessage != null -> LmsEmptyState(
                icon = Icons.Default.Description,
                title = column.title,
                message = errorMessage!!,
                modifier = Modifier.padding(24.dp),
            )
            else -> Column(
                modifier = Modifier
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                ScoreHeader(grade = grade, column = column)
                if (grade?.gradedByAi == true) {
                    LmsCard {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Icon(Icons.Default.AutoAwesome, contentDescription = null, tint = accentColor())
                            Text("Graded with AI assistance", fontSize = 14.sp)
                        }
                    }
                }
                column.rubric?.takeIf { it.criteria.isNotEmpty() }?.let { RubricSection(it, grade) }
                grade?.instructorComment?.trim()?.takeIf { it.isNotEmpty() }?.let { comment ->
                    FeedbackCommentCard("Instructor comment", comment)
                }
                grade?.comments?.takeIf { it.isNotEmpty() }?.let { CommentsSection(it) }
                if (feedbackMediaEnabled && feedbackMedia.isNotEmpty()) {
                    FeedbackMediaSection(
                        session = session,
                        media = feedbackMedia,
                        playbackPaths = playbackPaths,
                    )
                }
                submission?.let { sub ->
                    SubmissionSection(sub) { filename, path, mime ->
                        openPreview = FilePreviewTarget.submissionContentPath(
                            courseCode = course.courseCode,
                            contentPath = path,
                            fileName = filename,
                            mimeType = mime,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ScoreHeader(grade: SubmissionGrade?, column: GradeColumn) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("Your grade", fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textSecondary())
            when {
                grade?.excused == true -> Text("Excused", style = LexturesType.display(22, FontWeight.Bold))
                grade?.pointsEarned != null -> {
                    val max = grade.maxPoints ?: column.maxPoints ?: 0.0
                    Text(
                        "${formatPts(grade.pointsEarned!!)} / ${formatPts(max)}",
                        style = LexturesType.display(22, FontWeight.Bold),
                    )
                    if (max > 0) {
                        Text(
                            "${"%.1f".format(grade.pointsEarned!! / max * 100)}%",
                            fontSize = 14.sp,
                            color = accentColor(),
                        )
                    }
                }
                else -> Text("Not graded", color = textSecondary())
            }
        }
    }
}

@Composable
private fun RubricSection(rubric: RubricDefinition, grade: SubmissionGrade?) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text(rubric.title?.takeIf { it.isNotBlank() } ?: "Rubric", style = LexturesType.display(18, FontWeight.Bold))
            for (criterion in rubric.criteria) {
                RubricCriterionRow(criterion, grade?.rubricScores?.get(criterion.id))
            }
        }
    }
}

@Composable
private fun RubricCriterionRow(criterion: RubricCriterion, score: Double?) {
    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(criterion.title, fontWeight = FontWeight.SemiBold, fontSize = 14.sp, modifier = Modifier.weight(1f))
            Text(
                score?.let { formatPts(it) } ?: "—",
                fontWeight = FontWeight.Bold,
                color = if (score != null) accentColor() else textSecondary(),
            )
        }
        criterion.description?.takeIf { it.isNotBlank() }?.let {
            Text(it, fontSize = 12.sp, color = textSecondary())
        }
        if (score != null) {
            matchedLevel(criterion, score)?.let { level ->
                Text(level.label, fontSize = 12.sp, fontWeight = FontWeight.Medium)
                level.description?.takeIf { it.isNotBlank() }?.let { note ->
                    Text(note, fontSize = 12.sp, color = textSecondary())
                }
            }
        }
    }
}

@Composable
private fun FeedbackCommentCard(title: String, body: String) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(title, style = LexturesType.display(16, FontWeight.Bold))
            Text(body, fontSize = 14.sp)
        }
    }
}

@Composable
private fun CommentsSection(comments: List<GradeComment>) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Feedback thread", style = LexturesType.display(18, FontWeight.Bold))
            for (comment in comments) {
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    comment.displayName?.takeIf { it.isNotBlank() }?.let {
                        Text(it, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                    }
                    Text(comment.body, fontSize = 14.sp)
                }
            }
        }
    }
}

@OptIn(UnstableApi::class)
@Composable
private fun FeedbackMediaSection(
    session: AuthSession,
    media: List<SubmissionFeedbackMedia>,
    playbackPaths: Map<String, String>,
) {
    val accessToken by session.accessToken.collectAsState()
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("Audio / video feedback", style = LexturesType.display(18, FontWeight.Bold))
            for (item in media) {
                val path = playbackPaths[item.id]
                Text(item.mediaType.replaceFirstChar { it.uppercase() }, fontSize = 12.sp, color = textSecondary())
                val token = accessToken
                if (path != null && token != null) {
                    val url = AppConfiguration.apiUrl(path).toString()
                    AndroidView(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(if (item.mediaType == "video") 180.dp else 48.dp),
                        factory = { context ->
                            val dataSource = DefaultHttpDataSource.Factory()
                                .setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
                            val mediaSource = ProgressiveMediaSource.Factory(dataSource)
                                .createMediaSource(MediaItem.fromUri(url))
                            PlayerView(context).apply {
                                player = ExoPlayer.Builder(context).build().apply {
                                    setMediaSource(mediaSource)
                                    prepare()
                                }
                                useController = true
                            }
                        },
                    )
                } else {
                    CircularProgressIndicator()
                }
            }
        }
    }
}

@Composable
private fun SubmissionSection(
    submission: AssignmentSubmission,
    onOpenFile: (filename: String, path: String, mime: String?) -> Unit,
) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Your submission", style = LexturesType.display(18, FontWeight.Bold))
            val filename = submission.attachmentFilename?.takeIf { it.isNotBlank() }
            val path = submission.attachmentContentPath?.trim()?.takeIf { it.isNotEmpty() }
            if (filename != null && path != null) {
                Button(onClick = { onOpenFile(filename, path, submission.attachmentMimeType) }) {
                    Text("View submitted file")
                }
            }
            submission.bodyText?.trim()?.takeIf { it.isNotEmpty() }?.let {
                Text(it, fontSize = 14.sp, color = textSecondary())
            }
        }
    }
}

private fun matchedLevel(criterion: RubricCriterion, score: Double): RubricLevel? =
    criterion.levels.minByOrNull { kotlin.math.abs(it.points - score) }

private fun formatPts(value: Double): String =
    if (value % 1.0 == 0.0) value.toLong().toString() else "%.1f".format(value)
