package com.lextures.android.features.live

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Redo
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.ChangeHistory
import androidx.compose.material.icons.filled.CleaningServices
import androidx.compose.material.icons.filled.CropSquare
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Draw
import androidx.compose.material.icons.filled.HorizontalRule
import androidx.compose.material.icons.filled.NearMe
import androidx.compose.material.icons.outlined.Circle
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CourseWhiteboard
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.WhiteboardElement
import com.lextures.android.core.lms.WhiteboardLogic
import com.lextures.android.core.lms.WhiteboardObservability
import com.lextures.android.core.lms.WhiteboardRenderer
import com.lextures.android.core.lms.WhiteboardTool
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private fun toolIcon(tool: WhiteboardTool): ImageVector = when (tool) {
    WhiteboardTool.SELECT -> Icons.Default.NearMe
    WhiteboardTool.PEN -> Icons.Default.Draw
    WhiteboardTool.LINE -> Icons.Default.HorizontalRule
    WhiteboardTool.RECT -> Icons.Default.CropSquare
    WhiteboardTool.CIRCLE -> Icons.Outlined.Circle
    WhiteboardTool.TRIANGLE -> Icons.Default.ChangeHistory
    WhiteboardTool.ERASER -> Icons.Default.CleaningServices
}

private fun parseHexColor(raw: String): Color {
    var hex = raw.trim()
    if (hex.startsWith("#")) hex = hex.drop(1)
    if (hex.length != 6) return Color.Black
    val value = hex.toLongOrNull(16) ?: return Color.Black
    return Color(
        red = ((value shr 16) and 0xFF) / 255f,
        green = ((value shr 8) and 0xFF) / 255f,
        blue = (value and 0xFF) / 255f,
    )
}

@Composable
fun WhiteboardDialog(
    session: AuthSession,
    course: CourseSummary,
    board: CourseWhiteboard,
    canEdit: Boolean,
    onDismiss: () -> Unit,
    onDeleted: (() -> Unit)? = null,
) {
    if (canEdit) {
        WhiteboardEditorDialog(
            session = session,
            course = course,
            board = board,
            onDismiss = onDismiss,
            onDeleted = onDeleted,
        )
    } else {
        WhiteboardReadOnlyDialog(
            session = session,
            course = course,
            board = board,
            onDismiss = onDismiss,
        )
    }
}

@Composable
private fun WhiteboardReadOnlyDialog(
    session: AuthSession,
    course: CourseSummary,
    board: CourseWhiteboard,
    onDismiss: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    var loaded by remember(board.id) { mutableStateOf(board) }
    var scale by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    val isDark = isDarkTheme()

    LaunchedEffect(accessToken, board.id) {
        val token = accessToken ?: return@LaunchedEffect
        loaded = runCatching { LmsApi.fetchCourseWhiteboard(course.courseCode, board.id, token) }.getOrDefault(board)
    }

    androidx.compose.material3.AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(loaded.title) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(320.dp)
                        .graphicsLayer {
                            scaleX = scale
                            scaleY = scale
                            translationX = offset.x
                            translationY = offset.y
                        }
                        .pointerInput(Unit) {
                            detectTransformGestures { _, pan, zoom, _ ->
                                scale = (scale * zoom).coerceIn(0.5f, 4f)
                                offset += pan
                            }
                        },
                ) {
                    Canvas(modifier = Modifier.fillMaxSize()) {
                        WhiteboardRenderer.drawGrid(this, isDark)
                        loaded.canvasData.orEmpty().forEach { WhiteboardRenderer.drawElement(this, it) }
                    }
                }
                Text(L.text(R.string.mobile_live_whiteboard_readOnlyNotice), fontSize = 12.sp, color = textSecondary())
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text(liveClose()) } },
    )
}

private enum class SaveState { Idle, Saving, Saved, Failed }

