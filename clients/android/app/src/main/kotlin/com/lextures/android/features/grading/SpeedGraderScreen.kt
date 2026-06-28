package com.lextures.android.features.grading

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.SubmissionGradePut
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

/**
 * SpeedGrader-style flow: page through an assignment's submissions, read each one,
 * and enter a score/feedback without returning to the list. Seeded with the loaded
 * submissions snapshot and the index of the tapped student.
 */
@Composable
fun SpeedGraderScreen(
    session: AuthSession,
    course: CourseSummary,
    assignmentId: String,
    submissions: List<AssignmentSubmission>,
    startIndex: Int,
    onSaved: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var index by remember { mutableStateOf(startIndex.coerceIn(0, (submissions.size - 1).coerceAtLeast(0))) }
    var gradedIds by remember {
        mutableStateOf(submissions.filter { it.isGraded == true }.map { it.id }.toSet())
    }
    var pointsText by remember { mutableStateOf("") }
    var comment by remember { mutableStateOf("") }
    var maxPoints by remember { mutableStateOf<Double?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    BackHandler(onBack = onBack)

    val current = submissions.getOrNull(index)
    val remainingUngraded = submissions.count { it.id !in gradedIds }

    LaunchedEffect(accessToken, index) {
        val token = accessToken ?: return@LaunchedEffect
        val submission = current ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        pointsText = ""
        comment = ""
        maxPoints = null
        runCatching {
            LmsApi.fetchSubmissionGrade(course.courseCode, assignmentId, submission.id, token)
        }.getOrNull()?.let { grade ->
            maxPoints = grade.maxPoints
            grade.pointsEarned?.let {
                pointsText = formatGradePoints(it)
                gradedIds = gradedIds + submission.id
            }
            comment = grade.instructorComment.orEmpty()
        }
        loading = false
    }

    val pointsValue = pointsText.replace(',', '.').toDoubleOrNull()
    val pointsValid = pointsValue != null && pointsValue >= 0 &&
        (maxPoints == null || pointsValue <= maxPoints!!)

    fun nextUngradedIndex(from: Int): Int? {
        if (submissions.isEmpty()) return null
        for (offset in 1..submissions.size) {
            val candidate = (from + offset) % submissions.size
            if (candidate == from) break
            if (submissions[candidate].id !in gradedIds) return candidate
        }
        return null
    }

    fun save() {
        val token = accessToken ?: return
        val points = pointsValue ?: return
        val submission = current ?: return
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
                gradedIds = gradedIds + submission.id
                onSaved()
                nextUngradedIndex(index)?.let { index = it }
            }.onFailure {
                saving = false
                errorMessage = session.mapError(it)
            }
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
                text = "Speed grader",
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
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "${index + 1} of ${submissions.size}",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Box(modifier = Modifier.weight(1f))
                    val tint = if (remainingUngraded == 0) accentColor() else LexturesColors.Amber
                    Text(
                        text = if (remainingUngraded == 0) "All graded" else "$remainingUngraded ungraded",
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = tint,
                        modifier = Modifier
                            .clip(RoundedCornerShape(50))
                            .background(tint.copy(alpha = 0.14f))
                            .padding(horizontal = 9.dp, vertical = 3.dp),
                    )
                }
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            val submission = current
            if (submission == null) {
                item {
                    Text(
                        text = "No submissions in this view.",
                        fontSize = 14.sp,
                        color = textSecondary(),
                    )
                }
            } else {
                item {
                    LmsCard {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(42.dp)
                                    .clip(CircleShape)
                                    .background(coverBrush(submission.displayName)),
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(
                                    text = submission.displayName.take(2).uppercase(),
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.Bold,
                                    color = Color.White,
                                )
                            }
                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(text = submission.displayName, style = LexturesType.display(18), color = textPrimary())
                                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                    Text(
                                        text = "Submitted ${LmsDates.shortDateTime(submission.submittedAt)}",
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                    )
                                    submission.versionNumber?.takeIf { it > 1 }?.let {
                                        Text(
                                            text = "v$it",
                                            fontSize = 12.sp,
                                            fontWeight = FontWeight.SemiBold,
                                            color = accentColor(),
                                        )
                                    }
                                }
                            }
                            if (submission.id in gradedIds) {
                                Icon(
                                    Icons.Default.CheckCircle,
                                    contentDescription = "Graded",
                                    tint = accentColor(),
                                    modifier = Modifier.size(20.dp),
                                )
                            }
                        }
                    }
                }

                item {
                    LmsCard {
                        Text(text = "Submission", style = LexturesType.display(17), color = textPrimary())
                        val body = submission.bodyText?.trim().orEmpty()
                        val filename = submission.attachmentFilename?.takeIf { it.isNotEmpty() }
                        when {
                            body.isNotEmpty() || filename != null -> {
                                if (body.isNotEmpty()) {
                                    Text(text = body, fontSize = 14.sp, color = textPrimary())
                                }
                                filename?.let {
                                    Text(text = it, fontSize = 12.sp, color = textSecondary())
                                    Text(
                                        text = "Open the web app to review file submissions in full.",
                                        fontSize = 11.sp,
                                        fontStyle = FontStyle.Italic,
                                        color = textSecondary(),
                                    )
                                }
                            }
                            else -> Text(
                                text = "No text or attachment was submitted.",
                                fontSize = 14.sp,
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
                                    text = "/ ${formatGradePoints(it)} pts",
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = textSecondary(),
                                )
                            }
                        }
                        if (pointsText.isNotEmpty() && !pointsValid) {
                            Text(
                                text = maxPoints?.let { "Enter a number between 0 and ${formatGradePoints(it)}." }
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
                            .clickable(enabled = pointsValid && !saving) { save() }
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
                                text = if (remainingUngraded <= 1) "Save grade" else "Save & next",
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = Color.White,
                            )
                        }
                    }
                }

                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        SpeedGraderNavButton(
                            label = "Previous",
                            icon = Icons.Default.ChevronLeft,
                            enabled = index > 0 && !saving,
                            modifier = Modifier.weight(1f),
                        ) { index = (index - 1).coerceAtLeast(0) }
                        SpeedGraderNavButton(
                            label = "Next",
                            icon = Icons.Default.ChevronRight,
                            enabled = index < submissions.size - 1 && !saving,
                            modifier = Modifier.weight(1f),
                        ) { index = (index + 1).coerceAtMost(submissions.size - 1) }
                    }
                }
            }
        }
    }
}

@Composable
private fun SpeedGraderNavButton(
    label: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val tint = if (enabled) accentColor() else textSecondary().copy(alpha = 0.5f)
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(vertical = 12.dp),
        contentAlignment = Alignment.Center,
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(18.dp))
            Text(text = label, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = tint)
        }
    }
}

private fun formatGradePoints(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()
