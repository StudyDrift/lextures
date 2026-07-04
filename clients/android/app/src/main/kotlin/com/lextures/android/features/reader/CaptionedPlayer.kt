package com.lextures.android.features.reader

import androidx.annotation.OptIn
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.ui.window.Dialog
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.CaptionRecord
import com.lextures.android.core.lms.LmsApi
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive

@OptIn(UnstableApi::class)
@Composable
fun ContentVideoPlayer(
    url: String,
    accessToken: String?,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val resolvedUrl = remember(url) {
        if (url.startsWith("http")) url else AppConfiguration.apiUrl(url).toString()
    }
    val player = remember(resolvedUrl, accessToken) {
        val dataSourceFactory = DefaultHttpDataSource.Factory().apply {
            if (accessToken != null) {
                setDefaultRequestProperties(mapOf("Authorization" to "Bearer $accessToken"))
            }
        }
        val mediaSource = ProgressiveMediaSource.Factory(dataSourceFactory)
            .createMediaSource(MediaItem.fromUri(resolvedUrl))
        ExoPlayer.Builder(context).build().apply {
            setMediaSource(mediaSource)
            prepare()
        }
    }
    DisposableEffect(player) {
        onDispose { player.release() }
    }
    AndroidView(
        factory = { ctx ->
            PlayerView(ctx).apply {
                this.player = player
                useController = true
            }
        },
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(16f / 9f)
            .clip(RoundedCornerShape(12.dp))
            .semantics { contentDescription = "Embedded video" },
    )
}

@OptIn(UnstableApi::class)
@Composable
fun CaptionedPlayer(
    url: String,
    accessToken: String?,
    storageObjectId: String? = null,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val resolvedUrl = remember(url) {
        if (url.startsWith("http")) url else AppConfiguration.apiUrl(url).toString()
    }
    var captions by remember { mutableStateOf<List<CaptionRecord>>(emptyList()) }
    var cues by remember { mutableStateOf<List<ReaderLogic.VttCue>>(emptyList()) }
    var selectedCaptionId by remember { mutableStateOf<String?>(null) }
    var captionsEnabled by remember { mutableStateOf(false) }
    var currentCue by remember { mutableStateOf<ReaderLogic.VttCue?>(null) }
    var showTranscript by remember { mutableStateOf(false) }
    var playbackPosition by remember { mutableDoubleStateOf(0.0) }

    val player = remember(resolvedUrl, accessToken) {
        val dataSourceFactory = DefaultHttpDataSource.Factory().apply {
            if (accessToken != null) {
                setDefaultRequestProperties(mapOf("Authorization" to "Bearer $accessToken"))
            }
        }
        val mediaSource = ProgressiveMediaSource.Factory(dataSourceFactory)
            .createMediaSource(MediaItem.fromUri(resolvedUrl))
        ExoPlayer.Builder(context).build().apply {
            setMediaSource(mediaSource)
            prepare()
        }
    }

    DisposableEffect(player) {
        onDispose { player.release() }
    }

    LaunchedEffect(player) {
        while (isActive) {
            playbackPosition = player.currentPosition / 1000.0
            currentCue = ReaderLogic.activeCue(playbackPosition, cues)
            delay(250)
        }
    }

    LaunchedEffect(accessToken, resolvedUrl) {
        val token = accessToken ?: return@LaunchedEffect
        val objectId = storageObjectId ?: ReaderLogic.storageObjectId(resolvedUrl) ?: return@LaunchedEffect
        val records = runCatching { LmsApi.fetchCaptions(objectId, token) }.getOrDefault(emptyList())
        captions = ReaderLogic.readyCaptions(records)
        selectedCaptionId = captions.firstOrNull()?.id
        captions.firstOrNull()?.id?.let { captionId ->
            val raw = runCatching { LmsApi.fetchCaptionVtt(objectId, captionId, token) }.getOrDefault("")
            cues = ReaderLogic.parseVtt(raw)
        }
    }

    LaunchedEffect(selectedCaptionId, accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        val objectId = storageObjectId ?: ReaderLogic.storageObjectId(resolvedUrl) ?: return@LaunchedEffect
        val captionId = selectedCaptionId ?: return@LaunchedEffect
        val raw = runCatching { LmsApi.fetchCaptionVtt(objectId, captionId, token) }.getOrDefault("")
        cues = ReaderLogic.parseVtt(raw)
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 9f)
                .clip(RoundedCornerShape(12.dp)),
        ) {
            AndroidView(
                factory = { ctx ->
                    PlayerView(ctx).apply {
                        this.player = player
                        useController = true
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .semantics { contentDescription = "Course video" },
            )
            if (captionsEnabled && currentCue != null) {
                Text(
                    text = currentCue!!.text,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = Color.White,
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(horizontal = 12.dp, vertical = 10.dp)
                        .background(Color.Black.copy(alpha = 0.72f), RoundedCornerShape(8.dp))
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }

        if (captions.isNotEmpty()) {
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                TextButton(onClick = { captionsEnabled = !captionsEnabled }) {
                    Text(if (captionsEnabled) "Captions on" else "Captions off")
                }
                if (captions.size > 1) {
                    captions.forEach { caption ->
                        val selected = caption.id == selectedCaptionId
                        TextButton(
                            onClick = { selectedCaptionId = caption.id },
                            enabled = !selected,
                        ) {
                            Text(ReaderLogic.localeLabel(caption.lang))
                        }
                    }
                }
                TextButton(onClick = { showTranscript = true }) {
                    Text("Transcript")
                }
            }
        }
    }

    if (showTranscript) {
        Dialog(onDismissRequest = { showTranscript = false }) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(cardBackground())
                    .padding(horizontal = 16.dp, vertical = 8.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text("Transcript", fontWeight = FontWeight.SemiBold, color = textPrimary())
                cues.forEach { cue ->
                    Text(text = cue.text, fontSize = 14.sp, color = textPrimary())
                }
                Button(
                    onClick = { showTranscript = false },
                    modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
                ) {
                    Text("Done")
                }
            }
        }
    }
}