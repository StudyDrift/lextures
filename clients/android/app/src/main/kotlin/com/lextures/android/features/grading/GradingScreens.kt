package com.lextures.android.features.grading

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Assignment
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GradingBacklogItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.SubmissionGradePut
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

/** "Grading" section of course detail (staff): assignments with ungraded work. */
@Composable
fun GradingBacklogSection(
    session: AuthSession,
    course: CourseSummary,
    onOpenItem: (GradingBacklogItem) -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    var items by remember { mutableStateOf<List<GradingBacklogItem>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            items = LmsApi.fetchGradingBacklog(course.courseCode, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        errorMessage?.let { LmsErrorBanner(it) }
        when {
            loading && items.isEmpty() -> LmsSkeletonList(count = 3)
            items.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.FactCheck,
                title = "All caught up",
                message = "Submissions waiting for a grade will appear here.",
            )
            else -> items.forEach { item ->
                LmsCard(
                    accent = if (item.ungradedCount > 0) LexturesColors.Amber else null,
                    onClick = { onOpenItem(item) },
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(32.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(LexturesColors.Amber.copy(alpha = 0.13f)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                Icons.AutoMirrored.Filled.Assignment,
                                contentDescription = null,
                                tint = LexturesColors.Amber,
                                modifier = Modifier.size(16.dp),
                            )
                        }
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(6.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    text = item.assignmentTitle,
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.Medium,
                                    color = textPrimary(),
                                    modifier = Modifier.weight(1f, fill = false),
                                )
                                if (item.isQuiz) {
                                    Text(
                                        text = "Quiz",
                                        fontSize = 10.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = LexturesColors.Amber,
                                        modifier = Modifier
                                            .clip(RoundedCornerShape(50))
                                            .background(LexturesColors.Amber.copy(alpha = 0.14f))
                                            .padding(horizontal = 6.dp, vertical = 2.dp),
                                    )
                                }
                            }
                            Text(
                                text = "${item.ungradedCount} ungraded submission${if (item.ungradedCount == 1) "" else "s"}",
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        Text(
                            text = "${item.ungradedCount}",
                            style = LexturesType.display(16, FontWeight.Bold),
                            color = LexturesColors.Amber,
                            modifier = Modifier
                                .clip(RoundedCornerShape(50))
                                .background(LexturesColors.Amber.copy(alpha = 0.14f))
                                .padding(horizontal = 9.dp, vertical = 3.dp),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = textSecondary().copy(alpha = 0.6f),
                            modifier = Modifier.size(16.dp),
                        )
                    }
                }
            }
        }
    }
}

/** Standalone backlog screen (pushed from the dashboard teacher snapshot). */
@Composable
fun GradingBacklogScreen(
    session: AuthSession,
    course: CourseSummary,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var openItem by remember { mutableStateOf<GradingBacklogItem?>(null) }

    BackHandler(onBack = onBack)

    openItem?.let { item ->
        SubmissionsListScreen(
            session = session,
            course = course,
            backlogItem = item,
            onBack = { openItem = null },
            modifier = modifier,
        )
        return
    }

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = "Grading · ${course.displayTitle}",
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
        ) {
            item {
                GradingBacklogSection(
                    session = session,
                    course = course,
                    onOpenItem = { openItem = it },
                )
            }
        }
    }
}

