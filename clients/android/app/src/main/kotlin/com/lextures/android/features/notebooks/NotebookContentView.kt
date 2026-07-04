package com.lextures.android.features.notebooks

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.CheckBox
import androidx.compose.material.icons.filled.CheckBoxOutlineBlank
import androidx.compose.material.icons.filled.Draw
import androidx.compose.material.icons.filled.Image
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.notebook.NotebookBlock
import com.lextures.android.core.notebook.NotebookDrawing
import com.lextures.android.core.notebook.NotebookMarkdown
import com.lextures.android.core.notebook.ParsedNotebookTask
import com.lextures.android.features.reader.CaptionedPlayer
import com.lextures.android.features.reader.ContentVideoPlayer
import com.lextures.android.features.reader.ReaderLogic
import java.time.Instant

/**
 * Rendered reading view for a notebook page: headings, lists, quotes, code,
 * and interactive task checkboxes (parity with the web notebook editor output).
 */
@Composable
fun NotebookContentView(
    markdown: String,
    onToggleTask: (ParsedNotebookTask) -> Unit,
    onEditTaskDue: (ParsedNotebookTask) -> Unit,
    modifier: Modifier = Modifier,
    accessToken: String? = null,
    captionsEnabled: Boolean = false,
    onEditDrawing: ((index: Int, elementsJson: String) -> Unit)? = null,
) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        NotebookMarkdown.parseBlocks(markdown).forEach { block ->
            when (block) {
                is NotebookBlock.Heading -> Text(
                    text = inlineMarkdown(block.text),
                    style = LexturesType.display(if (block.level == 1) 24 else if (block.level == 2) 19 else 16),
                    color = textPrimary(),
                    modifier = Modifier.padding(top = if (block.level == 1) 6.dp else 2.dp),
                )

                is NotebookBlock.Paragraph -> {
                    val videoUrl = ReaderLogic.videoUrl(block.text)
                    when {
                        videoUrl != null && captionsEnabled && accessToken != null ->
                            CaptionedPlayer(url = videoUrl, accessToken = accessToken)
                        videoUrl != null && accessToken != null ->
                            ContentVideoPlayer(url = videoUrl, accessToken = accessToken)
                        videoUrl != null ->
                            Text(
                                text = inlineMarkdown(block.text),
                                fontSize = 14.sp,
                                lineHeight = 21.sp,
                                color = textPrimary(),
                            )
                        else ->
                            Text(
                                text = inlineMarkdown(block.text),
                                fontSize = 14.sp,
                                lineHeight = 21.sp,
                                color = textPrimary(),
                            )
                    }
                }

                is NotebookBlock.BulletItem -> Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    modifier = Modifier.padding(start = 4.dp),
                ) {
                    Box(
                        Modifier
                            .padding(top = 8.dp)
                            .size(5.dp)
                            .clip(CircleShape)
                            .background(accentColor()),
                    )
                    Text(text = inlineMarkdown(block.text), fontSize = 14.sp, color = textPrimary())
                }

                is NotebookBlock.OrderedItem -> Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.padding(start = 4.dp),
                ) {
                    Text(
                        text = "${block.number}.",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                    )
                    Text(text = inlineMarkdown(block.text), fontSize = 14.sp, color = textPrimary())
                }

                is NotebookBlock.Quote -> Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    modifier = Modifier.padding(vertical = 2.dp),
                ) {
                    Box(
                        Modifier
                            .width(3.dp)
                            .height(24.dp)
                            .clip(RoundedCornerShape(2.dp))
                            .background(LexturesColors.BrandAmber),
                    )
                    Text(
                        text = inlineMarkdown(block.text),
                        fontSize = 14.sp,
                        fontStyle = FontStyle.Italic,
                        color = textSecondary(),
                    )
                }

                is NotebookBlock.Code -> Text(
                    text = block.text,
                    fontSize = 12.sp,
                    fontFamily = FontFamily.Monospace,
                    color = textPrimary(),
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
                        .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
                        .padding(12.dp),
                )

                NotebookBlock.Divider -> Box(
                    Modifier
                        .fillMaxWidth()
                        .padding(vertical = 4.dp)
                        .height(1.dp)
                        .background(fieldBorder()),
                )

                is NotebookBlock.TaskBlock -> NotebookTaskRow(
                    task = block.task,
                    onToggle = { onToggleTask(block.task) },
                    onEditDue = { onEditTaskDue(block.task) },
                )

                is NotebookBlock.Image -> AuthorizedNotebookImage(
                    url = block.url,
                    alt = block.alt,
                    accessToken = accessToken,
                )

                is NotebookBlock.Drawing -> NotebookDrawingBlock(
                    elementsJson = block.elementsJson,
                    onClick = { onEditDrawing?.invoke(block.index, block.elementsJson) },
                )
            }
        }
    }
}

/**
 * Notebook image loader. Web stores relative course-file paths (`/api/v1/...`) that need the
 * bearer token (parity with web's authorized blob fetch).
 */
