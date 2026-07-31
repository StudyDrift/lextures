package com.lextures.android.features.contenttools

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolPack2Logic

/** Keyboard-aware free-text composer with local draft persistence (CT.M6 FR-2 / FR-3). */
@Composable
fun ToolComposer(
    placeholder: String,
    sendLabel: String,
    text: String,
    onTextChange: (String) -> Unit,
    draftKey: String,
    enabled: Boolean,
    online: Boolean,
    busy: Boolean,
    onSend: () -> Unit,
    modifier: Modifier = Modifier,
    cancelLabel: String? = null,
    showCancel: Boolean = false,
    onCancel: (() -> Unit)? = null,
    onDraftChange: ((String) -> Unit)? = null,
) {
    val context = LocalContext.current
    val draftStore = androidx.compose.runtime.remember(context) { ContentToolDraftStore.create(context) }
    val offlineLabel = L.text(R.string.mobile_contentTools_runtime_offlineComposer)
    val cancel = cancelLabel ?: L.text(R.string.mobile_contentTools_runtime_cancel)

    LaunchedEffect(draftKey) {
        if (text.isEmpty()) {
            val restored = draftStore.load(draftKey)
            if (restored.isNotEmpty()) onTextChange(restored)
        }
    }

    Column(modifier = modifier.fillMaxWidth()) {
        if (!online) {
            Text(
                offlineLabel,
                color = textSecondary(),
                fontSize = 12.sp,
                modifier = Modifier
                    .padding(bottom = 4.dp)
                    .semantics { contentDescription = offlineLabel },
            )
        }
        Row(
            verticalAlignment = Alignment.Bottom,
            modifier = Modifier.fillMaxWidth(),
        ) {
            OutlinedTextField(
                value = text,
                onValueChange = { next ->
                    onTextChange(next)
                    draftStore.save(draftKey, next)
                    onDraftChange?.invoke(next)
                },
                modifier = Modifier
                    .weight(1f)
                    .semantics { contentDescription = placeholder },
                placeholder = { Text(placeholder) },
                enabled = enabled && !busy,
                maxLines = 5,
            )
            if (busy && showCancel && onCancel != null) {
                TextButton(onClick = onCancel, modifier = Modifier.padding(start = 4.dp)) {
                    Text(cancel)
                }
            } else {
                Button(
                    onClick = onSend,
                    enabled = ContentToolPack2Logic.composerSendEnabled(
                        text = text,
                        readOnly = !enabled,
                        online = online,
                        busy = busy,
                    ),
                    modifier = Modifier
                        .padding(start = 4.dp)
                        .semantics { contentDescription = sendLabel },
                ) {
                    Text(sendLabel)
                }
            }
        }
    }
}

fun ContentToolDraftStore.clearDraft(key: String) {
    clear(key)
}
