package com.lextures.android.features.assignments

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.RubricCriterion
import com.lextures.android.core.lms.RubricDefinition
import com.lextures.android.features.home.LmsCard

/** Read-only rubric sheet: criteria, descriptions, and rating bands. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AssignmentRubricViewerSheet(
    rubric: RubricDefinition,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val totalMax = rubric.criteria.sumOf { criterionMaxPoints(it) }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = 28.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(R.string.mobile_assignment_rubricTitle),
                style = LexturesType.display(20, FontWeight.Bold),
                color = textPrimary(),
            )
            rubric.title?.takeIf { it.isNotBlank() }?.let { title ->
                Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
            }
            Text(
                text = summaryLine(rubric.criteria.size, totalMax),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            rubric.criteria.forEachIndexed { index, criterion ->
                CriterionCard(criterion = criterion, index = index)
            }
        }
    }
}

@Composable
private fun CriterionCard(criterion: RubricCriterion, index: Int) {
    val maxPts = criterionMaxPoints(criterion)
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Top,
            ) {
                Text(
                    text = "${index + 1}. ${criterion.title}",
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 14.sp,
                    color = textPrimary(),
                    modifier = Modifier.weight(1f),
                )
                if (maxPts > 0) {
                    Text(
                        text = L.format(R.string.mobile_assignment_rubricPoints, formatPoints(maxPts)),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textSecondary(),
                    )
                }
            }
            criterion.description?.takeIf { it.isNotBlank() }?.let {
                Text(it, fontSize = 12.sp, color = textSecondary())
            }
            if (criterion.levels.isNotEmpty()) {
                HorizontalDivider()
                criterion.levels.forEach { level ->
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.Top,
                    ) {
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(level.label, fontWeight = FontWeight.Medium, fontSize = 13.sp, color = textPrimary())
                            level.description?.takeIf { it.isNotBlank() }?.let { note ->
                                Text(note, fontSize = 12.sp, color = textSecondary())
                            }
                        }
                        Text(
                            text = L.format(R.string.mobile_assignment_rubricPoints, formatPoints(level.points)),
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textSecondary(),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun summaryLine(count: Int, totalMax: Double): String {
    val criteriaPart = if (count == 1) {
        L.format(R.string.mobile_assignment_rubricCriterionCount, count)
    } else {
        L.format(R.string.mobile_assignment_rubricCriteriaCount, count)
    }
    return if (totalMax > 0) {
        "$criteriaPart · ${L.format(R.string.mobile_assignment_rubricPointsTotal, formatPoints(totalMax))}"
    } else {
        criteriaPart
    }
}

private fun criterionMaxPoints(criterion: RubricCriterion): Double =
    criterion.levels.maxOfOrNull { it.points } ?: 0.0

private fun formatPoints(value: Double): String {
    return if (value == value.toLong().toDouble()) {
        value.toLong().toString()
    } else {
        value.toString().trimEnd('0').trimEnd('.')
    }
}
