package com.lextures.android.features.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.coverBrush

/**
 * Course banner image with auth for `/course-files/.../content` URLs (parity with web).
 * Falls back to the course cover gradient when no image is set or loading fails.
 */
@Composable
fun CourseHeroImage(
    url: String?,
    fallbackKey: String,
    accessToken: String?,
    modifier: Modifier = Modifier,
    height: Dp? = 84.dp,
) {
    val context = LocalContext.current
    val trimmed = url?.trim().orEmpty()
    val resolved = when {
        trimmed.startsWith("/") -> AppConfiguration.apiUrl(trimmed).toString()
        trimmed.startsWith("http://") || trimmed.startsWith("https://") -> trimmed
        else -> null
    }

    Box(
        modifier = modifier
            .fillMaxWidth()
            .then(if (height != null) Modifier.height(height) else Modifier.fillMaxSize()),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(coverBrush(fallbackKey)),
        )
        if (resolved != null) {
            AsyncImage(
                model = ImageRequest.Builder(context)
                    .data(resolved)
                    .apply {
                        if (!accessToken.isNullOrBlank()) {
                            setHeader("Authorization", "Bearer $accessToken")
                        }
                    }
                    .crossfade(true)
                    .build(),
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}
