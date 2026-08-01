package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack2Logic
import com.lextures.android.core.lms.ContentToolsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.contenttools.ContentToolDraftStore
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.contenttools.LocalContentToolsPage
import com.lextures.android.features.contenttools.ToolComposer
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject

private data class ExplainFeedback(
    val covered: List<String>,
    val missing: List<String>,
    val strength: String,
    val suggestion: String,
    val probe: String?,
    val mode: String,
)

private data class ExplainAttempt(
    val at: String,
    val text: String,
    val feedback: ExplainFeedback?,
)

@Composable
fun ExplainItBackTool(props: ContentToolRendererProps) {
    val page = LocalContentToolsPage.current
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val offline = remember { OfflineService.get(context.applicationContext) }
    val online by offline.networkMonitor.isOnline.collectAsState()
    val draftStore = remember { ContentToolDraftStore.create(context) }
    val draftKey = ContentToolPack2Logic.draftStorageKey(props.instanceId)

    var draft by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var revising by remember { mutableStateOf(false) }
    var disclosureMode by remember { mutableStateOf<String?>(null) }
    var decision by remember { mutableStateOf<String?>(null) }
    var consentFetched by remember { mutableStateOf(false) }
    val labels = remember { mutableStateMapOf<String, String>() }

    val aiFeedback = ContentToolPack2Logic.boolField(props.config, "aiFeedback") != false
    val consentAllowed = if (!aiFeedback) {
        true
    } else {
        ContentToolPack2Logic.composerAIAllowed(disclosureMode, decision, consentFetched)
    }
    // CT.M9: disclosure lives in ToolFrame chrome so sandboxed tools cannot cover it.
    val showDisclosure = false

    val minWords = (ContentToolPack2Logic.numberField(props.config, "minWords") ?: 25.0).toInt()
    val maxWords = (ContentToolPack2Logic.numberField(props.config, "maxWords") ?: 150.0).toInt()
    val maxAttempts = (ContentToolPack2Logic.numberField(props.config, "attempts") ?: 3.0).toInt()

    val attempts = remember(props.state) {
        ContentToolPack2Logic.arrayField(props.state, "attempts").mapNotNull { raw ->
            val o = ContentToolPack2Logic.objectMap(raw)
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val at = (o["at"] as? JsonPrimitive)?.contentOrNull.orEmpty()
            ExplainAttempt(at, text, parseFeedback(o["feedback"]))
        }
    }
    val latest = attempts.lastOrNull()
    val wordCount = ContentToolPack2Logic.wordCount(draft)
    val attemptsLeft = maxOf(0, maxAttempts - attempts.size)

    val prompt = ContentToolHostLogic.stringField(props.config, "prompt")
        ?: L.text(R.string.mobile_contentTools_tools_explain_it_back_defaultPrompt)
    val lengthGuide = L.format(R.string.mobile_contentTools_tools_explain_it_back_lengthGuide, minWords, maxWords)
    val practiceNote = L.text(R.string.mobile_contentTools_tools_explain_it_back_practiceNote)
    val consentRequired = L.text(R.string.mobile_contentTools_ai_consentRequired)
    val submitFeedback = L.text(R.string.mobile_contentTools_tools_explain_it_back_submitFeedback)
    val submitReview = L.text(R.string.mobile_contentTools_tools_explain_it_back_submitReview)
    val inputLabel = L.text(R.string.mobile_contentTools_tools_explain_it_back_inputLabel)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val offlineLabel = L.text(R.string.mobile_contentTools_runtime_offlineComposer)
    val feedbackReceived = L.text(R.string.mobile_contentTools_tools_explain_it_back_feedbackReceived)
    val ackLabel = L.text(R.string.mobile_contentTools_ai_acknowledge)
    val optOutLabel = L.text(R.string.mobile_contentTools_ai_optOut)
    val disclosureTitle = L.text(R.string.mobile_contentTools_ai_disclosureTitle)
    val disclosureBody = L.text(R.string.mobile_contentTools_ai_disclosureBody)
    val tooShortLabel = L.text(R.string.mobile_contentTools_tools_explain_it_back_error_tooShort)
    val tooLongLabel = L.text(R.string.mobile_contentTools_tools_explain_it_back_error_tooLong)

    LaunchedEffect(props.instanceId, page?.courseCode, page?.accessToken) {
        if (draft.isEmpty()) {
            draft = ContentToolHostLogic.stringField(props.state, "draft")
                ?: draftStore.load(draftKey)
        }
        if (!aiFeedback) {
            consentFetched = true
            return@LaunchedEffect
        }
        val token = page?.accessToken
        val course = page?.courseCode
        if (token.isNullOrBlank() || course.isNullOrBlank()) {
            consentFetched = false
            return@LaunchedEffect
        }
        runCatching { ContentToolsApi.fetchAIConsent(course, props.toolId, token) }
            .onSuccess {
                disclosureMode = it.aiDisclosureMode
                decision = it.decision
                consentFetched = true
            }
            .onFailure { consentFetched = false }
    }

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        NotebookContentView(markdown = prompt, compact = true)
        Text(lengthGuide, fontSize = 11.sp, color = textSecondary())
        Text(practiceNote, fontSize = 11.sp, color = textSecondary())

        if (showDisclosure) {
            Column(modifier = Modifier.padding(8.dp)) {
                Text(disclosureTitle, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                Text(disclosureBody, color = textSecondary(), fontSize = 11.sp)
                Row {
                    TextButton(onClick = {
                        scope.launch {
                            val token = page?.accessToken ?: return@launch
                            runCatching {
                                ContentToolsApi.postAIConsent(page.courseCode, props.toolId, "acknowledged", token)
                            }.onSuccess {
                                disclosureMode = it.aiDisclosureMode
                                decision = it.decision
                                consentFetched = true
                            }
                        }
                    }) { Text(ackLabel) }
                    TextButton(onClick = {
                        scope.launch {
                            val token = page?.accessToken ?: return@launch
                            runCatching {
                                ContentToolsApi.postAIConsent(page.courseCode, props.toolId, "opted_out", token)
                            }.onSuccess {
                                disclosureMode = it.aiDisclosureMode
                                decision = it.decision
                                consentFetched = true
                            }
                        }
                    }) { Text(optOutLabel) }
                }
            }
        }

        instructorNoteText(props)?.let { note ->
            Text(
                L.text(R.string.mobile_contentTools_tools_explain_it_back_instructorNote),
                fontWeight = FontWeight.SemiBold,
                fontSize = 12.sp,
            )
            NotebookContentView(markdown = note, compact = true)
        }

        if (latest?.feedback != null && !revising) {
            FeedbackCard(latest.feedback, labels)
            if (attemptsLeft > 0 && !props.readOnly) {
                TextButton(onClick = {
                    revising = true
                    draft = latest.text
                }) {
                    Text(L.format(R.string.mobile_contentTools_tools_explain_it_back_revise, attemptsLeft))
                }
            }
        } else if (!props.readOnly) {
            if (consentAllowed) {
                ToolComposer(
                    placeholder = inputLabel,
                    sendLabel = if (aiFeedback) submitFeedback else submitReview,
                    text = draft,
                    onTextChange = { draft = it },
                    draftKey = draftKey,
                    enabled = true,
                    online = online,
                    busy = busy,
                    onSend = {
                        scope.launch {
                            val text = draft.trim()
                            if (!ContentToolPack2Logic.canSubmitExplanation(
                                    text,
                                    minWords,
                                    maxWords,
                                    attempts.size,
                                    maxAttempts,
                                    props.readOnly,
                                    online,
                                    consentAllowed,
                                )
                            ) {
                                errorText = if (wordCount < minWords) tooShortLabel else tooLongLabel
                                return@launch
                            }
                            busy = true
                            errorText = null
                            try {
                                val raw = props.runAction(
                                    "submit",
                                    buildJsonObject { put("text", JsonPrimitive(text)) },
                                )
                                val result = ContentToolPack2Logic.objectMap(raw)
                                val code = (result["error"] as? JsonPrimitive)?.contentOrNull
                                if (code != null) {
                                    errorText = L.text(context, localePrefs, pack2ErrorRes(code))
                                } else {
                                    result["keyPointLabels"]?.jsonObject?.forEach { (k, v) ->
                                        (v as? JsonPrimitive)?.contentOrNull?.let { labels[k] = it }
                                    }
                                    draft = ""
                                    revising = false
                                    draftStore.clear(draftKey)
                                    props.save(mapOf("draft" to JsonPrimitive("")))
                                    props.announce(feedbackReceived, false)
                                }
                            } catch (_: Exception) {
                                errorText = if (!online) offlineLabel else retryLabel
                            } finally {
                                busy = false
                            }
                        }
                    },
                )
                Text(
                    L.format(
                        R.string.mobile_contentTools_tools_explain_it_back_wordCount,
                        wordCount,
                        minWords,
                        maxWords,
                    ),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
                if (attemptsLeft < maxAttempts) {
                    Text(
                        L.format(R.string.mobile_contentTools_tools_explain_it_back_attemptsLeft, attemptsLeft),
                        fontSize = 11.sp,
                    )
                }
            } else if (consentFetched) {
                Text(consentRequired, color = textSecondary(), fontSize = 12.sp)
            }
        } else if (latest != null) {
            NotebookContentView(markdown = latest.text, compact = true)
            latest.feedback?.let { FeedbackCard(it, labels) }
        }

        errorText?.let {
            Text(it, color = LexturesColors.Coral, fontSize = 12.sp)
        }
    }
}

@Composable
private fun FeedbackCard(feedback: ExplainFeedback, labels: Map<String, String>) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(
            if (feedback.mode == "review") {
                L.text(R.string.mobile_contentTools_tools_explain_it_back_reviewTitle)
            } else {
                L.text(R.string.mobile_contentTools_tools_explain_it_back_feedbackTitle)
            },
            fontWeight = FontWeight.SemiBold,
        )
        if (feedback.covered.isNotEmpty()) {
            Text(L.text(R.string.mobile_contentTools_tools_explain_it_back_whatYouGot), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            feedback.covered.forEach { Text("• ${labels[it] ?: it}", fontSize = 12.sp) }
        }
        if (feedback.missing.isNotEmpty()) {
            Text(L.text(R.string.mobile_contentTools_tools_explain_it_back_whatsMissing), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            feedback.missing.forEach { Text("• ${labels[it] ?: it}", fontSize = 12.sp) }
        }
        if (feedback.strength.isNotBlank()) {
            Text(L.text(R.string.mobile_contentTools_tools_explain_it_back_strength), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            NotebookContentView(markdown = feedback.strength, compact = true)
        }
        if (feedback.suggestion.isNotBlank()) {
            Text(L.text(R.string.mobile_contentTools_tools_explain_it_back_suggestion), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            NotebookContentView(markdown = feedback.suggestion, compact = true)
        }
        feedback.probe?.takeIf { it.isNotBlank() }?.let {
            Text(L.text(R.string.mobile_contentTools_tools_explain_it_back_probe), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            NotebookContentView(markdown = it, compact = true)
        }
    }
}

private fun instructorNoteText(props: ContentToolRendererProps): String? {
    val note = ContentToolPack2Logic.objectMap(ContentToolPack2Logic.objectMap(props.state)["instructorNote"])
    return (note["text"] as? JsonPrimitive)?.contentOrNull?.takeIf { it.isNotBlank() }
}

private fun parseFeedback(raw: kotlinx.serialization.json.JsonElement?): ExplainFeedback? {
    if (raw == null) return null
    val o = ContentToolPack2Logic.objectMap(raw)
    val covered = ContentToolPack2Logic.arrayField(raw, "covered").mapNotNull {
        (it as? JsonPrimitive)?.contentOrNull
    }
    val missing = ContentToolPack2Logic.arrayField(raw, "missing").mapNotNull {
        (it as? JsonPrimitive)?.contentOrNull
    }
    return ExplainFeedback(
        covered = covered,
        missing = missing,
        strength = (o["strength"] as? JsonPrimitive)?.contentOrNull.orEmpty(),
        suggestion = (o["suggestion"] as? JsonPrimitive)?.contentOrNull.orEmpty(),
        probe = (o["probe"] as? JsonPrimitive)?.contentOrNull,
        mode = (o["mode"] as? JsonPrimitive)?.contentOrNull ?: "ai",
    )
}
