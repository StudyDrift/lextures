package com.lextures.android.features.navigation

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.features.home.HomeShellState

/**
 * Course-scoped navigation drawer: the first-level menu shown while inside a course.
 * Mirrors the web course sidebar with a Back affordance (→ global menu) and a Dashboard
 * shortcut (→ leave course).
 */
@Composable
fun CourseDrawer(shell: HomeShellState, modifier: Modifier = Modifier) {
    val groups = MobileDestinations.courseDrawerGroups(shell.activeCourseSections)
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                painter = painterResource(R.drawable.launch_logo),
                contentDescription = null,
                tint = Color.Unspecified,
                modifier = Modifier.size(32.dp),
            )
            Text(
                text = shell.activeCourse?.displayTitle ?: "Lextures",
                fontSize = 17.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 2,
            )
        }

        Column(modifier = Modifier.padding(horizontal = 10.dp)) {
            DrawerRow(
                label = L.text(R.string.mobile_drawer_back),
                icon = Icons.AutoMirrored.Filled.ArrowBack,
                selected = false,
                onClick = { shell.openGlobalDrawer() },
            )
            DrawerRow(
                label = L.text(R.string.mobile_drawer_dashboard),
                icon = Icons.Filled.GridView,
                selected = false,
                onClick = { shell.exitCourseToDashboard() },
            )
        }

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 10.dp, vertical = 4.dp),
        ) {
            groups.forEach { group ->
                DrawerGroupHeader(drawerString(group.titleRes))
                group.sections.forEach { section ->
                    DrawerRow(
                        label = L.text(courseSectionLabelRes(section)),
                        icon = courseSectionIcon(section),
                        selected = shell.activeCourseSection == section,
                        onClick = {
                            shell.activeCourseSection = section
                            shell.closeDrawer()
                        },
                    )
                }
            }
        }
    }
}
