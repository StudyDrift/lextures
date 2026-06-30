package com.lextures.android.features.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.ProfileAvatar
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary

/** Floating card: generous radius, soft shadow, optional left accent stripe. */
@Composable
fun LmsCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    accent: Color? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = RoundedCornerShape(18.dp)
    val dark = isDarkTheme()
    var base = modifier
        .fillMaxWidth()
        .shadow(
            elevation = if (dark) 0.dp else 3.dp,
            shape = shape,
            clip = false,
            ambientColor = Color(0xFF3A2E18).copy(alpha = 0.25f),
            spotColor = Color(0xFF3A2E18).copy(alpha = 0.25f),
        )
        .clip(shape)
        .background(cardBackground())
        .border(1.dp, fieldBorder().copy(alpha = if (dark) 0.9f else 0.45f), shape)
    if (onClick != null) {
        base = base.clickable(onClick = onClick)
    }
    Row(modifier = base.height(IntrinsicSize.Min)) {
        if (accent != null) {
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .fillMaxHeight()
                    .background(accent),
            )
        }
        Column(
            modifier = Modifier
                .weight(1f)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
            content = content,
        )
    }
}

/** Serif section header — editorial, like a textbook chapter heading. */
@Composable
fun LmsSectionHeader(title: String, icon: ImageVector? = null, modifier: Modifier = Modifier) {
    Row(
        modifier = modifier.padding(top = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (icon != null) {
            Box(
                modifier = Modifier
                    .size(26.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.18f else 0.16f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(15.dp))
            }
        }
        Text(
            text = title,
            style = LexturesType.display(19),
            color = textPrimary(),
        )
    }
}

@Composable
fun LmsErrorBanner(message: String, modifier: Modifier = Modifier) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(LexturesColors.Error.copy(alpha = 0.09f))
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(
            Icons.Default.Warning,
            contentDescription = null,
            tint = LexturesColors.Error,
            modifier = Modifier.size(18.dp),
        )
        Text(text = message, fontSize = 14.sp, color = LexturesColors.Error)
    }
}

@Composable
fun LmsEmptyState(
    icon: ImageVector,
    title: String,
    message: String,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 36.dp, horizontal = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Box(
            modifier = Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.16f else 0.14f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = null, tint = accentColor(), modifier = Modifier.size(30.dp))
        }
        Text(text = title, style = LexturesType.display(18), color = textPrimary())
        Text(text = message, fontSize = 14.sp, color = textSecondary(), textAlign = TextAlign.Center)
    }
}

/** Rounded gradient tile used as a course "cover" thumbnail. */
@Composable
fun LmsCoverTile(
    key: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    size: Int = 48,
) {
    Box(
        modifier = modifier
            .size(size.dp)
            .clip(RoundedCornerShape((size * 0.28).dp))
            .background(coverBrush(key)),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            icon,
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size((size * 0.42).dp),
        )
    }
}

@Composable
fun LmsChipRow(
    options: List<Pair<String, String>>, // id to label
    selectedId: String,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        options.forEach { (id, label) ->
            val selected = id == selectedId
            Text(
                text = label,
                fontSize = 13.sp,
                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                color = if (selected) Color.White else textSecondary(),
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(
                        if (selected) LexturesColors.Primary else cardBackground(),
                    )
                    .border(
                        1.dp,
                        if (selected) LexturesColors.Primary else fieldBorder(),
                        RoundedCornerShape(50),
                    )
                    .clickable { onSelect(id) }
                    .padding(horizontal = 14.dp, vertical = 7.dp),
            )
        }
    }
}

/**
 * Segmented chip row — solid deep teal when selected, white card with hairline
 * border otherwise. Same look as iOS `LMSSegmentedChips`.
 */
