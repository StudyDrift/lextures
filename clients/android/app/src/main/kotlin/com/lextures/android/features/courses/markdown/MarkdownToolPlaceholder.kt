package com.lextures.android.features.courses.markdown

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.i18n.L

/** Neutral placeholder for ```lex-tool fences until CT.M3 hosts the live tool. */
@Composable
fun MarkdownToolPlaceholder(
    toolId: String,
    modifier: Modifier = Modifier,
) {
    val label = L.format(R.string.mobile_markdown_tool_placeholder, toolId)
    Text(
        text = label,
        fontSize = 14.sp,
        color = textSecondary(),
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
            .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
            .padding(12.dp)
            .semantics { contentDescription = label },
    )
}
