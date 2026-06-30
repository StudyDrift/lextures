package com.lextures.android.core.lms

import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.temporal.TemporalAdjusters
import kotlinx.serialization.Serializable

enum class StudentTodoKind { DueItem, NotebookTask }

enum class StudentTodoCompletion { Open, Submitted, Completed }

enum class StudentTodoBucket { Overdue, Today, ThisWeek, Later }

enum class PlannerCalendarEventKind {
    Assignment,
    Quiz,
    ContentPage,
    NotebookTask,
    Academic,
}

data class StudentTodoItem(
    val key: String,
    val kind: StudentTodoKind,
    val title: String,
    val courseCode: String,
    val courseTitle: String,
    val dueAt: Instant?,
    val structureKind: String? = null,
    val structureItemId: String? = null,
    val notebookPageId: String? = null,
    val notebookTaskId: String? = null,
    val completion: StudentTodoCompletion = StudentTodoCompletion.Open,
) {
    val isCompleted: Boolean
        get() = completion == StudentTodoCompletion.Completed || completion == StudentTodoCompletion.Submitted
}

data class PlannerCalendarEvent(
    val id: String,
    val title: String,
    val courseCode: String? = null,
    val courseTitle: String? = null,
    val startsAt: Instant,
    val endsAt: Instant? = null,
    val allDay: Boolean = false,
    val kind: PlannerCalendarEventKind,
    val structureKind: String? = null,
    val structureItemId: String? = null,
    val notebookPageId: String? = null,
)

data class PlannerCourseFilter(
    val courseCode: String,
    val title: String,
)

@Serializable
data class PlannerSnapshot(
    val fetchedAtEpochMs: Long,
    val todos: List<CachedStudentTodoItem> = emptyList(),
    val events: List<CachedPlannerCalendarEvent> = emptyList(),
)

@Serializable
data class CachedStudentTodoItem(
    val key: String,
    val kind: String,
    val title: String,
    val courseCode: String,
    val courseTitle: String,
    val dueAt: String? = null,
    val structureKind: String? = null,
    val structureItemId: String? = null,
    val notebookPageId: String? = null,
    val notebookTaskId: String? = null,
    val completion: String = StudentTodoCompletion.Open.name,
)

@Serializable
data class CachedPlannerCalendarEvent(
    val id: String,
    val title: String,
    val courseCode: String? = null,
    val courseTitle: String? = null,
    val startsAt: String,
    val endsAt: String? = null,
    val allDay: Boolean = false,
    val kind: String,
    val structureKind: String? = null,
    val structureItemId: String? = null,
    val notebookPageId: String? = null,
)

enum class DueReminderLeadTime(val minutes: Int) {
    None(0),
    FifteenMinutes(15),
    OneHour(60),
    OneDay(1440),
}

object PlannerLogic {
    private const val GLOBAL_NOTEBOOK_KEY = "__global__"

    fun dueItemKey(courseCode: String, itemId: String): String = "due:$courseCode:$itemId"

    fun notebookTaskKey(taskId: String): String = "notebook:$taskId"

    fun collectTodos(
        studentCourses: List<CourseSummary>,
        structureByCourseCode: Map<String, List<CourseStructureItem>>,
        notebookTasks: List<NotebookTask>,
        gradesByCourseCode: Map<String, MyGradesResponse>,
    ): List<StudentTodoItem> {
        val courseTitles = studentCourses.associate { it.courseCode to it.displayTitle }
        val studentCodes = studentCourses.map { it.courseCode }.toSet()
        val items = mutableListOf<StudentTodoItem>()

        for (task in notebookTasks) {
            if (task.completed) continue
            if (task.courseCode != GLOBAL_NOTEBOOK_KEY && task.courseCode !in studentCodes) continue
            val title = task.taskText.trim().ifEmpty { "Untitled task" }
            items += StudentTodoItem(
                key = notebookTaskKey(task.id),
                kind = StudentTodoKind.NotebookTask,
                title = title,
                courseCode = task.courseCode,
                courseTitle = courseTitle(task.courseCode, courseTitles),
                dueAt = LmsDates.parse(task.dueAt),
                notebookPageId = task.notebookPageId,
                notebookTaskId = task.id,
                completion = if (task.completed) StudentTodoCompletion.Completed else StudentTodoCompletion.Open,
            )
        }

        for (course in studentCourses) {
            if (!course.isCalendarEnabled) continue
            val structure = structureByCourseCode[course.courseCode].orEmpty()
            val grades = gradesByCourseCode[course.courseCode]
            for (row in structure) {
                if (!isDueStructureItem(row)) continue
                items += StudentTodoItem(
                    key = dueItemKey(course.courseCode, row.id),
                    kind = StudentTodoKind.DueItem,
                    title = row.title.trim().ifEmpty { "Untitled" },
                    courseCode = course.courseCode,
                    courseTitle = course.displayTitle,
                    dueAt = LmsDates.parse(row.dueAt),
                    structureKind = row.kind,
                    structureItemId = row.id,
                    completion = completionStatus(row.id, grades),
                )
            }
        }

        return items.sortedWith(compareBy(nullsLast()) { it.dueAt })
    }

