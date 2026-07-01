package com.lextures.android.features.home

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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GradingBacklogItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.grading.GradingBacklogSection

@Composable
fun TeachHubScreen(
    session: AuthSession,
    shell: HomeShellState,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var staffCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        runCatching { LmsApi.fetchCourses(token) }
            .onSuccess { list ->
                staffCourses = list.filter { it.viewerIsStaff }
            }
        loading = false
    }

    openCourse?.let { course ->
        CourseDetailScreen(
            session = session,
            course = course,
            onBack = { openCourse = null },
            modifier = modifier,
        )
        return
    }

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_ia_teach_title),
                fontSize = 20.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }
        if (loading) {
            item { LmsSkeletonList(count = 3) }
        } else if (staffCourses.isEmpty()) {
            item {
                LmsEmptyState(
                    icon = Icons.Default.CheckCircle,
                    title = L.text(context, localePrefs, R.string.mobile_ia_teach_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_ia_teach_emptyMessage),
                )
            }
        } else {
            items(staffCourses, key = { it.id }) { course ->
                LmsCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 4.dp),
                ) {
                    GradingBacklogSection(
                        session = session,
                        course = course,
                        onOpenItem = { _: GradingBacklogItem -> openCourse = course },
                    )
                }
            }
        }
    }
}