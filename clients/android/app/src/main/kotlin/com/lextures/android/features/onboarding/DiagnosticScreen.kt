package com.lextures.android.features.onboarding

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.DiagnosticQuestion

/** Multiple-choice placement diagnostic (parity with web onboarding step 3). */
@Composable
fun DiagnosticScreen(
    questions: List<DiagnosticQuestion>,
    questionIndex: Int,
    answers: Map<String, Int>,
    submitting: Boolean,
    onSelectAnswer: (questionId: String, choiceIndex: Int) -> Unit,
    onNextQuestion: () -> Unit,
    onSubmitAnswers: () -> Unit,
    onSkip: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePreferences = LocalLocalePreferences.current
    val question = questions.getOrNull(questionIndex)

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = L.text(R.string.mobile_onboarding_diagnostic_subtitle),
            fontSize = 14.sp,
            color = textSecondary(),
        )

        when {
            question != null -> {
                Text(
                    text = localePreferences.localizedContext(context).getString(
                        R.string.mobile_onboarding_diagnostic_questionOf,
                        questionIndex + 1,
                        questions.size,
                    ),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                Text(
                    text = question.prompt,
                    fontSize = 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                question.choices.forEachIndexed { index, choice ->
                    val selected = answers[question.id] == index
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .border(
                                1.dp,
                                if (selected) LexturesColors.Primary else LexturesColors.FieldBorder,
                                RoundedCornerShape(12.dp),
                            )
                            .clickable(enabled = !submitting) { onSelectAnswer(question.id, index) }
                            .padding(horizontal = 12.dp, vertical = 10.dp)
                            .semantics { contentDescription = choice },
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        RadioButton(selected = selected, onClick = null)
                        Text(text = choice, fontSize = 14.sp, color = textPrimary())
                    }
                }
                AuthPrimaryButton(
                    text = if (questionIndex < questions.lastIndex) {
                        L.text(R.string.mobile_onboarding_diagnostic_nextQuestion)
                    } else {
                        L.text(R.string.mobile_onboarding_continue)
                    },
                    enabled = answers.containsKey(question.id) && !submitting,
                    onClick = {
                        if (questionIndex < questions.lastIndex) onNextQuestion() else onSubmitAnswers()
                    },
                )
            }
            questions.isEmpty() -> CircularProgressIndicator(color = LexturesColors.Primary)
        }

        TextButton(onClick = onSkip, enabled = !submitting) {
            Text(text = L.text(R.string.mobile_onboarding_skipDiagnostic), color = textSecondary())
        }
    }
}
