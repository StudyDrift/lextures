package com.lextures.android.core.notebook

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import kotlin.math.abs
import kotlin.math.hypot
import kotlin.math.max
import kotlin.math.min

/** One whiteboard element from a ```drawing fenced block (parity with web `whiteboard/types`). */
sealed interface NotebookDrawEl {
    val color: String
    val width: Double

    data class StrokeEl(override val color: String, override val width: Double, val pts: List<Offset>) : NotebookDrawEl
    data class RectEl(override val color: String, override val width: Double, val x: Double, val y: Double, val w: Double, val h: Double) : NotebookDrawEl
    data class CircleEl(override val color: String, override val width: Double, val cx: Double, val cy: Double, val rx: Double, val ry: Double) : NotebookDrawEl
    data class TriangleEl(
        override val color: String,
        override val width: Double,
        val x1: Double, val y1: Double, val x2: Double, val y2: Double, val x3: Double, val y3: Double,
    ) : NotebookDrawEl
    data class LineEl(override val color: String, override val width: Double, val x1: Double, val y1: Double, val x2: Double, val y2: Double) : NotebookDrawEl
}

object NotebookDrawing {
    val colors: List<String> = listOf(
        "#1e293b", "#ef4444", "#f97316", "#eab308", "#22c55e",
        "#3b82f6", "#a855f7", "#ec4899", "#ffffff",
    )
    val strokeWidths: List<Double> = listOf(2.0, 4.0, 8.0)

    private val json = Json { ignoreUnknownKeys = true }

    // JSON (parity with web `whiteboard/serialize`)

    fun parseElements(raw: String): List<NotebookDrawEl> {
        val array = runCatching { json.parseToJsonElement(raw.trim()) as? JsonArray }.getOrNull() ?: return emptyList()
        return array.mapNotNull { el -> (el as? JsonObject)?.let(::parseElement) }
    }

    private fun JsonObject.num(key: String): Double? = (get(key) as? kotlinx.serialization.json.JsonPrimitive)?.doubleOrNull

    private fun parseElement(obj: JsonObject): NotebookDrawEl? {
        val color = (obj["color"] as? kotlinx.serialization.json.JsonPrimitive)?.contentOrNull ?: "#1e293b"
        val width = obj.num("width") ?: 2.0
        return when ((obj["type"] as? kotlinx.serialization.json.JsonPrimitive)?.contentOrNull) {
            "stroke" -> {
                val pts = (obj["pts"] as? JsonArray ?: return null).mapNotNull { pair ->
                    val arr = runCatching { pair.jsonArray }.getOrNull() ?: return@mapNotNull null
                    if (arr.size < 2) return@mapNotNull null
                    val x = arr[0].jsonPrimitive.doubleOrNull ?: return@mapNotNull null
                    val y = arr[1].jsonPrimitive.doubleOrNull ?: return@mapNotNull null
                    Offset(x.toFloat(), y.toFloat())
                }
                NotebookDrawEl.StrokeEl(color, width, pts)
            }
            "rect" -> NotebookDrawEl.RectEl(
                color, width,
                obj.num("x") ?: return null, obj.num("y") ?: return null,
                obj.num("w") ?: return null, obj.num("h") ?: return null,
            )
            "circle" -> NotebookDrawEl.CircleEl(
                color, width,
                obj.num("cx") ?: return null, obj.num("cy") ?: return null,
                obj.num("rx") ?: return null, obj.num("ry") ?: return null,
            )
            "triangle" -> NotebookDrawEl.TriangleEl(
                color, width,
                obj.num("x1") ?: return null, obj.num("y1") ?: return null,
                obj.num("x2") ?: return null, obj.num("y2") ?: return null,
                obj.num("x3") ?: return null, obj.num("y3") ?: return null,
            )
            "line" -> NotebookDrawEl.LineEl(
                color, width,
                obj.num("x1") ?: return null, obj.num("y1") ?: return null,
                obj.num("x2") ?: return null, obj.num("y2") ?: return null,
            )
            else -> null
        }
    }

