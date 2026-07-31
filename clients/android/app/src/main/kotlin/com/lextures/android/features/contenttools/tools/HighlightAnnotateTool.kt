package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material.icons.filled.Warning
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
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack3Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.contenttools.interaction.PassageAnnotationMark
import com.lextures.android.features.contenttools.interaction.PassageSelectionView
import com.lextures.android.features.notebooks.NotebookContentView
import java.time.Instant
import java.util.UUID
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull

private data class HighlightAnn(
    val id: String,
    val tagId: String,
    val quote: String,
    val prefix: String,
    val suffix: String,
    val approxOffset: Int,
    val unitIndex: Int?,
    val note: String?,
    val orphaned: Boolean,
)

@Composable
fun HighlightAnnotateTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val scope = rememberCoroutineScope()

    var selectedUnitIndex by remember(props.instanceId) { mutableStateOf<Int?>(null) }
    var selectedTagId by remember(props.instanceId) { mutableStateOf<String?>(null) }
    var noteDraft by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    val prompt = ContentToolHostLogic.stringField(props.config, "prompt").orEmpty()
    val passageMarkdown = ContentToolHostLogic.stringField(props.config, "passageMarkdown").orEmpty()
    val passage = remember(passageMarkdown) {
        ContentToolPack3Logic.plainPassageFromMarkdown(passageMarkdown)
    }
    val granularity = ContentToolHostLogic.stringField(props.config, "unitGranularity") ?: "sentence"
    val units = remember(passage, granularity) {
        ContentToolPack3Logic.segmentPassage(passage, granularity)
    }
    val tags = remember(props.config) {
        ContentToolPack3Logic.arrayField(props.config, "tags").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val label = (o["label"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val color = (o["color"] as? JsonPrimitive)?.contentOrNull.orEmpty()
            Triple(id, label, color)
        }
    }
    val maxAnnotations = (ContentToolPack3Logic.numberField(props.config, "maxAnnotations") ?: 20.0).toInt()
    val requireNote = ContentToolPack3Logic.boolField(props.config, "requireNote") == true
    val canEdit = !props.readOnly

    val annotations = remember(props.state) {
        ContentToolPack3Logic.arrayField(props.state, "annotations").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val tagId = (o["tagId"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val quote = (o["quote"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val anchor = ContentToolPack3Logic.objectMap(o["anchor"])
            val prefix = (anchor["prefix"] as? JsonPrimitive)?.contentOrNull.orEmpty()
            val suffix = (anchor["suffix"] as? JsonPrimitive)?.contentOrNull.orEmpty()
            val offset = (ContentToolPack3Logic.numberField(o["anchor"], "approxOffset") ?: 0.0).toInt()
            val unitIndex = ContentToolPack3Logic.numberField(o["anchor"], "unitIndex")?.toInt()
            val note = (o["note"] as? JsonPrimitive)?.contentOrNull
            val orphaned = ContentToolPack3Logic.boolField(raw, "orphaned") == true
            HighlightAnn(id, tagId, quote, prefix, suffix, offset, unitIndex, note, orphaned)
        }
    }

    val resolvedAnnotations = remember(annotations, passage, tags) {
        annotations.mapNotNull { ann ->
            val anchor = ContentToolPack3Logic.QuoteAnchor(
                prefix = ann.prefix,
                suffix = ann.suffix,
                approxOffset = ann.approxOffset,
                unitIndex = ann.unitIndex,
            )
            val range = ContentToolPack3Logic.resolveQuoteAnchor(passage, ann.quote, anchor)
                ?: return@mapNotNull null
            val tag = tags.firstOrNull { it.first == ann.tagId }
            PassageAnnotationMark(
                id = ann.id,
                start = range.start,
                end = range.end,
                tagLabel = tag?.second ?: ann.tagId,
                tagColor = tag?.third.orEmpty(),
            )
        }
    }

    val notePlaceholder = L.text(R.string.mobile_contentTools_tools_highlight_annotate_notePlaceholder)
    val addLabel = L.text(R.string.mobile_contentTools_tools_highlight_annotate_add)
    val yourAnnotations = L.text(R.string.mobile_contentTools_tools_highlight_annotate_yourAnnotations)
    val chooseTag = L.text(R.string.mobile_contentTools_tools_highlight_annotate_chooseTag)
    val deleteLabel = L.text(R.string.mobile_contentTools_tools_highlight_annotate_delete)
    val noteRequired = L.text(R.string.mobile_contentTools_tools_highlight_annotate_noteRequired)
    val noteFiltered = L.text(R.string.mobile_contentTools_tools_highlight_annotate_noteFiltered)
    val addedAnnounce = L.text(R.string.mobile_contentTools_tools_highlight_annotate_added)
    val deletedAnnounce = L.text(R.string.mobile_contentTools_tools_highlight_annotate_deleted)
    val unitSelectedAnnounce = L.text(R.string.mobile_contentTools_tools_highlight_annotate_unitSelected)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)

    fun persist(list: List<HighlightAnn>) {
        val arr = list.map { ann ->
            buildJsonObject {
                put("id", JsonPrimitive(ann.id))
                put("tagId", JsonPrimitive(ann.tagId))
                put("quote", JsonPrimitive(ann.quote))
                put(
                    "anchor",
                    buildJsonObject {
                        put("prefix", JsonPrimitive(ann.prefix))
                        put("suffix", JsonPrimitive(ann.suffix))
                        put("approxOffset", JsonPrimitive(ann.approxOffset))
                        ann.unitIndex?.let { put("unitIndex", JsonPrimitive(it)) }
                    },
                )
                put("createdAt", JsonPrimitive(Instant.now().toString()))
                ann.note?.let { put("note", JsonPrimitive(it)) }
                if (ann.orphaned) put("orphaned", JsonPrimitive(true))
            }
        }
        val patch = ContentToolPack3Logic.mergePreservingUnknown(
            ContentToolPack3Logic.objectMap(props.state),
            mapOf(
                "v" to JsonPrimitive(1),
                "annotations" to JsonArray(arr),
            ),
        )
        props.save(patch)
    }

    fun deleteAnnotation(id: String) {
        if (!canEdit) return
        persist(annotations.filter { it.id != id })
        props.announce(deletedAnnounce, false)
    }

    fun addAnnotation() {
        if (!canEdit) return
        val unitIdx = selectedUnitIndex ?: return
        val tagId = selectedTagId ?: return
        if (unitIdx !in units.indices) return
        val unit = units[unitIdx]
        val built = ContentToolPack3Logic.buildQuoteAnchor(
            passage,
            unit.start,
            unit.end,
            unit.index,
        ) ?: return

        if (requireNote && noteDraft.trim().isEmpty()) {
            errorText = noteRequired
            return
        }

        busy = true
        errorText = null
        scope.launch {
            try {
                val note = noteDraft.trim()
                if (note.isNotEmpty()) {
                    val filtered = props.runAction(
                        "filter_note",
                        buildJsonObject { put("note", JsonPrimitive(note)) },
                    )
                    val err = ContentToolHostLogic.stringField(filtered, "error")
                    if (err == "filtered") {
                        errorText = ContentToolHostLogic.stringField(filtered, "message") ?: noteFiltered
                        return@launch
                    }
                }
                val (quote, anchor) = built
                val list = annotations + HighlightAnn(
                    id = UUID.randomUUID().toString(),
                    tagId = tagId,
                    quote = quote,
                    prefix = anchor.prefix,
                    suffix = anchor.suffix,
                    approxOffset = anchor.approxOffset,
                    unitIndex = anchor.unitIndex,
                    note = note.ifEmpty { null },
                    orphaned = false,
                )
                persist(list)
                noteDraft = ""
                selectedUnitIndex = null
                Haptics.trigger(view, Haptics.Kind.Success)
                props.announce(addedAnnounce, false)
            } catch (_: Exception) {
                errorText = retryLabel
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(tags) {
        if (selectedTagId == null) {
            selectedTagId = tags.firstOrNull()?.first
        }
    }

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (prompt.isNotEmpty()) {
            NotebookContentView(markdown = prompt, compact = true)
        }

        PassageSelectionView(
            passage = passage,
            units = units,
            annotations = resolvedAnnotations,
            selectedUnitIndex = selectedUnitIndex,
            readOnly = props.readOnly,
            onSelectUnit = { idx ->
                selectedUnitIndex = idx
                props.announce(unitSelectedAnnounce, false)
            },
        )

        if (canEdit && selectedUnitIndex != null) {
            Text(chooseTag, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
            tags.forEach { (id, label, _) ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 44.dp)
                        .clickable(role = Role.Button) { selectedTagId = id }
                        .semantics { selected = selectedTagId == id }
                        .padding(vertical = 6.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        imageVector = if (selectedTagId == id) {
                            Icons.Default.CheckCircle
                        } else {
                            Icons.Default.RadioButtonUnchecked
                        },
                        contentDescription = null,
                        tint = LexturesColors.Primary,
                    )
                    Text(label, fontSize = 14.sp)
                }
            }
            OutlinedTextField(
                value = noteDraft,
                onValueChange = { noteDraft = it },
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text(notePlaceholder) },
                enabled = !busy,
                minLines = 2,
                maxLines = 5,
            )
            Button(
                onClick = { addAnnotation() },
                enabled = !busy && selectedTagId != null && annotations.size < maxAnnotations,
            ) {
                Text(addLabel)
            }
        }

        if (annotations.isNotEmpty()) {
            Text(yourAnnotations, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
            annotations.forEach { ann ->
                val tag = tags.firstOrNull { it.first == ann.tagId }
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .border(1.dp, fieldBorder(), RoundedCornerShape(8.dp))
                        .padding(8.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            imageVector = if (ann.orphaned) Icons.Default.Warning else Icons.Default.Edit,
                            contentDescription = null,
                            tint = if (ann.orphaned) LexturesColors.Amber else LexturesColors.Primary,
                        )
                        Text(
                            text = tag?.second ?: ann.tagId,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                    Text("“${ann.quote}”", fontSize = 14.sp)
                    if (!ann.note.isNullOrEmpty()) {
                        Text(ann.note, fontSize = 12.sp, color = textSecondary())
                    }
                    if (canEdit) {
                        TextButton(onClick = { deleteAnnotation(ann.id) }) {
                            Text(deleteLabel, color = LexturesColors.Coral)
                        }
                    }
                }
            }
        }

        errorText?.let {
            Text(it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}
