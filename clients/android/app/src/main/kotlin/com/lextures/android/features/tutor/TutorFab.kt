package com.lextures.android.features.tutor

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.TutorLogic

@Composable
fun TutorFab(
    course: CourseSummary,
    item: CourseStructureItem? = null,
    onOpen: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (!TutorLogic.shouldShowFab(course)) return
    FloatingActionButton(
        onClick = onOpen,
        modifier = modifier.padding(20.dp),
        containerColor = LexturesColors.Primary,
        contentColor = androidx.compose.ui.graphics.Color.White,
    ) {
        Icon(
            Icons.Default.AutoAwesome,
            contentDescription = stringResource(R.string.mobile_tutor_open),
        )
    }
}