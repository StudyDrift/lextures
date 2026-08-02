package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
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
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack1Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull

private data class InlineQuestion(
    val id: String,
    val type: String,
    val prompt: String,
    val options: List<Pair<String, String>>,
    val unit: String?,
)

@Composable
fun InlineQuestionsTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val drafts = remember(props.instanceId) { mutableStateMapOf<String, JsonElement>() }
    var busyId by remember { mutableStateOf<String?>(null) }
    val lastResults = remember(props.instanceId) { mutableStateMapOf<String, JsonElement>() }
    var gradingPending by remember { mutableStateOf(false) }

    val checkLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_checkLabel)
    val gradingPendingLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_gradingPending)
    val sequentialLockedLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_sequentialLocked)
    val submitLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_submit)
    val yourAnswerLabel = L.text(R.string.mobile_contentTools_runtime_yourAnswer)
    val correctLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_correct)
    val incorrectLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_incorrect)
    val correctAnnounce = L.text(R.string.mobile_contentTools_tools_inline_questions_correctAnnounce)
    val incorrectAnnounce = L.text(R.string.mobile_contentTools_tools_inline_questions_incorrectAnnounce)
    val needsConnection = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val previousLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_previous)
    val nextLabel = L.text(R.string.mobile_contentTools_tools_inline_questions_next)

    val label = ContentToolHostLogic.stringField(props.config, "label")
        ?.trim()
        ?.takeIf { it.isNotEmpty() }
        ?: checkLabel
    val sequential = ContentToolPack1Logic.boolField(props.config, "sequential") == true
    val shuffleOptions = ContentToolPack1Logic.boolField(props.config, "shuffleOptions") == true
    val configMap = ContentToolPack1Logic.objectMap(props.config)
    val maxAttempts = ContentToolPack1Logic.parseAttemptsConfig(configMap["attempts"])
    val questionsAtATime = ContentToolPack1Logic.parseQuestionsAtATime(configMap["questionsAtATime"])
    val answers = ContentToolPack1Logic.objectMap(
        ContentToolPack1Logic.objectMap(props.state)["answers"],
    )
    val questions = remember(props.config, props.instanceId, shuffleOptions) {
        parseQuestions(props.config, props.instanceId, shuffleOptions)
    }
    val questionIds = remember(questions) { questions.map { it.id } }
    val firstIncompleteIndex = remember(questions, answers) {
        val idx = questions.indexOfFirst {
            ContentToolPack1Logic.attemptsUsed(answers, it.id) == 0
        }
        if (idx < 0) maxOf(0, questions.lastIndex) else idx
    }
    var pageIndex by remember(props.instanceId, questionsAtATime, questions.size) {
        mutableStateOf(
            ContentToolPack1Logic.initialPageIndex(
                questions.size,
                questionsAtATime,
                firstIncompleteIndex,
            ),
        )
    }
    val pagingEnabled =
        questionsAtATime != null && questionsAtATime > 0 && questions.size > questionsAtATime
    val pageCount =
        if (pagingEnabled && questionsAtATime != null) {
            maxOf(1, (questions.size + questionsAtATime - 1) / questionsAtATime)
        } else {
            1
        }
    val safePage = pageIndex.coerceIn(0, pageCount - 1)
    val window = ContentToolPack1Logic.pageWindow(questions.size, questionsAtATime, safePage)
    val visibleQuestions = if (pagingEnabled) questions.slice(window) else questions

    fun hydrate() {
        val remote = ContentToolPack1Logic.objectMap(
            ContentToolPack1Logic.objectMap(props.state)["drafts"],
        )
        if (drafts.isEmpty() || drafts != remote) {
            drafts.clear()
            drafts.putAll(remote)
        }
    }

    LaunchedEffect(props.state) { hydrate() }

    fun persistDrafts() {
        if (props.readOnly) return
        val patch = ContentToolPack1Logic.mergePreservingUnknown(
            ContentToolPack1Logic.objectMap(props.state),
            mapOf(
                "v" to JsonPrimitive(1),
                "drafts" to JsonObject(drafts.toMap()),
                "answers" to JsonObject(answers),
            ),
        )
        props.save(patch)
    }

    fun setDraft(qid: String, value: JsonElement) {
        if (props.readOnly) return
        drafts[qid] = value
        persistDrafts()
    }

    fun hasDraft(qid: String): Boolean {
        val v = drafts[qid] ?: return false
        return when (v) {
            is JsonPrimitive -> {
                if (v.isString) v.content.trim().isNotEmpty() else true
            }
            is JsonArray -> v.isNotEmpty()
            else -> false
        }
    }

    fun stringDraft(qid: String): String =
        (drafts[qid] as? JsonPrimitive)?.contentOrNull.orEmpty()

    fun multiSelected(qid: String): Set<String> {
        val arr = drafts[qid] as? JsonArray ?: return emptySet()
        return arr.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }.toSet()
    }

    fun toggleMulti(qid: String, optionId: String) {
        val set = multiSelected(qid).toMutableSet()
        if (!set.add(optionId)) set.remove(optionId)
        setDraft(qid, JsonArray(set.sorted().map { JsonPrimitive(it) }))
    }

    fun feedbackFromAnswers(qid: String): JsonElement? =
        ContentToolPack1Logic.arrayField(answers[qid], "attempts").lastOrNull()

    fun submit(q: InlineQuestion) {
        val value = drafts[q.id] ?: return
        if (busyId != null) return
        busyId = q.id
        scope.launch {
            try {
                val raw = props.runAction(
                    "submit",
                    buildJsonObject {
                        put("questionId", JsonPrimitive(q.id))
                        put("value", value)
                    },
                )
                if (raw != null) {
                    lastResults[q.id] = raw
                    val correct = ContentToolPack1Logic.boolField(raw, "correct") == true
                    Haptics.trigger(view, if (correct) Haptics.Kind.Success else Haptics.Kind.Error)
                    props.announce(if (correct) correctAnnounce else incorrectAnnounce, false)
                }
                gradingPending = false
            } catch (_: Exception) {
                if (ContentToolPack1Logic.canQueueActionOffline("inline_questions", "submit")) {
                    gradingPending = true
                    props.announce(gradingPendingLabel, false)
                } else {
                    props.announce(needsConnection, true)
                }
            } finally {
                busyId = null
            }
        }
    }

    Column(
        verticalArrangement = Arrangement.spacedBy(16.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            text = label,
            fontSize = 14.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        if (gradingPending) {
            Text(text = gradingPendingLabel, fontSize = 12.sp, color = textSecondary())
        }
        visibleQuestions.forEach { q ->
            val unlocked = ContentToolPack1Logic.isSequentiallyUnlocked(
                questionIds,
                answers,
                q.id,
                sequential,
            )
            val canSubmit = ContentToolPack1Logic.canSubmit(
                answers,
                q.id,
                maxAttempts,
                props.readOnly,
            ) && unlocked
            val used = ContentToolPack1Logic.attemptsUsed(answers, q.id)
            val enabled = canSubmit && busyId == null

            Column(
                verticalArrangement = Arrangement.spacedBy(10.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                NotebookContentView(markdown = q.prompt, compact = true)
                if (!unlocked) {
                    Text(text = sequentialLockedLabel, fontSize = 12.sp, color = textSecondary())
                } else {
                    when (q.type) {
                        "multi" -> {
                            q.options.forEach { (oid, text) ->
                                val selected = multiSelected(q.id).contains(oid)
                                Row(
                                    Modifier
                                        .fillMaxWidth()
                                        .heightIn(min = 44.dp)
                                        .semantics { this.selected = selected }
                                        .clickable(enabled = enabled, role = Role.Checkbox) {
                                            toggleMulti(q.id, oid)
                                        },
                                    verticalAlignment = Alignment.Top,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    Checkbox(checked = selected, onCheckedChange = null, enabled = enabled)
                                    NotebookContentView(markdown = text, compact = true)
                                }
                            }
                        }
                        "short_text", "numeric" -> {
                            OutlinedTextField(
                                value = stringDraft(q.id),
                                onValueChange = { setDraft(q.id, JsonPrimitive(it)) },
                                enabled = enabled,
                                label = { Text(yourAnswerLabel) },
                                modifier = Modifier.fillMaxWidth(),
                                minLines = 1,
                                maxLines = 4,
                            )
                        }
                        else -> {
                            q.options.forEach { (oid, text) ->
                                val selected = stringDraft(q.id) == oid
                                Row(
                                    Modifier
                                        .fillMaxWidth()
                                        .heightIn(min = 44.dp)
                                        .semantics { this.selected = selected }
                                        .clickable(enabled = enabled, role = Role.RadioButton) {
                                            setDraft(q.id, JsonPrimitive(oid))
                                        },
                                    verticalAlignment = Alignment.Top,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    RadioButton(selected = selected, onClick = null, enabled = enabled)
                                    NotebookContentView(markdown = text, compact = true)
                                }
                            }
                        }
                    }
                }
                q.unit?.takeIf { it.isNotBlank() }?.let {
                    Text(text = it, fontSize = 11.sp, color = textSecondary())
                }
                Row(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Button(
                        onClick = { submit(q) },
                        enabled = canSubmit && busyId == null && hasDraft(q.id),
                    ) {
                        Text(submitLabel)
                    }
                    if (maxAttempts != null) {
                        Text(
                            text = L.format(
                                context,
                                localePrefs,
                                R.string.mobile_contentTools_tools_inline_questions_attemptsLeft,
                                maxOf(0, maxAttempts - used),
                            ),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
                val result = lastResults[q.id] ?: feedbackFromAnswers(q.id)
                if (result != null) {
                    val correct = ContentToolPack1Logic.boolField(result, "correct") == true
                    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(
                            text = if (correct) correctLabel else incorrectLabel,
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = if (correct) LexturesColors.Primary else LexturesColors.Coral,
                        )
                        ContentToolHostLogic.stringField(result, "feedback")
                            ?.takeIf { it.isNotBlank() }
                            ?.let { NotebookContentView(markdown = it, compact = true) }
                        ContentToolHostLogic.stringField(result, "explanation")
                            ?.takeIf { it.isNotBlank() }
                            ?.let { NotebookContentView(markdown = it, compact = true) }
                    }
                }
            }
        }
        if (pagingEnabled) {
            val canAdvance = !sequential || visibleQuestions.all {
                ContentToolPack1Logic.attemptsUsed(answers, it.id) > 0
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = L.format(
                        context,
                        localePrefs,
                        R.string.mobile_contentTools_tools_inline_questions_pageOf,
                        window.first + 1,
                        window.last + 1,
                        questions.size,
                    ),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(
                        onClick = { pageIndex = maxOf(0, safePage - 1) },
                        enabled = safePage > 0,
                    ) {
                        Text(previousLabel)
                    }
                    Button(
                        onClick = { pageIndex = minOf(pageCount - 1, safePage + 1) },
                        enabled = safePage < pageCount - 1 && canAdvance,
                    ) {
                        Text(nextLabel)
                    }
                }
            }
        }
    }
}

private fun parseQuestions(
    config: JsonElement,
    instanceId: String,
    shuffleOptions: Boolean,
): List<InlineQuestion> =
    ContentToolPack1Logic.arrayField(config, "questions").mapNotNull { raw ->
        val obj = ContentToolPack1Logic.objectMap(raw)
        val id = (obj["id"] as? JsonPrimitive)?.contentOrNull?.takeIf { it.isNotEmpty() }
            ?: return@mapNotNull null
        val type = (obj["type"] as? JsonPrimitive)?.contentOrNull ?: "single"
        val prompt = (obj["prompt"] as? JsonPrimitive)?.contentOrNull.orEmpty()
        var options = ContentToolPack1Logic.arrayField(raw, "options").mapNotNull { opt ->
            val o = ContentToolPack1Logic.objectMap(opt)
            val oid = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            oid to text
        }
        if (shuffleOptions) {
            options = ContentToolPack1Logic.shuffleStable(options, "$instanceId:$id")
        }
        val unit = (obj["unit"] as? JsonPrimitive)?.contentOrNull
        InlineQuestion(id, type, prompt, options, unit)
    }
