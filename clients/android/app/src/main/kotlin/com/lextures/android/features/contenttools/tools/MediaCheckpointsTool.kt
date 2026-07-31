package com.lextures.android.features.contenttools.tools

import android.net.Uri
import androidx.annotation.OptIn
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack4Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.contenttools.LocalContentToolsPage
import com.lextures.android.features.contenttools.ToolPlaceholder
import com.lextures.android.features.contenttools.ToolPlaceholderReason
import com.lextures.android.features.notebooks.NotebookContentView
import kotlin.math.abs
import kotlin.math.max
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull

private data class CheckpointQuestion(
    val type: String,
    val prompt: String,
    val options: List<Pair<String, String>>,
)

@OptIn(UnstableApi::class)
@Composable
fun MediaCheckpointsTool(props: ContentToolRendererProps) {
    val scope = rememberCoroutineScope()
    val page = LocalContentToolsPage.current
    val accessToken = page?.accessToken
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current

    var currentTime by remember { mutableDoubleStateOf(0.0) }
    var promptedIds by remember(props.instanceId) { mutableStateOf<Set<String>>(emptySet()) }
    var activeCheckpoint by remember(props.instanceId) {
        mutableStateOf<ContentToolPack4Logic.Checkpoint?>(null)
    }
    var answerDraft by remember { mutableStateOf("") }
    var selectedOptions by remember { mutableStateOf<Set<String>>(emptySet()) }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var feedbackText by remember { mutableStateOf<String?>(null) }
    var lastProgressMs by remember { mutableStateOf<Long?>(null) }
    var localSegments by remember(props.instanceId) {
        mutableStateOf<List<List<Double>>>(emptyList())
    }
    var segmentStart by remember { mutableStateOf<Double?>(null) }
    var blocked by remember { mutableStateOf(false) }
    var returnFromBackgroundPrompt by remember { mutableStateOf(false) }
    var captionsOn by remember { mutableStateOf(false) }
    var didResume by remember(props.instanceId) { mutableStateOf(false) }

    val unavailableBody = L.text(R.string.mobile_contentTools_tools_media_checkpoints_unavailableBody)
    val missingMedia = L.text(R.string.mobile_contentTools_tools_media_checkpoints_missingMedia)
    val playerLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_playerLabel)
    val captionsOnLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_captionsOn)
    val captionsOffLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_captionsOff)
    val blockedLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_blocked)
    val returnToApp = L.text(R.string.mobile_contentTools_tools_media_checkpoints_returnToApp)
    val submitLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_submit)
    val yourAnswerLabel = L.text(R.string.mobile_contentTools_runtime_yourAnswer)
    val correctLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_correct)
    val continueLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_continue)
    val incorrectLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_incorrect)
    val errorLabel = L.text(R.string.mobile_contentTools_tools_media_checkpoints_error)

    val mediaObj = ContentToolPack4Logic.objectMap(
        ContentToolPack4Logic.objectMap(props.config)["media"],
    )
    val mediaUrl = (mediaObj["url"] as? JsonPrimitive)?.contentOrNull
    val mediaSource = ContentToolHostLogic.stringField(JsonObject(mediaObj), "source")
    val mediaProvider = ContentToolHostLogic.stringField(JsonObject(mediaObj), "provider")
    val captionUrl = (mediaObj["captionUrl"] as? JsonPrimitive)?.contentOrNull
    val reliable = ContentToolPack4Logic.hasReliableCheckpointTiming(
        source = mediaSource,
        url = mediaUrl,
        provider = mediaProvider,
    )
    val preventSkip = ContentToolPack4Logic.boolField(props.config, "preventSkipPastUnanswered") == true
    val checkpoints = remember(props.config) { ContentToolPack4Logic.parseCheckpoints(props.config) }
    val answers = ContentToolPack4Logic.parseAnswers(props.state)

    fun formatTime(sec: Double): String {
        val s = max(0, sec.toInt())
        val m = s / 60
        val r = s % 60
        return "%d:%02d".format(m, r)
    }

    fun checkpointQuestion(id: String): CheckpointQuestion {
        val cps = ContentToolPack4Logic.arrayField(props.config, "checkpoints")
        val raw = cps.firstOrNull { ContentToolHostLogic.stringField(it, "id") == id }
        val q = ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(raw)["question"])
        val type = (q["type"] as? JsonPrimitive)?.contentOrNull ?: "single"
        val promptText = (q["prompt"] as? JsonPrimitive)?.contentOrNull.orEmpty()
        val options = ContentToolPack4Logic.arrayField(JsonObject(q), "options").mapNotNull { opt ->
            val o = ContentToolPack4Logic.objectMap(opt)
            val oid = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            oid to text
        }
        return CheckpointQuestion(type = type, prompt = promptText, options = options)
    }

    fun canSubmit(q: CheckpointQuestion): Boolean =
        when (q.type) {
            "multi" -> selectedOptions.isNotEmpty()
            else -> answerDraft.trim().isNotEmpty()
        }

    fun hydrate() {
        localSegments = ContentToolPack4Logic.parseWatchedSegments(props.state)
        val active = activeCheckpoint
        if (active != null && ContentToolPack4Logic.isCheckpointDone(answers, active)) {
            activeCheckpoint = null
            blocked = false
            returnFromBackgroundPrompt = false
        }
    }

    val resolvedUrl = remember(mediaUrl) {
        mediaUrl?.let { url ->
            if (url.startsWith("http")) url else AppConfiguration.apiUrl(url).toString()
        }
    }

    val player = remember(resolvedUrl, accessToken) {
        if (resolvedUrl.isNullOrBlank()) return@remember null
        val dataSourceFactory = DefaultHttpDataSource.Factory().apply {
            if (!accessToken.isNullOrBlank()) {
                setDefaultRequestProperties(mapOf("Authorization" to "Bearer $accessToken"))
            }
        }
        val mediaSourceFactory = ProgressiveMediaSource.Factory(dataSourceFactory)
            .createMediaSource(MediaItem.fromUri(resolvedUrl))
        ExoPlayer.Builder(context).build().apply {
            setMediaSource(mediaSourceFactory)
            prepare()
        }
    }

    fun seekTo(seconds: Double) {
        val p = player ?: return
        p.seekTo((max(0.0, seconds) * 1000).toLong())
    }

    fun pausePlayback() {
        player?.pause()
    }

    fun playPlayback() {
        if (!blocked) player?.play()
    }

    fun recordProgress() {
        if (props.readOnly) return
        val now = System.currentTimeMillis()
        lastProgressMs = now
        val start = segmentStart ?: max(0.0, currentTime - 1)
        val end = max(start, currentTime)
        localSegments = ContentToolPack4Logic.mergeLocalSegments(
            existing = localSegments,
            start = start,
            end = end,
        )
        segmentStart = currentTime
        val segmentsJson = JsonArray(
            localSegments.map { seg -> JsonArray(seg.map { JsonPrimitive(it) }) },
        )
        scope.launch {
            try {
                props.runAction(
                    "record_progress",
                    buildJsonObject {
                        put("currentSec", JsonPrimitive(currentTime))
                        put("watchedSegments", segmentsJson)
                        put("furthestSec", JsonPrimitive(currentTime))
                    },
                )
            } catch (_: Exception) {
                // Best-effort; position also survives via state merge on next successful save.
            }
        }
    }

    fun flushProgress() {
        recordProgress()
    }

    fun presentCheckpoint(cp: ContentToolPack4Logic.Checkpoint) {
        pausePlayback()
        activeCheckpoint = cp
        promptedIds = promptedIds + cp.id
        answerDraft = ""
        selectedOptions = emptySet()
        feedbackText = null
        blocked = ContentToolPack4Logic.shouldBlockPlayback(cp, answers)
        props.announce(
            L.format(
                R.string.mobile_contentTools_tools_media_checkpoints_checkpointAnnounce,
                formatTime(cp.atSec),
            ),
            true,
        )
        if (lifecycleOwner.lifecycle.currentState != Lifecycle.State.RESUMED) {
            returnFromBackgroundPrompt = true
        }
    }

    fun handleTime(seconds: Double) {
        val clamped = ContentToolPack4Logic.clampSeekTime(
            preventSkip = preventSkip,
            checkpoints = checkpoints,
            answers = answers,
            targetSec = seconds,
        )
        if (clamped.clamped && abs(clamped.time - seconds) > 0.1) {
            seekTo(clamped.time)
            currentTime = clamped.time
        } else {
            currentTime = seconds
        }

        if (segmentStart == null) {
            segmentStart = currentTime
        }

        if (activeCheckpoint == null) {
            val due = ContentToolPack4Logic.findDueCheckpoint(
                checkpoints = checkpoints,
                answers = answers,
                currentTime = currentTime,
                alreadyPromptedIds = promptedIds,
            )
            if (due != null) presentCheckpoint(due)
        }

        val now = System.currentTimeMillis()
        if (ContentToolPack4Logic.shouldFireProgressThrottle(lastProgressMs, now)) {
            recordProgress()
        }
    }

    fun submitCheckpoint(cp: ContentToolPack4Logic.Checkpoint) {
        busy = true
        errorText = null
        scope.launch {
            try {
                val question = checkpointQuestion(cp.id)
                val value: JsonElement = when (question.type) {
                    "multi" -> JsonArray(selectedOptions.sorted().map { JsonPrimitive(it) })
                    "numeric" -> {
                        val n = answerDraft.trim().toDoubleOrNull()
                        if (n != null) JsonPrimitive(n) else JsonPrimitive(answerDraft)
                    }
                    else -> JsonPrimitive(answerDraft)
                }
                val result = props.runAction(
                    "answer_checkpoint",
                    buildJsonObject {
                        put("checkpointId", JsonPrimitive(cp.id))
                        put("value", value)
                    },
                )
                val parsed = ContentToolPack4Logic.parseAnswerResult(result)
                feedbackText = parsed.feedback ?: parsed.message
                val done = parsed.done == true ||
                    parsed.correct == true ||
                    (parsed.attemptsRemaining ?: 1) <= 0
                if (done) {
                    activeCheckpoint = null
                    blocked = false
                    returnFromBackgroundPrompt = false
                    props.announce(
                        if (parsed.correct == true) correctLabel else continueLabel,
                        false,
                    )
                    if (parsed.correct == true || !cp.required) {
                        playPlayback()
                    }
                } else if (parsed.correct == false) {
                    props.announce(incorrectLabel, true)
                }
            } catch (_: Exception) {
                errorText = errorLabel
            } finally {
                busy = false
            }
        }
    }

    DisposableEffect(player) {
        onDispose {
            flushProgress()
            player?.release()
        }
    }

    val onBackground by rememberUpdatedState(newValue = {
        flushProgress()
        if (activeCheckpoint != null) {
            pausePlayback()
            returnFromBackgroundPrompt = true
        }
    })
    val onForeground by rememberUpdatedState(newValue = {
        if (returnFromBackgroundPrompt) {
            val clamped = ContentToolPack4Logic.clampSeekTime(
                preventSkip = preventSkip,
                checkpoints = checkpoints,
                answers = answers,
                targetSec = currentTime,
            )
            if (clamped.clamped) seekTo(clamped.time)
        }
    })
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_PAUSE, Lifecycle.Event.ON_STOP -> onBackground()
                Lifecycle.Event.ON_RESUME -> onForeground()
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    val onTick by rememberUpdatedState(newValue = { seconds: Double -> handleTime(seconds) })
    LaunchedEffect(player) {
        val p = player ?: return@LaunchedEffect
        while (isActive) {
            if (p.playbackState != Player.STATE_IDLE) {
                onTick(p.currentPosition / 1000.0)
            }
            delay(250)
        }
    }

    LaunchedEffect(player, props.state) {
        if (didResume || player == null) return@LaunchedEffect
        val resume = ContentToolPack4Logic.resumePosition(
            furthestSec = ContentToolPack4Logic.numberField(props.state, "furthestSec"),
            watchedSegments = localSegments,
        )
        if (resume > 0) {
            seekTo(resume)
            currentTime = resume
        }
        didResume = true
    }

    LaunchedEffect(props.state) { hydrate() }
    LaunchedEffect(props.instanceId) { hydrate() }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        when {
            !reliable -> {
                val path = ContentToolHostLogic.webActivityPath(
                    page?.courseCode.orEmpty(),
                    page?.itemId.orEmpty(),
                    props.instanceId,
                )
                ToolPlaceholder(
                    reason = ToolPlaceholderReason.OPEN_IN_BROWSER,
                    toolName = "media_checkpoints",
                    message = unavailableBody,
                    onOpenInBrowser = page?.let { ctx ->
                        { ctx.onOpenBrowser(Uri.parse(AppConfiguration.webUrl(path))) }
                    },
                )
            }
            resolvedUrl != null && player != null -> {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    AndroidView(
                        factory = { ctx ->
                            PlayerView(ctx).apply {
                                this.player = player
                                useController = true
                            }
                        },
                        update = { view ->
                            view.useController = !blocked
                            view.player = player
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .aspectRatio(16f / 9f)
                            .clip(RoundedCornerShape(12.dp))
                            .semantics { contentDescription = playerLabel },
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = formatTime(currentTime),
                            fontSize = 12.sp,
                            fontFamily = FontFamily.Monospace,
                            color = textSecondary(),
                        )
                        if (!captionUrl.isNullOrBlank()) {
                            TextButton(onClick = { captionsOn = !captionsOn }) {
                                Text(if (captionsOn) captionsOnLabel else captionsOffLabel)
                            }
                        }
                    }
                    if (blocked) {
                        Text(
                            text = blockedLabel,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = LexturesColors.Coral,
                        )
                    }
                }
            }
            else -> {
                Text(text = missingMedia, fontSize = 12.sp, color = LexturesColors.Coral)
            }
        }

        activeCheckpoint?.let { cp ->
            val question = checkpointQuestion(cp.id)
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(cardBackground())
                    .padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(
                    text = L.format(
                        R.string.mobile_contentTools_tools_media_checkpoints_checkpointAt,
                        formatTime(cp.atSec),
                    ),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                NotebookContentView(markdown = question.prompt, compact = true)

                when (question.type) {
                    "multi" -> {
                        question.options.forEach { (id, text) ->
                            val selected = id in selectedOptions
                            Row(
                                Modifier
                                    .fillMaxWidth()
                                    .heightIn(min = 44.dp)
                                    .semantics { this.selected = selected }
                                    .clickable(
                                        enabled = !busy && !props.readOnly,
                                        role = Role.Checkbox,
                                    ) {
                                        selectedOptions = if (selected) {
                                            selectedOptions - id
                                        } else {
                                            selectedOptions + id
                                        }
                                    },
                                verticalAlignment = Alignment.Top,
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                Checkbox(
                                    checked = selected,
                                    onCheckedChange = null,
                                    enabled = !busy && !props.readOnly,
                                )
                                NotebookContentView(markdown = text, compact = true)
                            }
                        }
                    }
                    "short_text", "numeric" -> {
                        OutlinedTextField(
                            value = answerDraft,
                            onValueChange = { answerDraft = it },
                            enabled = !busy && !props.readOnly,
                            label = { Text(yourAnswerLabel) },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = question.type == "numeric",
                        )
                    }
                    else -> {
                        question.options.forEach { (id, text) ->
                            val selected = answerDraft == id
                            Row(
                                Modifier
                                    .fillMaxWidth()
                                    .heightIn(min = 44.dp)
                                    .semantics { this.selected = selected }
                                    .clickable(
                                        enabled = !busy && !props.readOnly,
                                        role = Role.RadioButton,
                                    ) {
                                        answerDraft = id
                                    },
                                verticalAlignment = Alignment.Top,
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                RadioButton(
                                    selected = selected,
                                    onClick = null,
                                    enabled = !busy && !props.readOnly,
                                )
                                NotebookContentView(markdown = text, compact = true)
                            }
                        }
                    }
                }

                Button(
                    onClick = { submitCheckpoint(cp) },
                    enabled = !busy && !props.readOnly && canSubmit(question),
                ) {
                    Text(submitLabel)
                }

                feedbackText?.takeIf { it.isNotEmpty() }?.let {
                    NotebookContentView(markdown = it, compact = true)
                }
            }
        }

        if (returnFromBackgroundPrompt) {
            Text(
                text = returnToApp,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = accentColor(),
            )
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
