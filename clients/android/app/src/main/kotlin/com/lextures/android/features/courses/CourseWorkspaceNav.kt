package com.lextures.android.features.courses

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobileDestinations
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import com.lextures.android.features.home.LmsSegmentedChips

@Composable
fun CourseWorkspaceNav(
    sections: List<CourseWorkspaceSection>,
    overflow: List<CourseWorkspaceSection>,
    selected: CourseWorkspaceSection,
    onSelect: (CourseWorkspaceSection) -> Unit,
    onOpenOverflow: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val chipOptions = buildList {
        addAll(sections.map { it.name to sectionLabel(context, localePrefs, it) })
        if (overflow.isNotEmpty()) {
            add("more" to L.text(context, localePrefs, R.string.mobile_ia_more_title))
        }
    }
    val selectedId = if (overflow.contains(selected)) "more" else selected.name
    LmsSegmentedChips(
        options = chipOptions,
        selectedId = selectedId,
        onSelect = { id ->
            if (id == "more") {
                onOpenOverflow()
            } else {
                CourseWorkspaceSection.entries.firstOrNull { it.name == id }?.let(onSelect)
            }
        },
        modifier = modifier.horizontalScroll(rememberScrollState()),
    )
}

@Composable
fun CourseDestinationPlaceholder(
    section: CourseWorkspaceSection,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    com.lextures.android.features.home.LmsEmptyState(
        icon = Icons.AutoMirrored.Filled.MenuBook,
        title = sectionLabel(context, localePrefs, section),
        message = L.text(context, localePrefs, R.string.mobile_ia_placeholder_message),
        modifier = modifier,
    )
}

private fun sectionLabel(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    section: CourseWorkspaceSection,
): String {
    val res = when (section) {
        CourseWorkspaceSection.Overview -> R.string.mobile_ia_course_overview
        CourseWorkspaceSection.Modules -> R.string.mobile_ia_course_modules
        CourseWorkspaceSection.Grades -> R.string.mobile_ia_course_grades
        CourseWorkspaceSection.Mastery -> R.string.mobile_ia_course_mastery
        CourseWorkspaceSection.Discussions -> R.string.mobile_ia_course_discussions
        CourseWorkspaceSection.Feed -> R.string.mobile_ia_course_feed
        CourseWorkspaceSection.Live -> R.string.mobile_ia_course_live
        CourseWorkspaceSection.People -> R.string.mobile_ia_course_people
        CourseWorkspaceSection.Files -> R.string.mobile_ia_course_files
        CourseWorkspaceSection.Attendance -> R.string.mobile_ia_course_attendance
        CourseWorkspaceSection.Evaluations -> R.string.mobile_ia_course_evaluations
        CourseWorkspaceSection.Library -> R.string.mobile_ia_course_library
        CourseWorkspaceSection.OfficeHours -> R.string.mobile_ia_course_officeHours
        CourseWorkspaceSection.Groups -> R.string.mobile_ia_course_groups
        CourseWorkspaceSection.CollabDocs -> R.string.mobile_ia_course_collabDocs
        CourseWorkspaceSection.Boards -> R.string.mobile_ia_course_boards
        CourseWorkspaceSection.Grading -> R.string.mobile_ia_course_grading
        CourseWorkspaceSection.InstructorInsights -> R.string.mobile_ia_course_insights
        CourseWorkspaceSection.Settings -> R.string.mobile_ia_course_settings
        CourseWorkspaceSection.Behavior -> R.string.mobile_ia_course_behavior
        CourseWorkspaceSection.HallPass -> R.string.mobile_ia_course_hallPass
    }
    return L.text(context, localePrefs, res)
}