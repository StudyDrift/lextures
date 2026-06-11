package com.lextures.android.features.courses

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.activity.compose.BackHandler
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.isDarkTheme
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
    var openItem by remember { mutableStateOf<CourseStructureItem?>(null) }

    BackHandler(onBack = onBack)

    openItem?.let { selected ->
        ItemDetailScreen(
            session = session,
            courseCode = course.courseCode,
            item = selected,
            onBack = { openItem = null },
            modifier = modifier,
        )
        return
    }

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
                // Gradient cover banner — matches the course's tile color across the app.
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(24.dp))
                        .background(coverBrush(course.courseCode)),
                ) {
                    Box(
                        modifier = Modifier
                            .size(140.dp)
                            .offset(x = 260.dp, y = (-52).dp)
                            .clip(CircleShape)
                            .background(Color.White.copy(alpha = 0.08f)),
                    )
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(20.dp),
                        verticalArrangement = Arrangement.spacedBy(7.dp),
                    ) {
                        Text(
                            text = course.courseCode.uppercase(),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.SemiBold,
                            letterSpacing = 1.2.sp,
                            color = Color.White.copy(alpha = 0.8f),
                        )
                        Text(
                            text = course.title,
                            style = LexturesType.display(22),
                            color = Color.White,
                        )
                        if (course.description.isNotEmpty()) {
                            Text(
                                text = course.description,
                                fontSize = 13.sp,
                                color = Color.White.copy(alpha = 0.85f),
                                maxLines = 3,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        LmsDates.parse(course.startsAt)?.let {
                            Text(
                                text = "Starts ${LmsDates.shortDate(course.startsAt)}",
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Medium,
                                color = Color.White,
                                modifier = Modifier
                                    .padding(top = 4.dp)
                                    .clip(RoundedCornerShape(50))
                                    .background(Color.White.copy(alpha = 0.16f))
                                    .padding(horizontal = 9.dp, vertical = 4.dp),
                            )
                        }
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

            itemsIndexed(groups, key = { _, group -> group.id }) { index, group ->
                LmsCard {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(26.dp)
                                .clip(RoundedCornerShape(8.dp))
                                .background(coverBrush(course.courseCode)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                text = "${index + 1}",
                                style = LexturesType.display(13, FontWeight.Bold),
                                color = Color.White,
                            )
                        }
                        Text(
                            text = group.title,
                            style = LexturesType.display(17),
                            color = textPrimary(),
                        )
                    }

                    if (group.items.isEmpty()) {
                        Text(
                            text = "Nothing in this module yet",
                            fontSize = 12.sp,
                            fontStyle = FontStyle.Italic,
                            color = textSecondary(),
                        )
                    } else {
                        group.items.forEachIndexed { itemIndex, item ->
                            if (itemIndex > 0) {
                                HorizontalDivider()
                            }
                            ModuleItemRow(
                                item = item,
                                openable = ItemKind.isOpenable(item.kind),
                                onClick = { openItem = item },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ModuleItemRow(
    item: CourseStructureItem,
    openable: Boolean,
    onClick: () -> Unit,
) {
    var rowModifier = Modifier.fillMaxWidth()
    if (openable) {
        rowModifier = rowModifier.clickable(onClick = onClick)
    }
    Row(
        modifier = rowModifier.padding(vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(32.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.16f else 0.13f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                ItemKind.icon(item.kind),
                contentDescription = null,
                tint = accentColor(),
                modifier = Modifier.size(16.dp),
            )
        }
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                text = item.title,
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(text = ItemKind.label(item.kind), fontSize = 11.sp, color = textSecondary())
                LmsDates.parse(item.dueAt)?.let {
                    Text(
                        text = "Due ${LmsDates.shortDateTime(item.dueAt)}",
                        fontSize = 11.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = LexturesColors.Coral,
                    )
                }
            }
        }
        val points = item.pointsWorth ?: item.pointsPossible
        if (points != null) {
            Text(
                text = "${formatPoints(points)} pts",
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = LexturesColors.Amber,
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(LexturesColors.Amber.copy(alpha = 0.13f))
                    .padding(horizontal = 7.dp, vertical = 3.dp),
            )
        }
        if (openable) {
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = textSecondary().copy(alpha = 0.6f),
                modifier = Modifier.size(16.dp),
            )
        }
    }
}

private fun formatPoints(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()
