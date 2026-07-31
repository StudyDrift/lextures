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

@Composable
fun ToolErrorCard(
    toolName: String,
    onRetry: () -> Unit,
    message: String? = null,
    modifier: Modifier = Modifier,
) {
    val title = message?.takeIf { it.isNotBlank() }
        ?: L.text(R.string.mobile_contentTools_runtime_errorTitle)
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
            .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
            .padding(12.dp)
            .semantics { contentDescription = "$toolName. $title" },
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = toolName, fontSize = 14.sp, color = textSecondary())
        Text(text = title, fontSize = 13.sp, color = textSecondary())
        OutlinedButton(onClick = onRetry) {
            Text(L.text(R.string.mobile_contentTools_runtime_retry))
        }
    }
}
