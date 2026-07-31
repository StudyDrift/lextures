package com.lextures.android.features.contenttools.interaction

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolPack3Logic

data class PassageAnnotationMark(
    val id: String,
    val start: Int,
    val end: Int,
    val tagLabel: String,
    val tagColor: String = "",
)

/**
 * Purpose-built passage selection for highlight_annotate — sentence/paragraph tap
 * by default; avoids the OS copy/paste callout (CT.M7 FR-12 / FR-13).
 */
@Composable
@Suppress("UNUSED_PARAMETER")
fun PassageSelectionView(
    passage: String,
    units: List<ContentToolPack3Logic.PassageUnit>,
    annotations: List<PassageAnnotationMark>,
    selectedUnitIndex: Int?,
    readOnly: Boolean,
    onSelectUnit: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accent = accentColor()
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            text = L.text(R.string.mobile_contentTools_tools_highlight_annotate_sentenceTapHint),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            units.forEach { unit ->
                val isSelected = selectedUnitIndex == unit.index
                val covering = annotations.filter { it.start < unit.end && it.end > unit.start }
                val shape = RoundedCornerShape(6.dp)
                val bg = when {
                    isSelected -> accent.copy(alpha = 0.2f)
                    covering.isNotEmpty() -> LexturesColors.Amber.copy(alpha = 0.18f)
                    else -> Color.Transparent
                }
                val borderColor = if (isSelected) accent else Color.Transparent
                val a11y = unitAccessibility(unit, covering, isSelected, readOnly)
                Text(
                    text = unit.text,
                    fontSize = 16.sp,
                    color = textPrimary(),
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 44.dp)
                        .background(bg, shape)
                        .border(2.dp, borderColor, shape)
                        .clickable(
                            enabled = !readOnly,
                            role = Role.Button,
                            onClick = { onSelectUnit(unit.index) },
                        )
                        .padding(10.dp)
                        .semantics {
                            selected = isSelected
                            contentDescription = a11y
                        },
                )
            }
        }
    }
}

@Composable
private fun unitAccessibility(
    unit: ContentToolPack3Logic.PassageUnit,
    covering: List<PassageAnnotationMark>,
    selected: Boolean,
    readOnly: Boolean,
): String {
    val parts = mutableListOf<String>()
    if (selected) {
        parts += L.text(R.string.mobile_contentTools_interaction_selected)
    }
    parts += unit.text
    covering.firstOrNull()?.let { first ->
        parts += L.format(R.string.mobile_contentTools_tools_highlight_annotate_taggedAs, first.tagLabel)
    }
    if (!readOnly) {
        parts += L.text(R.string.mobile_contentTools_tools_highlight_annotate_doubleTapToSelect)
    }
    return parts.joinToString(". ")
}
