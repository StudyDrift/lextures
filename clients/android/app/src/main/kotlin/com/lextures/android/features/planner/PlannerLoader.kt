package com.lextures.android.features.planner

import com.lextures.android.core.lms.AcademicCalendarEvent
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MyGradesResponse
import com.lextures.android.core.lms.NotebookTask
import com.lextures.android.core.lms.NotebookTasksApi
import com.lextures.android.core.lms.PlannerCalendarEvent
import com.lextures.android.core.lms.PlannerCourseFilter
import com.lextures.android.core.lms.PlannerLogic
import com.lextures.android.core.lms.PlannerSnapshot
import com.lextures.android.core.offline.fetchCoursesCached
import com.lextures.android.core.lms.StudentTodoItem
import com.lextures.android.core.lms.isCalendarEnabled
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope

object PlannerLoader {
    suspend fun load(
        accessToken: String,
        offline: OfflineService,
        isOnline: Boolean,
    ): PlannerLoadResult {
        val coursesResult = offline.fetchCoursesCached(accessToken) {
            LmsApi.fetchCourses(accessToken)
        }
        val enriched = coroutineScope {
            coursesResult.first.map { course ->
                async { runCatching { LmsApi.fetchCourse(course.courseCode, accessToken) }.getOrDefault(course) }
            }.awaitAll()
        }
        val students = enriched.filter { it.viewerIsStudent && it.isCalendarEnabled }

        val snapshotResult = offline.cachedFetch(
            key = OfflineCacheKey.plannerSnapshot(),
            accessToken = accessToken,
            serializer = PlannerSnapshot.serializer(),
            fetch = { fetchSnapshot(students, accessToken) },
        )
        val (todos, events) = PlannerLogic.decodeSnapshot(snapshotResult.first)
        val staleLabel = snapshotResult.second
            ?.takeIf { it.isStale(isOnline) }
            ?.lastUpdatedLabel()

        return PlannerLoadResult(
            courses = enriched,
            courseFilters = students.map { PlannerCourseFilter(it.courseCode, it.displayTitle) },
            todos = todos,
            events = events,
            staleLabel = staleLabel,
        )
    }

    private suspend fun fetchSnapshot(
        studentCourses: List<CourseSummary>,
        accessToken: String,
    ): PlannerSnapshot = coroutineScope {
        val notebookTasks = runCatching { NotebookTasksApi.fetch(accessToken) }.getOrDefault(emptyList())
        val structures = mutableMapOf<String, List<CourseStructureItem>>()
        val grades = mutableMapOf<String, MyGradesResponse>()
        studentCourses.map { course ->
            async {
                val structure = runCatching { LmsApi.fetchCourseStructure(course.courseCode, accessToken) }
                    .getOrDefault(emptyList())
                val myGrades = runCatching { LmsApi.fetchMyGrades(course.courseCode, accessToken) }.getOrNull()
                Triple(course.courseCode, structure, myGrades)
            }
        }.awaitAll().forEach { (code, structure, myGrades) ->
            structures[code] = structure
            if (myGrades != null) grades[code] = myGrades
        }

        val academic = mutableListOf<AcademicCalendarEvent>()
        val seen = mutableSetOf<String>()
        for (course in studentCourses) {
            val orgId = course.orgId ?: continue
            val key = "$orgId:${course.termId.orEmpty()}"
            if (!seen.add(key)) continue
            academic += runCatching {
                LmsApi.fetchAcademicCalendarEvents(orgId, course.termId, accessToken)
            }.getOrDefault(emptyList())
        }

        val todos = PlannerLogic.collectTodos(studentCourses, structures, notebookTasks, grades)
        val events = PlannerLogic.collectCalendarEvents(studentCourses, structures, notebookTasks, academic)
        PlannerLogic.encodeSnapshot(todos, events)
    }
}

data class PlannerLoadResult(
    val courses: List<CourseSummary>,
    val courseFilters: List<PlannerCourseFilter>,
    val todos: List<StudentTodoItem>,
    val events: List<PlannerCalendarEvent>,
    val staleLabel: String?,
)
