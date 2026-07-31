package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
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
import com.lextures.android.features.contenttools.interaction.DragOrTapAssignBar
import com.lextures.android.features.contenttools.interaction.PlacementChip
import com.lextures.android.features.notebooks.NotebookContentView
import kotlin.math.roundToInt
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive

@Composable
fun SortSequenceTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val scope = rememberCoroutineScope()

    var engine by remember(props.instanceId) {
        mutableStateOf(
            ContentToolPack3Logic.createInitialEngineState(
                ContentToolPack3Logic.PlacementMode.CATEGORIZE,
                emptyList(),
            ),
        )
    }
    var lastCheck by remember(props.instanceId) {
        mutableStateOf<ContentToolPack3Logic.CheckResultView?>(null)
    }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    val mode = if (ContentToolHostLogic.stringField(props.config, "mode") == "order") {
        ContentToolPack3Logic.PlacementMode.ORDER
    } else {
        ContentToolPack3Logic.PlacementMode.CATEGORIZE
    }
    val prompt = ContentToolHostLogic.stringField(props.config, "prompt").orEmpty()
    val items = remember(props.config) {
        ContentToolPack3Logic.arrayField(props.config, "items").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            id to text
        }
    }
    val buckets = remember(props.config) {
        ContentToolPack3Logic.arrayField(props.config, "buckets").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val label = (o["label"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            id to label
        }
    }
    val itemIds = remember(items) { items.map { it.first } }
    val lockedItemIds = ContentToolPack3Logic.arrayField(props.state, "lockedItemIds").mapNotNull {
        (it as? JsonPrimitive)?.contentOrNull
    }
    val attemptsUsed = ContentToolPack3Logic.arrayField(props.state, "attempts").size
    val maxAttempts = ContentToolPack3Logic.parseAttemptsConfig(
        ContentToolPack3Logic.objectMap(props.config)["attempts"],
    )
    val attemptsExhausted = maxAttempts != null && attemptsUsed >= maxAttempts
    val canEdit = !props.readOnly && !attemptsExhausted
    val allPlaced = ContentToolPack3Logic.allPlaced(mode, itemIds, engine.placement)
    val selectedItemLabel = engine.grabbedId?.let { id -> items.firstOrNull { it.first == id }?.second }
    val lastPerItem: Map<String, Boolean> = lastCheck?.perItem ?: run {
        val map = ContentToolPack3Logic.objectMap(
            ContentToolPack3Logic.objectMap(props.state)["lastPerItem"],
        )
        map.mapNotNull { (k, v) ->
            val flag = v.jsonPrimitive.booleanOrNull
            if (flag != null) k to flag else null
        }.toMap()
    }

    val trayLabel = L.text(R.string.mobile_contentTools_tools_sort_sequence_tray)
    val trayEmptyLabel = L.text(R.string.mobile_contentTools_tools_sort_sequence_trayEmpty)
    val sequenceLabel = L.text(R.string.mobile_contentTools_tools_sort_sequence_sequence)
    val insertEndLabel = L.text(R.string.mobile_contentTools_tools_sort_sequence_insertEnd)
    val checkLabel = L.text(R.string.mobile_contentTools_runtime_checkAnswer)
    val resetLabel = L.text(R.string.mobile_contentTools_runtime_reset)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val maxAttemptsError = L.text(R.string.mobile_contentTools_tools_sort_sequence_error_maxAttempts)
    val allCorrectAnnounce = L.text(R.string.mobile_contentTools_tools_sort_sequence_allCorrect)
    val checkedAnnounce = L.text(R.string.mobile_contentTools_tools_sort_sequence_checked)
    val resetAnnounce = L.text(R.string.mobile_contentTools_tools_sort_sequence_reset)
    val placedAnnounce = L.text(R.string.mobile_contentTools_interaction_placed)
    val movedAnnounce = L.text(R.string.mobile_contentTools_interaction_moved)
    val tapToSelect = L.text(R.string.mobile_contentTools_interaction_tapToSelect)
    val tapToPlace = L.text(R.string.mobile_contentTools_interaction_tapToPlace)
    val moveUpLabel = L.text(R.string.mobile_contentTools_interaction_moveUp)
    val moveDownLabel = L.text(R.string.mobile_contentTools_interaction_moveDown)
    val helperText = L.text(
        if (mode == ContentToolPack3Logic.PlacementMode.ORDER) {
            R.string.mobile_contentTools_tools_sort_sequence_tapOrderHint
        } else {
            R.string.mobile_contentTools_tools_sort_sequence_tapCategorizeHint
        },
    )

    fun hydrate() {
        val raw = ContentToolPack3Logic.objectMap(props.state)["placement"]
        val placement: ContentToolPack3Logic.Placement = when (mode) {
            ContentToolPack3Logic.PlacementMode.ORDER ->
                ContentToolPack3Logic.Placement.Order(ContentToolPack3Logic.parseOrderPlacement(raw))
            ContentToolPack3Logic.PlacementMode.CATEGORIZE -> {
                var map = ContentToolPack3Logic.parseCategorizePlacement(raw)
                if (map.isEmpty()) {
                    map = itemIds.associateWith { null }
                }
                ContentToolPack3Logic.Placement.Categorize(map)
            }
        }
        engine = ContentToolPack3Logic.createInitialEngineState(mode, itemIds, placement)
    }

    fun persist() {
        if (props.readOnly) return
        val placementJson: JsonElement = when (mode) {
            ContentToolPack3Logic.PlacementMode.ORDER ->
                ContentToolPack3Logic.orderPlacementJson(
                    (engine.placement as? ContentToolPack3Logic.Placement.Order)?.ids.orEmpty(),
                )
            ContentToolPack3Logic.PlacementMode.CATEGORIZE ->
                ContentToolPack3Logic.categorizePlacementJson(
                    (engine.placement as? ContentToolPack3Logic.Placement.Categorize)?.map.orEmpty(),
                )
        }
        val stateMap = ContentToolPack3Logic.objectMap(props.state)
        val patch = ContentToolPack3Logic.mergePreservingUnknown(
            stateMap,
            mapOf(
                "v" to JsonPrimitive(1),
                "placement" to placementJson,
                "attempts" to (stateMap["attempts"] ?: JsonArray(emptyList())),
                "lockedItemIds" to (stateMap["lockedItemIds"] ?: JsonArray(emptyList())),
            ),
        )
        props.save(patch)
    }

    fun tap(hit: ContentToolPack3Logic.PlacementHit) {
        if (!canEdit) return
        val next = ContentToolPack3Logic.tapItemOrTarget(engine, mode, lockedItemIds, hit)
        if (next.grabbedId != engine.grabbedId) {
            Haptics.trigger(
                view,
                if (next.grabbedId == null) Haptics.Kind.Selection else Haptics.Kind.Tap,
            )
        }
        engine = next
        if (next.grabbedId == null) {
            persist()
            props.announce(placedAnnounce, false)
        }
    }

    fun move(itemId: String, direction: Int) {
        if (!canEdit || mode != ContentToolPack3Logic.PlacementMode.ORDER) return
        val order = (engine.placement as? ContentToolPack3Logic.Placement.Order)?.ids.orEmpty()
        val next = ContentToolPack3Logic.moveInOrder(order, itemId, direction, lockedItemIds)
        if (next == order) return
        engine = engine.copy(placement = ContentToolPack3Logic.Placement.Order(next))
        Haptics.trigger(view, Haptics.Kind.Selection)
        persist()
        props.announce(movedAnnounce, false)
    }

    fun check() {
        if (!ContentToolPack3Logic.canCheck(attemptsUsed, maxAttempts, props.readOnly)) {
            errorText = maxAttemptsError
            return
        }
        busy = true
        errorText = null
        scope.launch {
            try {
                val placementJson: JsonElement = when (mode) {
                    ContentToolPack3Logic.PlacementMode.ORDER ->
                        ContentToolPack3Logic.orderPlacementJson(
                            (engine.placement as? ContentToolPack3Logic.Placement.Order)?.ids.orEmpty(),
                        )
                    ContentToolPack3Logic.PlacementMode.CATEGORIZE ->
                        ContentToolPack3Logic.categorizePlacementJson(
                            (engine.placement as? ContentToolPack3Logic.Placement.Categorize)?.map.orEmpty(),
                        )
                }
                val result = props.runAction(
                    "check",
                    buildJsonObject { put("placement", placementJson) },
                )
                val parsed = ContentToolPack3Logic.parseCheckResult(result)
                lastCheck = parsed
                if (parsed.error == ContentToolPack3Logic.CheckError.MAX_ATTEMPTS) {
                    errorText = parsed.message ?: maxAttemptsError
                    Haptics.trigger(view, Haptics.Kind.Error)
                } else {
                    val perfect = (parsed.scorePct ?: 0.0) >= 100
                    Haptics.trigger(view, if (perfect) Haptics.Kind.Success else Haptics.Kind.Selection)
                    props.announce(if (perfect) allCorrectAnnounce else checkedAnnounce, false)
                }
                hydrate()
            } catch (_: Exception) {
                errorText = retryLabel
                Haptics.trigger(view, Haptics.Kind.Error)
            } finally {
                busy = false
            }
        }
    }

    fun resetAttempt() {
        busy = true
        errorText = null
        scope.launch {
            try {
                props.runAction("reset_attempt", buildJsonObject { })
                lastCheck = null
                engine = ContentToolPack3Logic.restoreAfterDragInterrupt(
                    ContentToolPack3Logic.createInitialEngineState(mode, itemIds),
                )
                props.announce(resetAnnounce, false)
                hydrate()
            } catch (_: Exception) {
                errorText = retryLabel
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(props.state, props.config, props.instanceId) { hydrate() }

    val tray = ContentToolPack3Logic.trayItemIds(mode, itemIds, engine.placement)
    val order = (engine.placement as? ContentToolPack3Logic.Placement.Order)?.ids.orEmpty()

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (prompt.isNotEmpty()) {
            NotebookContentView(markdown = prompt, compact = true)
        }

        if (canEdit) {
            DragOrTapAssignBar(selectedLabel = selectedItemLabel, helperText = helperText)
        }

        if (mode == ContentToolPack3Logic.PlacementMode.CATEGORIZE) {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(trayLabel, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                tray.forEach { id ->
                    SortItemChip(
                        id = id,
                        items = items,
                        grabbedId = engine.grabbedId,
                        lockedItemIds = lockedItemIds,
                        correct = lastPerItem[id],
                        canEdit = canEdit,
                        tapToSelect = tapToSelect,
                        tapToPlace = tapToPlace,
                        onTap = ::tap,
                    )
                }
                if (tray.isEmpty()) {
                    Text(trayEmptyLabel, fontSize = 12.sp, color = textSecondary())
                }
                buckets.forEach { (bucketId, label) ->
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .border(1.dp, fieldBorder(), RoundedCornerShape(8.dp))
                            .padding(8.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Text(
                            text = label,
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold,
                            modifier = Modifier
                                .fillMaxWidth()
                                .heightIn(min = 44.dp)
                                .clickable(enabled = canEdit) {
                                    tap(ContentToolPack3Logic.PlacementHit.Bucket(bucketId))
                                }
                                .semantics { contentDescription = "$label. $tapToPlace" }
                                .padding(vertical = 8.dp),
                        )
                        ContentToolPack3Logic.itemsInBucket(engine.placement, bucketId).forEach { id ->
                            SortItemChip(
                                id = id,
                                items = items,
                                grabbedId = engine.grabbedId,
                                lockedItemIds = lockedItemIds,
                                correct = lastPerItem[id],
                                canEdit = canEdit,
                                tapToSelect = tapToSelect,
                                tapToPlace = tapToPlace,
                                onTap = ::tap,
                            )
                        }
                    }
                }
            }
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(trayLabel, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                tray.forEach { id ->
                    SortItemChip(
                        id = id,
                        items = items,
                        grabbedId = engine.grabbedId,
                        lockedItemIds = lockedItemIds,
                        correct = lastPerItem[id],
                        canEdit = canEdit,
                        tapToSelect = tapToSelect,
                        tapToPlace = tapToPlace,
                        onTap = ::tap,
                    )
                }
                Text(sequenceLabel, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                order.forEachIndexed { index, id ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = "${index + 1}.",
                            fontSize = 12.sp,
                            fontFamily = FontFamily.Monospace,
                            modifier = Modifier
                                .width(28.dp)
                                .clickable(enabled = canEdit) {
                                    tap(ContentToolPack3Logic.PlacementHit.Position(index))
                                },
                        )
                        SortItemChip(
                            id = id,
                            items = items,
                            grabbedId = engine.grabbedId,
                            lockedItemIds = lockedItemIds,
                            correct = lastPerItem[id],
                            canEdit = canEdit,
                            tapToSelect = tapToSelect,
                            tapToPlace = tapToPlace,
                            onTap = ::tap,
                            modifier = Modifier.weight(1f),
                        )
                        if (canEdit && id !in lockedItemIds) {
                            Column {
                                IconButton(onClick = { move(id, -1) }) {
                                    Icon(Icons.Default.KeyboardArrowUp, contentDescription = moveUpLabel)
                                }
                                IconButton(onClick = { move(id, 1) }) {
                                    Icon(Icons.Default.KeyboardArrowDown, contentDescription = moveDownLabel)
                                }
                            }
                        }
                    }
                }
                if (canEdit) {
                    Text(
                        text = insertEndLabel,
                        fontSize = 12.sp,
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(min = 44.dp)
                            .clickable(enabled = engine.grabbedId != null) {
                                tap(ContentToolPack3Logic.PlacementHit.Position(order.size))
                            }
                            .padding(vertical = 12.dp),
                    )
                }
            }
        }

        lastCheck?.let { result ->
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                result.scorePct?.let { score ->
                    val label = L.format(
                        R.string.mobile_contentTools_tools_sort_sequence_scorePct,
                        score.roundToInt(),
                    )
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            imageVector = if (score >= 100) Icons.Default.CheckCircle else Icons.Default.Info,
                            contentDescription = null,
                            tint = if (score >= 100) LexturesColors.Primary else textSecondary(),
                        )
                        Text(
                            text = label,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = if (score >= 100) LexturesColors.Primary else textSecondary(),
                        )
                    }
                }
                result.error?.let {
                    Text(
                        text = result.message ?: it.code,
                        fontSize = 12.sp,
                        color = LexturesColors.Coral,
                    )
                }
            }
        }

        if (canEdit) {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = { check() }, enabled = !busy && allPlaced) {
                    Text(checkLabel)
                }
                TextButton(onClick = { resetAttempt() }, enabled = !busy) {
                    Text(resetLabel)
                }
            }
        }

        errorText?.let {
            Text(it, fontSize = 12.sp, color = LexturesColors.Coral)
        }
    }
}

@Composable
private fun SortItemChip(
    id: String,
    items: List<Pair<String, String>>,
    grabbedId: String?,
    lockedItemIds: List<String>,
    correct: Boolean?,
    canEdit: Boolean,
    tapToSelect: String,
    tapToPlace: String,
    onTap: (ContentToolPack3Logic.PlacementHit) -> Unit,
    modifier: Modifier = Modifier,
) {
    val text = items.firstOrNull { it.first == id }?.second ?: id
    PlacementChip(
        title = text,
        selected = grabbedId == id,
        locked = id in lockedItemIds,
        correct = correct,
        disabled = !canEdit,
        onClick = { onTap(ContentToolPack3Logic.PlacementHit.Item(id)) },
        modifier = modifier.semantics {
            contentDescription = if (grabbedId == null) "$text. $tapToSelect" else "$text. $tapToPlace"
        },
    )
}
