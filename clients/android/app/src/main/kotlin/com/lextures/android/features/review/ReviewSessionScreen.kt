package com.lextures.android.features.review

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseFileLogic
import com.lextures.android.core.lms.QuizLogic
import com.lextures.android.core.lms.QuizQuestionKind
import com.lextures.android.core.lms.ReviewLogic
import com.lextures.android.core.lms.ReviewQueueItem
import com.lextures.android.core.lms.SrsGrade
import com.lextures.android.core.lms.SrsReviewSubmitBody
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Composable
fun ReviewSessionScreen(
    session: AuthSession,
    shell: HomeShellState?,
    initialQueue: List<ReviewQueueItem>,
    totalDue: Int,
    initialStreak: Int,
    onBack: () -> Unit,
    onFinished: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val json = remember { Json { ignoreUnknownKeys = true } }

    val queue = remember { mutableStateListOf(*initialQueue.toTypedArray()) }
    var revealed by remember { mutableStateOf(false) }
    var reviewedCount by remember { mutableStateOf(0) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var submitting by remember { mutableStateOf(false) }
    var finished by remember { mutableStateOf(false) }
    var shownAtMs by remember { mutableLongStateOf(System.currentTimeMillis()) }

    val userId = remember(shell?.profile?.id, accessToken) {
        shell?.profile?.id ?: NotebookStore.jwtSubject(accessToken)
    }
    val current = queue.firstOrNull()
    val progressLabel = remember(reviewedCount, totalDue, current) {
        val currentIndex = reviewedCount + if (current == null) 0 else 1
        context.getString(R.string.mobile_review_progress, currentIndex, totalDue)
    }

    fun submit(grade: SrsGrade) {
        val token = accessToken ?: return
        val learnerId = userId ?: return
        val item = current ?: return
        scope.launch {
            submitting = true
            errorMessage = null
            val ratedAtMs = System.currentTimeMillis()
            val body = SrsReviewSubmitBody(
                questionId = item.questionId,
                grade = grade.apiValue,
                responseMs = (ratedAtMs - shownAtMs).toInt(),
            )
            val path = "/api/v1/learners/${CourseFileLogic.encodePath(learnerId)}/review"
            val idempotencyKey = ReviewLogic.idempotencyKey(item.questionId, ratedAtMs)
            try {
                offline.enqueueMutation(
                    method = "POST",
                    path = path,
                    bodyJson = json.encodeToString(SrsReviewSubmitBody.serializer(), body),
                    label = L.text(context, localePrefs, R.string.mobile_review_submitLabel),
                    accessToken = token,
                    idempotencyKey = idempotencyKey,
                )
                reviewedCount += 1
                queue.removeAt(0)
                revealed = false
                shownAtMs = System.currentTimeMillis()
                if (queue.isEmpty()) finished = true
            } catch (_: Exception) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_review_error_submit)
            } finally {
                submitting = false
            }
        }
    }

    if (finished) {
        Column(
            modifier = modifier.fillMaxSize().padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(Icons.Default.CheckCircle, contentDescription = null, tint = LexturesColors.BrandTeal)
            Spacer(Modifier.height(16.dp))
            Text(
                L.text(context, localePrefs, R.string.mobile_review_summaryTitle),
                fontSize = 24.sp,
                fontWeight = FontWeight.Bold,
                color = textPrimary(),
            )
            Text(
                context.resources.getQuantityString(
                    R.plurals.mobile_review_summaryReviewed,
                    reviewedCount,
                    reviewedCount,
                ),
                color = textSecondary(),
            )
            Spacer(Modifier.height(16.dp))
            Button(onClick = onFinished) {
                Text(L.text(context, localePrefs, R.string.mobile_review_done))
            }
        }
        return
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(progressLabel, fontSize = 12.sp, color = textSecondary())
            current?.courseTitle?.let {
                Text(it, fontSize = 11.sp, color = textSecondary(), maxLines = 1)
            }
        }
        Spacer(Modifier.height(12.dp))
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            current?.let { item ->
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_review_question),
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textSecondary(),
                        )
                        Text(item.stem, color = textPrimary())
                        val quizQuestion = ReviewLogic.toQuizQuestion(item)
                        if (quizQuestion != null && !revealed) {
                            when (QuizQuestionKind.from(item.questionType)) {
                                QuizQuestionKind.MultipleChoice, QuizQuestionKind.TrueFalse -> {
                                    QuizLogic.visibleChoices(quizQuestion).forEach { choice ->
                                        Text("• $choice", color = textPrimary())
                                    }
                                }
                                QuizQuestionKind.Ordering -> {
                                    QuizLogic.orderingItems(quizQuestion).forEachIndexed { index, label ->
                                        Text("${index + 1}. $label", color = textPrimary())
                                    }
                                }
                                else -> Unit
                            }
                        }
                        if (!revealed) {
                            OutlinedButton(onClick = { revealed = true }, modifier = Modifier.fillMaxWidth()) {
                                Text(L.text(context, localePrefs, R.string.mobile_review_reveal))
                            }
                        }
                    }
                }
                if (revealed) {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_review_answer),
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = LexturesColors.BrandTeal,
                            )
                            Text(ReviewLogic.formatAnswerPreview(item.correctAnswer), color = textPrimary())
                            item.explanation?.takeIf { it.isNotBlank() }?.let {
                                Text(it, fontSize = 12.sp, color = textSecondary())
                            }
                        }
                    }
                }
            }
            errorMessage?.let { LmsErrorBanner(message = it) }
        }
        if (revealed) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                    GradeButton(SrsGrade.Again, submitting, Modifier.weight(1f)) { submit(SrsGrade.Again) }
                    GradeButton(SrsGrade.Hard, submitting, Modifier.weight(1f)) { submit(SrsGrade.Hard) }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                    GradeButton(SrsGrade.Good, submitting, Modifier.weight(1f)) { submit(SrsGrade.Good) }
                    GradeButton(SrsGrade.Easy, submitting, Modifier.weight(1f)) { submit(SrsGrade.Easy) }
                }
            }
        }
    }
}

@Composable
private fun GradeButton(
    grade: SrsGrade,
    disabled: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val label = when (grade) {
        SrsGrade.Again -> R.string.mobile_review_grade_again
        SrsGrade.Hard -> R.string.mobile_review_grade_hard
        SrsGrade.Good -> R.string.mobile_review_grade_good
        SrsGrade.Easy -> R.string.mobile_review_grade_easy
    }
    Button(
        onClick = onClick,
        enabled = !disabled,
        modifier = modifier,
        colors = ButtonDefaults.buttonColors(
            containerColor = when (grade) {
                SrsGrade.Again -> LexturesColors.Coral
                SrsGrade.Hard -> LexturesColors.Amber
                SrsGrade.Good -> LexturesColors.BrandTeal
                SrsGrade.Easy -> LexturesColors.BrandTeal
            },
        ),
    ) {
        Text(L.text(context, localePrefs, label))
    }
}
