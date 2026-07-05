package com.lextures.android.features.evaluations

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.EvaluationLogic
import com.lextures.android.core.lms.EvaluationQuestion
import com.lextures.android.core.lms.EvaluationQuestionResult
import com.lextures.android.core.lms.EvaluationQuestionType
import com.lextures.android.core.lms.EvaluationResults
import com.lextures.android.core.lms.EvaluationStatus
import com.lextures.android.core.lms.EvaluationSubmitBody
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

private val offlineJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseEvaluationsSection(
    session: AuthSession,
    course: CourseSummary,
    showResults: Boolean = false,
    modifier: Modifier = Modifier,
) {
    if (course.viewerIsStaff || showResults) {
        EvaluationResultsScreen(session = session, course = course, modifier = modifier)
    } else {
        EvaluationFormScreen(session = session, course = course, modifier = modifier)
    }
}

@Composable
fun EvaluationFormScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()
    val accessToken = session.accessToken.value
    var status by remember { mutableStateOf<EvaluationStatus?>(null) }
    val answers = remember { mutableStateMapOf<String, String>() }
    var loading by remember { mutableStateOf(true) }
    var submitting by remember { mutableStateOf(false) }
    var submitted by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var validationError by remember { mutableStateOf<String?>(null) }
    var staleLabel by remember { mutableStateOf<String?>(null) }
    val loadErrorText = context.getString(R.string.mobile_evaluations_loadError)
    val validationRequiredText = context.getString(R.string.mobile_evaluations_validationRequired)
    val submitLabel = context.getString(R.string.mobile_evaluations_submit)
    val submitErrorText = context.getString(R.string.mobile_evaluations_submitError)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        runCatching {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.evaluationStatus(course.courseCode),
                accessToken = token,
                serializer = EvaluationStatus.serializer(),
            ) {
                LmsApi.fetchEvaluationStatus(course.courseCode, token)
            }
            status = result.first
            submitted = result.first.hasSubmitted
            result.first.windowId?.let { windowId ->
                if (!result.first.hasSubmitted) {
                    answers.clear()
                    answers.putAll(EvaluationLogic.loadDraft(context, course.courseCode, windowId))
                }
            }
            staleLabel = result.second?.takeIf { it.isStale(offline.networkMonitor.isOnline.value) }?.lastUpdatedLabel()
        }.onFailure {
            errorMessage = it.message ?: loadErrorText
        }
        loading = false
    }

    when {
        loading -> LmsSkeletonList(count = 3, modifier = modifier)
        submitted || status?.hasSubmitted == true -> LmsEmptyState(
            icon = Icons.Default.Star,
            title = evaluationsSubmittedTitle(),
            message = evaluationsSubmittedMessage(),
            modifier = modifier.fillMaxSize(),
        )
        status?.windowOpen != true -> LmsEmptyState(
            icon = Icons.Default.Star,
            title = evaluationsNotOpenTitle(),
            message = evaluationsNotOpenMessage(),
            modifier = modifier.fillMaxSize(),
        )
        else -> Column(
            modifier = modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            staleLabel?.let { StalenessChip(label = it) }
            LmsCard {
                Text(evaluationsAnonymityBanner(), fontSize = 14.sp, color = textPrimary())
            }
            status?.closesAt?.let {
                Text(evaluationsDeadline(EvaluationLogic.formatDeadline(it)), fontSize = 12.sp, color = textSecondary())
            }
            status?.questions.orEmpty().forEachIndexed { index, question ->
                EvaluationQuestionField(
                    question = question,
                    index = index,
                    value = answers[index.toString()].orEmpty(),
                    hasError = validationError != null &&
                        question.isRequired &&
                        answers[index.toString()].isNullOrBlank(),
                    onValueChange = { value ->
                        answers[index.toString()] = value
                        validationError = null
                        status?.windowId?.let { windowId ->
                            EvaluationLogic.saveDraft(context, course.courseCode, windowId, answers.toMap())
                        }
                    },
                )
            }
            validationError?.let { LmsErrorBanner(it) }
            errorMessage?.let { LmsErrorBanner(it) }
            Button(
                onClick = {
                    val token = accessToken ?: return@Button
                    val current = status ?: return@Button
                    val windowId = current.windowId ?: return@Button
                    val questions = current.questions.orEmpty()
                    if (EvaluationLogic.missingRequiredIndices(questions, answers.toMap()).isNotEmpty()) {
                        validationError = validationRequiredText
                        return@Button
                    }
                    scope.launch {
                        submitting = true
                        errorMessage = null
                        val path = "/api/v1/courses/${course.courseCode}/evaluations/$windowId/submit"
                        val body = EvaluationSubmitBody(answers.toMap())
                        runCatching {
                            if (offline.networkMonitor.isOnline.value) {
                                LmsApi.submitEvaluation(course.courseCode, windowId, answers.toMap(), token)
                            } else {
                                offline.enqueueMutation(
                                    method = "POST",
                                    path = path,
                                    bodyJson = offlineJson.encodeToString(EvaluationSubmitBody.serializer(), body),
                                    label = submitLabel,
                                    accessToken = token,
                                    preferQueue = true,
                                    idempotencyKey = EvaluationLogic.submitIdempotencyKey(course.courseCode, windowId),
                                )
                            }
                            EvaluationLogic.clearDraft(context, course.courseCode, windowId)
                            submitted = true
                            status = current.copy(hasSubmitted = true, questions = null)
                        }.onFailure {
                            errorMessage = it.message ?: submitErrorText
                        }
                        submitting = false
                    }
                },
                enabled = !submitting && !EvaluationLogic.isSubmitBlocked(status),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (submitting) evaluationsSubmitting() else evaluationsSubmit())
            }
        }
    }
}

