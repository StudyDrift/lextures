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
                            Text(
                                text = item.assignmentTitle,
                                fontSize = 14.sp,
                                fontWeight = FontWeight.Medium,
                                color = textPrimary(),
                            )
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
    var grading by remember { mutableStateOf<AssignmentSubmission?>(null) }
    var reloadKey by remember { mutableStateOf(0) }

    BackHandler(onBack = onBack)

    grading?.let { submission ->
        GradeSubmissionScreen(
            session = session,
            course = course,
            assignmentId = backlogItem.assignmentId,
            submission = submission,
            onDone = { saved ->
                grading = null
                if (saved) reloadKey++
            },
            modifier = modifier,
        )
        return
    }

    LaunchedEffect(accessToken, filter, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            submissions = LmsApi.fetchSubmissions(
                courseCode = course.courseCode,
                itemId = backlogItem.assignmentId,
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
                    LmsCard(onClick = { grading = submission }) {
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

/** Full-screen grade entry: points + comment for one submission. */
@Composable
fun GradeSubmissionScreen(
    session: AuthSession,
    course: CourseSummary,
    assignmentId: String,
    submission: AssignmentSubmission,
    onDone: (saved: Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var pointsText by remember { mutableStateOf("") }
    var comment by remember { mutableStateOf("") }
    var maxPoints by remember { mutableStateOf<Double?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }

    BackHandler { onDone(false) }

    LaunchedEffect(accessToken, submission.id) {
        val token = accessToken ?: return@LaunchedEffect
        // Pre-fill when a grade already exists; also learns maxPoints for validation.
        runCatching {
            LmsApi.fetchSubmissionGrade(course.courseCode, assignmentId, submission.id, token)
        }.getOrNull()?.let { grade ->
            maxPoints = grade.maxPoints
            grade.pointsEarned?.let { pointsText = formatPoints(it) }
            comment = grade.instructorComment.orEmpty()
        }
    }

    val pointsValue = pointsText.replace(',', '.').toDoubleOrNull()
    val pointsValid = pointsValue != null && pointsValue >= 0 &&
        (maxPoints == null || pointsValue <= maxPoints!!)

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { onDone(false) }) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = "Grade submission",
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
            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            item {
                LmsCard {
                    Text(text = submission.displayName, style = LexturesType.display(18), color = textPrimary())
                    Text(
                        text = "Submitted ${LmsDates.shortDateTime(submission.submittedAt)}",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    submission.attachmentFilename?.takeIf { it.isNotEmpty() }?.let { filename ->
                        Text(text = filename, fontSize = 12.sp, color = textSecondary())
                        Text(
                            text = "Open the web app to review file submissions in full.",
                            fontSize = 11.sp,
                            fontStyle = FontStyle.Italic,
                            color = textSecondary(),
                        )
                    }
                }
            }

            item {
                LmsCard {
                    Text(text = "Score", style = LexturesType.display(17), color = textPrimary())
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        OutlinedTextField(
                            value = pointsText,
                            onValueChange = { pointsText = it },
                            label = { Text("Points") },
                            isError = pointsText.isNotEmpty() && !pointsValid,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            singleLine = true,
                            colors = OutlinedTextFieldDefaults.colors(
                                focusedBorderColor = LexturesColors.Primary,
                                cursorColor = LexturesColors.Primary,
                            ),
                            modifier = Modifier.weight(1f),
                        )
                        maxPoints?.let {
                            Text(
                                text = "/ ${formatPoints(it)} pts",
                                fontSize = 14.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textSecondary(),
                            )
                        }
                    }
                    if (pointsText.isNotEmpty() && !pointsValid) {
                        Text(
                            text = maxPoints?.let { "Enter a number between 0 and ${formatPoints(it)}." }
                                ?: "Enter a valid number.",
                            fontSize = 12.sp,
                            color = LexturesColors.Error,
                        )
                    }
                }
            }

            item {
                LmsCard {
                    Text(text = "Feedback", style = LexturesType.display(17), color = textPrimary())
                    OutlinedTextField(
                        value = comment,
                        onValueChange = { comment = it },
                        label = { Text("Comment for the student (optional)") },
                        minLines = 3,
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = LexturesColors.Primary,
                            cursorColor = LexturesColors.Primary,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }

            item {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(14.dp))
                        .background(
                            if (pointsValid && !saving) LexturesColors.Primary
                            else LexturesColors.Primary.copy(alpha = 0.55f),
                        )
                        .clickable(enabled = pointsValid && !saving) {
                            val token = accessToken ?: return@clickable
                            val points = pointsValue ?: return@clickable
                            scope.launch {
                                saving = true
                                errorMessage = null
                                runCatching {
                                    LmsApi.putSubmissionGrade(
                                        courseCode = course.courseCode,
                                        itemId = assignmentId,
                                        submissionId = submission.id,
                                        gradeBody = SubmissionGradePut(
                                            pointsEarned = points,
                                            instructorComment = comment.trim().takeIf { it.isNotEmpty() },
                                        ),
                                        accessToken = token,
                                    )
                                }.onSuccess {
                                    saving = false
                                    onDone(true)
                                }.onFailure {
                                    saving = false
                                    errorMessage = session.mapError(it)
                                }
                            }
                        }
                        .padding(vertical = 14.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    if (saving) {
                        CircularProgressIndicator(
                            color = Color.White,
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                        )
                    } else {
                        Text(
                            text = "Save grade",
                            fontSize = 15.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = Color.White,
                        )
                    }
                }
            }
        }
    }
}

private fun formatPoints(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()
