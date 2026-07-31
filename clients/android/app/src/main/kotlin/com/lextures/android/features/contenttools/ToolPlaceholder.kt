package com.lextures.android.features.contenttools

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L

enum class ToolPlaceholderReason {
    LOADING,
    UNAVAILABLE,
    OPEN_IN_BROWSER,
    READ_ONLY_ARCHIVED,
    READ_ONLY_PAST_DUE,
    READ_ONLY_PREVIEW,
    MAINTENANCE,
    UPDATE_REQUIRED,
}

@Composable
fun ToolPlaceholder(
    reason: ToolPlaceholderReason,
    toolName: String? = null,
    message: String? = null,
    onOpenInBrowser: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val text = message ?: when (reason) {
        ToolPlaceholderReason.LOADING -> L.text(R.string.mobile_contentTools_runtime_loading)
        ToolPlaceholderReason.UNAVAILABLE -> L.text(R.string.mobile_contentTools_runtime_unavailable)
        ToolPlaceholderReason.OPEN_IN_BROWSER ->
            if (toolName.isNullOrBlank()) L.text(R.string.mobile_contentTools_runtime_openInBrowser)
            else L.format(R.string.mobile_contentTools_runtime_openInBrowserNamed, toolName)
        ToolPlaceholderReason.READ_ONLY_ARCHIVED -> L.text(R.string.mobile_contentTools_runtime_readOnlyArchived)
        ToolPlaceholderReason.READ_ONLY_PAST_DUE -> L.text(R.string.mobile_contentTools_runtime_readOnlyPastDue)
        ToolPlaceholderReason.READ_ONLY_PREVIEW -> L.text(R.string.mobile_contentTools_runtime_readOnlyPreview)
        ToolPlaceholderReason.MAINTENANCE -> L.text(R.string.mobile_contentTools_runtime_unavailable)
        ToolPlaceholderReason.UPDATE_REQUIRED -> L.text(R.string.mobile_contentTools_runtime_openInBrowser)
    }
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
            .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
            .padding(12.dp)
            .semantics { contentDescription = text },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = text, fontSize = 14.sp, color = textSecondary())
        if (reason == ToolPlaceholderReason.OPEN_IN_BROWSER && onOpenInBrowser != null) {
            OutlinedButton(onClick = onOpenInBrowser) {
                Text(L.text(R.string.mobile_contentTools_runtime_openInBrowserAction))
            }
        }
    }
}
