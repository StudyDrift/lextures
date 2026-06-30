package com.lextures.android.features.planner

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.PlannerCalendarEvent
import com.lextures.android.core.lms.PlannerCourseFilter
import com.lextures.android.core.lms.PlannerLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsSectionHeader
import java.time.LocalDate
import java.time.YearMonth
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.TextStyle
import java.util.Locale

@Composable
fun CalendarScreen(
    events: List<PlannerCalendarEvent>,
    courseFilters: List<PlannerCourseFilter>,
    selectedCourseCode: String?,
    onCourseSelected: (String?) -> Unit,
    onEventSelected: (PlannerCalendarEvent) -> Unit,
    modifier: Modifier = Modifier,
) {
    var month by remember { mutableStateOf(YearMonth.now()) }
    var selectedDay by remember { mutableStateOf(LocalDate.now()) }
    val zone = ZoneId.systemDefault()
    val filtered = selectedCourseCode?.let { code ->
        events.filter { it.courseCode == null || it.courseCode == code }
    } ?: events
    val counts = PlannerLogic.eventCountsByDay(filtered, zone)
    val cells = PlannerLogic.monthGridCells(month.atDay(1), zone)
    val dayEvents = PlannerLogic.eventsOnDay(selectedDay, filtered, zone)

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            CourseFilterChips(
                courseFilters = courseFilters,
                selectedCourseCode = selectedCourseCode,
                onCourseSelected = onCourseSelected,
            )
        }
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = { month = month.minusMonths(1) }) {
                    Icon(Icons.Default.ChevronLeft, contentDescription = null)
                }
                Text(
                    text = month.month.getDisplayName(TextStyle.FULL, Locale.getDefault()) + " ${month.year}",
                    modifier = Modifier.weight(1f),
                    fontSize = 18.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                IconButton(onClick = { month = month.plusMonths(1) }) {
                    Icon(Icons.Default.ChevronRight, contentDescription = null)
                }
            }
        }
        item {
            LazyVerticalGrid(
                columns = GridCells.Fixed(7),
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(18.dp))
                    .background(cardBackground())
                    .padding(12.dp),
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                items(listOf("M", "T", "W", "T", "F", "S", "S")) { label ->
                    Text(label, fontSize = 11.sp, color = textSecondary(), modifier = Modifier.fillMaxWidth())
                }
                items(cells) { day ->
                    val inMonth = day.month == month.month
                    val selected = day == selectedDay
                    val count = counts[PlannerLogic.dateKeyLocal(day)] ?: 0
                    Column(
                        modifier = Modifier
                            .clip(RoundedCornerShape(10.dp))
                            .background(if (selected) accentColor() else cardBackground())
                            .clickable { selectedDay = day }
                            .padding(vertical = 6.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text(
                            text = "${day.dayOfMonth}",
                            fontSize = 14.sp,
                            fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
                            color = if (selected) LexturesColors.PrimaryDeep
                            else if (inMonth) textPrimary() else textSecondary(),
                        )
                        Box(
                            modifier = Modifier
                                .size(5.dp)
                                .clip(CircleShape)
                                .background(if (count > 0) LexturesColors.Coral else cardBackground()),
                        )
                    }
                }
            }
        }
        item {
            LmsSectionHeader(
                selectedDay.format(DateTimeFormatter.ofPattern("EEEE, MMM d")),
            )
        }
        if (dayEvents.isEmpty()) {
            item {
                LmsCard {
                    Text(plannerEmptyDayLabel(), fontSize = 14.sp, color = textSecondary())
                }
            }
        } else {
            items(dayEvents, key = { it.id }) { event ->
                LmsCard(onClick = { onEventSelected(event) }) {
                    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                        Text(event.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        event.courseTitle?.let {
                            Text(it, fontSize = 12.sp, color = textSecondary())
                        }
                        Text(calendarKindLabel(event.kind), fontSize = 11.sp, color = textSecondary())
                    }
                }
            }
        }
    }
}
