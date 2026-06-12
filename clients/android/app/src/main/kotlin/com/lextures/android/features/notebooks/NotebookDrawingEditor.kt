package com.lextures.android.features.notebooks

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.ChangeHistory
import androidx.compose.material.icons.outlined.Circle
import androidx.compose.material.icons.filled.CleaningServices
import androidx.compose.material.icons.filled.CropSquare
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Draw
import androidx.compose.material.icons.filled.HorizontalRule
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectDragGestures
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.notebook.NotebookDrawEl
import com.lextures.android.core.notebook.NotebookDrawing
import com.lextures.android.core.notebook.NotebookDrawing.drawElements
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

private enum class DrawTool(val icon: ImageVector) {
    PEN(Icons.Default.Draw),
    LINE(Icons.Default.HorizontalRule),
    RECT(Icons.Default.CropSquare),
    CIRCLE(Icons.Outlined.Circle),
    TRIANGLE(Icons.Default.ChangeHistory),
    ERASER(Icons.Default.CleaningServices),
}

/**
 * Full-screen whiteboard editor: pen / line / rect / circle / triangle / eraser,
 * the web palette and stroke widths, undo and clear (parity with web `/drawing`).
 */
@Composable
fun NotebookDrawingEditor(
    initialElementsJson: String,
    onDismiss: () -> Unit,
    onSave: (String) -> Unit,
) {
    var elements by remember { mutableStateOf(NotebookDrawing.parseElements(initialElementsJson)) }
    var undoStack by remember { mutableStateOf(listOf<List<NotebookDrawEl>>()) }
    var tool by remember { mutableStateOf(DrawTool.PEN) }
    var color by remember { mutableStateOf(NotebookDrawing.colors[0]) }
    var lineWidth by remember { mutableStateOf(NotebookDrawing.strokeWidths[1]) }
    var draftElement by remember { mutableStateOf<NotebookDrawEl?>(null) }

    fun pushUndo() {
        undoStack = (undoStack + listOf(elements)).takeLast(50)
    }

    Dialog(onDismissRequest = onDismiss, properties = DialogProperties(usePlatformDefaultWidth = false)) {
        Column(modifier = Modifier.fillMaxSize().background(sceneBackground())) {
            // Header
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp),
            ) {
                TextButton(onClick = onDismiss) { Text("Cancel", color = textPrimary()) }
                Spacer(Modifier.weight(1f))
                Text("Drawing", style = LexturesType.display(17), color = textPrimary())
                Spacer(Modifier.weight(1f))
                TextButton(onClick = { onSave(NotebookDrawing.serializeElements(elements)) }) {
                    Text("Save", color = accentColor(), fontWeight = FontWeight.SemiBold)
                }
            }

            // Canvas
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .padding(12.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(if (isDarkTheme()) Color(0xFF1F1F1F) else Color.White),
            ) {
                Canvas(
                    modifier = Modifier
                        .fillMaxSize()
                        .pointerInput(tool, color, lineWidth) {
                            val content = NotebookDrawing.contentSize(
                                elements,
                                minWidth = size.width.toFloat(),
                                minHeight = size.height.toFloat(),
                            )
                            val scale = min(min(size.width / content.width, size.height / content.height), 1f)
                            var dragAnchor: Offset? = null

                            fun contentPoint(position: Offset) = Offset(position.x / scale, position.y / scale)

                            detectDragGestures(
                                onDragStart = { position ->
                                    val point = contentPoint(position)
                                    when (tool) {
                                        DrawTool.ERASER -> {
                                            val radius = max(lineWidth * 2, 12.0)
                                            val next = elements.filterNot { NotebookDrawing.hitTest(it, point, radius) }
                                            if (next.size != elements.size) {
                                                pushUndo()
                                                elements = next
                                            }
                                        }
                                        DrawTool.PEN ->
                                            draftElement = NotebookDrawEl.StrokeEl(color, lineWidth, listOf(point))
                                        DrawTool.LINE ->
                                            draftElement = NotebookDrawEl.LineEl(
                                                color, lineWidth,
                                                point.x.toDouble(), point.y.toDouble(), point.x.toDouble(), point.y.toDouble(),
                                            )
                                        DrawTool.RECT ->
                                            draftElement = NotebookDrawEl.RectEl(
                                                color, lineWidth, point.x.toDouble(), point.y.toDouble(), 0.0, 0.0,
                                            )
                                        DrawTool.CIRCLE ->
                                            draftElement = NotebookDrawEl.CircleEl(
                                                color, lineWidth, point.x.toDouble(), point.y.toDouble(), 0.0, 0.0,
                                            )
                                        DrawTool.TRIANGLE ->
                                            draftElement = NotebookDrawEl.TriangleEl(
                                                color, lineWidth,
                                                point.x.toDouble(), point.y.toDouble(),
                                                point.x.toDouble(), point.y.toDouble(),
                                                point.x.toDouble(), point.y.toDouble(),
                                            )
                                    }
                                    // Anchor for shape tools.
                                    dragAnchor = point
                                },
                                onDrag = { change, _ ->
                                    val point = contentPoint(change.position)
                                    val start = dragAnchor ?: point
                                    when (tool) {
                                        DrawTool.ERASER -> {
                                            val radius = max(lineWidth * 2, 12.0)
                                            val next = elements.filterNot { NotebookDrawing.hitTest(it, point, radius) }
                                            if (next.size != elements.size) {
                                                pushUndo()
                                                elements = next
                                            }
                                        }
                                        DrawTool.PEN -> {
                                            val stroke = draftElement as? NotebookDrawEl.StrokeEl ?: return@detectDragGestures
                                            draftElement = stroke.copy(pts = stroke.pts + point)
                                        }
                                        DrawTool.LINE -> draftElement = NotebookDrawEl.LineEl(
                                            color, lineWidth,
                                            start.x.toDouble(), start.y.toDouble(), point.x.toDouble(), point.y.toDouble(),
                                        )
                                        DrawTool.RECT -> draftElement = NotebookDrawEl.RectEl(
                                            color, lineWidth,
                                            min(start.x, point.x).toDouble(), min(start.y, point.y).toDouble(),
                                            abs(point.x - start.x).toDouble(), abs(point.y - start.y).toDouble(),
                                        )
                                        DrawTool.CIRCLE -> draftElement = NotebookDrawEl.CircleEl(
                                            color, lineWidth,
                                            ((start.x + point.x) / 2).toDouble(), ((start.y + point.y) / 2).toDouble(),
                                            (abs(point.x - start.x) / 2).toDouble(), (abs(point.y - start.y) / 2).toDouble(),
                                        )
                                        DrawTool.TRIANGLE -> draftElement = NotebookDrawEl.TriangleEl(
                                            color, lineWidth,
                                            ((start.x + point.x) / 2).toDouble(), min(start.y, point.y).toDouble(),
                                            start.x.toDouble(), max(start.y, point.y).toDouble(),
                                            point.x.toDouble(), max(start.y, point.y).toDouble(),
                                        )
                                    }
                                },
                                onDragEnd = {
                                    draftElement?.let {
                                        pushUndo()
                                        elements = elements + it
                                    }
                                    draftElement = null
                                    dragAnchor = null
                                },
                                onDragCancel = {
                                    draftElement = null
                                    dragAnchor = null
                                },
                            )
                        },
                ) {
                    val content = NotebookDrawing.contentSize(elements, minWidth = size.width, minHeight = size.height)
                    val scale = min(min(size.width / content.width, size.height / content.height), 1f)
                    drawElements(elements, scale)
                    draftElement?.let { drawElements(listOf(it), scale) }
                }
            }

            // Tools
            Column(
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth().background(cardBackground()).padding(horizontal = 14.dp, vertical = 10.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                    DrawTool.entries.forEach { item ->
                        Box(
                            contentAlignment = Alignment.Center,
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(if (tool == item) accentColor() else Color.Transparent)
                                .clickable { tool = item },
                        ) {
                            Icon(
                                item.icon,
                                contentDescription = item.name.lowercase(),
                                tint = if (tool == item) Color.White else textPrimary(),
                                modifier = Modifier.size(20.dp),
                            )
                        }
                    }
                    Spacer(Modifier.weight(1f))
                    IconButton(
                        onClick = {
                            undoStack.lastOrNull()?.let {
                                elements = it
                                undoStack = undoStack.dropLast(1)
                            }
                        },
                        enabled = undoStack.isNotEmpty(),
                    ) {
                        Icon(Icons.AutoMirrored.Filled.Undo, contentDescription = "Undo", tint = textPrimary())
                    }
                    IconButton(
                        onClick = {
                            if (elements.isNotEmpty()) {
                                pushUndo()
                                elements = emptyList()
                            }
                        },
                        enabled = elements.isNotEmpty(),
                    ) {
                        Icon(Icons.Default.Delete, contentDescription = "Clear", tint = textPrimary())
                    }
                }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    NotebookDrawing.colors.forEach { hex ->
                        Box(
                            modifier = Modifier
                                .size(24.dp)
                                .clip(CircleShape)
                                .background(NotebookDrawing.color(hex))
                                .border(
                                    width = if (color == hex) 2.5.dp else 1.dp,
                                    color = if (color == hex) accentColor() else fieldBorder(),
                                    shape = CircleShape,
                                )
                                .clickable { color = hex },
                        )
                    }
                    Spacer(Modifier.weight(1f))
                    NotebookDrawing.strokeWidths.forEach { w ->
                        Box(
                            contentAlignment = Alignment.Center,
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .background(if (lineWidth == w) accentColor().copy(alpha = 0.18f) else Color.Transparent)
                                .clickable { lineWidth = w },
                        ) {
                            Box(
                                Modifier
                                    .size((6 + w * 1.5).dp)
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
