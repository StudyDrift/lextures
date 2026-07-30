package com.lextures.android.features.quiz

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.lms.QuizAnswerState
import com.lextures.android.core.lms.QuizLogic
import com.lextures.android.core.lms.QuizQuestion
import com.lextures.android.core.lms.QuizQuestionKind
import com.lextures.android.core.lms.QuizResultsResponse
import com.lextures.android.core.lms.QuizSaveState
import com.lextures.android.features.courses.RowHeader
import com.lextures.android.features.courses.markdown.MarkdownRenderStyle
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.notebooks.NotebookContentView
import com.lextures.android.features.notebooks.inlineMarkdown

@Composable
fun QuizQuestionContent(
    question: QuizQuestion,
    answer: QuizAnswerState,
    saveState: QuizSaveState,
    onChange: (QuizAnswerState) -> Unit,
    codeRunContext: CodeQuestionRunContext? = null,
    modifier: Modifier = Modifier,
    suppressAffordances: Boolean = false,
) {
    val kind = QuizQuestionKind.from(question.questionType)
    LmsCard(modifier = modifier.padding(vertical = 8.dp)) {
        NotebookContentView(
            markdown = question.prompt,
            suppressAffordances = suppressAffordances,
        )
        when (saveState) {
            QuizSaveState.Saved -> Text(quizSavedLabel())
            QuizSaveState.Failed -> Text(quizSaveFailedLabel(), color = LexturesColors.Coral)
            QuizSaveState.Queued -> Text(quizNotYetSavedLabel(), color = LexturesColors.Coral)
            else -> Unit
        }
        if (!kind.supportsMobileInput) {
            Text(quizOpenOnWebLabel())
            Text(quizOpenOnWebHintLabel())
        } else when (kind) {
            QuizQuestionKind.MultipleChoice, QuizQuestionKind.TrueFalse -> {
                QuizLogic.visibleChoices(question).forEachIndexed { index, choice ->
                    QuizChoiceRow(
                        label = choice,
                        selected = answer.choice == index,
                        suppressLinks = suppressAffordances,
                        onClick = { onChange(answer.copy(choice = index)) },
                    )
                }
            }
            QuizQuestionKind.Numeric, QuizQuestionKind.Formula, QuizQuestionKind.ShortAnswer,
            QuizQuestionKind.FillInBlank, QuizQuestionKind.Essay, QuizQuestionKind.FileUpload,
            -> {
                OutlinedTextField(
                    value = answer.text.orEmpty(),
                    onValueChange = { onChange(answer.copy(text = it)) },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(quizShortAnswerPlaceholder()) },
                )
            }
            QuizQuestionKind.Ordering -> {
                val items = answer.ordering ?: QuizLogic.orderingItems(question)
                LaunchedEffect(question.id) {
                    if (answer.ordering == null) {
                        onChange(answer.copy(ordering = QuizLogic.orderingItems(question)))
                    }
                }
                items.forEachIndexed { index, item ->
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp)
                            .heightIn(min = 48.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("${index + 1}.", fontWeight = FontWeight.SemiBold)
                        Text(
                            text = inlineMarkdown(item, suppressLinks = suppressAffordances),
                            modifier = Modifier.semantics {
                                contentDescription = MarkdownRenderStyle.plainLabel(item)
                            },
                        )
                    }
                }
            }
            QuizQuestionKind.Code -> {
                CodeQuestionView(
                    question = question,
                    answer = answer,
                    runContext = codeRunContext,
                    onChange = onChange,
                )
            }
            QuizQuestionKind.Matching -> {
                val pairs = QuizLogic.matchingPairs(question)
                val rights = QuizLogic.sortedRightOptions(pairs)
                pairs.forEach { pair ->
                    Text(
                        text = inlineMarkdown(pair.left, suppressLinks = suppressAffordances),
                        fontWeight = FontWeight.Medium,
                        modifier = Modifier.semantics {
                            contentDescription = MarkdownRenderStyle.plainLabel(pair.left)
                        },
                    )
                    rights.forEach { right ->
                        QuizChoiceRow(
                            label = right,
                            selected = answer.matching?.get(pair.leftId) == right,
                            suppressLinks = suppressAffordances,
                            onClick = {
                                val map = (answer.matching ?: emptyMap()).toMutableMap()
                                map[pair.leftId] = right
                                onChange(answer.copy(matching = map))
                            },
                        )
                    }
                }
            }
            else -> Text(quizOpenOnWebLabel())
        }
    }
}

@Composable
private fun QuizChoiceRow(
    label: String,
    selected: Boolean,
    suppressLinks: Boolean,
    onClick: () -> Unit,
) {
    val plain = MarkdownRenderStyle.plainLabel(label)
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 48.dp)
            .clickable(onClick = onClick)
            .semantics(mergeDescendants = true) {
                role = Role.RadioButton
                contentDescription = plain
            },
    ) {
        RadioButton(selected = selected, onClick = onClick)
        Text(text = inlineMarkdown(label, suppressLinks = suppressLinks))
    }
}

@Composable
fun QuizResultsScreen(
    title: String,
    results: QuizResultsResponse,
    onDone: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        RowHeader(title = title, onBack = onDone)
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Text(quizSubmittedLabel(), fontWeight = FontWeight.Bold)
            results.score?.let { score ->
                LmsCard {
                    Text(quizYourScoreLabel(), fontWeight = FontWeight.SemiBold)
                    Text("${score.pointsEarned} / ${score.pointsPossible} (${score.scorePercent.toInt()}%)")
                }
            } ?: LmsCard { Text(quizPendingReviewLabel()) }
            results.questions.orEmpty().forEach { question ->
                LmsCard {
                    Text(quizQuestionNumberLabel(question.questionIndex + 1))
                    question.promptSnapshot?.let {
                        NotebookContentView(markdown = it, compact = true)
                    }
                    question.pointsAwarded?.let {
                        Text("$it / ${question.maxPoints} pts")
                    }
                }
            }
            AuthPrimaryButton(text = quizDoneLabel(), onClick = onDone, modifier = Modifier.fillMaxWidth())
        }
    }
}
