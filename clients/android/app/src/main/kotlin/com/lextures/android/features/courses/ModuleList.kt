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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LockReason
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModuleGroup
import com.lextures.android.core.lms.ModulesProgressSnapshot
import com.lextures.android.features.home.LmsCard

/** Sectioned module list with type icons, completion, and lock indicators (M3.1). */
@Composable
fun ModuleList(
    course: CourseSummary,
    groups: List<ModuleGroup>,
    progress: ModulesProgressSnapshot?,
    onSelectItem: (CourseStructureItem) -> Unit,
    onLockedItem: (CourseStructureItem, LockReason?) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
        groups.forEachIndexed { index, group ->
            LmsCard {
                val moduleState = ModuleContentLogic.moduleLockState(progress, group.id)
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
                            color = androidx.compose.ui.graphics.Color.White,
                        )
                    }
                    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(text = group.title, style = LexturesType.display(17), color = textPrimary())
                        moduleState?.reason?.message?.takeIf { moduleState.locked && it.isNotEmpty() }?.let { reason ->
                            Row(horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
                                Icon(Icons.Default.Lock, contentDescription = null, tint = LexturesColors.Coral, modifier = Modifier.size(12.dp))
                                Text(text = reason, fontSize = 11.sp, color = LexturesColors.Coral)
                            }
                        }
                    }
                }

                if (group.items.isEmpty()) {
                    Text(
                        text = moduleEmptyLabel(),
                        fontSize = 12.sp,
                        fontStyle = FontStyle.Italic,
                        color = textSecondary(),
                    )
                } else {
                    group.items.forEachIndexed { itemIndex, item ->
                        if (itemIndex > 0) HorizontalDivider()
                        ModuleListItemRow(
                            item = item,
                            progress = progress,
                            onSelect = { onSelectItem(item) },
                            onLocked = { onLockedItem(item, ModuleContentLogic.itemLockState(progress, item.id)?.reason) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ModuleListItemRow(
    item: CourseStructureItem,
    progress: ModulesProgressSnapshot?,
    onSelect: () -> Unit,
    onLocked: () -> Unit,
) {
    val navigable = ModuleContentLogic.isNavigable(item.kind)
    val locked = ModuleContentLogic.isLocked(progress, item.id)
    val complete = ModuleContentLogic.isComplete(progress, item.id)
    val completeLabel = moduleCompleteLabel()
    val a11y = buildString {
        append(ItemKind.label(item.kind))
        append(", ")
        append(item.title)
        if (complete) {
            append(", ")
            append(completeLabel)
        }
        ModuleContentLogic.itemLockState(progress, item.id)?.reason?.message?.takeIf { it.isNotEmpty() }?.let {
            append(", ")
            append(it)
        }
    }

    ModuleItemRow(
        item = item,
        openable = navigable && !locked,
        locked = locked,
        complete = complete,
        onClick = {
            when {
                locked -> onLocked()
                navigable -> onSelect()
            }
        },
        modifier = Modifier.semantics { contentDescription = a11y },
    )
}

@Composable
fun ModuleItemRow(
    item: CourseStructureItem,
    openable: Boolean,
    locked: Boolean = false,
    complete: Boolean = false,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var rowModifier = modifier.fillMaxWidth().padding(vertical = 6.dp)
    if (openable || locked) {
        rowModifier = rowModifier.clickable(onClick = onClick)
    }

    Row(
        modifier = rowModifier,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(contentAlignment = Alignment.BottomEnd) {
            Box(
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.16f else 0.13f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    if (locked) Icons.Default.Lock else ItemKind.icon(item.kind),
                    contentDescription = null,
                    tint = if (locked) textSecondary() else accentColor(),
                    modifier = Modifier.size(16.dp),
                )
            }
            if (complete) {
                Icon(
                    Icons.Default.CheckCircle,
                    contentDescription = null,
                    tint = LexturesColors.Primary,
                    modifier = Modifier.size(14.dp).offset(x = 4.dp, y = 4.dp),
                )
            }
        }
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                text = item.title,
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium,
                color = if (locked) textSecondary() else textPrimary(),
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
    }
}

private fun formatPoints(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()
