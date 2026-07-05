package com.lextures.android.core.ui

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.lms.QuizLogic

@Composable
fun CodeEditor(
    text: String,
    onTextChange: (String) -> Unit,
    onInsert: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            QuizLogic.codeSymbolSnippets.forEach { snippet ->
                TextButton(onClick = { onInsert(snippet) }) {
                    Text(
                        snippet.replace("\n", "↵").replace("    ", "⇥"),
                        fontFamily = FontFamily.Monospace,
                    )
                }
            }
        }
        BasicTextField(
            value = text,
            onValueChange = { next ->
                onTextChange(QuizLogic.applyAutoIndent(next))
            },
            textStyle = TextStyle(
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = LexturesColors.TextPrimary,
            ),
            cursorBrush = SolidColor(LexturesColors.Primary),
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(12.dp),
        )
    }
}
