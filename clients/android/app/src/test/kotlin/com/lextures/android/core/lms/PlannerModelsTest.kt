package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

class PlannerModelsTest {
    @Test
    fun bucketsOverdueTodayAndLater() {
        val zone = ZoneId.of("UTC")
        val now = LocalDate.of(2026, 6, 30).atTime(12, 0).atZone(zone).toInstant()
        val yesterday = now.minusSeconds(86_400)
        val later = now.plusSeconds(10 * 86_400)
        val items = listOf(
            todo("a", yesterday),
            todo("b", now),
            todo("c", later),
        )
        val buckets = PlannerLogic.bucketTodos(items, zone, now)
        assertEquals(listOf("a"), buckets[StudentTodoBucket.Overdue]?.map { it.key })
        assertEquals(listOf("b"), buckets[StudentTodoBucket.Today]?.map { it.key })
        assertEquals(listOf("c"), buckets[StudentTodoBucket.Later]?.map { it.key })
    }

    @Test
    fun collectTodosFromStructure() {
        val course = CourseSummary(
            id = "1",
            courseCode = "BIO101",
            title = "Biology",
            viewerEnrollmentRoles = listOf("student"),
            calendarEnabled = true,
        )
        val structure = listOf(
            CourseStructureItem(
                id = "q1",
                kind = "quiz",
                title = "Quiz 1",
                dueAt = "2026-07-01T23:59:00Z",
            ),
        )
        val todos = PlannerLogic.collectTodos(
            studentCourses = listOf(course),
            structureByCourseCode = mapOf("BIO101" to structure),
            notebookTasks = emptyList(),
            gradesByCourseCode = emptyMap(),
        )
        assertEquals(1, todos.size)
        assertEquals("Quiz 1", todos[0].title)
    }

    @Test
    fun monthGridHas42Cells() {
        val cells = PlannerLogic.monthGridCells(LocalDate.of(2026, 6, 1), ZoneId.of("UTC"))
        assertEquals(42, cells.size)
    }

    @Test
    fun snapshotRoundTrip() {
        val todo = todo("due:BIO101:q1", Instant.parse("2026-07-01T12:00:00Z"))
        val event = PlannerCalendarEvent(
            id = "due:BIO101:q1",
            title = "Quiz 1",
            courseCode = "BIO101",
            courseTitle = "Biology",
            startsAt = Instant.parse("2026-07-01T12:00:00Z"),
            kind = PlannerCalendarEventKind.Quiz,
            structureKind = "quiz",
            structureItemId = "q1",
        )
        val snapshot = PlannerLogic.encodeSnapshot(listOf(todo), listOf(event))
        val (decodedTodos, decodedEvents) = PlannerLogic.decodeSnapshot(snapshot)
        assertEquals(1, decodedTodos.size)
        assertEquals(1, decodedEvents.size)
        assertEquals(todo.key, decodedTodos[0].key)
    }

    @Test
    fun dueSoonItemsRespectsLimit() {
        val now = Instant.now()
        val items = (1..8).map { index ->
            todo("item-$index", now.plusSeconds(index * 3600L))
        }
        assertTrue(PlannerLogic.dueSoonItems(items, limit = 5).size <= 5)
    }

    private fun todo(key: String, due: Instant) = StudentTodoItem(
        key = key,
        kind = StudentTodoKind.DueItem,
        title = "Item",
        courseCode = "BIO101",
        courseTitle = "Biology",
        dueAt = due,
        structureKind = "assignment",
        structureItemId = "a1",
    )
}
