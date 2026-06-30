package com.lextures.android.features.courses

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowForward
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModuleGroup
import com.lextures.android.core.lms.ModulesProgressSnapshot
import com.lextures.android.core.lms.RequirementRow
import com.lextures.android.core.lms.RequirementsLogic
import com.lextures.android.core.lms.RequirementsSummary

/** Structured lock explanation with progress and deep-link to the next required item (M3.4). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RequirementsSheet(
    targetItem: CourseStructureItem,
    groups: List<ModuleGroup>,
    progress: ModulesProgressSnapshot?,
    onDismiss: () -> Unit,
    onGoToRequired: (String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val summary = RequirementsLogic.buildRequirements(targetItem, groups, progress)
    val titleLabel = moduleRequirementsTitle()
    val doneLabel = moduleRequirementsDoneLabel()
    val listLabel = moduleRequirementsListLabel()
    val goToNextLabel = moduleRequirementsGoToNextLabel()

    LaunchedEffect(progress, targetItem.id) {
        if (progress != null && !ModuleContentLogic.isLocked(progress, targetItem.id)) {
            onDismiss()
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = titleLabel,
                    style = LexturesType.display(18, FontWeight.SemiBold),
                    color = textPrimary(),
                )
                TextButton(onClick = onDismiss) {
                    Text(doneLabel)
                }
            }

            if (summary.totalCount > 0) {
                RequirementsProgressCard(targetItem = targetItem, summary = summary)
            }

            Column(
                modifier = Modifier.semantics { contentDescription = listLabel },
            ) {
                summary.rows.forEachIndexed { index, row ->
                    if (index > 0) HorizontalDivider()
                    RequirementRowView(row)
                }
            }

            summary.nextRequiredItemId?.let { nextId ->
                Button(
                    onClick = {
                        onDismiss()
                        onGoToRequired(nextId)
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.Default.ArrowForward, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text(goToNextLabel)
                    }
                }
            }

            Spacer(modifier = Modifier.size(16.dp))
        }
    }
}

@Composable
private fun RequirementsProgressCard(
    targetItem: CourseStructureItem,
    summary: RequirementsSummary,
) {
    val progressLabel = moduleRequirementsProgressLabel(summary.metCount, summary.totalCount)
    val a11y = moduleRequirementsProgressA11yLabel(summary.metCount, summary.totalCount)

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.12f else 0.08f))
            .padding(14.dp)
            .semantics { contentDescription = a11y },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            text = progressLabel,
            fontSize = 14.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        LinearProgressIndicator(
            progress = { summary.metCount.toFloat() / maxOf(summary.totalCount, 1) },
            modifier = Modifier.fillMaxWidth(),
        )
        Text(text = targetItem.title, fontSize = 12.sp, color = textSecondary())
    }
}

@Composable
private fun RequirementRowView(row: RequirementRow) {
    val status = if (row.met) moduleRequirementsMetLabel() else moduleRequirementsUnmetLabel()
    val a11y = buildString {
        append(row.title)
        row.detail?.takeIf { it.isNotEmpty() }?.let { append(", $it") }
        append(", $status")
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 10.dp)
            .semantics { contentDescription = a11y },
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(
            if (row.met) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
            contentDescription = null,
            tint = if (row.met) LexturesColors.Primary else textSecondary(),
            modifier = Modifier.size(20.dp),
        )
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = row.title, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary())
            row.detail?.takeIf { it.isNotEmpty() }?.let {
                Text(text = it, fontSize = 12.sp, color = textSecondary())
            }
            Text(
                text = status,
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = if (row.met) LexturesColors.Primary else LexturesColors.Coral,
            )
        }
    }
}
