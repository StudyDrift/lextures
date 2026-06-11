package com.lextures.android.features.courses

import androidx.compose.foundation.background
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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsCoverTile
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner

@Composable
fun CoursesTab(session: AuthSession, modifier: Modifier = Modifier) {
    val accessToken by session.accessToken.collectAsState()

    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var searchText by remember { mutableStateOf("") }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            courses = LmsApi.fetchCourses(token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
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

    val filtered = remember(courses, searchText) {
        val q = searchText.trim().lowercase()
        if (q.isEmpty()) {
            courses
        } else {
            courses.filter {
                it.displayTitle.lowercase().contains(q) ||
                    it.courseCode.lowercase().contains(q) ||
                    it.description.lowercase().contains(q)
            }
        }
    }

    Column(modifier = modifier) {
        Text(
            text = "Courses",
            style = LexturesType.display(24),
            color = textPrimary(),
            modifier = Modifier.padding(start = 16.dp, top = 12.dp),
        )

        OutlinedTextField(
            value = searchText,
            onValueChange = { searchText = it },
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 10.dp),
            placeholder = { Text("Search courses", color = textSecondary()) },
            leadingIcon = { Icon(Icons.Default.Search, contentDescription = null, tint = textSecondary()) },
            singleLine = true,
            shape = RoundedCornerShape(14.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = LexturesColors.Primary,
                unfocusedBorderColor = fieldBorder(),
                focusedContainerColor = cardBackground(),
                unfocusedContainerColor = cardBackground(),
            ),
        )

        when {
            loading && courses.isEmpty() -> Box(
                modifier = Modifier.fillMaxSize(),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }

            filtered.isEmpty() -> Column {
                errorMessage?.let { LmsErrorBanner(it, Modifier.padding(horizontal = 16.dp)) }
                LmsEmptyState(
                    icon = Icons.AutoMirrored.Filled.MenuBook,
                    title = if (searchText.isBlank()) "No courses yet" else "No matching courses",
                    message = if (searchText.isBlank()) {
                        "Courses you enroll in will show up here."
                    } else {
                        "Try different keywords, or clear search."
                    },
                )
            }

            else -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                errorMessage?.let { message ->
                    item { LmsErrorBanner(message) }
                }
                items(filtered, key = { it.id }) { course ->
                    CourseRowCard(course = course, onClick = { openCourse = course })
                }
            }
        }
    }
}

@Composable
fun CourseRowCard(course: CourseSummary, onClick: () -> Unit, modifier: Modifier = Modifier) {
    LmsCard(modifier = modifier, onClick = onClick) {
        Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            LmsCoverTile(key = course.courseCode, icon = Icons.AutoMirrored.Filled.MenuBook, size = 52)
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    text = course.displayTitle,
                    style = LexturesType.display(16),
                    color = textPrimary(),
                )
                Text(
                    text = course.courseCode.uppercase(),
                    fontSize = 11.sp,
                    fontWeight = FontWeight.SemiBold,
                    letterSpacing = 0.8.sp,
                    color = accentColor(),
                )
                if (course.description.isNotEmpty()) {
                    Text(
                        text = course.description,
                        fontSize = 12.sp,
                        color = textSecondary(),
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}
