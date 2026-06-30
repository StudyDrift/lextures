package com.lextures.android.core.design

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.SubcomposeAsyncImage

/** Circular profile image with initials fallback (http(s) or data:image URLs). */
@Composable
fun ProfileAvatar(
    avatarUrl: String?,
    initials: String,
    modifier: Modifier = Modifier,
    size: Dp = 84.dp,
    initialsBackground: Color = LexturesColors.BrandTeal.copy(alpha = 0.16f),
    initialsForeground: Color = accentColor(),
) {
    val resolvedUrl = avatarUrl?.trim().orEmpty().takeIf { it.isNotEmpty() }
    Box(
        modifier = modifier
            .size(size)
            .clip(CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (resolvedUrl != null) {
            SubcomposeAsyncImage(
                model = resolvedUrl,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape),
                loading = {
                    CircularProgressIndicator(
                        modifier = Modifier.size(size * 0.35f),
                        strokeWidth = 2.dp,
                        color = initialsForeground,
                    )
                },
                error = {
                    InitialsCircle(
                        initials = initials,
                        size = size,
                        background = initialsBackground,
                        foreground = initialsForeground,
                    )
                },
            )
        } else {
            InitialsCircle(
                initials = initials,
                size = size,
                background = initialsBackground,
                foreground = initialsForeground,
            )
        }
    }
}

@Composable
private fun InitialsCircle(
    initials: String,
    size: Dp,
    background: Color,
    foreground: Color,
) {
    Box(
        modifier = Modifier
            .size(size)
            .clip(CircleShape)
            .background(background),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = initials,
            fontSize = (size.value * 0.33).sp,
            fontWeight = FontWeight.Bold,
            color = foreground,
        )
    }
}
