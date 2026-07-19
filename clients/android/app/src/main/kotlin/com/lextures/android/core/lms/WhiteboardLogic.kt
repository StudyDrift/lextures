package com.lextures.android.core.lms

import androidx.compose.ui.geometry.Offset
import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.util.concurrent.ConcurrentHashMap
import kotlin.math.abs
import kotlin.math.hypot
import kotlin.math.max
import kotlin.math.min

enum class WhiteboardTool {
    SELECT,
    PEN,
    LINE,
    RECT,
    CIRCLE,
    TRIANGLE,
    ERASER,
    ;

    val wireName: String
        get() = name.lowercase()

    val accessibilityKey: String
        get() = "mobile.whiteboard.tool.$wireName"
}

/** Pure helpers for course whiteboard authoring (MOB.6). */
object WhiteboardLogic {
    val colors: List<String> = listOf(
        "#1e293b", "#ef4444", "#f97316", "#eab308", "#22c55e",
        "#3b82f6", "#a855f7", "#ec4899", "#ffffff",
    )
    val strokeWidths: List<Double> = listOf(2.0, 4.0, 8.0)
    val eraserSizes: List<Double> = listOf(8.0, 16.0, 32.0)
    const val undoLimit = 50
    const val autosaveDelayMs = 800L
    const val pickHitRadius = 8.0

    fun canEdit(viewerIsStaff: Boolean, features: MobilePlatformFeatures): Boolean =
        viewerIsStaff && features.ffMobileWhiteboardEdit

    fun normalizeTitle(raw: String): String = raw.trim()

    fun isValidTitle(raw: String): Boolean = normalizeTitle(raw).isNotEmpty()

    fun defaultTitle(existingCount: Int): String = "Whiteboard ${max(1, existingCount + 1)}"

    data class History(
        val undoStack: List<List<WhiteboardElement>> = emptyList(),
        val redoStack: List<List<WhiteboardElement>> = emptyList(),
    ) {
        fun push(elements: List<WhiteboardElement>): History {
            val nextUndo = (undoStack + listOf(elements)).takeLast(undoLimit)
            return copy(undoStack = nextUndo, redoStack = emptyList())
        }

        fun undo(current: List<WhiteboardElement>): Pair<History, List<WhiteboardElement>?> {
            if (undoStack.isEmpty()) return this to null
            val previous = undoStack.last()
            return copy(
                undoStack = undoStack.dropLast(1),
                redoStack = redoStack + listOf(current),
            ) to previous
        }

        fun redo(current: List<WhiteboardElement>): Pair<History, List<WhiteboardElement>?> {
            if (redoStack.isEmpty()) return this to null
            val next = redoStack.last()
            return copy(
                undoStack = undoStack + listOf(current),
                redoStack = redoStack.dropLast(1),
            ) to next
        }

        val canUndo: Boolean get() = undoStack.isNotEmpty()
        val canRedo: Boolean get() = redoStack.isNotEmpty()
    }

    fun stroke(color: String, width: Double, pts: List<Offset>): WhiteboardElement =
        WhiteboardElement(
            type = "stroke",
            color = color,
            width = width,
            pts = pts.map { listOf(it.x.toDouble(), it.y.toDouble()) },
        )

    fun line(color: String, width: Double, from: Offset, to: Offset): WhiteboardElement =
        WhiteboardElement(
            type = "line",
            color = color,
            width = width,
            x1 = from.x.toDouble(),
            y1 = from.y.toDouble(),
            x2 = to.x.toDouble(),
            y2 = to.y.toDouble(),
        )

    fun rect(color: String, width: Double, from: Offset, to: Offset): WhiteboardElement =
        WhiteboardElement(
            type = "rect",
            color = color,
            width = width,
            x = min(from.x, to.x).toDouble(),
            y = min(from.y, to.y).toDouble(),
            w = abs(to.x - from.x).toDouble(),
            h = abs(to.y - from.y).toDouble(),
        )

