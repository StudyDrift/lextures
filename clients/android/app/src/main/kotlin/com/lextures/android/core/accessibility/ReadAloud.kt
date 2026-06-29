package com.lextures.android.core.accessibility

import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.VolumeUp
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.lextures.android.core.design.textSecondary
import java.util.Locale
import java.util.UUID

private enum class ReadAloudStatus { Idle, Playing, Paused }

@Composable
fun ReadAloudControls(
    text: String,
    modifier: Modifier = Modifier,
    speed: Float = 1f,
) {
    val context = LocalContext.current
    var status by remember { mutableStateOf(ReadAloudStatus.Idle) }
    var sentenceIndex by remember { mutableIntStateOf(0) }
    val sentences = remember(text) {
        AccessibilitySupport.chunkSentences(
            AccessibilitySupport.plainTextFromMarkdown(text),
        )
    }

    val tts = remember {
        var engine: TextToSpeech? = null
        engine = TextToSpeech(context) { initialized ->
            if (initialized == TextToSpeech.SUCCESS) {
                engine?.language = Locale.US
            }
        }
        engine
    }

    DisposableEffect(tts) {
        val listener = object : UtteranceProgressListener() {
            override fun onStart(utteranceId: String?) = Unit

            override fun onDone(utteranceId: String?) {
                if (status != ReadAloudStatus.Playing) return
                sentenceIndex += 1
                if (sentenceIndex < sentences.size) {
                    speakSentence(tts, sentences[sentenceIndex], speed)
                } else {
                    status = ReadAloudStatus.Idle
                    sentenceIndex = 0
                }
            }

            @Deprecated("Deprecated in Java")
            override fun onError(utteranceId: String?) {
                status = ReadAloudStatus.Idle
            }
        }
        tts.setOnUtteranceProgressListener(listener)
        onDispose {
            tts.stop()
            tts.shutdown()
        }
    }

    if (sentences.isEmpty()) return

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        val playing = status == ReadAloudStatus.Playing
        Button(
            onClick = {
                when (status) {
                    ReadAloudStatus.Idle, ReadAloudStatus.Paused -> {
                        if (status == ReadAloudStatus.Idle && sentenceIndex == 0) {
                            speakSentence(tts, sentences[sentenceIndex], speed)
                        } else {
                            tts.stop()
                            speakSentence(tts, sentences[sentenceIndex], speed)
                        }
                        status = ReadAloudStatus.Playing
                    }
                    ReadAloudStatus.Playing -> {
                        tts.stop()
                        status = ReadAloudStatus.Paused
                    }
                }
            },
            modifier = Modifier
                .sizeIn(
                    minWidth = AccessibilitySupport.MINIMUM_TAP_TARGET_DP.dp,
                    minHeight = AccessibilitySupport.MINIMUM_TAP_TARGET_DP.dp,
                )
                .semantics {
                    contentDescription = if (playing) "Pause read aloud" else "Read aloud"
                },
        ) {
            Icon(
                imageVector = if (playing) Icons.Default.Pause else Icons.Default.VolumeUp,
                contentDescription = null,
            )
            Text(if (playing) "Pause" else "Read aloud")
        }

        Text(
            text = "Sentence ${sentenceIndex + 1} of ${sentences.size}",
            style = MaterialTheme.typography.labelSmall,
            color = textSecondary(),
        )

        if (status != ReadAloudStatus.Idle) {
            TextButton(
                onClick = {
                    tts.stop()
                    sentenceIndex = 0
                    speakSentence(tts, sentences[0], speed)
                    status = ReadAloudStatus.Playing
                },
                modifier = Modifier.sizeIn(
                    minWidth = AccessibilitySupport.MINIMUM_TAP_TARGET_DP.dp,
                    minHeight = AccessibilitySupport.MINIMUM_TAP_TARGET_DP.dp,
                ),
            ) {
                Text("Restart")
            }
        }
    }
}

private fun speakSentence(tts: TextToSpeech, sentence: String, speed: Float) {
    tts.setSpeechRate(speed.coerceIn(0.5f, 2f))
    tts.speak(sentence, TextToSpeech.QUEUE_FLUSH, null, UUID.randomUUID().toString())
}