    fun serializeElements(elements: List<NotebookDrawEl>): String {
        val array = buildJsonArray {
            for (el in elements) {
                add(
                    buildJsonObject {
                        put("color", el.color)
                        put("width", el.width)
                        when (el) {
                            is NotebookDrawEl.StrokeEl -> {
                                put("type", "stroke")
                                put(
                                    "pts",
                                    buildJsonArray {
                                        for (p in el.pts) {
                                            add(
                                                buildJsonArray {
                                                    add(kotlinx.serialization.json.JsonPrimitive(p.x.toDouble()))
                                                    add(kotlinx.serialization.json.JsonPrimitive(p.y.toDouble()))
                                                },
                                            )
                                        }
                                    },
                                )
                            }
                            is NotebookDrawEl.RectEl -> {
                                put("type", "rect"); put("x", el.x); put("y", el.y); put("w", el.w); put("h", el.h)
                            }
                            is NotebookDrawEl.CircleEl -> {
                                put("type", "circle"); put("cx", el.cx); put("cy", el.cy); put("rx", el.rx); put("ry", el.ry)
                            }
                            is NotebookDrawEl.TriangleEl -> {
                                put("type", "triangle")
                                put("x1", el.x1); put("y1", el.y1); put("x2", el.x2); put("y2", el.y2); put("x3", el.x3); put("y3", el.y3)
                            }
                            is NotebookDrawEl.LineEl -> {
                                put("type", "line"); put("x1", el.x1); put("y1", el.y1); put("x2", el.x2); put("y2", el.y2)
                            }
                        }
                    },
                )
            }
        }
        return json.encodeToString(JsonArray.serializer(), array)
    }

    // Geometry

    /** Content extent of the elements (origin-anchored), with a sensible floor for empty boards. */
    fun contentSize(elements: List<NotebookDrawEl>, minWidth: Float = 320f, minHeight: Float = 220f): Size {
        var maxX = minWidth
        var maxY = minHeight
        fun grow(x: Double, y: Double) {
            maxX = max(maxX, x.toFloat())
            maxY = max(maxY, y.toFloat())
        }
        for (el in elements) {
            when (el) {
                is NotebookDrawEl.StrokeEl -> el.pts.forEach { grow(it.x.toDouble(), it.y.toDouble()) }
                is NotebookDrawEl.RectEl -> { grow(el.x + el.w, el.y + el.h); grow(el.x, el.y) }
                is NotebookDrawEl.CircleEl -> grow(el.cx + el.rx, el.cy + el.ry)
                is NotebookDrawEl.TriangleEl -> { grow(el.x1, el.y1); grow(el.x2, el.y2); grow(el.x3, el.y3) }
                is NotebookDrawEl.LineEl -> { grow(el.x1, el.y1); grow(el.x2, el.y2) }
            }
        }
        return Size(maxX, maxY)
    }

    fun color(hex: String, fallback: Color = Color.Black): Color {
        val value = hex.trim().removePrefix("#")
        if (value.length != 6) return fallback
        val rgb = value.toLongOrNull(16) ?: return fallback
        return Color(
            red = ((rgb shr 16) and 0xFF).toInt() / 255f,
            green = ((rgb shr 8) and 0xFF).toInt() / 255f,
            blue = (rgb and 0xFF).toInt() / 255f,
        )
    }

