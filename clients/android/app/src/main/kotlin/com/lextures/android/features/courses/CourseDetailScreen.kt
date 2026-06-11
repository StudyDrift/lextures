package com.lextures.android.features.courses

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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Assignment
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.Link
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.activity.compose.BackHandler
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner

private data class ModuleGroup(
    val id: String,
    val title: String,
    val items: List<CourseStructureItem>,
)

/** Course structure (modules and items) for one course. */
@Composable
fun CourseDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()

    var items by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            items = LmsApi.fetchCourseStructure(course.courseCode, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    val groups = remember(items) {
        val modules = items.filter { it.isModule }.sortedBy { it.sortOrder }
        val children = items.filter { !it.isModule && it.parentId != null }.groupBy { it.parentId }
        val grouped = modules.map { module ->
            ModuleGroup(module.id, module.title, (children[module.id] ?: emptyList()).sortedBy { it.sortOrder })
        }
        val orphans = items
            .filter { !it.isModule && it.parentId == null && it.kind != "heading" }
            .sortedBy { it.sortOrder }
        if (orphans.isEmpty()) grouped else grouped + ModuleGroup("__orphans__", "Other items", orphans)
    }

    Column(modifier = modifier) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = course.displayTitle,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        if (loading && items.isEmpty()) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            return
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LmsCard {
                    Text(
                        text = course.title,
                        fontSize = 16.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Text(text = course.courseCode, fontSize = 12.sp, color = textSecondary())
                    if (course.description.isNotEmpty()) {
                        Text(text = course.description, fontSize = 14.sp, color = textSecondary())
                    }
                    LmsDates.parse(course.startsAt)?.let {
                        Text(
                            text = "Starts ${LmsDates.shortDate(course.startsAt)}",
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (groups.isEmpty() && errorMessage == null) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.Layers,
                        title = "No content yet",
                        message = "Modules and assignments will appear here once published.",
                    )
                }
            }

            items(groups, key = { it.id }) { group ->
                LmsCard {
                    Text(
                        text = group.title,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    if (group.items.isEmpty()) {
                        Text(text = "Empty module", fontSize = 12.sp, color = textSecondary())
                    } else {
                        group.items.forEach { item ->
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 4.dp),
                                horizontalArrangement = Arrangement.spacedBy(10.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Icon(
                                    iconFor(item.kind),
                                    contentDescription = null,
                                    tint = LexturesColors.Primary,
                                    modifier = Modifier.size(18.dp),
                                )
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(text = item.title, fontSize = 14.sp, color = textPrimary())
                                    LmsDates.parse(item.dueAt)?.let {
                                        Text(
                                            text = "Due ${LmsDates.shortDateTime(item.dueAt)}",
                                            fontSize = 12.sp,
                                            color = textSecondary(),
                                        )
                                    }
                                }
                                val points = item.pointsWorth ?: item.pointsPossible
                                if (points != null) {
                                    Text(
                                        text = "${formatPoints(points)} pts",
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun formatPoints(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()

private fun iconFor(kind: String): ImageVector = when (kind) {
    "assignment" -> Icons.AutoMirrored.Filled.Assignment
    "quiz" -> Icons.Default.CheckCircle
    "content_page" -> Icons.Default.Description
    "external_link", "lti_link" -> Icons.Default.Link
    "library_resource", "textbook_resource" -> Icons.AutoMirrored.Filled.MenuBook
    else -> Icons.Default.Layers
}
