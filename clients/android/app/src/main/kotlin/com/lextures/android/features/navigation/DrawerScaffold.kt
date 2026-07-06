package com.lextures.android.features.navigation

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.heightIn
import com.lextures.android.core.design.UIMode
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.navigation.DrawerState
import kotlin.math.min
import kotlin.math.roundToInt

/**
 * Reusable left-drawer container implementing the web-parity, two-level navigation:
 * a leading-edge swipe reveals the drawer; when a course is active the first swipe opens
 * the course menu and a second edge swipe escalates to the global menu. Drag left on an
 * open panel, or tap the scrim, to close. Selecting a drawer row pushes main content
 * while the panel slides closed (X-style).
 */
@Composable
fun DrawerScaffold(
    state: DrawerState,
    courseAvailable: Boolean,
    onStateChange: (DrawerState) -> Unit,
    onDrawerProgress: (Float) -> Unit = {},
    globalPanel: @Composable () -> Unit,
    coursePanel: @Composable () -> Unit,
    content: @Composable () -> Unit,
) {
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val density = LocalDensity.current
        val panelW = min(maxWidth.value * 0.82f, 360f).dp
        val panelPx = with(density) { panelW.toPx() }
        val openThreshold = panelPx * 0.33f
        val escalateThreshold = panelPx * 0.25f

        val drawerProgress by animateFloatAsState(
            targetValue = if (state != DrawerState.None) 1f else 0f,
            animationSpec = tween(durationMillis = 350, easing = FastOutSlowInEasing),
            label = "drawerProgress",
        )
        SideEffect { onDrawerProgress(drawerProgress) }

        Box(Modifier.fillMaxSize()) {
            Box(
                Modifier
                    .fillMaxSize()
                    .offset { IntOffset((panelPx * drawerProgress).roundToInt(), 0) },
            ) {
                content()
            }

            if (drawerProgress > 0.001f) {
                Box(
                    Modifier
                        .fillMaxSize()
                        .background(Color.Black.copy(alpha = 0.45f * drawerProgress))
                        .clickable(
                            interactionSource = remember { MutableInteractionSource() },
                            indication = null,
                        ) { onStateChange(DrawerState.None) },
                )

                // Panel with drag-to-close.
                Box(
                    Modifier
                        .fillMaxHeight()
                        .width(panelW)
                        .offset {
                            IntOffset(
                                (-panelPx * (1f - drawerProgress)).roundToInt(),
                                0,
                            )
                        }
                        .pointerInput(state) {
                            var acc = 0f
                            detectHorizontalDragGestures(
                                onDragStart = { acc = 0f },
                                onHorizontalDrag = { change, delta -> acc += delta; change.consume() },
                                onDragEnd = {
                                    if (acc < -openThreshold) onStateChange(DrawerState.None)
                                    else if (state == DrawerState.Course && acc > escalateThreshold) {
                                        onStateChange(DrawerState.Global)
                                    }
                                },
                            )
                        },
                ) {
                    when (state) {
                        DrawerState.Course -> coursePanel()
                        else -> globalPanel()
                    }
                }
            }

            // Leading-edge catcher: opens (closed) or escalates (course menu open).
            if (state == DrawerState.None || state == DrawerState.Course) {
                Box(
                    Modifier
                        .fillMaxHeight()
                        .width(24.dp)
                        .pointerInput(state, courseAvailable) {
                            var acc = 0f
                            detectHorizontalDragGestures(
                                onDragStart = { acc = 0f },
                                onHorizontalDrag = { change, delta -> acc += delta; change.consume() },
                                onDragEnd = {
                                    if (state == DrawerState.Course) {
                                        if (acc > escalateThreshold) onStateChange(DrawerState.Global)
                                    } else if (acc > openThreshold) {
                                        onStateChange(if (courseAvailable) DrawerState.Course else DrawerState.Global)
                                    }
                                },
                            )
                        },
                )
            }
        }
    }
}

// MARK: - Shared drawer chrome

@Composable
fun DrawerGroupHeader(title: String) {
    Text(
        text = title.uppercase(),
        color = textSecondary(),
        fontSize = 11.sp,
        fontWeight = FontWeight.SemiBold,
        letterSpacing = 0.6.sp,
        modifier = Modifier
            .padding(start = 12.dp, end = 12.dp, top = 14.dp, bottom = 4.dp),
    )
}

@Composable
fun DrawerRow(
    label: String,
    icon: ImageVector,
    selected: Boolean,
    onClick: () -> Unit,
    badge: Int = 0,
    uiMode: UIMode = UIMode.Standard,
) {
    val dark = isDarkTheme()
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 4.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(if (selected) LexturesColors.BrandTeal.copy(alpha = if (dark) 0.22f else 0.16f) else Color.Transparent)
            .clickable(onClick = onClick)
            .semantics { contentDescription = label }
            .padding(horizontal = 12.dp, vertical = uiMode.drawerRowVerticalPadding)
            .heightIn(min = uiMode.minimumTapTarget),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(
            icon,
            contentDescription = null,
            tint = if (selected) accentColor() else textSecondary(),
            modifier = Modifier.size(uiMode.drawerIconDp),
        )
        Text(
            label,
            color = textPrimary(),
            fontSize = uiMode.baseBodySp,
            fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
            modifier = Modifier.weight(1f),
        )
        if (badge > 0) {
            Text(
                text = if (badge > 99) "99+" else "$badge",
                fontSize = 11.sp,
                fontWeight = FontWeight.Bold,
                color = Color.White,
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(LexturesColors.Coral)
                    .padding(horizontal = 6.dp, vertical = 2.dp),
            )
        }
    }
}