package com.lextures.android.features.planner

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.PlannerCourseFilter
import com.lextures.android.core.lms.StudentTodoBucket
import com.lextures.android.core.lms.StudentTodoItem
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun TodosScreen(
    todos: List<StudentTodoItem>,
    courseFilters: List<PlannerCourseFilter>,
    selectedCourseCode: String?,
    showCompleted: Boolean,
    loading: Boolean,
    onCourseSelected: (String?) -> Unit,
    onShowCompletedChange: (Boolean) -> Unit,
    onOpenItem: (StudentTodoItem, CourseSummary?) -> Unit,
    courses: List<CourseSummary>,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val filtered = todos
        .let { list -> selectedCourseCode?.let { code -> list.filter { it.courseCode == code } } ?: list }
        .let { list -> if (showCompleted) list else list.filter { !it.isCompleted } }
    val buckets = com.lextures.android.core.lms.PlannerLogic.bucketTodos(filtered)

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            CourseFilterChips(
                courseFilters = courseFilters,
                selectedCourseCode = selectedCourseCode,
                onCourseSelected = onCourseSelected,
            )
            Column(modifier = Modifier.padding(top = 8.dp)) {
                Text(plannerShowCompletedLabel(), fontSize = 14.sp, color = textSecondary())
                Switch(checked = showCompleted, onCheckedChange = onShowCompletedChange)
            }
        }
        if (loading && todos.isEmpty()) {
            item { LmsSkeletonList(count = 4) }
            return@LazyColumn
        }
        if (filtered.isEmpty()) {
            item {
                LmsEmptyState(
                    icon = Icons.Default.CheckCircle,
                    title = plannerEmptyTodosTitle(),
                    message = plannerEmptyTodosMessage(),
                )
            }
            return@LazyColumn
        }
        StudentTodoBucket.entries.forEach { bucket ->
            val items = buckets[bucket].orEmpty()
            if (items.isEmpty()) return@forEach
            item { LmsSectionHeader(bucketLabel(bucket)) }
            items(items, key = { it.key }) { item ->
                LaunchedEffect(item.key) { DueReminderScheduler.scheduleReminder(context, item) }
                LmsCard(onClick = {
                    onOpenItem(item, courses.firstOrNull { it.courseCode == item.courseCode })
                }) {
                    TodoRow(item)
                }
            }
        }
    }
}

@Composable
private fun TodoRow(item: StudentTodoItem) {
    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
        Text(
            text = item.title,
            fontSize = 15.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            textDecoration = if (item.isCompleted) TextDecoration.LineThrough else TextDecoration.None,
        )
        Text(item.courseTitle, fontSize = 12.sp, color = textSecondary())
        Text(statusLabel(item.completion), fontSize = 11.sp, color = textSecondary())
        item.dueAt?.let { due ->
            Text(
                text = DateTimeFormatter.ofPattern("EEE h:mm a").withZone(ZoneId.systemDefault()).format(due),
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                color = LexturesColors.Coral,
            )
        }
    }
}

fun plannerStructureItem(item: StudentTodoItem): CourseStructureItem? {
    val kind = item.structureKind ?: return null
    val id = item.structureItemId ?: return null
    return CourseStructureItem(
        id = id,
        kind = kind,
        title = item.title,
        dueAt = item.dueAt?.toString(),
    )
}