@Composable
private fun AuthorizedNotebookImage(url: String, alt: String, accessToken: String?) {
    val context = LocalContext.current
    val resolved = when {
        url.startsWith("/") -> AppConfiguration.apiUrl(url).toString()
        url.startsWith("http://") || url.startsWith("https://") -> url
        else -> null
    }
    if (resolved == null) {
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Default.Image, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(16.dp))
            Text(text = alt.ifBlank { "Image" }, fontSize = 12.sp, color = textSecondary())
        }
        return
    }
    AsyncImage(
        model = ImageRequest.Builder(context)
            .data(resolved)
            .apply { if (!accessToken.isNullOrBlank()) setHeader("Authorization", "Bearer $accessToken") }
            .build(),
        contentDescription = alt.ifBlank { "Image" },
        contentScale = ContentScale.FillWidth,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp)),
    )
}

/** Rendered whiteboard drawing in the reading view; tap to edit. */
@Composable
private fun NotebookDrawingBlock(elementsJson: String, onClick: () -> Unit) {
    val elements = remember(elementsJson) { NotebookDrawing.parseElements(elementsJson) }
    val content = remember(elementsJson) { NotebookDrawing.contentSize(elements) }

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(content.width / content.height)
            .clip(RoundedCornerShape(12.dp))
            .background(if (isDarkTheme()) Color(0xFF1F1F1F) else Color.White)
            .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
            .clickable(onClick = onClick),
    ) {
        Canvas(modifier = Modifier.matchParentSize()) {
            val scale = minOf(size.width / content.width, 1f)
            with(NotebookDrawing) { drawElements(elements, scale) }
        }
        Row(
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(8.dp)
                .clip(RoundedCornerShape(50))
                .background(sceneBackground().copy(alpha = 0.85f))
                .padding(horizontal = 8.dp, vertical = 4.dp),
        ) {
            Icon(Icons.Default.Draw, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(12.dp))
            Text(
                text = if (elements.isEmpty()) "Tap to draw" else "Edit drawing",
                fontSize = 11.sp,
                fontWeight = FontWeight.Medium,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun NotebookTaskRow(
    task: ParsedNotebookTask,
    onToggle: () -> Unit,
    onEditDue: () -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(cardBackground())
            .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
            .padding(10.dp),
    ) {
        Icon(
            imageVector = if (task.checked) Icons.Default.CheckBox else Icons.Default.CheckBoxOutlineBlank,
            contentDescription = if (task.checked) "Mark task incomplete" else "Mark task complete",
            tint = if (task.checked) accentColor() else textSecondary(),
            modifier = Modifier
                .size(22.dp)
                .clickable(onClick = onToggle),
        )
        Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
            Text(
                text = inlineMarkdown(task.text.ifBlank { "Untitled task" }),
                fontSize = 14.sp,
                color = if (task.checked) textSecondary() else textPrimary(),
                textDecoration = if (task.checked) TextDecoration.LineThrough else TextDecoration.None,
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.clickable(onClick = onEditDue),
            ) {
                val overdue = task.dueAt != null && !task.checked && isPastDue(task.dueAt)
                val dueColor = when {
                    overdue -> LexturesColors.Coral
                    task.dueAt != null -> textSecondary()
                    else -> textSecondary().copy(alpha = 0.8f)
                }
                Icon(Icons.Default.CalendarMonth, contentDescription = "Edit due date", tint = dueColor, modifier = Modifier.size(13.dp))
                Text(
                    text = task.dueAt?.let { "Due ${LmsDates.shortDate(it)}" } ?: "Add due date",
                    fontSize = 12.sp,
                    color = dueColor,
                )
            }
        }
        Spacer(Modifier.weight(1f))
    }
}

private fun isPastDue(dueAt: String): Boolean =
    runCatching { Instant.parse(dueAt).isBefore(Instant.now()) }.getOrDefault(false)

/** Minimal inline markdown: **bold**, *italic*, `code`. */
fun inlineMarkdown(raw: String): AnnotatedString = buildAnnotatedString {
    var i = 0
    while (i < raw.length) {
        when {
            raw.startsWith("**", i) -> {
                val end = raw.indexOf("**", i + 2)
                if (end > i + 1) {
                    pushStyle(SpanStyle(fontWeight = FontWeight.Bold))
                    append(raw.substring(i + 2, end))
                    pop()
                    i = end + 2
                } else {
                    append(raw[i]); i++
                }
            }
            raw.startsWith("*", i) && !raw.startsWith("**", i) -> {
                val end = raw.indexOf('*', i + 1)
                if (end > i) {
                    pushStyle(SpanStyle(fontStyle = FontStyle.Italic))
                    append(raw.substring(i + 1, end))
                    pop()
                    i = end + 1
                } else {
                    append(raw[i]); i++
                }
            }
            raw.startsWith("`", i) -> {
                val end = raw.indexOf('`', i + 1)
                if (end > i) {
                    pushStyle(SpanStyle(fontFamily = FontFamily.Monospace, fontSize = 13.sp))
                    append(raw.substring(i + 1, end))
                    pop()
                    i = end + 1
                } else {
                    append(raw[i]); i++
                }
            }
            else -> {
                append(raw[i]); i++
            }
        }
    }
}