    fun circle(color: String, width: Double, from: Offset, to: Offset): WhiteboardElement =
        WhiteboardElement(
            type = "circle",
            color = color,
            width = width,
            cx = ((from.x + to.x) / 2).toDouble(),
            cy = ((from.y + to.y) / 2).toDouble(),
            rx = (abs(to.x - from.x) / 2).toDouble(),
            ry = (abs(to.y - from.y) / 2).toDouble(),
        )

    fun triangle(color: String, width: Double, from: Offset, to: Offset): WhiteboardElement =
        WhiteboardElement(
            type = "triangle",
            color = color,
            width = width,
            x1 = ((from.x + to.x) / 2).toDouble(),
            y1 = min(from.y, to.y).toDouble(),
            x2 = from.x.toDouble(),
            y2 = max(from.y, to.y).toDouble(),
            x3 = to.x.toDouble(),
            y3 = max(from.y, to.y).toDouble(),
        )

    fun hitTest(element: WhiteboardElement, point: Offset, radius: Double): Boolean {
        fun distToSegment(probe: Offset, start: Offset, end: Offset): Double {
            val dx = end.x - start.x
            val dy = end.y - start.y
            val lengthSq = dx * dx + dy * dy
            if (lengthSq == 0f) return hypot((probe.x - start.x).toDouble(), (probe.y - start.y).toDouble())
            val projection = (((probe.x - start.x) * dx + (probe.y - start.y) * dy) / lengthSq)
                .coerceIn(0f, 1f)
            return hypot(
                (probe.x - (start.x + projection * dx)).toDouble(),
                (probe.y - (start.y + projection * dy)).toDouble(),
            )
        }

        return when (element.type) {
            "stroke" -> {
                val pts = element.pointOffsets()
                val hitRadius = radius + element.width / 2
                when {
                    pts.size == 1 -> hypot((point.x - pts[0].x).toDouble(), (point.y - pts[0].y).toDouble()) <= hitRadius
                    else -> (0 until max(0, pts.size - 1)).any {
                        distToSegment(point, pts[it], pts[it + 1]) <= hitRadius
                    }
                }
            }
            "line" -> {
                val pts = element.pointOffsets()
                if (pts.size != 2) return false
                distToSegment(point, pts[0], pts[1]) <= radius + element.width / 2
            }
            "rect" -> {
                val x = element.x ?: return false
                val y = element.y ?: return false
                val w = element.w ?: return false
                val h = element.h ?: return false
                val corners = listOf(
                    Offset(x.toFloat(), y.toFloat()),
                    Offset((x + w).toFloat(), y.toFloat()),
                    Offset((x + w).toFloat(), (y + h).toFloat()),
                    Offset(x.toFloat(), (y + h).toFloat()),
                )
                val hitRadius = radius + element.width / 2
                (0 until 4).any { distToSegment(point, corners[it], corners[(it + 1) % 4]) <= hitRadius }
            }
            "circle" -> {
                val cx = element.cx ?: return false
                val cy = element.cy ?: return false
                val rx = abs(element.rx ?: return false)
                val ry = abs(element.ry ?: return false)
                if (rx <= 0 || ry <= 0) return false
                val norm = hypot((point.x - cx) / rx, (point.y - cy) / ry)
                abs(norm - 1) * min(rx, ry) <= radius + element.width / 2
            }
            "triangle" -> {
                val pts = element.pointOffsets()
                if (pts.size != 3) return false
                val hitRadius = radius + element.width / 2
                (0 until 3).any { distToSegment(point, pts[it], pts[(it + 1) % 3]) <= hitRadius }
            }
            else -> false
        }
    }

    fun erase(elements: List<WhiteboardElement>, point: Offset, radius: Double): List<WhiteboardElement> =
        elements.filterNot { hitTest(it, point, radius) }

    fun pickElement(elements: List<WhiteboardElement>, point: Offset): Int? {
        for (index in elements.indices.reversed()) {
            if (hitTest(elements[index], point, pickHitRadius)) return index
        }
        return null
    }