@Composable
fun LmsSegmentedChips(
    options: List<Pair<String, String>>, // id to label
    selectedId: String,
    onSelect: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val dark = isDarkTheme()
    Row(
        modifier = modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(vertical = 2.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        options.forEach { (id, label) ->
            val selected = id == selectedId
            Text(
                text = label,
                fontSize = 14.sp,
                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                color = when {
                    selected && dark -> LexturesColors.PrimaryDeep
                    selected -> Color.White
                    else -> textSecondary()
                },
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(if (selected) accentColor() else cardBackground())
                    .border(
                        1.dp,
                        if (selected) Color.Transparent else fieldBorder(),
                        RoundedCornerShape(50),
                    )
                    .clickable { onSelect(id) }
                    .padding(horizontal = 15.dp, vertical = 8.dp),
            )
        }
    }
}

/** Card-shaped placeholder shown while a list loads. */
@Composable
fun LmsSkeletonCard(modifier: Modifier = Modifier) {
    LmsCard(modifier = modifier) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(fieldBorder().copy(alpha = 0.6f)),
            )
            Column(verticalArrangement = Arrangement.spacedBy(7.dp)) {
                Box(
                    modifier = Modifier
                        .width(180.dp)
                        .height(11.dp)
                        .clip(RoundedCornerShape(4.dp))
                        .background(fieldBorder().copy(alpha = 0.7f)),
                )
                Box(
                    modifier = Modifier
                        .width(120.dp)
                        .height(11.dp)
                        .clip(RoundedCornerShape(4.dp))
                        .background(fieldBorder().copy(alpha = 0.45f)),
                )
            }
        }
    }
}

/** Stack of skeleton cards for list screens. */
@Composable
fun LmsSkeletonList(count: Int = 4, modifier: Modifier = Modifier) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        repeat(count) { LmsSkeletonCard() }
    }
}

/** Small circular progress ring (e.g. overall grade) — echoes the health-app ring. */
@Composable
fun LmsProgressRing(
    progress: Float, // 0f..1f
    modifier: Modifier = Modifier,
    size: Int = 38,
    tint: Color? = null,
) {
    val ringColor = tint ?: accentColor()
    val track = fieldBorder().copy(alpha = 0.8f)
    Box(modifier = modifier.size(size.dp), contentAlignment = Alignment.Center) {
        androidx.compose.foundation.Canvas(modifier = Modifier.size(size.dp)) {
            val stroke = androidx.compose.ui.graphics.drawscope.Stroke(
                width = 4.dp.toPx(),
                cap = androidx.compose.ui.graphics.StrokeCap.Round,
            )
            val inset = 2.dp.toPx()
            val arcSize = androidx.compose.ui.geometry.Size(
                this.size.width - inset * 2,
                this.size.height - inset * 2,
            )
            drawArc(
                color = track,
                startAngle = 0f,
                sweepAngle = 360f,
                useCenter = false,
                topLeft = androidx.compose.ui.geometry.Offset(inset, inset),
                size = arcSize,
                style = stroke,
            )
            drawArc(
                color = ringColor,
                startAngle = -90f,
                sweepAngle = 360f * progress.coerceIn(0.02f, 1f),
                useCenter = false,
                topLeft = androidx.compose.ui.geometry.Offset(inset, inset),
                size = arcSize,
                style = stroke,
            )
        }
        Text(
            text = "${(progress * 100).toInt()}",
            fontSize = (size * 0.3).sp,
            fontWeight = FontWeight.Bold,
            color = textPrimary(),
        )
    }
}

/** Initials avatar chip on the hero gradient; tapping jumps to the Profile tab. */
@Composable
fun LmsAvatarChip(
    initials: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    size: Int = 34,
    avatarUrl: String? = null,
) {
    Box(
        modifier = modifier
            .size(size.dp)
            .clip(CircleShape)
            .border(1.dp, Color.White.copy(alpha = 0.35f), CircleShape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (avatarUrl.isNullOrBlank()) {
            Box(
                modifier = Modifier
                    .matchParentSize()
                    .clip(CircleShape)
                    .background(Color.White.copy(alpha = 0.16f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = initials,
                    fontSize = (size * 0.36).sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                )
            }
        } else {
            ProfileAvatar(
                avatarUrl = avatarUrl,
                initials = initials,
                size = size.dp,
                initialsBackground = Color.White.copy(alpha = 0.16f),
                initialsForeground = Color.White,
            )
        }
    }
}
