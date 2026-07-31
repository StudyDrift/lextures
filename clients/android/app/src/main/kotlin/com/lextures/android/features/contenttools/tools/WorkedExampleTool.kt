package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack4Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

@Composable
fun WorkedExampleTool(props: ContentToolRendererProps) {
    val scope = rememberCoroutineScope()

    var draft by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var prepared by remember(props.instanceId) { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var lastFeedback by remember { mutableStateOf<String?>(null) }
    var lastResult by remember { mutableStateOf<String?>(null) }
    var hintText by remember { mutableStateOf<String?>(null) }
    var revealText by remember { mutableStateOf<String?>(null) }

    val answerPlaceholder = L.text(R.string.mobile_contentTools_tools_worked_example_answerPlaceholder)
    val checkLabel = L.text(R.string.mobile_contentTools_tools_worked_example_check)
    val hintLabel = L.text(R.string.mobile_contentTools_tools_worked_example_hint)
    val revealLabel = L.text(R.string.mobile_contentTools_tools_worked_example_reveal)
    val revealAllLabel = L.text(R.string.mobile_contentTools_tools_worked_example_revealAll)
    val statusRevealed = L.text(R.string.mobile_contentTools_tools_worked_example_statusRevealed)
    val statusSolved = L.text(R.string.mobile_contentTools_tools_worked_example_statusSolved)
    val statusCurrent = L.text(R.string.mobile_contentTools_tools_worked_example_statusCurrent)
    val statusLocked = L.text(R.string.mobile_contentTools_tools_worked_example_statusLocked)
    val statusScaffolded = L.text(R.string.mobile_contentTools_tools_worked_example_statusScaffolded)
    val statusComplete = L.text(R.string.mobile_contentTools_tools_worked_example_statusComplete)
    val correctLabel = L.text(R.string.mobile_contentTools_tools_worked_example_correct)
    val needsReviewLabel = L.text(R.string.mobile_contentTools_tools_worked_example_needsReview)
    val incorrectLabel = L.text(R.string.mobile_contentTools_tools_worked_example_incorrect)
    val advancedAnnounce = L.text(R.string.mobile_contentTools_tools_worked_example_advanced)
    val revealAllDone = L.text(R.string.mobile_contentTools_tools_worked_example_revealAllDone)
    val errorLabel = L.text(R.string.mobile_contentTools_tools_worked_example_error)

    val title = ContentToolHostLogic.stringField(props.config, "title")
        ?.trim()
        .orEmpty()
    val problem = ContentToolHostLogic.stringField(props.config, "problem").orEmpty()
    val allowRevealAll = ContentToolPack4Logic.boolField(props.config, "allowRevealAll") == true
    val allStepIds = remember(props.config) { ContentToolPack4Logic.parseStepIds(props.config) }
    val blanked = ContentToolPack4Logic.blankedStepIds(props.config, props.state)
    val progress = ContentToolPack4Logic.stepProgressMap(props.state)
    val currentStepId = ContentToolPack4Logic.resolveCurrentStepId(
        blankedStepIds = blanked,
        currentStepId = ContentToolHostLogic.stringField(props.state, "currentStepId"),
        progress = progress,
    )

    fun stepObject(stepId: String): JsonElement? =
        ContentToolPack4Logic.arrayField(props.config, "steps").firstOrNull { raw ->
            ContentToolHostLogic.stringField(raw, "id") == stepId
        }

    fun statusLabel(status: ContentToolPack4Logic.StepStatus): String =
        when (status) {
            ContentToolPack4Logic.StepStatus.SOLVED -> statusSolved
            ContentToolPack4Logic.StepStatus.REVEALED -> statusRevealed
            ContentToolPack4Logic.StepStatus.CURRENT -> statusCurrent
            ContentToolPack4Logic.StepStatus.LOCKED -> statusLocked
            ContentToolPack4Logic.StepStatus.SCAFFOLDED -> statusScaffolded
            ContentToolPack4Logic.StepStatus.ALL_COMPLETE -> statusComplete
        }

    fun resultLabel(result: String): String =
        when (result) {
            "correct" -> correctLabel
            "needs_review" -> needsReviewLabel
            else -> incorrectLabel
        }

    fun hydrate() {
        val sp = progress[currentStepId]
        ContentToolHostLogic.stringField(sp, "draft")?.let { draft = it }
        if (ContentToolPack4Logic.arrayField(props.state, "blankedStepIds").isNotEmpty()) {
            prepared = true
        }
    }

    fun persistDraft(value: String) {
        if (props.readOnly || currentStepId.isEmpty()) return
        val base = ContentToolPack4Logic.objectMap(props.state)
        val patch = ContentToolPack4Logic.mergeStepDraft(base, currentStepId, value)
        props.save(patch)
    }

    fun prepareIfNeeded() {
        if (prepared || props.readOnly) return
        busy = true
        scope.launch {
            try {
                props.runAction("prepare", buildJsonObject { })
                prepared = true
            } catch (_: Exception) {
                prepared = true
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    fun checkStep() {
        if (!ContentToolPack4Logic.canCheckStep(draft, props.readOnly, busy, stepDone = false)) return
        busy = true
        errorText = null
        scope.launch {
            try {
                val result = props.runAction(
                    "check_step",
                    buildJsonObject {
                        put("stepId", JsonPrimitive(currentStepId))
                        put("value", JsonPrimitive(draft))
                    },
                )
                val parsed = ContentToolPack4Logic.parseCheckStepResult(result)
                lastResult = parsed.result
                lastFeedback = parsed.feedback
                if (parsed.result == "correct" || parsed.result == "needs_review") {
                    props.announce(advancedAnnounce, false)
                    hintText = null
                    revealText = null
                    draft = ""
                } else {
                    props.announce(incorrectLabel, true)
                }
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    fun hint() {
        if (props.readOnly || busy) return
        busy = true
        scope.launch {
            try {
                val result = props.runAction(
                    "hint",
                    buildJsonObject { put("stepId", JsonPrimitive(currentStepId)) },
                )
                val parsed = ContentToolPack4Logic.parseHintResult(result)
                hintText = parsed.hint
                parsed.hint?.let { props.announce(it, false) }
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    fun revealStep() {
        if (props.readOnly || busy) return
        busy = true
        scope.launch {
            try {
                val result = props.runAction(
                    "reveal_step",
                    buildJsonObject { put("stepId", JsonPrimitive(currentStepId)) },
                )
                revealText = ContentToolHostLogic.stringField(result, "explanation")
                    ?: ContentToolHostLogic.stringField(result, "expectedDisplay")
                props.announce(statusRevealed, false)
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    fun revealAll() {
        if (props.readOnly || busy) return
        busy = true
        scope.launch {
            try {
                props.runAction("reveal_all", buildJsonObject { })
                props.announce(revealAllDone, false)
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.instanceId) {
        hydrate()
        prepareIfNeeded()
    }
    LaunchedEffect(props.state) { hydrate() }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        if (title.isNotEmpty()) {
            Text(
                text = title,
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }
        if (problem.isNotEmpty()) {
            NotebookContentView(markdown = problem, compact = true)
        }

        allStepIds.forEach { stepId ->
            val status = ContentToolPack4Logic.stepStatus(
                stepId = stepId,
                blankedStepIds = blanked,
                currentStepId = currentStepId,
                progress = progress,
                allStepIds = allStepIds,
            )
            val step = stepObject(stepId)
            val label = ContentToolHostLogic.stringField(step, "label")
                ?: ContentToolHostLogic.stringField(step, "text")
                ?: stepId
            val (icon, tint) = statusIcon(status)

            Column(
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier
                    .fillMaxWidth()
                    .alpha(if (status == ContentToolPack4Logic.StepStatus.LOCKED) 0.55f else 1f)
                    .semantics {
                        contentDescription = "${statusLabel(status)}: $label"
                    },
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.Top,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Icon(
                        imageVector = icon,
                        contentDescription = null,
                        tint = tint,
                    )
                    NotebookContentView(
                        markdown = label,
                        compact = true,
                        modifier = Modifier.weight(1f),
                    )
                }

                when (status) {
                    ContentToolPack4Logic.StepStatus.CURRENT -> {
                        OutlinedTextField(
                            value = draft,
                            onValueChange = {
                                draft = it
                                persistDraft(it)
                            },
                            enabled = !props.readOnly && !busy,
                            label = { Text(answerPlaceholder) },
                            modifier = Modifier.fillMaxWidth(),
                            minLines = 1,
                            maxLines = 4,
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            Button(
                                onClick = { checkStep() },
                                enabled = ContentToolPack4Logic.canCheckStep(
                                    draft = draft,
                                    readOnly = props.readOnly,
                                    busy = busy,
                                    stepDone = false,
                                ),
                            ) {
                                Text(checkLabel)
                            }
                            TextButton(
                                onClick = { hint() },
                                enabled = !props.readOnly && !busy,
                            ) {
                                Text(hintLabel)
                            }
                            TextButton(
                                onClick = { revealStep() },
                                enabled = !props.readOnly && !busy,
                            ) {
                                Text(revealLabel)
                            }
                        }
                        lastResult?.let { result ->
                            Text(
                                text = resultLabel(result),
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = if (result == "correct") LexturesColors.Primary else LexturesColors.Coral,
                            )
                        }
                        lastFeedback?.takeIf { it.isNotEmpty() }?.let {
                            NotebookContentView(markdown = it, compact = true)
                        }
                        hintText?.takeIf { it.isNotEmpty() }?.let {
                            NotebookContentView(markdown = it, compact = true)
                        }
                        revealText?.takeIf { it.isNotEmpty() }?.let {
                            NotebookContentView(markdown = it, compact = true)
                        }
                        if (allowRevealAll) {
                            TextButton(
                                onClick = { revealAll() },
                                enabled = !props.readOnly && !busy,
                            ) {
                                Text(revealAllLabel)
                            }
                        }
                    }
                    ContentToolPack4Logic.StepStatus.REVEALED -> {
                        Text(text = statusRevealed, fontSize = 11.sp, color = textSecondary())
                    }
                    ContentToolPack4Logic.StepStatus.SOLVED -> {
                        Text(text = statusSolved, fontSize = 11.sp, color = LexturesColors.Primary)
                    }
                    ContentToolPack4Logic.StepStatus.LOCKED,
                    ContentToolPack4Logic.StepStatus.SCAFFOLDED,
                    ContentToolPack4Logic.StepStatus.ALL_COMPLETE,
                    -> Unit
                }
            }
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}

@Composable
private fun statusIcon(
    status: ContentToolPack4Logic.StepStatus,
): Pair<ImageVector, Color> =
    when (status) {
        ContentToolPack4Logic.StepStatus.SOLVED ->
            Icons.Default.CheckCircle to LexturesColors.Primary
        ContentToolPack4Logic.StepStatus.REVEALED ->
            Icons.Default.Visibility to textSecondary()
        ContentToolPack4Logic.StepStatus.CURRENT ->
            Icons.Default.Edit to accentColor()
        ContentToolPack4Logic.StepStatus.SCAFFOLDED ->
            Icons.Default.MenuBook to textSecondary()
        ContentToolPack4Logic.StepStatus.LOCKED,
        ContentToolPack4Logic.StepStatus.ALL_COMPLETE,
        -> Icons.Default.Lock to textSecondary()
    }
