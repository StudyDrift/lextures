package com.lextures.android.features.settings.admin

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary

data class AdminMetricCardModel(
    val id: String,
    val title: String,
    val hint: String?,
    val value: Long?,
    val icon: ImageVector,
    val selected: Boolean,
)

@Composable
fun AdminMetricCardsGrid(
    cards: List<AdminMetricCardModel>,
    loading: Boolean,
    hintLine: String,
    viewListLabel: String,
    hideListLabel: String,
    onSelect: (String) -> Unit,
    formatCount: (Long) -> String,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(hintLine, fontSize = 12.sp, color = textSecondary())
        cards.chunked(2).forEach { row ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                row.forEach { card ->
                    AdminMetricCard(
                        card = card,
                        loading = loading && card.value == null,
                        viewListLabel = viewListLabel,
                        hideListLabel = hideListLabel,
                        formatCount = formatCount,
                        onClick = { onSelect(card.id) },
                        modifier = Modifier.weight(1f),
                    )
                }
                if (row.size == 1) Box(modifier = Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun AdminMetricCard(
    card: AdminMetricCardModel,
    loading: Boolean,
    viewListLabel: String,
    hideListLabel: String,
    formatCount: (Long) -> String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val shape = RoundedCornerShape(18.dp)
    val dark = isDarkTheme()
    Column(
        modifier = modifier
            .clip(shape)
            .background(cardBackground())
            .border(
                width = if (card.selected) 2.dp else 1.dp,
                color = if (card.selected) LexturesColors.BrandTeal.copy(alpha = 0.85f)
                else fieldBorder().copy(alpha = if (dark) 0.9f else 0.45f),
                shape = shape,
            )
            .clickable(onClick = onClick)
            .padding(14.dp)
            .height(148.dp),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                card.title.uppercase(),
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
                modifier = Modifier.weight(1f),
                maxLines = 2,
            )
            Box(
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(LexturesColors.BrandTeal.copy(alpha = if (dark) 0.18f else 0.14f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(card.icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(16.dp))
            }
        }
        if (loading) {
            Box(
                modifier = Modifier
                    .width(56.dp)
                    .height(28.dp)
                    .clip(RoundedCornerShape(6.dp))
                    .background(fieldBorder().copy(alpha = 0.35f)),
            )
        } else {
            Text(card.value?.let(formatCount) ?: "—", style = LexturesType.display(26), color = textPrimary())
        }
        Text(card.hint ?: " ", fontSize = 11.sp, color = textSecondary(), maxLines = 2, minLines = 2)
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                if (card.selected) hideListLabel else viewListLabel,
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
            Icon(
                Icons.Default.KeyboardArrowDown,
                contentDescription = null,
                tint = textSecondary(),
                modifier = Modifier.size(16.dp).rotate(if (card.selected) 180f else 0f),
            )
        }
    }
}
