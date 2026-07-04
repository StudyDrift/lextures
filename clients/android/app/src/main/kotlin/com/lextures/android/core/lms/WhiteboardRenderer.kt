package com.lextures.android.core.lms

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke

/** Read-only whiteboard canvas renderer (M7.5). */
object WhiteboardRenderer {
    fun drawGrid(scope: DrawScope, isDark: Boolean) {
        val background = if (isDark) Color(0xFF171717) else Color.White
        scope.drawRect(background)
        val dot = if (isDark) Color.White.copy(alpha = 0.18f) else Color.Black.copy(alpha = 0.18f)
        val spacing = 24f
        var x = spacing
        while (x < scope.size.width) {
            var y = spacing
            while (y < scope.size.height) {
                scope.drawCircle(dot, 1f, Offset(x, y))
                y += spacing
            }
            x += spacing
        }
    }

    fun drawElement(scope: DrawScope, element: WhiteboardElement) {
        val color = parseColor(element.color) ?: return
        val stroke = Stroke(width = element.width.toFloat(), cap = androidx.compose.ui.graphics.StrokeCap.Round)
        when (element.type) {
            "stroke" -> {
                val points = element.points()
                if (points.size < 2) return
                val path = Path().apply {
                    moveTo(points[0].x, points[0].y)
                    points.drop(1).forEach { lineTo(it.x, it.y) }
                }
                scope.drawPath(path, color, style = stroke)
            }
            "rect" -> {
                val rect = element.rect() ?: return
                scope.drawRect(color, topLeft = Offset(rect.left, rect.top), size = rect.size, style = stroke)
            }
            "circle" -> {
                val rect = element.ellipse() ?: return
                scope.drawOval(color, topLeft = Offset(rect.left, rect.top), size = rect.size, style = stroke)
            }
            "triangle" -> {
                val points = element.points()
                if (points.size != 3) return
                val path = Path().apply {
                    moveTo(points[0].x, points[0].y)
                    lineTo(points[1].x, points[1].y)
                    lineTo(points[2].x, points[2].y)
                    close()
                }
                scope.drawPath(path, color, style = stroke)
            }
            "line" -> {
                val points = element.points()
                if (points.size != 2) return
                scope.drawLine(color, points[0], points[1], strokeWidth = element.width.toFloat())
            }
        }
    }

    private fun parseColor(raw: String): Color? {
        var hex = raw.trim()
        if (hex.startsWith("#")) hex = hex.drop(1)
        if (hex.length != 6) return null
        val value = hex.toLongOrNull(16) ?: return null
        val red = ((value shr 16) and 0xFF) / 255f
        val green = ((value shr 8) and 0xFF) / 255f
        val blue = (value and 0xFF) / 255f
        return Color(red, green, blue)
    }
}

private fun WhiteboardElement.points(): List<Offset> {
    pts?.let { raw ->
        return raw.mapNotNull { pair ->
            if (pair.size < 2) return@mapNotNull null
            Offset(pair[0].toFloat(), pair[1].toFloat())
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

private fun WhiteboardElement.rect(): Rect? {
    if (x == null || y == null || w == null || h == null) return null
    return Rect(x.toFloat(), y.toFloat(), (x + w).toFloat(), (y + h).toFloat())
}

private fun WhiteboardElement.ellipse(): Rect? {
    if (cx == null || cy == null || rx == null || ry == null) return null
    return Rect(
        (cx - kotlin.math.abs(rx)).toFloat(),
        (cy - kotlin.math.abs(ry)).toFloat(),
        (cx + kotlin.math.abs(rx)).toFloat(),
        (cy + kotlin.math.abs(ry)).toFloat(),
    )
}