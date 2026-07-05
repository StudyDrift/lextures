package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test

class WhiteboardElementTest {
    @Test
    fun decodesStrokePointsFromPtsArray() {
        val element = WhiteboardElement(
            type = "stroke",
            color = "#ff0000",
            width = 2.0,
            pts = listOf(listOf(1.0, 2.0), listOf(3.0, 4.0)),
        )
        val points = element.pts.orEmpty()
        assertEquals(2, points.size)
        assertEquals(1.0, points[0][0], 0.001)
        assertEquals(4.0, points[1][1], 0.001)
    }

    @Test
    fun decodesRectCoordinates() {
        val element = WhiteboardElement(
            type = "rect",
            color = "#00ff00",
            width = 1.0,
            x = 10.0,
            y = 20.0,
            w = 30.0,
            h = 40.0,
        )
        assertNotNull(element.x)
        assertEquals(30.0, element.w)
    }
}
