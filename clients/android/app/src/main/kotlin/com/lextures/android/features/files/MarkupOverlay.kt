package com.lextures.android.features.files

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import com.lextures.android.core.lms.AnnotationRect
import com.lextures.android.core.lms.SubmissionAnnotation

/** Read-only overlay for instructor markups on PDF/image previews (M6.1). */
@Composable
fun MarkupOverlay(
    annotations: List<SubmissionAnnotation>,
    page: Int = 1,
    modifier: Modifier = Modifier,
) {
    val pageAnnotations = annotations.filter { it.page == page }
    Canvas(modifier = modifier.fillMaxSize()) {
        for (annotation in pageAnnotations) {
            val color = parseHexColor(annotation.colour)
            when (annotation.toolType) {
                "highlight" -> {
                    for (rect in highlightRects(annotation)) {
                        drawRect(
                            color = color.copy(alpha = 0.35f),
                            topLeft = Offset(
                                (rect.x1 * size.width).toFloat(),
                                (rect.y1 * size.height).toFloat(),
                            ),
                            size = Size(
                                ((rect.x2 - rect.x1) * size.width).toFloat(),
                                ((rect.y2 - rect.y1) * size.height).toFloat(),
                            ),
                        )
                    }
                }
                "draw" -> {
                    val points = annotation.coordsJson?.points.orEmpty()
                    if (points.size >= 2) {
                        val path = Path().apply {
                            moveTo(
                                (points[0].x * size.width).toFloat(),
                                (points[0].y * size.height).toFloat(),
                            )
                            for (point in points.drop(1)) {
                                lineTo(
                                    (point.x * size.width).toFloat(),
                                    (point.y * size.height).toFloat(),
                                )
                            }
                        }
                        drawPath(path, color = color, style = Stroke(width = 2f))
                    }
                }
                "pin", "text" -> {
                    val coords = annotation.coordsJson
                    val x = coords?.x ?: coords?.x1
                    val y = coords?.y ?: coords?.y1
                    if (x != null && y != null) {
                        drawCircle(
                            color = color,
                            radius = 5f,
                            center = Offset(
                                (x * size.width).toFloat(),
                                (y * size.height).toFloat(),
                            ),
                        )
                    }
                }
            }
        }
    }
}

private fun highlightRects(annotation: SubmissionAnnotation): List<AnnotationRect> {
    val coords = annotation.coordsJson ?: return emptyList()
    if (!coords.rects.isNullOrEmpty()) return coords.rects
    val x1 = coords.x1
    val y1 = coords.y1
    val x2 = coords.x2
    val y2 = coords.y2
    return if (x1 != null && y1 != null && x2 != null && y2 != null) {
        listOf(AnnotationRect(x1, y1, x2, y2))
    } else {
        emptyList()
    }
}

private fun parseHexColor(hex: String): Color {
    val cleaned = hex.trim().removePrefix("#")
    if (cleaned.length != 6) return Color.Yellow
    val value = cleaned.toLongOrNull(16) ?: return Color.Yellow
    val r = ((value shr 16) and 0xFF) / 255f
    val g = ((value shr 8) and 0xFF) / 255f
    val b = (value and 0xFF) / 255f
    return Color(r, g, b)
}