@Composable
private fun WhiteboardEditorDialog(
    session: AuthSession,
    course: CourseSummary,
    board: CourseWhiteboard,
    onDismiss: () -> Unit,
    onDeleted: (() -> Unit)?,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val isDark = isDarkTheme()

    var title by remember(board.id) { mutableStateOf(board.title) }
    var elements by remember(board.id) { mutableStateOf(board.canvasData.orEmpty()) }
    var history by remember { mutableStateOf(WhiteboardLogic.History()) }
    var tool by remember { mutableStateOf(WhiteboardTool.PEN) }
    var color by remember { mutableStateOf(WhiteboardLogic.colors[0]) }
    var strokeWidth by remember { mutableStateOf(WhiteboardLogic.strokeWidths[1]) }
    var eraserSize by remember { mutableStateOf(WhiteboardLogic.eraserSizes[1]) }
    var draft by remember { mutableStateOf<WhiteboardElement?>(null) }
    var dragStart by remember { mutableStateOf<Offset?>(null) }
    var selectedIdx by remember { mutableStateOf<Int?>(null) }
    var selectStart by remember { mutableStateOf<Offset?>(null) }
    var selectOrig by remember { mutableStateOf<WhiteboardElement?>(null) }
    var saveState by remember { mutableStateOf(SaveState.Idle) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var autosaveJob by remember { mutableStateOf<Job?>(null) }
    var scale by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }

    fun scheduleAutosave() {
        autosaveJob?.cancel()
        autosaveJob = scope.launch {
            delay(WhiteboardLogic.autosaveDelayMs)
            val token = accessToken ?: return@launch
            saveState = SaveState.Saving
            runCatching {
                LmsApi.updateCourseWhiteboard(course.courseCode, board.id, title, elements, token)
            }.onSuccess {
                elements = it.canvasData ?: elements
                title = it.title
                saveState = SaveState.Saved
            }.onFailure {
                saveState = SaveState.Failed
                errorMessage = L.text(R.string.mobile_whiteboard_error_save)
            }
        }
    }

    LaunchedEffect(accessToken, board.id) {
        val token = accessToken ?: return@LaunchedEffect
        runCatching { LmsApi.fetchCourseWhiteboard(course.courseCode, board.id, token) }
            .onSuccess {
                title = it.title
                elements = it.canvasData.orEmpty()
            }
    }

    Dialog(onDismissRequest = onDismiss, properties = DialogProperties(usePlatformDefaultWidth = false)) {
        Column(modifier = Modifier.fillMaxSize().background(sceneBackground())) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp),
            ) {
                TextButton(onClick = onDismiss) { Text(liveClose(), color = textPrimary()) }
                Spacer(Modifier.weight(1f))
                Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                Spacer(Modifier.weight(1f))
                IconButton(
                    onClick = {
                        val token = accessToken ?: return@IconButton
                        scope.launch {
                            runCatching {
                                LmsApi.deleteCourseWhiteboard(course.courseCode, board.id, token)
                            }.onSuccess {
                                onDeleted?.invoke()
                                onDismiss()
                            }.onFailure {
                                errorMessage = L.text(R.string.mobile_whiteboard_error_delete)
                            }
                        }
                    },
                ) {
                    Icon(Icons.Default.Delete, contentDescription = L.text(R.string.mobile_whiteboard_delete))
                }
            }

            Text(
                text = when (saveState) {
                    SaveState.Idle -> L.text(R.string.mobile_whiteboard_status_idle)
                    SaveState.Saving -> L.text(R.string.mobile_whiteboard_status_saving)
                    SaveState.Saved -> L.text(R.string.mobile_whiteboard_status_saved)
                    SaveState.Failed -> L.text(R.string.mobile_whiteboard_status_failed)
                },
                fontSize = 12.sp,
                color = textSecondary(),
                modifier = Modifier.padding(horizontal = 16.dp),
            )
            errorMessage?.let {
                Text(it, color = Color.Red, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 16.dp))
            }

            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .padding(12.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(if (isDark) Color(0xFF171717) else Color.White)
                    .semantics { contentDescription = L.text(R.string.mobile_whiteboard_canvas) }
                    .pointerInput(tool, color, strokeWidth, eraserSize, elements) {
                        var anchor: Offset? = null
                        detectDragGestures(
                            onDragStart = { position ->
                                val point = Offset(
                                    (position.x - offset.x) / scale,
                                    (position.y - offset.y) / scale,
                                )
                                when (tool) {
                                    WhiteboardTool.SELECT -> {
                                        selectedIdx = WhiteboardLogic.pickElement(elements, point)
                                        selectStart = point
                                        selectOrig = selectedIdx?.let { elements[it] }
                                        if (selectedIdx != null) history = history.push(elements)
                                    }
                                    WhiteboardTool.ERASER -> {
                                        val next = WhiteboardLogic.erase(elements, point, eraserSize)
                                        if (next.size != elements.size) {
                                            history = history.push(elements)
                                            elements = next
                                            WhiteboardObservability.record("whiteboard_edited", mapOf("tool" to "eraser"))
                                        }
                                    }
                                    WhiteboardTool.PEN -> {
                                        draft = WhiteboardLogic.stroke(color, strokeWidth, listOf(point))
                                        anchor = point
                                    }
                                    WhiteboardTool.LINE -> {
                                        draft = WhiteboardLogic.line(color, strokeWidth, point, point)
                                        anchor = point
                                    }
                                    WhiteboardTool.RECT -> {
                                        draft = WhiteboardLogic.rect(color, strokeWidth, point, point)
                                        anchor = point
                                    }
                                    WhiteboardTool.CIRCLE -> {
                                        draft = WhiteboardLogic.circle(color, strokeWidth, point, point)
                                        anchor = point
                                    }
                                    WhiteboardTool.TRIANGLE -> {
                                        draft = WhiteboardLogic.triangle(color, strokeWidth, point, point)
                                        anchor = point
                                    }
                                }
                                dragStart = point
                            },
                            onDrag = { change, _ ->
                                val point = Offset(
                                    (change.position.x - offset.x) / scale,
                                    (change.position.y - offset.y) / scale,
                                )
                                when (tool) {
                                    WhiteboardTool.SELECT -> {
                                        val idx = selectedIdx
                                        val start = selectStart
                                        val orig = selectOrig
                                        if (idx != null && start != null && orig != null) {
                                            val dx = (point.x - start.x).toDouble()
                                            val dy = (point.y - start.y).toDouble()
                                            elements = elements.toMutableList().also {
                                                it[idx] = WhiteboardLogic.translate(orig, dx, dy)
                                            }
                                        }
                                    }
                                    WhiteboardTool.ERASER -> {
                                        val next = WhiteboardLogic.erase(elements, point, eraserSize)
                                        if (next.size != elements.size) {
                                            history = history.push(elements)
                                            elements = next
                                        }
                                    }
                                    WhiteboardTool.PEN -> {
                                        val stroke = draft ?: return@detectDragGestures
                                        val pts = stroke.pts.orEmpty().toMutableList()
                                        pts.add(listOf(point.x.toDouble(), point.y.toDouble()))
                                        draft = stroke.copy(pts = pts)
                                    }
                                    WhiteboardTool.LINE -> {
                                        val start = anchor ?: point
                                        draft = WhiteboardLogic.line(color, strokeWidth, start, point)
                                    }
                                    WhiteboardTool.RECT -> {
                                        val start = anchor ?: point
                                        draft = WhiteboardLogic.rect(color, strokeWidth, start, point)
                                    }
                                    WhiteboardTool.CIRCLE -> {
                                        val start = anchor ?: point
                                        draft = WhiteboardLogic.circle(color, strokeWidth, start, point)
                                    }
                                    WhiteboardTool.TRIANGLE -> {
                                        val start = anchor ?: point
                                        draft = WhiteboardLogic.triangle(color, strokeWidth, start, point)
                                    }
                                }
                            },
                            onDragEnd = {
                                if (tool != WhiteboardTool.SELECT && tool != WhiteboardTool.ERASER) {
                                    draft?.let {
                                        history = history.push(elements)
                                        elements = elements + it
                                        WhiteboardObservability.record("whiteboard_edited", mapOf("tool" to tool.wireName))
                                    }
                                }
                                draft = null
                                dragStart = null
                                selectStart = null
                                selectOrig = null
                                scheduleAutosave()
                            },
                        )
                    }
                    .pointerInput(Unit) {
                        detectTransformGestures { _, pan, zoom, _ ->
                            if (tool == WhiteboardTool.SELECT) {
                                scale = (scale * zoom).coerceIn(0.5f, 4f)
                                offset += pan
                            }
                        }
                    },
            ) {
                Canvas(
                    modifier = Modifier
                        .fillMaxSize()
                        .graphicsLayer {
                            scaleX = scale
                            scaleY = scale
                            translationX = offset.x
                            translationY = offset.y
                        },
                ) {
                    WhiteboardRenderer.drawGrid(this, isDark)
                    elements.forEach { WhiteboardRenderer.drawElement(this, it) }
                    draft?.let { WhiteboardRenderer.drawElement(this, it) }
                }
            }

            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(cardBackground())
                    .padding(vertical = 10.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                        .padding(horizontal = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    WhiteboardTool.entries.forEach { item ->
                        val selected = tool == item
                        Box(
                            modifier = Modifier
                                .size(44.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(if (selected) accentColor() else Color.Transparent)
                                .clickable { tool = item }
                                .semantics {
                                    contentDescription = L.text(
                                        when (item) {
                                            WhiteboardTool.SELECT -> R.string.mobile_whiteboard_tool_select
                                            WhiteboardTool.PEN -> R.string.mobile_whiteboard_tool_pen
                                            WhiteboardTool.LINE -> R.string.mobile_whiteboard_tool_line
                                            WhiteboardTool.RECT -> R.string.mobile_whiteboard_tool_rect
                                            WhiteboardTool.CIRCLE -> R.string.mobile_whiteboard_tool_circle
                                            WhiteboardTool.TRIANGLE -> R.string.mobile_whiteboard_tool_triangle
                                            WhiteboardTool.ERASER -> R.string.mobile_whiteboard_tool_eraser
                                        },
                                    )
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                toolIcon(item),
                                contentDescription = null,
                                tint = if (selected) Color.White else textPrimary(),
                            )
                        }
                    }
                    IconButton(
                        onClick = {
                            val (nextHistory, previous) = history.undo(elements)
                            if (previous != null) {
                                history = nextHistory
                                elements = previous
                                WhiteboardObservability.record("whiteboard_undo")
                                scheduleAutosave()
                            }
                        },
                        enabled = history.canUndo,
                    ) {
                        Icon(Icons.AutoMirrored.Filled.Undo, contentDescription = L.text(R.string.mobile_whiteboard_undo))
                    }
                    IconButton(
                        onClick = {
                            val (nextHistory, next) = history.redo(elements)
                            if (next != null) {
                                history = nextHistory
                                elements = next
                                scheduleAutosave()
                            }
                        },
                        enabled = history.canRedo,
                    ) {
                        Icon(Icons.AutoMirrored.Filled.Redo, contentDescription = L.text(R.string.mobile_whiteboard_redo))
                    }
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                        .padding(horizontal = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    WhiteboardLogic.colors.forEach { hex ->
                        Box(
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .background(parseHexColor(hex))
                                .border(
                                    width = if (color == hex) 2.5.dp else 1.dp,
                                    color = if (color == hex) accentColor() else fieldBorder(),
                                    shape = CircleShape,
                                )
                                .clickable { color = hex },
                        )
                    }
                    Spacer(Modifier.weight(1f))
                    val widths = if (tool == WhiteboardTool.ERASER) WhiteboardLogic.eraserSizes else WhiteboardLogic.strokeWidths
                    widths.forEach { width ->
                        val selected = if (tool == WhiteboardTool.ERASER) eraserSize == width else strokeWidth == width
                        Box(
                            modifier = Modifier
                                .size(44.dp)
                                .clip(CircleShape)
                                .background(if (selected) accentColor().copy(alpha = 0.18f) else Color.Transparent)
                                .clickable {
                                    if (tool == WhiteboardTool.ERASER) eraserSize = width else strokeWidth = width
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Box(
                                modifier = Modifier
                                    .size((6 + width * 1.5).dp)
                                    .clip(CircleShape)
                                    .background(textPrimary()),
                            )
                        }
                    }
                }
            }
        }
    }
}
