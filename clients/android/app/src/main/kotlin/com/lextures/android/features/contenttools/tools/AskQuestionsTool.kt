package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
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

private data class AskTurn(
    val id: String,
    val role: String,
    val text: String,
    val citations: List<Pair<String, String?>>,
)

@Composable
fun AskQuestionsTool(props: ContentToolRendererProps) {
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
    var askInstructor by remember { mutableStateOf(false) }
    var showClearConfirm by remember { mutableStateOf(false) }
    var disclosureMode by remember { mutableStateOf<String?>(null) }
    var decision by remember { mutableStateOf<String?>(null) }
    var consentFetched by remember { mutableStateOf(false) }

    val consentAllowed = ContentToolPack2Logic.composerAIAllowed(disclosureMode, decision, consentFetched)
    // CT.M9: disclosure lives in ToolFrame chrome so sandboxed tools cannot cover it.
    val showDisclosure = false

    val turns = remember(props.state) {
        ContentToolPack2Logic.arrayField(props.state, "turns").mapNotNull { raw ->
            val o = ContentToolPack2Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val role = (o["role"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val citations = ContentToolPack2Logic.arrayField(raw, "citations").mapNotNull { cite ->
                val c = ContentToolPack2Logic.objectMap(cite)
                val title = (c["title"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                title to (c["url"] as? JsonPrimitive)?.contentOrNull
            }
            AskTurn(id, role, text, citations)
        }
    }

    val emptyLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_empty)
    val messagesLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_messagesLabel)
    val thinkingLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_thinking)
    val askLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_ask)
    val clearLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_clear)
    val clearConfirm = L.text(R.string.mobile_contentTools_tools_ask_questions_clearConfirm)
    val cancelLabel = L.text(R.string.mobile_contentTools_runtime_cancel)
    val consentRequired = L.text(R.string.mobile_contentTools_ai_consentRequired)
    val ackLabel = L.text(R.string.mobile_contentTools_ai_acknowledge)
    val optOutLabel = L.text(R.string.mobile_contentTools_ai_optOut)
    val disclosureTitle = L.text(R.string.mobile_contentTools_ai_disclosureTitle)
    val disclosureBody = L.text(R.string.mobile_contentTools_ai_disclosureBody)
    val youLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_you)
    val aiBadge = L.text(R.string.mobile_contentTools_tools_ask_questions_aiBadge)
    val sourcesLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_sources)
    val askInstructorLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_askInstructor)
    val clearedLabel = L.text(R.string.mobile_contentTools_tools_ask_questions_cleared)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val offlineLabel = L.text(R.string.mobile_contentTools_runtime_offlineComposer)
    val placeholder = ContentToolHostLogic.stringField(props.config, "placeholder")
        ?: L.text(R.string.mobile_contentTools_tools_ask_questions_placeholder)

    LaunchedEffect(props.instanceId, page?.courseCode, page?.accessToken) {
        if (draft.isEmpty()) {
            draft = ContentToolHostLogic.stringField(props.state, "draft")
                ?: draftStore.load(draftKey)
        }
        val token = page?.accessToken
        val course = page?.courseCode
        if (token.isNullOrBlank() || course.isNullOrBlank()) {
            consentFetched = false
            return@LaunchedEffect
        }
        runCatching {
            ContentToolsApi.fetchAIConsent(course, props.toolId, token)
        }.onSuccess {
            disclosureMode = it.aiDisclosureMode
            decision = it.decision
            consentFetched = true
        }.onFailure {
            consentFetched = false
        }
    }

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        ContentToolHostLogic.stringField(props.config, "intro")?.takeIf { it.isNotBlank() }?.let {
            NotebookContentView(markdown = it, compact = true)
        }

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

        if (turns.isEmpty()) {
            Text(emptyLabel, color = textSecondary(), fontSize = 12.sp)
        } else {
            LazyColumn(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = 280.dp)
                    .semantics { contentDescription = messagesLabel },
            ) {
                items(turns, key = { it.id }) { turn ->
                    val roleLabel = if (turn.role == "user") youLabel else aiBadge
                    Column(modifier = Modifier.padding(vertical = 4.dp)) {
                        Text(
                            roleLabel,
                            fontWeight = FontWeight.SemiBold,
                            fontSize = 11.sp,
                            color = textSecondary(),
                        )
                        NotebookContentView(markdown = turn.text, compact = true)
                        if (ContentToolPack2Logic.boolField(props.config, "showCitations") != false &&
                            turn.citations.isNotEmpty()
                        ) {
                            Text(sourcesLabel, fontWeight = FontWeight.SemiBold, fontSize = 11.sp)
                            turn.citations.forEachIndexed { idx, cite ->
                                Text(
                                    // sync-mobile-locales expands "Source %lld: %@" → "%2$d: %1$s"
                                    L.format(
                                        R.string.mobile_contentTools_tools_ask_questions_sourceChip,
                                        cite.first,
                                        idx + 1,
                                    ),
                                    fontSize = 11.sp,
                                    color = textSecondary(),
                                )
                            }
                        }
                    }
                }
            }
        }

        if (busy) {
            Row {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp))
                Text(thinkingLabel, fontSize = 12.sp, color = textSecondary())
            }
        }

        errorText?.let {
            Text(it, color = LexturesColors.Coral, fontSize = 12.sp)
            if (askInstructor) {
                Text(askInstructorLabel, color = textSecondary(), fontSize = 11.sp)
            }
        }

        if (!props.readOnly) {
            if (consentAllowed) {
                ToolComposer(
                    placeholder = placeholder,
                    sendLabel = askLabel,
                    text = draft,
                    onTextChange = { draft = it },
                    draftKey = draftKey,
                    enabled = true,
                    online = online,
                    busy = busy,
                    showCancel = true,
                    cancelLabel = cancelLabel,
                    onCancel = { busy = false },
                    onSend = {
                        scope.launch {
                            val question = draft.trim()
                            if (!ContentToolPack2Logic.canAsk(question, props.readOnly, online, consentAllowed, busy)) {
                                return@launch
                            }
                            busy = true
                            errorText = null
                            askInstructor = false
                            try {
                                val raw = props.runAction(
                                    "ask",
                                    buildJsonObject { put("question", JsonPrimitive(question)) },
                                )
                                val result = ContentToolPack2Logic.objectMap(raw)
                                val code = (result["error"] as? JsonPrimitive)?.contentOrNull
                                if (code != null) {
                                    errorText = L.text(context, localePrefs, pack2ErrorRes(code))
                                    askInstructor = ContentToolPack2Logic.boolField(raw, "askInstructor") == true
                                } else {
                                    draft = ""
                                    draftStore.clear(draftKey)
                                    props.save(mapOf("draft" to JsonPrimitive("")))
                                    val citeCount = ContentToolPack2Logic.numberField(raw, "citationCount")?.toInt() ?: 0
                                    props.announce(
                                        L.format(
                                            context,
                                            localePrefs,
                                            R.string.mobile_contentTools_tools_ask_questions_answerReceived,
                                            citeCount,
                                        ),
                                        false,
                                    )
                                }
                            } catch (_: Exception) {
                                errorText = if (!online) offlineLabel else retryLabel
                            } finally {
                                busy = false
                            }
                        }
                    },
                )
            } else if (consentFetched) {
                Text(consentRequired, color = textSecondary(), fontSize = 12.sp)
            }

            if (turns.isNotEmpty()) {
                TextButton(onClick = { showClearConfirm = true }, enabled = !busy) {
                    Text(clearLabel, color = LexturesColors.Coral)
                }
            }
        }
    }

    if (showClearConfirm) {
        AlertDialog(
            onDismissRequest = { showClearConfirm = false },
            title = { Text(clearConfirm) },
            confirmButton = {
                TextButton(onClick = {
                    showClearConfirm = false
                    scope.launch {
                        busy = true
                        try {
                            props.runAction("clear", buildJsonObject { })
                            draft = ""
                            draftStore.clear(draftKey)
                            props.announce(clearedLabel, false)
                        } catch (_: Exception) {
                            errorText = retryLabel
                        } finally {
                            busy = false
                        }
                    }
                }) { Text(clearLabel) }
            },
            dismissButton = {
                TextButton(onClick = { showClearConfirm = false }) { Text(cancelLabel) }
            },
        )
    }
}

