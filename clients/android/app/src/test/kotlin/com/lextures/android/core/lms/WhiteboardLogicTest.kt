package com.lextures.android.core.lms

import androidx.compose.ui.geometry.Offset
import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class WhiteboardLogicTest {
    @Before
    fun setUp() {
        WhiteboardObservability.resetForTests()
    }

    @Test
    fun canEditRequiresFlagAndStaff() {
        assertFalse(
            WhiteboardLogic.canEdit(true, MobilePlatformFeatures(ffMobileWhiteboardEdit = false)),
        )
        assertTrue(
            WhiteboardLogic.canEdit(true, MobilePlatformFeatures(ffMobileWhiteboardEdit = true)),
        )
        assertFalse(
            WhiteboardLogic.canEdit(false, MobilePlatformFeatures(ffMobileWhiteboardEdit = true)),
        )
    }

    @Test
    fun titleValidation() {
        assertFalse(WhiteboardLogic.isValidTitle("  "))
        assertTrue(WhiteboardLogic.isValidTitle(" Board "))
        assertEquals("Board", WhiteboardLogic.normalizeTitle(" Board "))
        assertEquals("Whiteboard 3", WhiteboardLogic.defaultTitle(2))
    }

    @Test
    fun serializeStrokeHasWebKeys() {
        val stroke = WhiteboardLogic.stroke(
            color = "#ef4444",
            width = 4.0,
            pts = listOf(Offset(1f, 2f), Offset(3f, 4f)),
        )
        val dict = WhiteboardLogic.serializeElement(stroke)!!
        assertEquals("stroke", dict["type"])
        assertEquals("#ef4444", dict["color"])
        @Suppress("UNCHECKED_CAST")
        val pts = dict["pts"] as List<List<Double>>
        assertEquals(2, pts.size)
        assertEquals(1.0, pts[0][0], 0.001)
    }

    @Test
    fun serializeRectMatchesWebSchema() {
        val rect = WhiteboardLogic.rect(
            color = "#22c55e",
            width = 2.0,
            from = Offset(10f, 20f),
            to = Offset(40f, 50f),
        )
        val dict = WhiteboardLogic.serializeElement(rect)!!
        assertEquals("rect", dict["type"])
        assertEquals(10.0, dict["x"] as Double, 0.001)
        assertEquals(20.0, dict["y"] as Double, 0.001)
        assertEquals(30.0, dict["w"] as Double, 0.001)
        assertEquals(30.0, dict["h"] as Double, 0.001)
    }

    @Test
    fun undoRedoStack() {
        var history = WhiteboardLogic.History()
        val a = listOf(WhiteboardLogic.stroke("#000000", 2.0, listOf(Offset.Zero, Offset(1f, 1f))))
        val b = a + WhiteboardLogic.line("#111111", 2.0, Offset.Zero, Offset(5f, 5f))
        history = history.push(a)
        assertTrue(history.canUndo)
        val (afterUndo, undone) = history.undo(b)
        assertEquals(1, undone!!.size)
        history = afterUndo
        val (_, redone) = history.redo(undone)
        assertEquals(2, redone!!.size)
    }

    @Test
    fun eraseRemovesHitStroke() {
        val stroke = WhiteboardLogic.stroke("#000000", 2.0, listOf(Offset(0f, 0f), Offset(10f, 0f)))
        val kept = WhiteboardLogic.line("#000000", 2.0, Offset(100f, 100f), Offset(120f, 120f))
        val next = WhiteboardLogic.erase(listOf(stroke, kept), Offset(5f, 0f), 8.0)
        assertEquals(1, next.size)
        assertEquals("line", next[0].type)
    }

    @Test
    fun translateAndPick() {
        val rect = WhiteboardLogic.rect("#000", 2.0, Offset(0f, 0f), Offset(20f, 20f))
        val moved = WhiteboardLogic.translate(rect, 5.0, 10.0)
        assertEquals(5.0, moved.x!!, 0.001)
        assertEquals(10.0, moved.y!!, 0.001)
        assertEquals(0, WhiteboardLogic.pickElement(listOf(moved), Offset(5f, 10f)))
        assertNull(WhiteboardLogic.pickElement(listOf(moved), Offset(200f, 200f)))
        assertNotNull(moved)
    }

    @Test
    fun stylusExclusiveDrawing() {
        assertTrue(WhiteboardLogic.shouldAcceptTouch(isStylus = false, stylusExclusiveDrawing = false))
        assertFalse(WhiteboardLogic.shouldAcceptTouch(isStylus = false, stylusExclusiveDrawing = true))
        assertTrue(WhiteboardLogic.shouldAcceptTouch(isStylus = true, stylusExclusiveDrawing = true))
    }

    @Test
    fun observabilityCounters() {
        WhiteboardObservability.record("whiteboard_edited", mapOf("tool" to "pen"))
        WhiteboardObservability.record("whiteboard_undo")
        assertEquals(1, WhiteboardObservability.count("whiteboard_edited"))
        assertEquals(1, WhiteboardObservability.count("whiteboard_undo"))
    }
}
