package com.lextures.android.features.quiz

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.QuizAnswerState
import com.lextures.android.core.lms.QuizCodeRunResponse
import com.lextures.android.core.lms.QuizLogic
import com.lextures.android.core.lms.QuizQuestion
import com.lextures.android.core.ui.CodeEditor
import com.lextures.android.R
import kotlinx.coroutines.launch

data class CodeQuestionRunContext(
    val courseCode: String,
    val itemId: String,
    val attemptId: String,
    val accessToken: String,
)

@Composable
fun CodeQuestionView(
    question: QuizQuestion,
    answer: QuizAnswerState,
    runContext: CodeQuestionRunContext?,
    onChange: (QuizAnswerState) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (QuizLogic.isCodeQuestionOversized(question)) {
        Column(modifier = modifier.padding(vertical = 4.dp)) {
            Text(quizCodeOversizedTitle(), fontWeight = FontWeight.SemiBold)
            Text(quizCodeOversizedHint(), color = LexturesColors.TextSecondary)
        }
        return
    }

    val scope = rememberCoroutineScope()
    val runFailedMessage = quizCodeRunFailedLabel()
    var running by remember(question.id) { mutableStateOf(false) }
    var runError by remember(question.id) { mutableStateOf<String?>(null) }
    var runResult by remember(question.id) { mutableStateOf<QuizCodeRunResponse?>(null) }
    var seeded by remember(question.id) { mutableStateOf(false) }
    val language = QuizLogic.codeLanguageLabel(question)

    LaunchedEffect(question.id) {
        if (!seeded) {
            seeded = true
            val initial = QuizLogic.initialCodeAnswer(question, answer)
            if (initial != answer) onChange(initial)
        }
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(L.format(R.string.mobile_quiz_code_language, language), fontWeight = FontWeight.Medium)

        CodeEditor(
            text = answer.text.orEmpty(),
            onTextChange = { onChange(answer.copy(text = it)) },
            onInsert = { snippet -> onChange(answer.copy(text = answer.text.orEmpty() + snippet)) },
        )

        AuthPrimaryButton(
            text = if (running) quizCodeRunningLabel() else quizCodeRunLabel(),
            onClick = {
                val ctx = runContext ?: return@AuthPrimaryButton
                scope.launch {
                    running = true
                    runError = null
                    try {
                        runResult = LmsApi.postQuizQuestionRun(
                            courseCode = ctx.courseCode,
                            itemId = ctx.itemId,
                            attemptId = ctx.attemptId,
                            questionId = question.id,
                            code = answer.text.orEmpty(),
                            languageId = question.typeConfig?.languageId,
                            accessToken = ctx.accessToken,
                        )
                    } catch (e: Exception) {
                        runError = e.message ?: runFailedMessage
                    } finally {
                        running = false
                    }
                }
            },
            enabled = !running && runContext != null && !answer.text.isNullOrBlank(),
            modifier = Modifier.fillMaxWidth(),
        )

        if (running) {
            CircularProgressIndicator()
        }

        runError?.let {
            Text(it, color = LexturesColors.Coral)
        }

        runResult?.let { response ->
            CodeRunResultsPanel(response)
        }
    }
}

@Composable
private fun CodeRunResultsPanel(response: QuizCodeRunResponse) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(L.format(R.string.mobile_quiz_code_runScore, response.pointsEarned, response.pointsPossible))
        response.results.forEachIndexed { index, result ->
            Column(
                Modifier
                    .fillMaxWidth()
                    .padding(10.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(L.format(R.string.mobile_quiz_code_testNumber, index + 1), fontWeight = FontWeight.SemiBold)
                    Text(
                        quizCodeStatusLabel(result.status),
                        color = if (result.passed) LexturesColors.Primary else LexturesColors.Coral,
                        fontWeight = FontWeight.Bold,
                    )
                }
                if (result.expectedOutput.isNotEmpty()) {
                    Text(
                        L.format(R.string.mobile_quiz_code_expected, result.expectedOutput),
                        fontFamily = FontFamily.Monospace,
                    )
                }
                Text(
                    L.format(R.string.mobile_quiz_code_actual, result.actualOutput),
                    fontFamily = FontFamily.Monospace,
                )
                result.stderr?.takeIf { it.isNotEmpty() }?.let { stderr ->
                    Text(
                        L.format(R.string.mobile_quiz_code_stderr, stderr),
                        fontFamily = FontFamily.Monospace,
                        color = LexturesColors.Coral,
                    )
                }
            }
        }
    }
}

@Composable
private fun quizCodeStatusLabel(status: String): String = when (status) {
    "pass" -> quizCodeStatusPassLabel()
    "fail" -> quizCodeStatusFailLabel()
    "tle" -> quizCodeStatusTleLabel()
    "mle" -> quizCodeStatusMleLabel()
    "re" -> quizCodeStatusReLabel()
    "ce" -> quizCodeStatusCeLabel()
    else -> status.uppercase()
}