/** Submissions for one assignment, filterable by graded state. */
@Composable
fun SubmissionsListScreen(
    session: AuthSession,
    course: CourseSummary,
    backlogItem: GradingBacklogItem,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()

    var filter by remember { mutableStateOf("ungraded") }
    var submissions by remember { mutableStateOf<List<AssignmentSubmission>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var quizInfo by remember { mutableStateOf<AssignmentSubmission?>(null) }
    // Frozen submissions snapshot + starting index for the speed grader.
    var speedGrader by remember { mutableStateOf<Pair<List<AssignmentSubmission>, Int>?>(null) }
    var reloadKey by remember { mutableStateOf(0) }

    BackHandler(onBack = onBack)

    quizInfo?.let { submission ->
        QuizAttemptInfoScreen(
            backlogItem = backlogItem,
            submission = submission,
            onDone = { quizInfo = null },
            modifier = modifier,
        )
        return
    }

    speedGrader?.let { (snapshot, startIndex) ->
        SpeedGraderScreen(
            session = session,
            course = course,
            assignmentId = backlogItem.resolvedItemId,
            submissions = snapshot,
            startIndex = startIndex,
            onSaved = { reloadKey++ },
            onBack = { speedGrader = null },
            modifier = modifier,
        )
        return
    }

    LaunchedEffect(accessToken, filter, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            submissions = LmsApi.fetchGradingSubmissions(
                courseCode = course.courseCode,
                backlogItem = backlogItem,
                graded = filter.takeIf { it != "all" },
                accessToken = token,
            )
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = backlogItem.assignmentTitle,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LmsSegmentedChips(
                    options = listOf("ungraded" to "Ungraded", "graded" to "Graded", "all" to "All"),
                    selectedId = filter,
                    onSelect = { filter = it },
                )
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (loading && submissions.isEmpty()) {
                item { LmsSkeletonList(count = 4) }
            } else if (submissions.isEmpty()) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.Inbox,
                        title = "No submissions",
                        message = if (filter == "ungraded") {
                            "Nothing waiting for a grade right now."
                        } else {
                            "No submissions in this view."
                        },
                    )
                }
            } else {
                items(submissions, key = { it.id }) { submission ->
                    LmsCard(onClick = {
                        if (backlogItem.isQuiz) {
                            quizInfo = submission
                        } else {
                            speedGrader = submissions to submissions.indexOf(submission)
                        }
                    }) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(38.dp)
                                    .clip(CircleShape)
                                    .background(coverBrush(submission.displayName)),
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(
                                    text = submission.displayName.take(2).uppercase(),
                                    fontSize = 12.sp,
                                    fontWeight = FontWeight.Bold,
                                    color = Color.White,
                                )
                            }
                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(
                                    text = submission.displayName,
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.Medium,
                                    color = textPrimary(),
                                )
                                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                    Text(
                                        text = "Submitted ${LmsDates.relative(submission.submittedAt)}",
                                        fontSize = 11.sp,
                                        color = textSecondary(),
                                    )
                                    submission.versionNumber?.takeIf { it > 1 }?.let {
                                        Text(
                                            text = "v$it",
                                            fontSize = 11.sp,
                                            fontWeight = FontWeight.SemiBold,
                                            color = accentColor(),
                                        )
                                    }
                                }
                                submission.attachmentFilename?.takeIf { it.isNotEmpty() }?.let { filename ->
                                    Row(
                                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Icon(
                                            Icons.Default.AttachFile,
                                            contentDescription = null,
                                            tint = textSecondary(),
                                            modifier = Modifier.size(12.dp),
                                        )
                                        Text(
                                            text = filename,
                                            fontSize = 11.sp,
                                            color = textSecondary(),
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis,
                                        )
                                    }
                                }
                            }
                            Icon(
                                Icons.Default.Edit,
                                contentDescription = "Grade",
                                tint = accentColor(),
                                modifier = Modifier.size(18.dp),
                            )
                        }
                    }
                }
            }
        }
    }
}

/** Quiz attempts must be graded per question on web (mobile v1 is read-only). */
@Composable
fun QuizAttemptInfoScreen(
    backlogItem: GradingBacklogItem,
    submission: AssignmentSubmission,
    onDone: () -> Unit,
    modifier: Modifier = Modifier,
) {
    BackHandler(onBack = onDone)

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onDone) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = "Quiz attempt",
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                LmsCard {
                    Text(text = submission.displayName, style = LexturesType.display(18), color = textPrimary())
                    Text(
                        text = "Submitted ${LmsDates.shortDateTime(submission.submittedAt)}",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    submission.versionNumber?.takeIf { it > 1 }?.let { attempt ->
                        Text(
                            text = "Attempt $attempt",
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
            }
            item {
                LmsCard {
                    Text(text = "Grade on web", style = LexturesType.display(17), color = textPrimary())
                    Text(
                        text = "Quiz answers are graded question by question. Open the web app to review responses and enter scores for ${backlogItem.assignmentTitle}.",
                        fontSize = 14.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}
