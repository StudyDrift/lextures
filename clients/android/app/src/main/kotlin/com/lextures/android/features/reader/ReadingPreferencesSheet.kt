package com.lextures.android.features.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.core.lms.ReadingPreferencesPatch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReadingPreferencesSheet(
    visible: Boolean,
    store: ReadingPreferencesStore,
    accessToken: String?,
    onDismiss: () -> Unit,
) {
    if (!visible) return
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    val fontOptions = listOf("default", "open-dyslexic", "atkinson", "system")
    val spacingOptions = listOf("normal", "wide", "wider")

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("Reading preferences")
            Text("Font")
            fontOptions.forEach { face ->
                Button(
                    onClick = {
                        store.updateAsync(
                            scope,
                            ReadingPreferencesPatch(
                                fontFace = face,
                                dyslexiaDisplayEnabled = ReaderLogic.dyslexiaFromFontFace(face),
                            ),
                            accessToken,
                        )
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(face.replaceFirstChar { it.uppercase() }) }
            }
            Text("Letter spacing")
            spacingOptions.forEach { spacing ->
                Button(
                    onClick = {
                        store.updateAsync(scope, ReadingPreferencesPatch(letterSpacing = spacing), accessToken)
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(spacing) }
            }
            Text("Read aloud speed: ${"%.1f".format(store.row.ttsSpeed)}×")
            Slider(
                value = store.row.ttsSpeed.toFloat(),
                onValueChange = { speed ->
                    store.updateAsync(scope, ReadingPreferencesPatch(ttsSpeed = speed.toDouble()), accessToken)
                },
                valueRange = 0.5f..2f,
            )
            Button(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) { Text("Done") }
        }
    }
}