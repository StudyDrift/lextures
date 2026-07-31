package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Slider
import androidx.compose.material3.Switch
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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack4Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlin.math.max
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull

private data class NumberParam(
    val id: String,
    val label: String,
    val unit: String?,
    val min: Double,
    val max: Double,
    val step: Double,
)

private data class BoolParam(
    val id: String,
    val label: String,
)

private data class ChoiceParam(
    val id: String,
    val label: String,
    val options: List<Pair<String, String>>,
)

private data class PromptItem(
    val id: String,
    val text: String,
    val kind: String,
    val options: List<Pair<String, String>>,
    val unlockWhen: String?,
)

@Composable
fun ParameterExplorerTool(props: ContentToolRendererProps) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current

    var params by remember(props.instanceId) { mutableStateOf<Map<String, JsonElement>>(emptyMap()) }
    var dragging by remember { mutableStateOf(false) }
    var dirty by remember { mutableStateOf(false) }
    var lastRecomputeMs by remember { mutableStateOf<Long?>(null) }
    var plotPoints by remember { mutableStateOf<List<Pair<Double, Double>>>(emptyList()) }
    val answerDrafts = remember(props.instanceId) { mutableStateMapOf<String, String>() }
    val numberTextEntries = remember(props.instanceId) { mutableStateMapOf<String, String>() }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var statusText by remember { mutableStateOf<String?>(null) }

    val resetDefaultsLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_resetDefaults)
    val dataTableLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_dataTable)
    val lockedLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_locked)
    val answerPlaceholder = L.text(R.string.mobile_contentTools_tools_parameter_explorer_answerPlaceholder)
    val submitAnswerLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_submitAnswer)
    val answeredLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_answered)
    val checkpointReached = L.text(R.string.mobile_contentTools_tools_parameter_explorer_checkpointReached)
    val resetAnnounce = L.text(R.string.mobile_contentTools_tools_parameter_explorer_resetAnnounce)
    val errorLabel = L.text(R.string.mobile_contentTools_tools_parameter_explorer_error)

    val prompt = ContentToolHostLogic.stringField(props.config, "prompt").orEmpty()
    val numberParams = remember(props.config) {
        ContentToolPack4Logic.arrayField(props.config, "parameters").mapNotNull { raw ->
            val obj = ContentToolPack4Logic.objectMap(raw)
            if ((obj["kind"] as? JsonPrimitive)?.contentOrNull != "number") return@mapNotNull null
            val id = (obj["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            NumberParam(
                id = id,
                label = ContentToolHostLogic.stringField(raw, "label") ?: id,
                unit = ContentToolHostLogic.stringField(raw, "unit"),
                min = ContentToolPack4Logic.numberField(raw, "min") ?: 0.0,
                max = ContentToolPack4Logic.numberField(raw, "max") ?: 1.0,
                step = ContentToolPack4Logic.numberField(raw, "step") ?: 0.1,
            )
        }
    }
    val booleanParams = remember(props.config) {
        ContentToolPack4Logic.arrayField(props.config, "parameters").mapNotNull { raw ->
            val obj = ContentToolPack4Logic.objectMap(raw)
            if ((obj["kind"] as? JsonPrimitive)?.contentOrNull != "boolean") return@mapNotNull null
            val id = (obj["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            BoolParam(id = id, label = ContentToolHostLogic.stringField(raw, "label") ?: id)
        }
    }
    val choiceParams = remember(props.config) {
        ContentToolPack4Logic.arrayField(props.config, "parameters").mapNotNull { raw ->
            val obj = ContentToolPack4Logic.objectMap(raw)
            if ((obj["kind"] as? JsonPrimitive)?.contentOrNull != "choice") return@mapNotNull null
            val id = (obj["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val options = ContentToolPack4Logic.arrayField(raw, "options").mapNotNull { opt ->
                val o = ContentToolPack4Logic.objectMap(opt)
                val value = (o["value"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                value to (ContentToolHostLogic.stringField(opt, "label") ?: value)
            }
            ChoiceParam(
                id = id,
                label = ContentToolHostLogic.stringField(raw, "label") ?: id,
                options = options,
            )
        }
    }
    val noticingPrompts = remember(props.config) {
        ContentToolPack4Logic.arrayField(props.config, "noticingPrompts").mapNotNull { raw ->
            val id = ContentToolPack4Logic.objectMap(raw)["id"]
                .let { (it as? JsonPrimitive)?.contentOrNull }
                ?: return@mapNotNull null
            val options = ContentToolPack4Logic.arrayField(raw, "options").mapNotNull { opt ->
                val o = ContentToolPack4Logic.objectMap(opt)
                val value = (o["value"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                value to (ContentToolHostLogic.stringField(opt, "label") ?: value)
            }
            PromptItem(
                id = id,
                text = ContentToolHostLogic.stringField(raw, "text") ?: id,
                kind = ContentToolHostLogic.stringField(raw, "kind") ?: "text",
                options = options,
                unlockWhen = ContentToolHostLogic.stringField(raw, "unlockWhen"),
            )
        }
    }
    val checkpoints = ContentToolPack4Logic.objectMap(
        ContentToolPack4Logic.objectMap(props.state)["checkpoints"],
    )
    val answers = ContentToolPack4Logic.objectMap(
        ContentToolPack4Logic.objectMap(props.state)["answers"],
    )

    fun numberValue(id: String): Double? =
        ContentToolPack4Logic.numberField(JsonObject(params), id)

    fun evaluateUnlock(expr: String?): Boolean {
        if (expr == null) return true
        val trimmed = expr.trim()
        val match = Regex("""^a\s*>\s*([0-9.]+)$""").matchEntire(trimmed)
        if (match == null) return checkpoints.isNotEmpty()
        val threshold = match.groupValues.getOrNull(1)?.toDoubleOrNull() ?: return false
        return (numberValue("a") ?: 0.0) > threshold
    }

    fun settle() {
        if (!ContentToolPack4Logic.shouldAutosaveOnSettle(dragging, dirty)) return
        val base = ContentToolPack4Logic.objectMap(props.state)
        val patch = ContentToolPack4Logic.mergeParamsPreservingUnknown(base, params)
        props.save(patch)
        dirty = false
    }

    fun maybeCheckpoint() {
        noticingPrompts.forEach { item ->
            val unlock = item.unlockWhen ?: return@forEach
            if (checkpoints[item.id] != null) return@forEach
            if (!evaluateUnlock(unlock)) return@forEach
            scope.launch {
                try {
                    props.runAction(
                        "checkpoint",
                        buildJsonObject {
                            put("promptId", JsonPrimitive(item.id))
                            put("params", JsonObject(params))
                        },
                    )
                    props.announce(checkpointReached, false)
                } catch (_: Exception) {
                    // Non-fatal — unlock UI still works locally.
                }
            }
        }
    }

    fun recompute(force: Boolean) {
        val now = System.currentTimeMillis()
        if (!force && !ContentToolPack4Logic.shouldRecompute(lastRecomputeMs, now)) return
        lastRecomputeMs = now
        val a = numberValue("a") ?: 1.0
        val b = numberValue("b") ?: 0.0
        val c = numberValue("c") ?: 0.0
        val pts = (0..40).map { i ->
            val x = -10.0 + (20.0 * i) / 40.0
            x to (a * x * x + b * x + c)
        }
        plotPoints = pts
        maybeCheckpoint()
    }

    fun setNumber(id: String, value: Double, isDragging: Boolean) {
        params = params + (id to JsonPrimitive(value))
        dragging = isDragging
        dirty = true
        recompute(force = false)
        props.announce(
            L.format(
                context,
                localePrefs,
                R.string.mobile_contentTools_tools_parameter_explorer_valueAnnounce,
                id,
                value,
            ),
            false,
        )
    }

    fun hydrate() {
        val stateParams = ContentToolPack4Logic.objectMap(
            ContentToolPack4Logic.objectMap(props.state)["params"],
        )
        params = if (stateParams.isEmpty()) {
            ContentToolPack4Logic.defaultParams(props.config)
        } else {
            stateParams
        }
        answers.forEach { (id, value) ->
            (value as? JsonPrimitive)?.contentOrNull?.let { answerDrafts[id] = it }
        }
        if (!dragging) {
            numberParams.forEach { param ->
                val v = numberValue(param.id) ?: param.min
                numberTextEntries[param.id] = v.toString()
            }
        }
    }

    fun submitAnswer(promptId: String) {
        val answer = answerDrafts[promptId]?.trim().orEmpty()
        if (answer.isEmpty()) return
        busy = true
        errorText = null
        scope.launch {
            try {
                props.runAction(
                    "submit_answer",
                    buildJsonObject {
                        put("promptId", JsonPrimitive(promptId))
                        put("answer", JsonPrimitive(answer))
                        put("params", JsonObject(params))
                    },
                )
                statusText = answeredLabel
                props.announce(answeredLabel, false)
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    fun resetDefaults() {
        busy = true
        errorText = null
        scope.launch {
            try {
                props.runAction("reset_defaults", buildJsonObject { })
                params = ContentToolPack4Logic.defaultParams(props.config)
                numberParams.forEach { param ->
                    numberTextEntries[param.id] = (numberValue(param.id) ?: param.min).toString()
                }
                dirty = false
                recompute(force = true)
                props.announce(resetAnnounce, false)
            } catch (_: Exception) {
                params = ContentToolPack4Logic.defaultParams(props.config)
                numberParams.forEach { param ->
                    numberTextEntries[param.id] = (numberValue(param.id) ?: param.min).toString()
                }
                dirty = true
                settle()
                recompute(force = true)
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.instanceId) {
        hydrate()
        recompute(force = true)
    }
    LaunchedEffect(props.state) {
        if (!dragging) hydrate()
    }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        if (prompt.isNotEmpty()) {
            NotebookContentView(markdown = prompt, compact = true)
        }

        numberParams.forEach { param ->
            val value = numberValue(param.id) ?: param.min
            val title = param.unit?.let { "${param.label} ($it)" } ?: param.label
            val textEntry = numberTextEntries[param.id] ?: value.toString()
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    text = title,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    val stepSize = max(param.step, 0.0001)
                    val steps = max(0, (((param.max - param.min) / stepSize).toInt()) - 1)
                    Slider(
                        value = value.toFloat(),
                        onValueChange = {
                            val clamped = ContentToolPack4Logic.clampNumber(
                                it.toDouble(),
                                param.min,
                                param.max,
                                param.step,
                            )
                            setNumber(param.id, clamped, isDragging = true)
                            numberTextEntries[param.id] = clamped.toString()
                        },
                        onValueChangeFinished = {
                            dragging = false
                            settle()
                        },
                        valueRange = param.min.toFloat()..param.max.toFloat(),
                        steps = steps,
                        enabled = !props.readOnly,
                        modifier = Modifier
                            .weight(1f)
                            .semantics {
                                contentDescription = title
                            },
                    )
                    OutlinedTextField(
                        value = textEntry,
                        onValueChange = { raw ->
                            numberTextEntries[param.id] = raw
                            val parsed = raw.toDoubleOrNull() ?: return@OutlinedTextField
                            val clamped = ContentToolPack4Logic.clampNumber(
                                parsed,
                                param.min,
                                param.max,
                                param.step,
                            )
                            setNumber(param.id, clamped, isDragging = false)
                            settle()
                        },
                        enabled = !props.readOnly,
                        singleLine = true,
                        modifier = Modifier
                            .width(88.dp)
                            .semantics {
                                contentDescription = L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_contentTools_tools_parameter_explorer_directEntry,
                                    param.label,
                                )
                            },
                    )
                }
                Text(
                    text = L.format(
                        R.string.mobile_contentTools_tools_parameter_explorer_sliderSemantics,
                        value,
                        param.min,
                        param.max,
                        param.step,
                    ),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
            }
        }

        booleanParams.forEach { param ->
            val checked = (params[param.id] as? JsonPrimitive)?.booleanOrNull ?: false
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 44.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(text = param.label, color = textPrimary(), modifier = Modifier.weight(1f))
                Switch(
                    checked = checked,
                    onCheckedChange = { newValue ->
                        params = params + (param.id to JsonPrimitive(newValue))
                        dirty = true
                        recompute(force = true)
                        settle()
                    },
                    enabled = !props.readOnly,
                )
            }
        }

        choiceParams.forEach { param ->
            val selected = (params[param.id] as? JsonPrimitive)?.contentOrNull
                ?: param.options.firstOrNull()?.first.orEmpty()
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    text = param.label,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                param.options.forEach { (value, optLabel) ->
                    val isSelected = selected == value
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .heightIn(min = 44.dp)
                            .semantics { this.selected = isSelected }
                            .clickable(enabled = !props.readOnly, role = Role.RadioButton) {
                                params = params + (param.id to JsonPrimitive(value))
                                dirty = true
                                recompute(force = true)
                                settle()
                            },
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        RadioButton(selected = isSelected, onClick = null, enabled = !props.readOnly)
                        Text(text = optLabel, color = textPrimary())
                    }
                }
            }
        }

        if (plotPoints.isNotEmpty()) {
            val first = plotPoints.first()
            val last = plotPoints.last()
            Text(
                text = L.format(
                    R.string.mobile_contentTools_tools_parameter_explorer_plotSummary,
                    first.first,
                    last.first,
                    first.second,
                    last.second,
                ),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState())
                    .semantics { contentDescription = dataTableLabel },
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("x", fontSize = 12.sp, fontWeight = FontWeight.SemiBold, modifier = Modifier.width(64.dp))
                    Text("y", fontSize = 12.sp, fontWeight = FontWeight.SemiBold, modifier = Modifier.width(80.dp))
                }
                plotPoints.forEachIndexed { index, pt ->
                    if (index % 4 != 0) return@forEachIndexed
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            text = String.format("%.2f", pt.first),
                            fontSize = 11.sp,
                            color = textPrimary(),
                            modifier = Modifier.width(64.dp),
                        )
                        Text(
                            text = String.format("%.3f", pt.second),
                            fontSize = 11.sp,
                            color = textPrimary(),
                            modifier = Modifier.width(80.dp),
                        )
                    }
                }
            }
        }

        noticingPrompts.forEach { item ->
            val unlocked = item.unlockWhen == null ||
                checkpoints[item.id] != null ||
                evaluateUnlock(item.unlockWhen)
            val answered = (answers[item.id] as? JsonPrimitive)?.contentOrNull?.isNotEmpty() == true
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = item.text,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                if (!unlocked) {
                    Text(text = lockedLabel, fontSize = 11.sp, color = textSecondary())
                } else {
                    OutlinedTextField(
                        value = answerDrafts[item.id].orEmpty(),
                        onValueChange = { answerDrafts[item.id] = it },
                        enabled = !props.readOnly && !busy,
                        label = { Text(answerPlaceholder) },
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 2,
                        maxLines = 5,
                    )
                    Button(
                        onClick = { submitAnswer(item.id) },
                        enabled = !props.readOnly &&
                            !busy &&
                            answerDrafts[item.id].orEmpty().trim().isNotEmpty(),
                    ) {
                        Text(submitAnswerLabel)
                    }
                    if (answered) {
                        Text(text = answeredLabel, fontSize = 11.sp, color = LexturesColors.Primary)
                    }
                }
            }
        }

        Row(
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Button(onClick = { resetDefaults() }, enabled = !props.readOnly && !busy) {
                Text(resetDefaultsLabel)
            }
            statusText?.let {
                Text(text = it, fontSize = 12.sp, color = textSecondary())
            }
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
