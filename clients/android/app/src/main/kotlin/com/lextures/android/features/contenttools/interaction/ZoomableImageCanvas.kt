package com.lextures.android.features.contenttools.interaction

import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Image
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textSecondary
import com.lextures.android.features.contenttools.LocalContentToolsPage

private const val MIN_ZOOM = 1f
private const val MAX_ZOOM = 4f

/**
 * Fit-to-width image canvas with pinch-zoom / pan for diagram_hotspot (CT.M7).
 * [overlay] receives the unscaled canvas size in pixels for region hit targets.
 */
@Composable
fun ZoomableImageCanvas(
    urlString: String,
    alt: String,
    naturalWidth: Double,
    naturalHeight: Double,
    modifier: Modifier = Modifier,
    overlay: @Composable (Size) -> Unit = {},
) {
    var zoom by remember { mutableFloatStateOf(1f) }
    var pan by remember { mutableStateOf(Offset.Zero) }
    val transformState = rememberTransformableState { zoomChange, panChange, _ ->
        val nextZoom = (zoom * zoomChange).coerceIn(MIN_ZOOM, MAX_ZOOM)
        zoom = nextZoom
        pan = if (nextZoom > MIN_ZOOM) pan + panChange else Offset.Zero
    }

    val ratio = if (naturalWidth > 0 && naturalHeight > 0) {
        (naturalWidth / naturalHeight).toFloat()
    } else {
        4f / 3f
    }
    val accessToken = LocalContentToolsPage.current?.accessToken

    BoxWithConstraints(
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(ratio)
            .clipToBounds(),
    ) {
        val sizePx = Size(constraints.maxWidth.toFloat(), constraints.maxHeight.toFloat())

        Box(
            modifier = Modifier
                .fillMaxSize()
                .graphicsLayer {
                    scaleX = zoom
                    scaleY = zoom
                    translationX = pan.x
                    translationY = pan.y
                }
                .transformable(state = transformState),
        ) {
            AuthorizedToolImage(
                url = urlString,
                alt = alt,
                accessToken = accessToken,
                modifier = Modifier.fillMaxSize(),
            )
            overlay(sizePx)
        }
    }
}

/**
 * Minimal authenticated AsyncImage (parity with NotebookContentView's private
 * AuthorizedNotebookImage — duplicated here because that helper is private).
 */
@Composable
private fun AuthorizedToolImage(
    url: String,
    alt: String,
    accessToken: String?,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val resolved = when {
        url.startsWith("/") -> AppConfiguration.apiUrl(url).toString()
        url.startsWith("http://") || url.startsWith("https://") -> url
        else -> null
    }
    if (resolved == null) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = modifier,
        ) {
            Icon(
                Icons.Default.Image,
                contentDescription = null,
                tint = textSecondary(),
                modifier = Modifier.size(16.dp),
            )
            Text(text = alt.ifBlank { "Image" }, fontSize = 12.sp, color = textSecondary())
        }
        return
    }
    AsyncImage(
        model = ImageRequest.Builder(context)
            .data(resolved)
            .apply { if (!accessToken.isNullOrBlank()) setHeader("Authorization", "Bearer $accessToken") }
            .build(),
        contentDescription = alt.ifBlank { "Image" },
        contentScale = ContentScale.Fit,
        modifier = modifier.fillMaxWidth(),
    )
}
