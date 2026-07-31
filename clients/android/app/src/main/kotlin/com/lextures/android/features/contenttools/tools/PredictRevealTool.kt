package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.Haptics
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack1Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonObject

@Composable
fun PredictRevealTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val scope = rememberCoroutineScope()

    var outcomeId by remember(props.instanceId) { mutableStateOf("") }
    var text by remember(props.instanceId) { mutableStateOf("") }
    var confidence by remember(props.instanceId) { mutableStateOf<Double?>(null) }
    var reflection by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var revealMarkdown by remember { mutableStateOf<String?>(null) }
    var peerRows by remember { mutableStateOf<List<Pair<String, Int>>>(emptyList()) }
    var peersSuppressed by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    val needsConnection = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val yourPredictionLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_yourPrediction)
    val howSureLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_howSure)
    val commitLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_commit)
    val commitHelper = L.text(R.string.mobile_contentTools_tools_predict_reveal_commitHelper)
    val whatHappensLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_whatHappens)
    val peersSuppressedLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_peersSuppressed)
    val peerResultsLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_peerResults)
    val reflectionPlaceholder = L.text(R.string.mobile_contentTools_tools_predict_reveal_reflectionPlaceholder)
    val saveReflectionLabel = L.text(R.string.mobile_contentTools_tools_predict_reveal_saveReflection)
    val openPlaceholderDefault = L.text(R.string.mobile_contentTools_tools_predict_reveal_openPlaceholder)
    val revealedAnnounce = L.text(R.string.mobile_contentTools_tools_predict_reveal_revealedAnnounce)
    val confidenceGuessing = L.text(R.string.mobile_contentTools_tools_predict_reveal_confidence_guessing)
    val confidenceFairlySure = L.text(R.string.mobile_contentTools_tools_predict_reveal_confidence_fairlySure)
    val confidenceCertain = L.text(R.string.mobile_contentTools_tools_predict_reveal_confidence_certain)

    val question = ContentToolHostLogic.stringField(props.config, "question").orEmpty()
    val mode = if (ContentToolHostLogic.stringField(props.config, "mode") == "open") "open" else "choice"
    val confidenceScale = ContentToolHostLogic.stringField(props.config, "confidenceScale") ?: "three"
    val confidenceRequired = ContentToolPack1Logic.boolField(props.config, "confidenceRequired") != false
    val reflectionPrompt = ContentToolHostLogic.stringField(props.config, "reflectionPrompt").orEmpty()
    val outcomes = remember(props.config) {
        ContentToolPack1Logic.arrayField(props.config, "outcomes").mapNotNull { raw ->
            val o = ContentToolPack1Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val t = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            id to t
        }
    }
    val committed = ContentToolPack1Logic.isCommitted(props.state)
    val canEdit = ContentToolPack1Logic.canEditPrediction(committed, props.readOnly)
    val confidenceOptions = when (confidenceScale) {
        "none" -> emptyList()
        "five" -> (1..5).map { it.toDouble() to it.toString() }
        "percent" -> listOf(0, 25, 50, 75, 100).map { it.toDouble() to "$it%" }
        else -> listOf(
            1.0 to confidenceGuessing,
            2.0 to confidenceFairlySure,
            3.0 to confidenceCertain,
        )
    }
    val canCommit = when {
        mode == "open" && text.trim().isEmpty() -> false
        mode != "open" && outcomeId.isEmpty() -> false
        confidenceRequired && confidenceScale != "none" && confidence == null -> false
        else -> true
    }

    fun hydrate() {
        val draft = ContentToolPack1Logic.objectMap(
            ContentToolPack1Logic.objectMap(props.state)["draft"],
        )
        val prediction = ContentToolPack1Logic.objectMap(
            ContentToolPack1Logic.objectMap(props.state)["prediction"],
        )
        (draft["outcomeId"] ?: prediction["outcomeId"])
            ?.let { (it as? JsonPrimitive)?.contentOrNull }
            ?.let { outcomeId = it }
        (draft["text"] ?: prediction["text"])
            ?.let { (it as? JsonPrimitive)?.contentOrNull }
            ?.let { text = it }
        (draft["confidence"] as? JsonPrimitive)?.doubleOrNull?.let { confidence = it }
        ContentToolHostLogic.stringField(props.state, "reflection")?.let { reflection = it }
    }

    fun persistDraft() {
        if (!canEdit) return
        val draft = buildMap {
            if (mode == "open") {
                put("text", JsonPrimitive(text))
            } else {
                put("outcomeId", JsonPrimitive(outcomeId))
            }
            confidence?.let { put("confidence", JsonPrimitive(it)) }
        }
        val patch = ContentToolPack1Logic.mergePreservingUnknown(
            ContentToolPack1Logic.objectMap(props.state),
            mapOf("v" to JsonPrimitive(1), "draft" to JsonObject(draft)),
        )
        props.save(patch)
    }

    fun applyCommitResult(raw: JsonElement?) {
        val obj = raw?.let { runCatching { it.jsonObject }.getOrNull() } ?: return
        val reveal = obj["reveal"]?.let { runCatching { it.jsonObject }.getOrNull() }
        (reveal?.get("markdown") as? JsonPrimitive)?.contentOrNull?.let { revealMarkdown = it }
        val peers = obj["peerResults"]?.let { runCatching { it.jsonObject }.getOrNull() }
        if (peers != null) {
            if (ContentToolPack1Logic.boolField(peers, "suppressed") == true) {
                peersSuppressed = true
                peerRows = emptyList()
            } else {
                peersSuppressed = false
                peerRows = ContentToolPack1Logic.arrayField(peers, "outcomes").mapNotNull { row ->
                    val o = ContentToolPack1Logic.objectMap(row)
                    val id = (o["outcomeId"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                    val count = ContentToolPack1Logic.numberField(row, "count")?.toInt() ?: 0
                    val label = outcomes.firstOrNull { it.first == id }?.second ?: id
                    label to count
                }
            }
        }
        val pred = ContentToolPack1Logic.objectMap(
            ContentToolPack1Logic.objectMap(props.state)["prediction"],
        )
        (pred["outcomeId"] as? JsonPrimitive)?.contentOrNull?.let { outcomeId = it }
        (pred["text"] as? JsonPrimitive)?.contentOrNull?.let { text = it }
    }

    fun commit(reloading: Boolean = false) {
        if (busy) return
        busy = true
        errorText = null
        scope.launch {
            try {
                val input = buildJsonObject {
                    if (!reloading) {
                        if (mode == "open") {
                            put("prediction", buildJsonObject { put("text", JsonPrimitive(text)) })
                        } else {
                            put(
                                "prediction",
                                buildJsonObject { put("outcomeId", JsonPrimitive(outcomeId)) },
                            )
                        }
                        confidence?.let { put("confidence", JsonPrimitive(it)) }
                    }
                }
                val raw = props.runAction("commit", input)
                applyCommitResult(raw)
                Haptics.trigger(view, Haptics.Kind.Success)
                if (revealMarkdown != null) {
                    props.announce(revealedAnnounce, false)
                }
            } catch (_: Exception) {
                errorText = needsConnection
                props.announce(needsConnection, true)
            } finally {
                busy = false
            }
        }
    }

    fun reflect() {
        if (busy || props.readOnly) return
        busy = true
        scope.launch {
            try {
                props.runAction(
                    "reflect",
                    buildJsonObject { put("text", JsonPrimitive(reflection)) },
                )
                Haptics.trigger(view, Haptics.Kind.Success)
            } catch (_: Exception) {
                errorText = needsConnection
                props.announce(needsConnection, true)
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.state) { hydrate() }
    LaunchedEffect(props.instanceId) {
        hydrate()
        if (ContentToolPack1Logic.isCommitted(props.state)) {
            commit(reloading = true)
        }
    }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        NotebookContentView(markdown = question, compact = true)

        if (canEdit) {
            Text(
                text = yourPredictionLabel,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            if (mode == "open") {
                OutlinedTextField(
                    value = text,
                    onValueChange = {
                        text = it
                        persistDraft()
                    },
                    enabled = !busy,
                    label = {
                        Text(
                            ContentToolHostLogic.stringField(props.config, "openPlaceholder")
                                ?: openPlaceholderDefault,
                        )
                    },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    maxLines = 6,
                )
            } else {
                outcomes.forEach { (id, outcomeText) ->
                    val selected = outcomeId == id
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .heightIn(min = 44.dp)
                            .semantics { this.selected = selected }
                            .clickable(enabled = !busy, role = Role.RadioButton) {
                                outcomeId = id
                                persistDraft()
                            },
                        verticalAlignment = Alignment.Top,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        RadioButton(selected = selected, onClick = null, enabled = !busy)
                        NotebookContentView(markdown = outcomeText, compact = true)
                    }
                }
            }
            if (confidenceOptions.isNotEmpty()) {
                Text(
                    text = howSureLabel,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                confidenceOptions.forEach { (value, optLabel) ->
                    val selected = confidence == value
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .heightIn(min = 44.dp)
                            .semantics { this.selected = selected }
                            .clickable(enabled = !busy, role = Role.RadioButton) {
                                confidence = value
                                persistDraft()
                            },
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        RadioButton(selected = selected, onClick = null, enabled = !busy)
                        Text(text = optLabel, color = textPrimary())
                    }
                }
            }
            Button(onClick = { commit() }, enabled = !busy && canCommit) {
                Text(commitLabel)
            }
            Text(text = commitHelper, fontSize = 11.sp, color = textSecondary())
        } else if (committed) {
            Text(
                text = yourPredictionLabel,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            if (mode == "open") {
                Text(text = text, color = textPrimary())
            } else {
                val locked = outcomes.firstOrNull { it.first == outcomeId }?.second ?: outcomeId
                NotebookContentView(markdown = locked, compact = true)
            }
            if (ContentToolPack1Logic.canShowReveal(committed = true, hasRevealPayload = revealMarkdown != null)) {
                revealMarkdown?.let { md ->
                    Text(
                        text = whatHappensLabel,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    NotebookContentView(markdown = md, compact = true)
                }
            } else if (revealMarkdown == null) {
                Button(onClick = { commit(reloading = true) }, enabled = !busy) {
                    Text(commitLabel)
                }
            }
            if (peersSuppressed) {
                Text(text = peersSuppressedLabel, fontSize = 12.sp, color = textSecondary())
            } else if (peerRows.isNotEmpty()) {
                Text(
                    text = peerResultsLabel,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                peerRows.forEach { (rowLabel, count) ->
                    Text(text = "$rowLabel: $count", fontSize = 12.sp, color = textPrimary())
                }
            }
            if (reflectionPrompt.isNotEmpty()) {
                Text(text = reflectionPrompt, fontSize = 12.sp, color = textSecondary())
                OutlinedTextField(
                    value = reflection,
                    onValueChange = { reflection = it },
                    enabled = !props.readOnly && !busy,
                    label = { Text(reflectionPlaceholder) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    maxLines = 6,
                )
                Button(
                    onClick = { reflect() },
                    enabled = !props.readOnly && !busy && reflection.trim().isNotEmpty(),
                ) {
                    Text(saveReflectionLabel)
                }
            }
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
