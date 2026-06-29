package com.lextures.android.core.design

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.offline.OutboxStatus

@Composable
fun StalenessChip(label: String, modifier: Modifier = Modifier) {
    Text(
        text = label,
        modifier = modifier
            .background(LexturesColors.Amber.copy(alpha = 0.14f), RoundedCornerShape(999.dp))
            .padding(horizontal = 10.dp, vertical = 5.dp),
        color = LexturesColors.Amber,
        fontSize = 11.sp,
        fontWeight = FontWeight.SemiBold,
    )
}

@Composable
fun OutboxStatusChip(status: OutboxStatus, modifier: Modifier = Modifier) {
    val tint = when (status) {
        OutboxStatus.Queued -> LexturesColors.Amber
        OutboxStatus.Syncing -> accentColor()
        OutboxStatus.Synced -> LexturesColors.BrandTeal
        OutboxStatus.Failed -> LexturesColors.Error
        OutboxStatus.Conflict -> LexturesColors.Coral
    }
    Text(
        text = status.userLabel,
        modifier = modifier
            .background(tint.copy(alpha = 0.12f), RoundedCornerShape(999.dp))
            .padding(horizontal = 8.dp, vertical = 4.dp),
        color = tint,
        fontSize = 11.sp,
        fontWeight = FontWeight.SemiBold,
    )
}

@Composable
fun OfflineBanner(modifier: Modifier = Modifier) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(LexturesColors.Amber.copy(alpha = 0.14f), RoundedCornerShape(12.dp))
            .padding(horizontal = 12.dp, vertical = 10.dp),
    ) {
        Text(
            text = "You're offline — showing saved data",
            color = textPrimary(),
            fontSize = 12.sp,
            fontWeight = FontWeight.Medium,
        )
    }
}

@Composable
fun PendingSyncBadge(count: Int, modifier: Modifier = Modifier) {
    if (count <= 0) return
    Text(
        text = if (count > 99) "99+" else count.toString(),
        modifier = modifier
            .background(LexturesColors.Amber, RoundedCornerShape(999.dp))
            .padding(horizontal = 6.dp, vertical = 2.dp),
        color = Color.White,
        fontSize = 10.sp,
        fontWeight = FontWeight.Bold,
    )
}