    /** Draw all elements into a Compose draw scope at the given scale. */
    fun DrawScope.drawElements(elements: List<NotebookDrawEl>, scale: Float) {
        for (el in elements) {
            val c = color(el.color)
            val strokeWidth = (el.width * scale).toFloat()
            when (el) {
                is NotebookDrawEl.StrokeEl -> {
                    if (el.pts.size < 2) {
                        el.pts.firstOrNull()?.let { p ->
                            drawCircle(c, radius = max(strokeWidth, 2f) / 2, center = Offset(p.x * scale, p.y * scale))
                        }
                        continue
                    }
                    val path = Path()
                    path.moveTo(el.pts[0].x * scale, el.pts[0].y * scale)
                    for (p in el.pts.drop(1)) path.lineTo(p.x * scale, p.y * scale)
                    drawPath(path, c, style = Stroke(width = strokeWidth, cap = StrokeCap.Round, join = StrokeJoin.Round))
                }
                is NotebookDrawEl.RectEl -> drawRect(
                    c,
                    topLeft = Offset((el.x * scale).toFloat(), (el.y * scale).toFloat()),
                    size = Size((el.w * scale).toFloat(), (el.h * scale).toFloat()),
                    style = Stroke(width = strokeWidth),
                )
                is NotebookDrawEl.CircleEl -> drawOval(
                    c,
                    topLeft = Offset(((el.cx - el.rx) * scale).toFloat(), ((el.cy - el.ry) * scale).toFloat()),
                    size = Size((el.rx * 2 * scale).toFloat(), (el.ry * 2 * scale).toFloat()),
                    style = Stroke(width = strokeWidth),
                )
                is NotebookDrawEl.TriangleEl -> {
                    val path = Path()
                    path.moveTo((el.x1 * scale).toFloat(), (el.y1 * scale).toFloat())
                    path.lineTo((el.x2 * scale).toFloat(), (el.y2 * scale).toFloat())
                    path.lineTo((el.x3 * scale).toFloat(), (el.y3 * scale).toFloat())
                    path.close()
                    drawPath(path, c, style = Stroke(width = strokeWidth, join = StrokeJoin.Round))
                }
                is NotebookDrawEl.LineEl -> drawLine(
                    c,
                    start = Offset((el.x1 * scale).toFloat(), (el.y1 * scale).toFloat()),
                    end = Offset((el.x2 * scale).toFloat(), (el.y2 * scale).toFloat()),
                    strokeWidth = strokeWidth,
                    cap = StrokeCap.Round,
                )
            }
        }
    }

    /** Whether a point (content coordinates) is within `radius` of an element — eraser hit test. */
    fun hitTest(el: NotebookDrawEl, point: Offset, radius: Double): Boolean {
        fun distToSegment(p: Offset, ax: Double, ay: Double, bx: Double, by: Double): Double {
            val dx = bx - ax
            val dy = by - ay
            val lengthSq = dx * dx + dy * dy
            if (lengthSq == 0.0) return hypot(p.x - ax, p.y - ay)
            val t = (((p.x - ax) * dx + (p.y - ay) * dy) / lengthSq).coerceIn(0.0, 1.0)
            return hypot(p.x - (ax + t * dx), p.y - (ay + t * dy))
        }
        val r = radius + el.width / 2
        return when (el) {
            is NotebookDrawEl.StrokeEl -> {
                if (el.pts.size == 1) return hypot((point.x - el.pts[0].x).toDouble(), (point.y - el.pts[0].y).toDouble()) <= r
                el.pts.zipWithNext().any { (a, b) ->
                    distToSegment(point, a.x.toDouble(), a.y.toDouble(), b.x.toDouble(), b.y.toDouble()) <= r
                }
            }
            is NotebookDrawEl.LineEl -> distToSegment(point, el.x1, el.y1, el.x2, el.y2) <= r
            is NotebookDrawEl.RectEl -> {
                val corners = listOf(
                    el.x to el.y, (el.x + el.w) to el.y,
                    (el.x + el.w) to (el.y + el.h), el.x to (el.y + el.h),
                )
                (0 until 4).any { i ->
                    val (ax, ay) = corners[i]
                    val (bx, by) = corners[(i + 1) % 4]
                    distToSegment(point, ax, ay, bx, by) <= r
                }
            }
            is NotebookDrawEl.CircleEl -> {
                if (el.rx <= 0 || el.ry <= 0) return false
                val nx = (point.x - el.cx) / el.rx
                val ny = (point.y - el.cy) / el.ry
                abs(hypot(nx, ny) - 1) * min(el.rx, el.ry) <= r
            }
            is NotebookDrawEl.TriangleEl -> {
                val pts = listOf(el.x1 to el.y1, el.x2 to el.y2, el.x3 to el.y3)
                (0 until 3).any { i ->
                    val (ax, ay) = pts[i]
                    val (bx, by) = pts[(i + 1) % 3]
                    distToSegment(point, ax, ay, bx, by) <= r
                }
            }
        }
    }
}