@Composable
private fun EvaluationQuestionField(
    question: EvaluationQuestion,
    index: Int,
    value: String,
    hasError: Boolean,
    onValueChange: (String) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row {
            Text(
                "${index + 1}. ${question.text}",
                fontWeight = FontWeight.SemiBold,
                fontSize = 14.sp,
                color = if (hasError) LexturesColors.Error else textPrimary(),
            )
            if (question.isRequired) {
                Text(" *", color = LexturesColors.Error)
            }
        }
        when (question.type) {
            EvaluationQuestionType.Rating -> {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf("1", "2", "3", "4", "5").forEach { rating ->
                        val selected = value == rating
                        LmsCard(
                            modifier = Modifier
                                .heightIn(min = 44.dp)
                                .selectable(
                                    selected = selected,
                                    onClick = { onValueChange(rating) },
                                    role = Role.RadioButton,
                                ),
                        ) {
                            Text(rating, modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp))
                        }
                    }
                }
            }
            EvaluationQuestionType.MultipleChoice -> {
                question.options.orEmpty().forEach { option ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(min = 44.dp)
                            .selectable(
                                selected = value == option,
                                onClick = { onValueChange(option) },
                                role = Role.RadioButton,
                            ),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(selected = value == option, onClick = null)
                        Text(option, fontSize = 14.sp)
                    }
                }
            }
            EvaluationQuestionType.OpenText -> {
                OutlinedTextField(
                    value = value,
                    onValueChange = onValueChange,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(120.dp),
                    isError = hasError,
                )
            }
        }
    }
}

