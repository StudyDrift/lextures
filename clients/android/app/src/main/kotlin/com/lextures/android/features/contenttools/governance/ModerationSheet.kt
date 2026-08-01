package com.lextures.android.features.contenttools.governance

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolModerationAction
import kotlinx.coroutines.launch

/** Staff moderation controls (CT.M9 FR-11). Hidden for non-entitled viewers by the caller. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ModerationSheet(
    items: List<ContentToolModerationAction>,
    onModerate: suspend (action: String) -> ModerationResult,
    onDismiss: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var forbidden by remember { mutableStateOf(false) }
    val forbiddenLabel = L.text(R.string.mobile_contentTools_governance_moderateForbidden)
    val errorLabel = L.text(R.string.mobile_contentTools_governance_moderateError)

    fun run(action: String) {
        if (busy || forbidden) return
        scope.launch {
            busy = true
            errorText = null
            when (onModerate(action)) {
                ModerationResult.OK -> onDismiss()
                ModerationResult.FORBIDDEN -> {
                    forbidden = true
                    errorText = forbiddenLabel
                }
                ModerationResult.ERROR -> errorText = errorLabel
            }
            busy = false
        }
    }

    ModalBottomSheet(
        onDismissRequest = { if (!busy) onDismiss() },
        sheetState = sheetState,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(R.string.mobile_contentTools_governance_moderateTitle),
                fontSize = 16.sp,
                color = textPrimary(),
            )
            if (items.isEmpty()) {
                Text(
                    text = L.text(R.string.mobile_contentTools_governance_moderateEmpty),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(items, key = { it.id.ifEmpty { it.createdAt + it.action } }) { item ->
                        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(text = item.action, fontSize = 14.sp, color = textPrimary())
                            item.category?.takeIf { it.isNotBlank() }?.let {
                                Text(text = it, fontSize = 12.sp, color = textSecondary())
                            }
                            if (item.createdAt.isNotBlank()) {
                                Text(text = item.createdAt, fontSize = 11.sp, color = textSecondary())
                            }
                        }
                    }
                }
            }
            errorText?.let {
                Text(text = it, fontSize = 12.sp, color = LexturesColors.Coral)
            }
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                TextButton(
                    onClick = { run("hidden") },
                    enabled = !busy && !forbidden,
                ) {
                    Text(L.text(R.string.mobile_contentTools_governance_moderateHide))
                }
                TextButton(
                    onClick = { run("removed") },
                    enabled = !busy && !forbidden,
                ) {
                    Text(L.text(R.string.mobile_contentTools_governance_moderateRemove))
                }
                TextButton(
                    onClick = { run("restored") },
                    enabled = !busy && !forbidden,
                ) {
                    Text(L.text(R.string.mobile_contentTools_governance_moderateRestore))
                }
                TextButton(onClick = onDismiss, enabled = !busy) {
                    Text(L.text(R.string.mobile_contentTools_runtime_cancel))
                }
            }
        }
    }
}

enum class ModerationResult {
    OK,
    FORBIDDEN,
    ERROR,
}
