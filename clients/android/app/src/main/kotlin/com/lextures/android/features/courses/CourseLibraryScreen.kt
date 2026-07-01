package com.lextures.android.features.courses

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LibraryResourceLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.library.libraryCourseEmptyMessage
import com.lextures.android.features.library.libraryCourseEmptyTitle
import com.lextures.android.features.library.libraryTypeGeneric

@Composable
fun CourseLibraryScreen(
    course: CourseSummary,
    items: List<CourseStructureItem>,
    onSelectItem: (CourseStructureItem) -> Unit,
    modifier: Modifier = Modifier,
) {
    val libraryItems = LibraryResourceLogic.libraryItems(items)
    if (libraryItems.isEmpty()) {
        LmsEmptyState(
            icon = Icons.AutoMirrored.Filled.MenuBook,
            title = libraryCourseEmptyTitle(),
            message = libraryCourseEmptyMessage(),
            modifier = modifier,
        )
    } else {
        Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
            libraryItems.forEach { item ->
                LmsCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { onSelectItem(item) },
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Icon(
                            Icons.AutoMirrored.Filled.MenuBook,
                            contentDescription = null,
                            tint = LexturesColors.PrimaryMuted,
                        )
                        Column(Modifier.weight(1f)) {
                            Text(item.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(libraryTypeGeneric(), fontSize = 12.sp, color = textSecondary())
                        }
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = textSecondary(),
                        )
                    }
                }
            }
        }
    }
}