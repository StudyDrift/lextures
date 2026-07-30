package com.lextures.android.features.courses.markdown

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.lextures.android.R
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.i18n.L
import com.lextures.android.core.notebook.MarkdownTableAlign
import com.lextures.android.features.notebooks.inlineMarkdown

/** Inline GFM table with horizontal scroll and full-screen expand (CT.M1 FR-5 / FR-6). */
@Composable
fun MarkdownTable(
    align: List<MarkdownTableAlign>,
    header: List<String>,
    rows: List<List<String>>,
    modifier: Modifier = Modifier,
) {
    var expanded by remember { mutableStateOf(false) }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(modifier = Modifier.horizontalScroll(rememberScrollState())) {
            TableGrid(align = align, header = header, rows = rows, minCellWidth = 96)
        }
        TextButton(onClick = { expanded = true }) {
            Text(
                text = L.text(R.string.mobile_markdown_table_expand),
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
            )
        }
    }

    if (expanded) {
        Dialog(
            onDismissRequest = { expanded = false },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
                    .padding(16.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = L.text(R.string.mobile_markdown_table_expand),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    TextButton(onClick = { expanded = false }) {
                        Text(L.text(R.string.mobile_markdown_table_close))
                    }
                }
                Row(
                    modifier = Modifier
                        .horizontalScroll(rememberScrollState())
                        .verticalScroll(rememberScrollState()),
                ) {
                    TableGrid(align = align, header = header, rows = rows, minCellWidth = 120)
                }
            }
        }
    }
}

@Composable
private fun TableGrid(
    align: List<MarkdownTableAlign>,
    header: List<String>,
    rows: List<List<String>>,
    minCellWidth: Int,
) {
    Column(
        modifier = Modifier
            .clip(RoundedCornerShape(8.dp))
            .border(1.dp, fieldBorder(), RoundedCornerShape(8.dp)),
    ) {
        Row(modifier = Modifier.background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)) {
            header.forEachIndexed { index, cell ->
                TableCell(
                    text = cell,
                    align = align.getOrElse(index) { MarkdownTableAlign.Default },
                    isHeader = true,
                    a11y = cell,
                    minCellWidth = minCellWidth,
                )
            }
        }
        rows.forEach { row ->
            Row {
                row.forEachIndexed { index, cell ->
                    val column = header.getOrElse(index) { "" }
                    val rowHeader = row.firstOrNull().orEmpty()
                    val label = if (rowHeader.isNotEmpty() && rowHeader != cell) {
                        "$column, $rowHeader: $cell"
                    } else {
                        "$column: $cell"
                    }
                    TableCell(
                        text = cell,
                        align = align.getOrElse(index) { MarkdownTableAlign.Default },
                        isHeader = false,
                        a11y = label,
                        minCellWidth = minCellWidth,
                    )
                }
            }
        }
    }
}

@Composable
private fun TableCell(
    text: String,
    align: MarkdownTableAlign,
    isHeader: Boolean,
    a11y: String,
    minCellWidth: Int,
) {
    val textAlign = when (align) {
        MarkdownTableAlign.Center -> TextAlign.Center
        MarkdownTableAlign.Right -> TextAlign.End
        MarkdownTableAlign.Left, MarkdownTableAlign.Default -> TextAlign.Start
    }
    Text(
        text = inlineMarkdown(text),
        fontSize = 12.sp,
        fontWeight = if (isHeader) FontWeight.SemiBold else FontWeight.Normal,
        color = textPrimary(),
        textAlign = textAlign,
        modifier = Modifier
            .widthIn(min = minCellWidth.dp)
            .padding(horizontal = 10.dp, vertical = 8.dp)
            .semantics { contentDescription = a11y },
    )
}