    fun bucketTodos(items: List<StudentTodoItem>, zone: ZoneId = ZoneId.systemDefault()): Map<StudentTodoBucket, List<StudentTodoItem>> {
        val now = Instant.now()
        val today = LocalDate.now(zone)
        val startOfToday = today.atStartOfDay(zone).toInstant()
        val endOfToday = today.plusDays(1).atStartOfDay(zone).minusSeconds(1).toInstant()
        val weekStart = today.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY)).atStartOfDay(zone).toInstant()
        val weekEnd = weekStart.plusSeconds(7 * 86_400 - 1)

        val buckets = StudentTodoBucket.entries.associateWith { mutableListOf<StudentTodoItem>() }
        for (item in items) {
            val due = item.dueAt
            val bucket = when {
                due == null -> StudentTodoBucket.Later
                due < startOfToday -> StudentTodoBucket.Overdue
                due <= endOfToday -> StudentTodoBucket.Today
                due <= weekEnd -> StudentTodoBucket.ThisWeek
                else -> StudentTodoBucket.Later
            }
            buckets.getValue(bucket) += item
        }
        return buckets
    }

    fun collectCalendarEvents(
        studentCourses: List<CourseSummary>,
        structureByCourseCode: Map<String, List<CourseStructureItem>>,
        notebookTasks: List<NotebookTask>,
        academicEvents: List<AcademicCalendarEvent>,
    ): List<PlannerCalendarEvent> {
        val courseTitles = studentCourses.associate { it.courseCode to it.displayTitle }
        val events = mutableListOf<PlannerCalendarEvent>()

        for (course in studentCourses) {
            if (!course.isCalendarEnabled) continue
            for (row in structureByCourseCode[course.courseCode].orEmpty()) {
                if (!isDueStructureItem(row)) continue
                val due = LmsDates.parse(row.dueAt) ?: continue
                events += PlannerCalendarEvent(
                    id = "due:${course.courseCode}:${row.id}",
                    title = row.title,
                    courseCode = course.courseCode,
                    courseTitle = course.displayTitle,
                    startsAt = due,
                    kind = calendarKind(row.kind),
                    structureKind = row.kind,
                    structureItemId = row.id,
                )
            }
        }

        for (task in notebookTasks) {
            if (task.completed) continue
            val due = LmsDates.parse(task.dueAt) ?: continue
            events += PlannerCalendarEvent(
                id = "notebook:${task.id}",
                title = task.taskText.trim().ifEmpty { "Notebook task" },
                courseCode = task.courseCode.takeIf { it != GLOBAL_NOTEBOOK_KEY },
                courseTitle = if (task.courseCode == GLOBAL_NOTEBOOK_KEY) null
                else courseTitle(task.courseCode, courseTitles),
                startsAt = due,
                kind = PlannerCalendarEventKind.NotebookTask,
                notebookPageId = task.notebookPageId,
            )
        }

        for (event in academicEvents) {
            val start = LmsDates.parse(event.startDate) ?: continue
            events += PlannerCalendarEvent(
                id = "academic:${event.id}",
                title = event.eventName,
                startsAt = start,
                endsAt = event.endDate?.let { LmsDates.parse(it) },
                allDay = event.allDay,
                kind = PlannerCalendarEventKind.Academic,
            )
        }

        return events.sortedBy { it.startsAt }
    }

    fun monthGridCells(monthAnchor: LocalDate, zone: ZoneId = ZoneId.systemDefault()): List<LocalDate> {
        val start = monthAnchor.withDayOfMonth(1)
        val mondayOffset = (start.dayOfWeek.value + 6) % 7
        val gridStart = start.minusDays(mondayOffset.toLong())
        return (0 until 42).map { gridStart.plusDays(it.toLong()) }
    }

    fun dateKeyLocal(instant: Instant, zone: ZoneId = ZoneId.systemDefault()): String =
        instant.atZone(zone).toLocalDate().toString()

    fun dateKeyLocal(date: LocalDate): String = date.toString()

    fun eventsOnDay(day: LocalDate, events: List<PlannerCalendarEvent>, zone: ZoneId = ZoneId.systemDefault()): List<PlannerCalendarEvent> {
        val key = day.toString()
        return events.filter { dateKeyLocal(it.startsAt, zone) == key }
    }

    fun eventCountsByDay(events: List<PlannerCalendarEvent>, zone: ZoneId = ZoneId.systemDefault()): Map<String, Int> =
        events.groupingBy { dateKeyLocal(it.startsAt, zone) }.eachCount()

    fun dueSoonItems(items: List<StudentTodoItem>, limit: Int = 5): List<StudentTodoItem> {
        val buckets = bucketTodos(items.filter { !it.isCompleted })
        return (buckets[StudentTodoBucket.Overdue].orEmpty() +
            buckets[StudentTodoBucket.Today].orEmpty() +
            buckets[StudentTodoBucket.ThisWeek].orEmpty())
            .take(limit)
    }

    fun encodeSnapshot(todos: List<StudentTodoItem>, events: List<PlannerCalendarEvent>): PlannerSnapshot =
        PlannerSnapshot(
            fetchedAtEpochMs = System.currentTimeMillis(),
            todos = todos.map(::cachedTodo),
            events = events.map(::cachedEvent),
        )

    fun decodeSnapshot(snapshot: PlannerSnapshot): Pair<List<StudentTodoItem>, List<PlannerCalendarEvent>> =
        snapshot.todos.map(::decodedTodo) to snapshot.events.map(::decodedEvent)

    private fun isDueStructureItem(item: CourseStructureItem): Boolean =
        item.kind in setOf("content_page", "assignment", "quiz") && !item.dueAt.isNullOrBlank()

    private fun calendarKind(structureKind: String): PlannerCalendarEventKind = when (structureKind) {
        "assignment" -> PlannerCalendarEventKind.Assignment
        "quiz" -> PlannerCalendarEventKind.Quiz
        else -> PlannerCalendarEventKind.ContentPage
    }

    private fun completionStatus(itemId: String, grades: MyGradesResponse?): StudentTodoCompletion {
        if (grades == null) return StudentTodoCompletion.Open
        if (grades.grades[itemId] != null || grades.displayGrades[itemId] != null) {
            return StudentTodoCompletion.Completed
        }
        val status = grades.gradeStatuses[itemId]?.lowercase().orEmpty()
        if (status.contains("submit")) return StudentTodoCompletion.Submitted
        if (status.contains("complete") || status.contains("graded")) return StudentTodoCompletion.Completed
        return StudentTodoCompletion.Open
    }

    private fun courseTitle(courseCode: String, titles: Map<String, String>): String =
        if (courseCode == GLOBAL_NOTEBOOK_KEY) "Notebook" else titles[courseCode] ?: courseCode

    private fun cachedTodo(item: StudentTodoItem) = CachedStudentTodoItem(
        key = item.key,
        kind = item.kind.name,
        title = item.title,
        courseCode = item.courseCode,
        courseTitle = item.courseTitle,
        dueAt = item.dueAt?.toString(),
        structureKind = item.structureKind,
        structureItemId = item.structureItemId,
        notebookPageId = item.notebookPageId,
        notebookTaskId = item.notebookTaskId,
        completion = item.completion.name,
    )

    private fun decodedTodo(cached: CachedStudentTodoItem) = StudentTodoItem(
        key = cached.key,
        kind = runCatching { StudentTodoKind.valueOf(cached.kind) }.getOrDefault(StudentTodoKind.DueItem),
        title = cached.title,
        courseCode = cached.courseCode,
        courseTitle = cached.courseTitle,
        dueAt = LmsDates.parse(cached.dueAt),
        structureKind = cached.structureKind,
        structureItemId = cached.structureItemId,
        notebookPageId = cached.notebookPageId,
        notebookTaskId = cached.notebookTaskId,
        completion = runCatching { StudentTodoCompletion.valueOf(cached.completion) }
            .getOrDefault(StudentTodoCompletion.Open),
    )

    private fun cachedEvent(event: PlannerCalendarEvent) = CachedPlannerCalendarEvent(
        id = event.id,
        title = event.title,
        courseCode = event.courseCode,
        courseTitle = event.courseTitle,
        startsAt = event.startsAt.toString(),
        endsAt = event.endsAt?.toString(),
        allDay = event.allDay,
        kind = event.kind.name,
        structureKind = event.structureKind,
        structureItemId = event.structureItemId,
        notebookPageId = event.notebookPageId,
    )

    private fun decodedEvent(cached: CachedPlannerCalendarEvent) = PlannerCalendarEvent(
        id = cached.id,
        title = cached.title,
        courseCode = cached.courseCode,
        courseTitle = cached.courseTitle,
        startsAt = LmsDates.parse(cached.startsAt) ?: Instant.now(),
        endsAt = cached.endsAt?.let { LmsDates.parse(it) },
        allDay = cached.allDay,
        kind = runCatching { PlannerCalendarEventKind.valueOf(cached.kind) }
            .getOrDefault(PlannerCalendarEventKind.Academic),
        structureKind = cached.structureKind,
        structureItemId = cached.structureItemId,
        notebookPageId = cached.notebookPageId,
    )
}

val CourseSummary.isCalendarEnabled: Boolean
    get() = calendarEnabled != false
