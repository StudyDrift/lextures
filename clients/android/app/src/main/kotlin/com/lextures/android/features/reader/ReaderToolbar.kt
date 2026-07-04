package com.lextures.android.features.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.TextFields
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.core.accessibility.ReadAloudControls
import com.lextures.android.core.lms.ImmersiveReaderCapabilities

@Composable
fun ReaderToolbar(
    text: String,
    accessToken: String?,
    capabilities: ImmersiveReaderCapabilities,
    modifier: Modifier = Modifier,
    courseCode: String? = null,
    ugcTranslation: UgcTranslationTarget? = null,
    onContentReload: (suspend () -> Unit)? = null,
    onOpenPreferences: () -> Unit = {},
    ttsSpeed: Float = 1f,
) {
    LaunchedEffect(accessToken, capabilities.preferencesEnabled) {
        // Parent provides store sync via composition locals when needed.
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            if (capabilities.readAloudEnabled && text.isNotBlank()) {
                ReadAloudControls(text = text, speed = ttsSpeed)
            }
            if (capabilities.preferencesEnabled) {
                TextButton(onClick = onOpenPreferences) {
                    Icon(Icons.Default.TextFields, contentDescription = null)
                    Text("Aa")
                }
            }
        }
        if (capabilities.translationEnabled) {
            when {
                ugcTranslation != null && accessToken != null ->
                    ContentTranslationControls(
                        mode = ContentTranslationMode.Ugc(ugcTranslation, accessToken),
                    )
                courseCode != null && accessToken != null ->
                    ContentTranslationControls(
                        mode = ContentTranslationMode.CourseContent(courseCode, accessToken, onContentReload),
                    )
            }
        }
    }
}

data class UgcTranslationTarget(
    val contentType: String,
    val contentId: String,
    val text: String,
    val targetLang: String,
)

@Composable
fun ReaderToolbarOrLegacy(
    text: String,
    accessToken: String?,
    capabilities: ImmersiveReaderCapabilities,
    modifier: Modifier = Modifier,
    courseCode: String? = null,
    ugcTranslation: UgcTranslationTarget? = null,
    onContentReload: (suspend () -> Unit)? = null,
    onOpenPreferences: () -> Unit = {},
    ttsSpeed: Float = 1f,
) {
    if (capabilities.toolbarEnabled) {
        ReaderToolbar(
            text = text,
            accessToken = accessToken,
            capabilities = capabilities,
            modifier = modifier,
            courseCode = courseCode,
            ugcTranslation = ugcTranslation,
            onContentReload = onContentReload,
            onOpenPreferences = onOpenPreferences,
            ttsSpeed = ttsSpeed,
        )
    } else if (text.isNotBlank()) {
        ReadAloudControls(text = text, speed = ttsSpeed)
    }
}