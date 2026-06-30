package com.lextures.android.features.planner

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.PlannerCalendarEventKind
import com.lextures.android.core.lms.StudentTodoBucket
import com.lextures.android.core.lms.StudentTodoCompletion

@Composable
fun plannerTitle(): String = L.text(R.string.mobile_planner_title)

@Composable
fun plannerTabTodos(): String = L.text(R.string.mobile_planner_tab_todos)

@Composable
fun plannerTabCalendar(): String = L.text(R.string.mobile_planner_tab_calendar)

@Composable
fun plannerAllCoursesLabel(): String = L.text(R.string.mobile_planner_filter_allCourses)

@Composable
fun plannerShowCompletedLabel(): String = L.text(R.string.mobile_planner_filter_showCompleted)

@Composable
fun plannerEmptyTodosTitle(): String = L.text(R.string.mobile_planner_todos_empty_title)

@Composable
fun plannerEmptyTodosMessage(): String = L.text(R.string.mobile_planner_todos_empty_message)

@Composable
fun plannerEmptyDayLabel(): String = L.text(R.string.mobile_planner_calendar_emptyDay)

@Composable
fun plannerSubscribeTitle(): String = L.text(R.string.mobile_planner_subscribe_title)

@Composable
fun plannerSubscribeMessage(): String = L.text(R.string.mobile_planner_subscribe_message)

@Composable
fun plannerSubscribeGenerate(): String = L.text(R.string.mobile_planner_subscribe_generate)

@Composable
fun plannerSubscribeCopy(): String = L.text(R.string.mobile_planner_subscribe_copy)

@Composable
fun plannerSubscribeOpen(): String = L.text(R.string.mobile_planner_subscribe_open)

@Composable
fun bucketLabel(bucket: StudentTodoBucket): String = when (bucket) {
    StudentTodoBucket.Overdue -> L.text(R.string.mobile_planner_bucket_overdue)
    StudentTodoBucket.Today -> L.text(R.string.mobile_planner_bucket_today)
    StudentTodoBucket.ThisWeek -> L.text(R.string.mobile_planner_bucket_thisWeek)
    StudentTodoBucket.Later -> L.text(R.string.mobile_planner_bucket_later)
}

@Composable
fun statusLabel(completion: StudentTodoCompletion): String = when (completion) {
    StudentTodoCompletion.Open -> L.text(R.string.mobile_planner_status_open)
    StudentTodoCompletion.Submitted -> L.text(R.string.mobile_planner_status_submitted)
    StudentTodoCompletion.Completed -> L.text(R.string.mobile_planner_status_completed)
}

@Composable
fun calendarKindLabel(kind: PlannerCalendarEventKind): String = when (kind) {
    PlannerCalendarEventKind.Assignment -> L.text(R.string.mobile_planner_kind_assignment)
    PlannerCalendarEventKind.Quiz -> L.text(R.string.mobile_planner_kind_quiz)
    PlannerCalendarEventKind.ContentPage -> L.text(R.string.mobile_planner_kind_page)
    PlannerCalendarEventKind.NotebookTask -> L.text(R.string.mobile_planner_kind_task)
    PlannerCalendarEventKind.Academic -> L.text(R.string.mobile_planner_kind_academic)
}
