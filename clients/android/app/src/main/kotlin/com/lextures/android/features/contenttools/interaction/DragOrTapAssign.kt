package com.lextures.android.features.contenttools.interaction

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

/** Shared select-then-place interaction for CT.M7 (visible by default, not a11y-only). */
@Composable
fun DragOrTapAssignBar(
    selectedLabel: String?,
    helperText: String,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(helperText, fontSize = 12.sp, color = textSecondary())
        if (!selectedLabel.isNullOrEmpty()) {
            val selectedText = L.format(R.string.mobile_contentTools_interaction_selectedItem, selectedLabel)
            val a11y = L.format(R.string.mobile_contentTools_interaction_selectedA11y, selectedLabel)
            Text(
                text = selectedText,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                modifier = Modifier.semantics { contentDescription = a11y },
            )
        }
    }
}

@Composable
fun PlacementChip(
    title: String,
    selected: Boolean,
    locked: Boolean,
    correct: Boolean?,
    disabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accent = accentColor()
    val shape = RoundedCornerShape(8.dp)
    val bg = if (selected) accent.copy(alpha = 0.18f) else cardBackground()
    val borderColor = if (selected) accent else fieldBorder()
    val borderWidth = if (selected) 2.dp else 1.dp
    Row(
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = 44.dp)
            .background(bg, shape)
            .border(borderWidth, borderColor, shape)
            .clickable(
                enabled = !disabled && !locked,
                role = Role.Button,
                onClick = onClick,
            )
            .padding(horizontal = 12.dp, vertical = 10.dp)
            .semantics { this.selected = selected },
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (correct != null) {
            Icon(
                imageVector = if (correct) Icons.Default.CheckCircle else Icons.Default.Cancel,
                contentDescription = null,
                tint = if (correct) LexturesColors.Primary else LexturesColors.Coral,
            )
        }
        Text(
            text = title,
            fontSize = 14.sp,
            color = textPrimary(),
            modifier = Modifier.weight(1f),
        )
        if (locked) {
            Icon(
                imageVector = Icons.Default.Lock,
                contentDescription = null,
                tint = textSecondary(),
            )
        }
    }
}
