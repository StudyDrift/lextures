package com.lextures.android.features.navigation

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.Accessible
import androidx.compose.material.icons.filled.Assignment
import androidx.compose.material.icons.filled.Autorenew
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.Book
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Checklist
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.DynamicFeed
import androidx.compose.material.icons.filled.DirectionsWalk
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.filled.EventAvailable
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.FamilyRestroom
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.Group
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Public
import androidx.compose.material.icons.filled.RateReview
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Sensors
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.ViewModule
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.R
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.RootDestination

/** Resolves a string-resource entry name (stored on the nav enums) to localized text. */
@Composable
fun drawerString(name: String): String {
    val context = LocalContext.current
    val prefs = LocalLocalePreferences.current
    val id = context.resources.getIdentifier(name, "string", context.packageName)
    return if (id != 0) prefs.localizedContext(context).getString(id) else name
}

fun rootDestinationIcon(dest: RootDestination): ImageVector = when (dest) {
    RootDestination.Dashboard -> Icons.Filled.GridView
    RootDestination.Courses -> Icons.AutoMirrored.Filled.MenuBook
    RootDestination.Calendar -> Icons.Filled.CalendarMonth
    RootDestination.Todos -> Icons.Filled.Checklist
    RootDestination.Review -> Icons.Filled.Autorenew
    RootDestination.Insights -> Icons.Filled.BarChart
    RootDestination.Notebooks -> Icons.Filled.Book
    RootDestination.GlobalNotebook -> Icons.Filled.Public
    RootDestination.Accommodations -> Icons.Filled.Accessible
    RootDestination.Inbox -> Icons.Filled.Inbox
    RootDestination.Settings -> Icons.Filled.Settings
    RootDestination.Profile -> Icons.Filled.Person
    RootDestination.Teach -> Icons.Filled.CheckCircle
    RootDestination.Children -> Icons.Filled.FamilyRestroom
}

fun courseSectionIcon(section: CourseWorkspaceSection): ImageVector = when (section) {
    CourseWorkspaceSection.Overview -> Icons.Filled.Description
    CourseWorkspaceSection.Modules -> Icons.Filled.ViewModule
    CourseWorkspaceSection.Files -> Icons.Filled.Folder
    CourseWorkspaceSection.Library -> Icons.AutoMirrored.Filled.MenuBook
    CourseWorkspaceSection.Discussions -> Icons.Filled.Forum
    CourseWorkspaceSection.Feed -> Icons.Filled.DynamicFeed
    CourseWorkspaceSection.Live -> Icons.Filled.Sensors
    CourseWorkspaceSection.OfficeHours -> Icons.Filled.Schedule
    CourseWorkspaceSection.Grades -> Icons.Filled.Assignment
    CourseWorkspaceSection.Mastery -> Icons.Filled.BarChart
    CourseWorkspaceSection.People -> Icons.Filled.Group
    CourseWorkspaceSection.Groups -> Icons.Filled.Group
    CourseWorkspaceSection.CollabDocs -> Icons.Filled.Description
    CourseWorkspaceSection.Grading -> Icons.Filled.FactCheck
    CourseWorkspaceSection.InstructorInsights -> Icons.Filled.BarChart
    CourseWorkspaceSection.Attendance -> Icons.Filled.EventAvailable
    CourseWorkspaceSection.Evaluations -> Icons.Filled.RateReview
    CourseWorkspaceSection.Behavior -> Icons.Filled.ThumbUp
    CourseWorkspaceSection.HallPass -> Icons.Filled.DirectionsWalk
}

/** R.string id for a course section label (reuses the existing `mobile_ia_course_*` keys). */
fun courseSectionLabelRes(section: CourseWorkspaceSection): Int = when (section) {
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
    CourseWorkspaceSection.Grading -> R.string.mobile_ia_course_grading
    CourseWorkspaceSection.InstructorInsights -> R.string.mobile_ia_course_insights
    CourseWorkspaceSection.Behavior -> R.string.mobile_ia_course_behavior
    CourseWorkspaceSection.HallPass -> R.string.mobile_ia_course_hallPass
}
