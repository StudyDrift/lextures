package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.Haptics
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack1Logic
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import org.json.JSONObject

private data class FlashcardCurrent(
    val cardId: String,
    val side: String,
    val prompt: String,
    val answer: String,
    val index: Int,
    val total: Int,
    val hint: String?,
)

@Composable
fun FlashcardsTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember(context) { OfflineService.get(context) }
    val scope = rememberCoroutineScope()

    var busy by remember { mutableStateOf(false) }
    var revealed by remember { mutableStateOf(false) }
    var showHint by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var caughtUp by remember { mutableStateOf(false) }
    var current by remember { mutableStateOf<FlashcardCurrent?>(null) }
    var statusText by remember { mutableStateOf("") }
    var summaryText by remember { mutableStateOf<String?>(null) }
    val pendingRatings = remember { mutableStateListOf<ContentToolPack1Logic.PendingAction>() }

    val needsConnection = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val ratingsPendingLabel = L.text(R.string.mobile_contentTools_tools_flashcards_ratingsPending)
    val caughtUpAnnounce = L.text(R.string.mobile_contentTools_tools_flashcards_caughtUpAnnounce)
    val answerRevealedLabel = L.text(R.string.mobile_contentTools_tools_flashcards_answerRevealed)
    val showAnswerLabel = L.text(R.string.mobile_contentTools_tools_flashcards_showAnswer)
    val showHintLabel = L.text(R.string.mobile_contentTools_tools_flashcards_showHint)
    val rateGroupLabel = L.text(R.string.mobile_contentTools_tools_flashcards_rateGroup)
    val endSessionLabel = L.text(R.string.mobile_contentTools_tools_flashcards_endSession)
    val startSessionLabel = L.text(R.string.mobile_contentTools_tools_flashcards_startSession)
    val caughtUpLabel = L.text(R.string.mobile_contentTools_tools_flashcards_caughtUp)
    val sessionCompleteAnnounce = L.text(R.string.mobile_contentTools_tools_flashcards_sessionCompleteAnnounce)
    val sessionEndedAnnounce = L.text(R.string.mobile_contentTools_tools_flashcards_sessionEndedAnnounce)
    val ratingAgain = L.text(R.string.mobile_contentTools_tools_flashcards_ratings_again)
    val ratingHard = L.text(R.string.mobile_contentTools_tools_flashcards_ratings_hard)
    val ratingGood = L.text(R.string.mobile_contentTools_tools_flashcards_ratings_good)
    val ratingEasy = L.text(R.string.mobile_contentTools_tools_flashcards_ratings_easy)

    fun ratingLabel(rating: String): String = when (rating.lowercase()) {
        "again" -> ratingAgain
        "hard" -> ratingHard
        "good" -> ratingGood
        "easy" -> ratingEasy
        else -> rating
    }

    fun invalidateReviewCaches() {
        offline.invalidateCache(ContentToolPack1Logic.reviewCacheKeysToInvalidate())
    }

    fun applyStatus(raw: JsonElement?) {
        val obj = raw?.let { runCatching { it.jsonObject }.getOrNull() } ?: return
        val status = obj["status"]?.let { runCatching { it.jsonObject }.getOrNull() } ?: return
        val newCount = ContentToolPack1Logic.numberField(status, "newCount")?.toInt() ?: 0
        val dueCount = ContentToolPack1Logic.numberField(status, "dueCount")?.toInt() ?: 0
        statusText = L.format(
            context,
            localePrefs,
            R.string.mobile_contentTools_tools_flashcards_statusChips,
            newCount,
            dueCount,
        )
    }

    fun applyCurrent(raw: JsonElement?) {
        applyStatus(raw)
        val obj = raw?.let { runCatching { it.jsonObject }.getOrNull() }
        val cur = obj?.get("current")?.let { runCatching { it.jsonObject }.getOrNull() }
        if (cur == null) {
            current = null
            return
        }
        val cardId = (cur["cardId"] as? JsonPrimitive)?.contentOrNull ?: return
        val prompt = (cur["prompt"] as? JsonPrimitive)?.contentOrNull ?: return
        val answer = (cur["answer"] as? JsonPrimitive)?.contentOrNull ?: return
        val side = (cur["side"] as? JsonPrimitive)?.contentOrNull ?: "forward"
        val index = ContentToolPack1Logic.numberField(cur, "index")?.toInt() ?: 0
        val total = ContentToolPack1Logic.numberField(cur, "total")?.toInt() ?: 0
        val hint = (cur["hint"] as? JsonPrimitive)?.contentOrNull
        current = FlashcardCurrent(cardId, side, prompt, answer, index, total, hint)
        caughtUp = false
    }

    suspend fun refreshStatus() {
        try {
            applyStatus(props.runAction("status", buildJsonObject { }))
        } catch (_: Exception) {
            // best-effort
        }
    }

    suspend fun flushPending() {
        val ordered = ContentToolPack1Logic.orderPendingActions(pendingRatings.toList())
        if (ordered.isEmpty()) return
        val remaining = mutableListOf<ContentToolPack1Logic.PendingAction>()
        for (item in ordered) {
            try {
                val dict = JSONObject(item.payloadJSON)
                val input = buildJsonObject {
                    dict.keys().forEach { key ->
                        put(key, JsonPrimitive(dict.getString(key)))
                    }
                }
                props.runAction(item.action, input)
            } catch (_: Exception) {
                remaining.add(item)
            }
        }
        pendingRatings.clear()
        pendingRatings.addAll(remaining)
        if (remaining.isEmpty()) {
            invalidateReviewCaches()
        }
    }

    fun startSession() {
        if (busy || props.readOnly) return
        busy = true
        errorText = null
        summaryText = null
        scope.launch {
            try {
                val raw = props.runAction("start_session", buildJsonObject { })
                if (ContentToolPack1Logic.boolField(raw, "caughtUp") == true) {
                    caughtUp = true
                    current = null
                    props.announce(caughtUpAnnounce, false)
                    return@launch
                }
                applyCurrent(raw)
                Haptics.trigger(view, Haptics.Kind.Tap)
            } catch (_: Exception) {
                errorText = needsConnection
                props.announce(needsConnection, true)
            } finally {
                busy = false
            }
        }
    }

    fun rate(rating: String) {
        val card = current ?: return
        if (!ContentToolPack1Logic.isValidRating(rating) || busy || !revealed) return
        busy = true
        errorText = null
        scope.launch {
            try {
                val input = buildJsonObject {
                    put("cardId", JsonPrimitive(card.cardId))
                    put("rating", JsonPrimitive(rating))
                    put("side", JsonPrimitive(card.side))
                    put("idempotencyKey", JsonPrimitive(ContentToolHostLogic.newIdempotencyKey()))
                }
                val raw = props.runAction("rate", input)
                props.announce(
                    L.format(
                        context,
                        localePrefs,
                        R.string.mobile_contentTools_tools_flashcards_ratedAnnounce,
                        ratingLabel(rating),
                    ),
                    false,
                )
                Haptics.trigger(view, Haptics.Kind.Success)
                if (ContentToolPack1Logic.boolField(raw, "sessionComplete") == true) {
                    val summary = raw?.let { runCatching { it.jsonObject }.getOrNull() }?.get("summary")
                    val reviewed = ContentToolPack1Logic.numberField(summary, "reviewed")?.toInt()
                        ?: (card.index + 1)
                    summaryText = L.format(
                        context,
                        localePrefs,
                        R.string.mobile_contentTools_tools_flashcards_sessionSummary,
                        reviewed,
                    )
                    current = null
                    revealed = false
                    props.announce(sessionCompleteAnnounce, false)
                    invalidateReviewCaches()
                    return@launch
                }
                applyCurrent(raw)
                revealed = false
                showHint = false
            } catch (_: Exception) {
                if (ContentToolPack1Logic.canQueueActionOffline("flashcards", "rate")) {
                    val key = ContentToolHostLogic.newIdempotencyKey()
                    val encoded = JSONObject()
                        .put("cardId", card.cardId)
                        .put("rating", rating)
                        .put("side", card.side)
                        .put("idempotencyKey", key)
                        .toString()
                    pendingRatings.add(
                        ContentToolPack1Logic.PendingAction(
                            instanceId = props.instanceId,
                            toolId = "flashcards",
                            action = "rate",
                            sequence = System.currentTimeMillis(),
                            payloadJSON = encoded,
                        ),
                    )
                    val ordered = ContentToolPack1Logic.orderPendingActions(pendingRatings.toList())
                    pendingRatings.clear()
                    pendingRatings.addAll(ordered)
                    props.announce(ratingsPendingLabel, false)
                } else {
                    errorText = needsConnection
                    props.announce(needsConnection, true)
                }
            } finally {
                busy = false
            }
        }
    }

    fun endSession() {
        if (busy) return
        busy = true
        scope.launch {
            try {
                val raw = props.runAction("end_session", buildJsonObject { })
                val summary = raw?.let { runCatching { it.jsonObject }.getOrNull() }?.get("summary")
                val reviewed = ContentToolPack1Logic.numberField(summary, "reviewed")?.toInt() ?: 0
                summaryText = L.format(
                    context,
                    localePrefs,
                    R.string.mobile_contentTools_tools_flashcards_sessionSummary,
                    reviewed,
                )
                current = null
                revealed = false
                props.announce(sessionEndedAnnounce, false)
                invalidateReviewCaches()
            } catch (_: Exception) {
                errorText = needsConnection
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.instanceId) {
        refreshStatus()
        flushPending()
    }

    val deckTitle = ContentToolHostLogic.stringField(props.config, "title")
        ?.trim()
        ?.takeIf { it.isNotEmpty() }

    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        deckTitle?.let {
            Text(
                text = it,
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }
        if (statusText.isNotEmpty()) {
            Text(text = statusText, fontSize = 12.sp, color = textSecondary())
        }
        if (pendingRatings.isNotEmpty()) {
            Text(text = ratingsPendingLabel, fontSize = 12.sp, color = textSecondary())
        }

        val card = current
        when {
            card != null -> {
                Text(
                    text = L.format(
                        context,
                        localePrefs,
                        R.string.mobile_contentTools_tools_flashcards_progress,
                        card.index + 1,
                        card.total,
                    ),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
                NotebookContentView(markdown = card.prompt, compact = true)
                if (revealed) {
                    HorizontalDivider()
                    NotebookContentView(markdown = card.answer, compact = true)
                }
                Button(
                    onClick = {
                        revealed = true
                        props.announce(answerRevealedLabel, false)
                        Haptics.trigger(view, Haptics.Kind.Selection)
                    },
                    enabled = !revealed,
                ) {
                    Text(if (revealed) answerRevealedLabel else showAnswerLabel)
                }
                card.hint?.takeIf { it.isNotBlank() }?.let { hint ->
                    OutlinedButton(onClick = { showHint = true }, enabled = !showHint) {
                        Text(showHintLabel)
                    }
                    if (showHint) {
                        NotebookContentView(markdown = hint, compact = true)
                    }
                }
                if (revealed && !props.readOnly) {
                    Text(
                        text = rateGroupLabel,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textSecondary(),
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        ContentToolPack1Logic.flashcardRatings.forEach { rating ->
                            OutlinedButton(
                                onClick = { rate(rating) },
                                enabled = !busy,
                                modifier = Modifier.heightIn(min = 44.dp),
                            ) {
                                Text(ratingLabel(rating))
                            }
                        }
                    }
                }
                OutlinedButton(onClick = { endSession() }, enabled = !busy) {
                    Text(endSessionLabel)
                }
            }
            caughtUp -> {
                Text(text = caughtUpLabel, fontSize = 14.sp, color = textPrimary())
            }
            summaryText != null -> {
                Text(text = summaryText.orEmpty(), fontSize = 14.sp, color = textPrimary())
                Button(
                    onClick = { startSession() },
                    enabled = !props.readOnly && !busy,
                ) {
                    Text(startSessionLabel)
                }
            }
            else -> {
                Button(
                    onClick = { startSession() },
                    enabled = !props.readOnly && !busy,
                ) {
                    Text(startSessionLabel)
                }
            }
        }

        errorText?.let {
            Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