@Composable
fun EvaluationResultsScreen(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val accessToken = session.accessToken.value
    var results by remember { mutableStateOf<EvaluationResults?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var staleLabel by remember { mutableStateOf<String?>(null) }
    val resultsLoadErrorText = context.getString(R.string.mobile_evaluations_resultsLoadError)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        runCatching {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.evaluationResults(course.courseCode),
                accessToken = token,
                serializer = EvaluationResults.serializer(),
            ) {
                LmsApi.fetchEvaluationResults(course.courseCode, token)
            }
            results = result.first
            staleLabel = result.second?.takeIf { it.isStale(offline.networkMonitor.isOnline.value) }?.lastUpdatedLabel()
        }.onFailure {
            errorMessage = it.message ?: resultsLoadErrorText
        }
        loading = false
    }

    when {
        loading -> LmsSkeletonList(count = 3, modifier = modifier)
        errorMessage != null && results == null -> LmsEmptyState(
            icon = Icons.Default.BarChart,
            title = evaluationsResultsErrorTitle(),
            message = errorMessage.orEmpty(),
            modifier = modifier.fillMaxSize(),
        )
        results == null -> LmsEmptyState(
            icon = Icons.Default.BarChart,
            title = evaluationsNoResultsTitle(),
            message = evaluationsNoResultsMessage(),
            modifier = modifier.fillMaxSize(),
        )
        else -> Column(
            modifier = modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            staleLabel?.let { StalenessChip(label = it) }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                SummaryTile(results!!.responseCount.toString(), evaluationsResponses(), Modifier.weight(1f))
                SummaryTile(results!!.enrolledCount.toString(), evaluationsEnrolled(), Modifier.weight(1f))
                SummaryTile("${results!!.completionPct.toInt()}%", evaluationsCompletion(), Modifier.weight(1f))
            }
            Text(
                evaluationsWindowRange(
                    EvaluationLogic.formatDeadline(results!!.opensAt),
                    EvaluationLogic.formatDeadline(results!!.closesAt),
                ),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            if (!results!!.meetsThreshold) {
                LmsCard {
                    Text(evaluationsThresholdTitle(), fontWeight = FontWeight.SemiBold)
                    Text(evaluationsThresholdMessage(), fontSize = 12.sp, color = textSecondary())
                }
            } else {
                results!!.questions.forEach { question -> QuestionResultCard(question) }
            }
        }
    }
}

@Composable
private fun SummaryTile(value: String, label: String, modifier: Modifier = Modifier) {
    LmsCard(modifier = modifier) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
            Text(value, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = textPrimary())
            Text(label, fontSize = 11.sp, color = textSecondary())
        }
    }
}

@Composable
private fun QuestionResultCard(question: EvaluationQuestionResult) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("${question.index + 1}. ${question.text}", fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
            when (question.type) {
                EvaluationQuestionType.Rating -> {
                    question.average?.let {
                        Text(evaluationsAverageRating(String.format("%.1f", it)), fontWeight = FontWeight.Bold)
                    }
                    question.distribution.orEmpty().let { dist ->
                        val max = (dist.values.maxOrNull() ?: 1).coerceAtLeast(1)
                        listOf("1", "2", "3", "4", "5").forEach { rating ->
                            val count = dist[rating] ?: 0
                            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text(rating, fontSize = 12.sp)
                                LinearProgressIndicator(
                                    progress = { count.toFloat() / max.toFloat() },
                                    modifier = Modifier
                                        .weight(1f)
                                        .height(8.dp),
                                )
                                Text("$count", fontSize = 11.sp)
                            }
                        }
                    }
                }
                EvaluationQuestionType.MultipleChoice -> {
                    val total = question.distribution.orEmpty().values.sum().coerceAtLeast(1)
                    question.distribution.orEmpty().toSortedMap().forEach { (option, count) ->
                        val pct = (count * 100) / total
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(option, fontSize = 12.sp, modifier = Modifier.weight(1f))
                            LinearProgressIndicator(
                                progress = { count.toFloat() / total.toFloat() },
                                modifier = Modifier
                                    .weight(2f)
                                    .height(8.dp),
                            )
                            Text("$count ($pct%)", fontSize = 11.sp)
                        }
                    }
                }
                EvaluationQuestionType.OpenText -> {
                    val texts = question.openTexts.orEmpty()
                    Text(evaluationsResponseCount(texts.size), fontSize = 12.sp, color = textSecondary())
                    if (texts.isEmpty()) {
                        Text(evaluationsNoOpenResponses(), fontSize = 12.sp, color = textSecondary())
                    } else {
                        texts.forEach { text ->
                            LmsCard { Text(text, fontSize = 12.sp) }
                        }
                    }
                }
            }
        }
    }
}
