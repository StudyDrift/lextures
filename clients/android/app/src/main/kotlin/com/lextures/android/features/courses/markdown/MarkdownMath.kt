package com.lextures.android.features.courses.markdown

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.i18n.L

/**
 * Math rendering with a safe monospace fallback (CT.M1 FR-10 / open question #1).
 * Unparseable LaTeX still shows source text — never blank or crash.
 */
@Composable
fun MarkdownMath(
    latex: String,
    display: Boolean,
    modifier: Modifier = Modifier,
) {
    val pretty = prettyLatex(latex)
    val label = L.format(
        if (display) R.string.mobile_markdown_math_display else R.string.mobile_markdown_math_inline,
        latex,
    )
    Text(
        text = pretty,
        fontSize = if (display) 15.sp else 13.sp,
        fontFamily = FontFamily.Monospace,
        color = textPrimary(),
        textAlign = if (display) TextAlign.Center else TextAlign.Start,
        modifier = modifier
            .then(if (display) Modifier.fillMaxWidth() else Modifier)
            .padding(vertical = if (display) 4.dp else 0.dp)
            .semantics { contentDescription = label },
    )
}

internal fun prettyLatex(latex: String): String {
    val trimmed = latex.trim()
    if (trimmed.isEmpty()) return latex
    return trimmed
        .replace("\\frac{", "(")
        .replace("}{", ")/(")
        .replace("\\cdot", "·")
        .replace("\\times", "×")
}
