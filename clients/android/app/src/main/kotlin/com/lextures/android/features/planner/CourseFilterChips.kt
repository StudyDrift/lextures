package com.lextures.android.features.planner

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.PlannerCourseFilter

@Composable
fun CourseFilterChips(
    courseFilters: List<PlannerCourseFilter>,
    selectedCourseCode: String?,
    onCourseSelected: (String?) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier.horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        FilterChip(
            selected = selectedCourseCode == null,
            onClick = { onCourseSelected(null) },
            label = { Text(L.text(R.string.mobile_planner_filter_allCourses)) },
            shape = RoundedCornerShape(50),
            colors = FilterChipDefaults.filterChipColors(
                selectedContainerColor = accentColor(),
                selectedLabelColor = textPrimary(),
            ),
        )
        courseFilters.forEach { filter ->
            FilterChip(
                selected = selectedCourseCode == filter.courseCode,
                onClick = { onCourseSelected(filter.courseCode) },
                label = { Text(filter.title) },
                shape = RoundedCornerShape(50),
                colors = FilterChipDefaults.filterChipColors(
                    selectedContainerColor = accentColor(),
                    selectedLabelColor = textPrimary(),
                ),
            )
        }
    }
}
