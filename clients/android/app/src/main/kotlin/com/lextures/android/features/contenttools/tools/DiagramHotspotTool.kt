package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
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
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.roundToInt
import com.lextures.android.R
import com.lextures.android.core.design.Haptics
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack3Logic
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.contenttools.interaction.DragOrTapAssignBar
import com.lextures.android.features.contenttools.interaction.PlacementChip
import com.lextures.android.features.contenttools.interaction.ZoomableImageCanvas
import com.lextures.android.features.notebooks.NotebookContentView
import kotlin.math.roundToInt
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonPrimitive

@Composable
fun DiagramHotspotTool(props: ContentToolRendererProps) {
    val view = LocalView.current
    val scope = rememberCoroutineScope()
    val density = LocalDensity.current

    var selectedItemId by remember(props.instanceId) { mutableStateOf<String?>(null) }
    var assignments by remember(props.instanceId) { mutableStateOf<Map<String, String?>>(emptyMap()) }
    var lastCheck by remember(props.instanceId) {
        mutableStateOf<ContentToolPack3Logic.CheckResultView?>(null)
    }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    val prompt = ContentToolHostLogic.stringField(props.config, "prompt").orEmpty()
    val imageObj = ContentToolPack3Logic.objectMap(
        ContentToolPack3Logic.objectMap(props.config)["image"],
    )
    val imageURL = (imageObj["url"] as? JsonPrimitive)?.contentOrNull.orEmpty()
    val imageAlt = (imageObj["alt"] as? JsonPrimitive)?.contentOrNull.orEmpty()
    val naturalWidth = (imageObj["naturalWidth"] as? JsonPrimitive)?.doubleOrNull ?: 1.0
    val naturalHeight = (imageObj["naturalHeight"] as? JsonPrimitive)?.doubleOrNull ?: 1.0

    val regions = remember(props.config) {
        ContentToolPack3Logic.arrayField(props.config, "regions").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val label = (o["label"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val description = (o["description"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val shape = ContentToolPack3Logic.parseShape(o["shape"]) ?: return@mapNotNull null
            ContentToolPack3Logic.DiagramRegion(id, label, description, shape)
        }
    }

    val placeableItems = remember(props.config) {
        val labels = ContentToolPack3Logic.arrayField(props.config, "labels").mapNotNull { raw ->
            val o = ContentToolPack3Logic.objectMap(raw)
            val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            id to text
        }
        if (labels.isNotEmpty()) {
            labels
        } else {
            ContentToolPack3Logic.arrayField(props.config, "prompts").mapNotNull { raw ->
                val o = ContentToolPack3Logic.objectMap(raw)
                val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                val text = (o["text"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
                id to text
            }
        }
    }

    val lockedIds = ContentToolPack3Logic.arrayField(props.state, "lockedIds").mapNotNull {
        (it as? JsonPrimitive)?.contentOrNull
    }
    val attemptsUsed = ContentToolPack3Logic.arrayField(props.state, "attempts").size
    val maxAttempts = ContentToolPack3Logic.parseAttemptsConfig(
        ContentToolPack3Logic.objectMap(props.config)["attempts"],
    )
    val attemptsExhausted = maxAttempts != null && attemptsUsed >= maxAttempts
    val canEdit = !props.readOnly && !attemptsExhausted
    val allAssigned = placeableItems.all { item ->
        val region = assignments[item.first]
        !region.isNullOrEmpty()
    }
    val selectedItemLabel = selectedItemId?.let { id ->
        placeableItems.firstOrNull { it.first == id }?.second
    }
    val lastPerItem: Map<String, Boolean> = lastCheck?.perItem ?: run {
        val map = ContentToolPack3Logic.objectMap(
            ContentToolPack3Logic.objectMap(props.state)["lastPerItem"],
        )
        map.mapNotNull { (k, v) ->
            val flag = v.jsonPrimitive.booleanOrNull
            if (flag != null) k to flag else null
        }.toMap()
    }

    val imageMissing = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_imageMissing)
    val listPickerHint = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_listPickerHint)
    val labelsTitle = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_labels)
    val targetsTitle = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_targets)
    val reviewTitle = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_reviewPlacements)
    val unplacedLabel = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_unplaced)
    val checkLabel = L.text(R.string.mobile_contentTools_runtime_checkAnswer)
    val resetLabel = L.text(R.string.mobile_contentTools_runtime_reset)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val maxAttemptsError = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_error_maxAttempts)
    val checkedAnnounce = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_checked)
    val resetAnnounce = L.text(R.string.mobile_contentTools_tools_diagram_hotspot_reset)
    val placedAnnounce = L.text(R.string.mobile_contentTools_interaction_placed)
    val tapToPlace = L.text(R.string.mobile_contentTools_interaction_tapToPlace)

    fun hydrate() {
        val raw = ContentToolPack3Logic.objectMap(props.state)["assignments"]
        val map = ContentToolPack3Logic.parseCategorizePlacement(raw).toMutableMap()
        for (item in placeableItems) {
            if (item.first !in map) map[item.first] = null
        }
        assignments = map
    }

    fun persist(usedListMode: Boolean) {
        if (props.readOnly) return
        val stateMap = ContentToolPack3Logic.objectMap(props.state)
        val patch = ContentToolPack3Logic.mergePreservingUnknown(
            stateMap,
            mapOf(
                "v" to JsonPrimitive(1),
                "assignments" to ContentToolPack3Logic.categorizePlacementJson(assignments),
                "usedListMode" to JsonPrimitive(usedListMode),
                "attempts" to (stateMap["attempts"] ?: JsonArray(emptyList())),
                "lockedIds" to (stateMap["lockedIds"] ?: JsonArray(emptyList())),
            ),
        )
        props.save(patch)
    }

    fun placeOnRegion(regionId: String) {
        val itemId = selectedItemId ?: return
        if (!canEdit || itemId in lockedIds) return
        assignments = assignments + (itemId to regionId)
        selectedItemId = null
        Haptics.trigger(view, Haptics.Kind.Success)
        persist(usedListMode = true)
        props.announce(placedAnnounce, false)
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
                val result = props.runAction(
                    "check",
                    buildJsonObject {
                        put("assignments", ContentToolPack3Logic.categorizePlacementJson(assignments))
                        put("usedListMode", JsonPrimitive(true))
                    },
                )
                val parsed = ContentToolPack3Logic.parseCheckResult(result)
                lastCheck = parsed
                if (parsed.error != null) {
                    Haptics.trigger(view, Haptics.Kind.Error)
                    errorText = parsed.message
                } else {
                    Haptics.trigger(
                        view,
                        if ((parsed.scorePct ?: 0.0) >= 100) Haptics.Kind.Success else Haptics.Kind.Selection,
                    )
                    props.announce(checkedAnnounce, false)
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
                props.announce(resetAnnounce, false)
                hydrate()
            } catch (_: Exception) {
                errorText = retryLabel
            } finally {
                busy = false
            }
        }
    }

    fun regionCenter(region: ContentToolPack3Logic.DiagramRegion, size: Size): Offset {
        val (nx, ny) = when (val shape = region.shape) {
            is ContentToolPack3Logic.RegionShape.Rect ->
                (shape.x + shape.w / 2) to (shape.y + shape.h / 2)
            is ContentToolPack3Logic.RegionShape.Circle ->
                shape.cx to shape.cy
            is ContentToolPack3Logic.RegionShape.Polygon -> {
                if (shape.points.isEmpty()) {
                    return Offset(size.width / 2, size.height / 2)
                }
                val avgX = shape.points.map { it.first }.average()
                val avgY = shape.points.map { it.second }.average()
                avgX to avgY
            }
        }
        val scale = minOf(size.width / naturalWidth.toFloat(), size.height / naturalHeight.toFloat())
        val drawW = naturalWidth.toFloat() * scale
        val drawH = naturalHeight.toFloat() * scale
        val offsetX = (size.width - drawW) / 2
        val offsetY = (size.height - drawH) / 2
        return Offset(offsetX + nx.toFloat() * drawW, offsetY + ny.toFloat() * drawH)
    }

    LaunchedEffect(props.state, props.instanceId) { hydrate() }

    val halfTargetPx = with(density) { 22.dp.toPx() }

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (prompt.isNotEmpty()) {
            NotebookContentView(markdown = prompt, compact = true)
        }

        if (imageURL.isNotEmpty()) {
            ZoomableImageCanvas(
                urlString = imageURL,
                alt = imageAlt,
                naturalWidth = naturalWidth,
                naturalHeight = naturalHeight,
            ) { size ->
                Box(modifier = Modifier.fillMaxSize()) {
                    regions.forEach { region ->
                        val center = regionCenter(region, size)
                        Box(
                            modifier = Modifier
                                .offset {
                                    IntOffset(
                                        (center.x - halfTargetPx).roundToInt(),
                                        (center.y - halfTargetPx).roundToInt(),
                                    )
                                }
                                .size(44.dp)
                                .clickable(
                                    enabled = canEdit && selectedItemId != null,
                                    onClick = { placeOnRegion(region.id) },
                                )
                                .semantics {
                                    contentDescription = "${region.label}. ${region.description}"
                                },
                        )
                    }
                }
            }
        } else {
            Text(
                text = imageAlt.ifEmpty { imageMissing },
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }

        if (canEdit) {
            DragOrTapAssignBar(
                selectedLabel = selectedItemLabel,
                helperText = listPickerHint,
            )
        }

        // List-based placement (FR-18) — always visible, not a11y-only.
        Text(labelsTitle, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        placeableItems.forEach { (id, text) ->
            PlacementChip(
                title = text,
                selected = selectedItemId == id,
                locked = id in lockedIds,
                correct = lastPerItem[id],
                disabled = !canEdit,
                onClick = {
                    if (selectedItemId == id) {
                        selectedItemId = null
                    } else {
                        selectedItemId = id
                        Haptics.trigger(view, Haptics.Kind.Tap)
                    }
                },
            )
        }

        Text(targetsTitle, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        regions.forEach { region ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 44.dp)
                    .clickable(
                        enabled = canEdit &&
                            selectedItemId != null &&
                            selectedItemId !in lockedIds,
                        onClick = { placeOnRegion(region.id) },
                    )
                    .semantics { contentDescription = "${region.label}. $tapToPlace" }
                    .padding(vertical = 6.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    Icons.Default.Place,
                    contentDescription = null,
                    tint = LexturesColors.Primary,
                )
                Column(modifier = Modifier.weight(1f)) {
                    Text(region.label, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                    Text(region.description, fontSize = 12.sp, color = textSecondary())
                }
            }
        }

        lastCheck?.let { result ->
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                result.scorePct?.let { score ->
                    val label = L.format(
                        R.string.mobile_contentTools_tools_diagram_hotspot_scorePct,
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
                        Text(label, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
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

        // Text list of placements / results (FR-20)
        Text(reviewTitle, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        placeableItems.forEach { (id, text) ->
            val regionId = assignments[id]
            val regionLabel = regions.firstOrNull { it.id == regionId }?.label
            val correct = lastPerItem[id]
            val dest = regionLabel ?: unplacedLabel
            val a11y = L.format(
                R.string.mobile_contentTools_tools_diagram_hotspot_placementA11y,
                text,
                dest,
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .semantics { contentDescription = a11y },
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (correct != null) {
                    Icon(
                        imageVector = if (correct) Icons.Default.CheckCircle else Icons.Default.Cancel,
                        contentDescription = null,
                        tint = if (correct) LexturesColors.Primary else LexturesColors.Coral,
                    )
                }
                Text(text, fontSize = 12.sp)
                Text("→", fontSize = 12.sp)
                Text(dest, fontSize = 12.sp, color = textSecondary())
            }
        }

        if (canEdit) {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = { check() }, enabled = !busy && allAssigned) {
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
