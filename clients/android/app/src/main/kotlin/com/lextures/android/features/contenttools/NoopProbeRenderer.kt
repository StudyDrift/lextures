package com.lextures.android.features.contenttools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

@Composable
fun NoopProbeRenderer(props: ContentToolRendererProps) {
    val prompt = ContentToolHostLogic.stringField(props.config, "prompt")
        ?.takeIf { it.isNotBlank() }
        ?: props.toolId
    val remote = ContentToolHostLogic.stringField(props.state, "response").orEmpty()
    var response by remember(props.instanceId) { mutableStateOf(remote) }
    var checking by remember { mutableStateOf(false) }
    var resultText by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val scoreLabel = L.text(R.string.mobile_contentTools_runtime_score)
    val needsConnectionLabel = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val yourAnswerLabel = L.text(R.string.mobile_contentTools_runtime_yourAnswer)
    val checkAnswerLabel = L.text(R.string.mobile_contentTools_runtime_checkAnswer)

    LaunchedEffect(remote) {
        if (remote != response) response = remote
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
        Text(text = prompt, fontSize = 14.sp, color = textPrimary())
        OutlinedTextField(
            value = response,
            onValueChange = { value ->
                response = value
                if (!props.readOnly) {
                    props.save(mapOf("response" to JsonPrimitive(value)))
                }
            },
            enabled = !props.readOnly && !checking,
            label = { Text(yourAnswerLabel) },
            modifier = Modifier.fillMaxWidth(),
            minLines = 3,
        )
        Button(
            onClick = {
                if (props.readOnly) return@Button
                checking = true
                resultText = null
                scope.launch {
                    try {
                        val raw = props.runAction(
                            "grade",
                            buildJsonObject { put("response", JsonPrimitive(response)) },
                        )
                        val obj = runCatching { raw?.jsonObject }.getOrNull()
                        val correct = obj?.get("correct")?.jsonPrimitive?.content
                        val reason = obj?.get("reason")?.jsonPrimitive?.content
                        resultText = reason ?: correct
                        if (correct == "true") {
                            props.announce(scoreLabel, false)
                        }
                    } catch (_: Exception) {
                        resultText = needsConnectionLabel
                    } finally {
                        checking = false
                    }
                }
            },
            enabled = !props.readOnly && !checking && response.isNotBlank(),
        ) {
            Text(checkAnswerLabel)
        }
        resultText?.let {
            Text(text = it, fontSize = 12.sp, color = textSecondary())
        }
    }
}