internal fun pack2ErrorRes(code: String?): Int =
    when (ContentToolPack2Logic.classifyAIError(code)) {
        ContentToolPack2Logic.AIErrorClass.RATE_LIMITED -> R.string.mobile_contentTools_ai_error_rateLimited
        ContentToolPack2Logic.AIErrorClass.BUDGET -> R.string.mobile_contentTools_ai_error_budget
        ContentToolPack2Logic.AIErrorClass.PROVIDER_UNAVAILABLE -> R.string.mobile_contentTools_ai_error_providerUnavailable
        ContentToolPack2Logic.AIErrorClass.FILTERED -> R.string.mobile_contentTools_ai_error_filtered
        ContentToolPack2Logic.AIErrorClass.OPT_OUT -> R.string.mobile_contentTools_ai_error_optOut
        ContentToolPack2Logic.AIErrorClass.COPPA -> R.string.mobile_contentTools_ai_error_coppa
        ContentToolPack2Logic.AIErrorClass.TOO_SHORT -> R.string.mobile_contentTools_tools_explain_it_back_error_tooShort
        ContentToolPack2Logic.AIErrorClass.TOO_LONG -> R.string.mobile_contentTools_tools_explain_it_back_error_tooLong
        ContentToolPack2Logic.AIErrorClass.MAX_ATTEMPTS -> R.string.mobile_contentTools_tools_explain_it_back_error_maxAttempts
        ContentToolPack2Logic.AIErrorClass.FORBIDDEN -> R.string.mobile_contentTools_tools_inline_discussion_error_forbidden
        ContentToolPack2Logic.AIErrorClass.OFFLINE -> R.string.mobile_contentTools_runtime_offlineComposer
        ContentToolPack2Logic.AIErrorClass.UNKNOWN -> R.string.mobile_contentTools_runtime_retry
    }
