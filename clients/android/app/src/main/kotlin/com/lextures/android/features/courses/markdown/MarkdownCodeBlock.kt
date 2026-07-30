package com.lextures.android.features.courses.markdown

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.design.LexturesColors

/** Fenced code block: language label, monospace, horizontal scroll, copy (CT.M1 FR-7). */
@Composable
fun MarkdownCodeBlock(
    source: String,
    language: String?,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    var copied by remember { mutableStateOf(false) }
    val languageLabel = if (!language.isNullOrBlank()) {
        L.format(R.string.mobile_markdown_code_language, language)
    } else {
        L.text(R.string.mobile_markdown_code_languageFallback)
    }
    val a11y = if (!language.isNullOrBlank()) {
        "$language ${L.text(R.string.mobile_markdown_code_block)}"
    } else {
        L.text(R.string.mobile_markdown_code_block)
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
            .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
            .semantics { contentDescription = a11y },
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = languageLabel,
                fontSize = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
                modifier = Modifier.padding(start = 4.dp),
            )
            TextButton(
                onClick = {
                    val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                    clipboard.setPrimaryClip(ClipData.newPlainText("code", source))
                    copied = true
                    Toast.makeText(
                        context,
                        context.getString(R.string.mobile_markdown_code_copied),
                        Toast.LENGTH_SHORT,
                    ).show()
                },
            ) {
                Text(
                    text = if (copied) L.text(R.string.mobile_markdown_code_copied) else L.text(R.string.mobile_markdown_code_copy),
                    fontSize = 11.sp,
                    fontWeight = FontWeight.SemiBold,
                )
            }
        }
        Row(
            modifier = Modifier
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 12.dp)
                .padding(bottom = 12.dp),
        ) {
            Text(
                text = source.ifEmpty { " " },
                fontSize = 12.sp,
                fontFamily = FontFamily.Monospace,
                color = textPrimary(),
                softWrap = false,
                overflow = TextOverflow.Visible,
            )
        }
    }
}
