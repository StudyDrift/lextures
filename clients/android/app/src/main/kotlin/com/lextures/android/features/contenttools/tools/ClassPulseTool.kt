package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.lextures.android.R
import com.lextures.android.core.design.Haptics
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack1Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlin.math.roundToInt

private data class PulseAggregate(
    val suppressed: Boolean,
    val learners: Int,
    val options: List<Triple<String, Int, Int?>>,
    val correctOptionId: String? = null,
)

@Composable
fun ClassPulseTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val lifecycleOwner = LocalLifecycleOwner.current

    var selectedOptionId by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var aggregate by remember { mutableStateOf<PulseAggregate?>(null) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var consecutiveFailures by remember { mutableIntStateOf(0) }
    var lifecycleStarted by remember {
        mutableStateOf(lifecycleOwner.lifecycle.currentState.isAtLeast(Lifecycle.State.STARTED))
    }

    val needsConnection = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val chooseOptionLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_chooseOption)
    val submitVoteLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_submitVote)
    val resultsLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_results)
    val suppressedLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_suppressed)
    val waitingLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_waitingForMore)
    val yourAnswerLabel = L.text(R.string.mobile_contentTools_tools_class_pulse_yourAnswer)
    val resultsAnnounce = L.text(R.string.mobile_contentTools_tools_class_pulse_resultsAnnounce)

    val question = ContentToolHostLogic.stringField(props.config, "question").orEmpty()
    val options = remember(props.config) {
        ContentToolPack1Logic.arrayField(props.config, "options").mapNotNull { raw ->
            val o = ContentToolPack1Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            id to text
        }
    }
    val showPercentages = ContentToolPack1Logic.boolField(props.config, "showPercentages") != false
    val votes = ContentToolPack1Logic.arrayField(props.state, "votes")
    val hasVoted = ContentToolPack1Logic.hasVoted(votes, round = 1)

    fun applyAggregate(raw: JsonElement?) {
        val obj = raw?.let { runCatching { it.jsonObject }.getOrNull() } ?: return
        val source = obj["aggregate"] ?: raw
        val agg = source?.let { runCatching { it.jsonObject }.getOrNull() } ?: return
        val suppressed = ContentToolPack1Logic.boolField(agg, "suppressed") == true
        val learners = ContentToolPack1Logic.numberField(agg, "learners")?.toInt() ?: 0
        val rows = ContentToolPack1Logic.arrayField(agg, "options").mapNotNull { opt ->
            val o = ContentToolPack1Logic.objectMap(opt)
            val id = (o["optionId"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val count = ContentToolPack1Logic.numberField(opt, "count")?.toInt() ?: 0
            val percent = ContentToolPack1Logic.numberField(opt, "percent")?.roundToInt()
            Triple(id, count, percent)
        }
        val reveal = obj["reveal"]?.let { runCatching { it.jsonObject }.getOrNull() }
        val correctId = (reveal?.get("correctOptionId") as? JsonPrimitive)?.contentOrNull
        aggregate = PulseAggregate(suppressed, learners, rows, correctId)
    }

    suspend fun fetchAggregate() {
        try {
            val raw = props.runAction("aggregate", buildJsonObject { })
            applyAggregate(raw)
            consecutiveFailures = 0
        } catch (_: Exception) {
            consecutiveFailures += 1
        }
    }

    fun hydrate() {
        val draft = ContentToolPack1Logic.objectMap(
            ContentToolPack1Logic.objectMap(props.state)["draft"],
        )
        (draft["optionId"] as? JsonPrimitive)?.contentOrNull?.let { selectedOptionId = it }
    }

    fun persistDraft() {
        if (props.readOnly) return
        val patch = ContentToolPack1Logic.mergePreservingUnknown(
            ContentToolPack1Logic.objectMap(props.state),
            mapOf(
                "v" to JsonPrimitive(1),
                "draft" to JsonObject(
                    mapOf(
                        "optionId" to JsonPrimitive(selectedOptionId),
                        "round" to JsonPrimitive(1),
                    ),
                ),
            ),
        )
        props.save(patch)
    }

    fun vote() {
        if (busy || props.readOnly) return
        busy = true
        errorText = null
        scope.launch {
            try {
                val raw = props.runAction(
                    "vote",
                    buildJsonObject {
                        put("optionId", JsonPrimitive(selectedOptionId))
                        put("round", JsonPrimitive(1))
                    },
                )
                applyAggregate(raw)
                Haptics.trigger(view, Haptics.Kind.Success)
                props.announce(resultsAnnounce, false)
            } catch (_: Exception) {
                // Needs connection — do NOT queue.
                errorText = needsConnection
                props.announce(needsConnection, true)
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.state) {
        hydrate()
        if (hasVoted && aggregate == null) {
            fetchAggregate()
        }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> lifecycleStarted = true
                Lifecycle.Event.ON_STOP -> lifecycleStarted = false
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
            lifecycleStarted = false
        }
    }

    LaunchedEffect(hasVoted, lifecycleStarted) {
        if (!ContentToolPack1Logic.shouldPollAggregate(lifecycleStarted, hasVoted)) return@LaunchedEffect
        while (true) {
            delay(ContentToolPack1Logic.nextPollDelayMs(consecutiveFailures).toLong())
            if (!lifecycleStarted) continue
            fetchAggregate()
        }
    }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        NotebookContentView(markdown = question, compact = true)

        if (!hasVoted) {
            Text(
                text = chooseOptionLabel,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            options.forEach { (id, optionText) ->
                val selected = selectedOptionId == id
                Row(
                    Modifier
                        .fillMaxWidth()
                        .heightIn(min = 44.dp)
                        .semantics { this.selected = selected }
                        .clickable(
                            enabled = !props.readOnly && !busy,
                            role = Role.RadioButton,
                        ) {
                            selectedOptionId = id
                            persistDraft()
                        },
                    verticalAlignment = Alignment.Top,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    RadioButton(
                        selected = selected,
                        onClick = null,
                        enabled = !props.readOnly && !busy,
                    )
                    NotebookContentView(markdown = optionText, compact = true)
                }
            }
            Button(
                onClick = { vote() },
                enabled = !props.readOnly && !busy && selectedOptionId.isNotEmpty(),
            ) {
                Text(submitVoteLabel)
            }
        } else {
            val agg = aggregate
            if (agg != null) {
                Text(
                    text = resultsLabel,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                if (agg.suppressed) {
                    Text(text = suppressedLabel, fontSize = 12.sp, color = textSecondary())
                } else {
                    Text(
                        text = L.format(
                            context,
                            localePrefs,
                            R.string.mobile_contentTools_tools_class_pulse_respondents,
                            agg.learners,
                        ),
                        fontSize = 11.sp,
                        color = textSecondary(),
                    )
                    agg.options.forEach { (optionId, count, percent) ->
                        val rowLabel = options.firstOrNull { it.first == optionId }?.second ?: optionId
                        val yours = votes.any { vote ->
                            ContentToolHostLogic.stringField(vote, "optionId") == optionId
                        }
                        val yoursSuffix = if (yours) " ($yourAnswerLabel)" else ""
                        val line = if (showPercentages && percent != null) {
                            "$rowLabel: $percent%$yoursSuffix"
                        } else {
                            "$rowLabel: $count$yoursSuffix"
                        }
                        Text(text = line, fontSize = 12.sp, color = textPrimary())
                    }
                }
                agg.correctOptionId?.let { correctId ->
                    val answerLabel = options.firstOrNull { it.first == correctId }?.second ?: correctId
                    Text(
                        text = L.format(
                            context,
                            localePrefs,
                            R.string.mobile_contentTools_tools_class_pulse_correctAnswer,
                            answerLabel,
                        ),
                        fontSize = 12.sp,
                        color = textPrimary(),
                    )
                }
            } else {
                Text(text = waitingLabel, fontSize = 12.sp, color = textSecondary())
            }
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