    fun translate(element: WhiteboardElement, dx: Double, dy: Double): WhiteboardElement =
        when (element.type) {
            "stroke" -> element.copy(
                pts = element.pts?.map { pair ->
                    if (pair.size < 2) pair else listOf(pair[0] + dx, pair[1] + dy)
                },
            )
            "rect" -> element.copy(
                x = element.x?.plus(dx),
                y = element.y?.plus(dy),
            )
            "circle" -> element.copy(
                cx = element.cx?.plus(dx),
                cy = element.cy?.plus(dy),
            )
            "line", "triangle" -> element.copy(
                x1 = element.x1?.plus(dx),
                y1 = element.y1?.plus(dy),
                x2 = element.x2?.plus(dx),
                y2 = element.y2?.plus(dy),
                x3 = element.x3?.plus(dx),
                y3 = element.y3?.plus(dy),
            )
            else -> element
        }

    fun serializeElements(elements: List<WhiteboardElement>): List<Map<String, Any>> =
        elements.mapNotNull { serializeElement(it) }

    fun serializeElement(element: WhiteboardElement): Map<String, Any>? {
        val base = mutableMapOf<String, Any>(
            "type" to element.type,
            "color" to element.color,
            "width" to element.width,
        )
        when (element.type) {
            "stroke" -> {
                val pts = element.pts ?: return null
                base["pts"] = pts
            }
            "rect" -> {
                base["x"] = element.x ?: return null
                base["y"] = element.y ?: return null
                base["w"] = element.w ?: return null
                base["h"] = element.h ?: return null
            }
            "circle" -> {
                base["cx"] = element.cx ?: return null
                base["cy"] = element.cy ?: return null
                base["rx"] = element.rx ?: return null
                base["ry"] = element.ry ?: return null
            }
            "line" -> {
                base["x1"] = element.x1 ?: return null
                base["y1"] = element.y1 ?: return null
                base["x2"] = element.x2 ?: return null
                base["y2"] = element.y2 ?: return null
            }
            "triangle" -> {
                base["x1"] = element.x1 ?: return null
                base["y1"] = element.y1 ?: return null
                base["x2"] = element.x2 ?: return null
                base["y2"] = element.y2 ?: return null
                base["x3"] = element.x3 ?: return null
                base["y3"] = element.y3 ?: return null
            }
            else -> return null
        }
        return base
    }

    fun shouldAcceptTouch(isStylus: Boolean, stylusExclusiveDrawing: Boolean): Boolean {
        if (!stylusExclusiveDrawing) return true
        return isStylus
    }
}

object WhiteboardObservability {
    private val counters = ConcurrentHashMap<String, Int>()

    fun record(event: String, attributes: Map<String, String> = emptyMap()) {
        val key = if (attributes.isEmpty()) {
            event
        } else {
            event + "|" + attributes.toSortedMap().entries.joinToString(",") { "${it.key}=${it.value}" }
        }
        counters.merge(key, 1, Int::plus)
    }

    fun count(event: String): Int =
        counters.entries
            .filter { it.key == event || it.key.startsWith("$event|") }
            .sumOf { it.value }

    fun resetForTests() {
        counters.clear()
    }
}

private fun WhiteboardElement.pointOffsets(): List<Offset> {
    pts?.let { raw ->
        return raw.mapNotNull { pair ->
            if (pair.size < 2) null else Offset(pair[0].toFloat(), pair[1].toFloat())
        }
    }
    if (x1 != null && y1 != null && x2 != null && y2 != null && x3 != null && y3 != null) {
        return listOf(
            Offset(x1.toFloat(), y1.toFloat()),
            Offset(x2.toFloat(), y2.toFloat()),
            Offset(x3.toFloat(), y3.toFloat()),
        )
    }
    if (x1 != null && y1 != null && x2 != null && y2 != null) {
        return listOf(Offset(x1.toFloat(), y1.toFloat()), Offset(x2.toFloat(), y2.toFloat()))
    }
    return emptyList()
}
