package com.lextures.android.features.notebooks

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.FormatListBulleted
import androidx.compose.material.icons.filled.AddTask
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.CheckBox
import androidx.compose.material.icons.filled.CheckBoxOutlineBlank
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Code
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Draw
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.FormatListNumbered
import androidx.compose.material.icons.filled.FormatQuote
import androidx.compose.material.icons.filled.HorizontalRule
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
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
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.NotebookTaskUpsert
import com.lextures.android.core.lms.NotebookTasksApi
import com.lextures.android.core.notebook.NotebookEditBlock
import com.lextures.android.core.notebook.NotebookMarkdown
import com.lextures.android.core.notebook.NotebookSlashCommand
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.notebook.NotebookSync
import com.lextures.android.core.notebook.NotebookTree
import com.lextures.android.core.notebook.ParsedNotebookTask
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.ZoneOffset

/**
 * Notion-style page editor: a large editable title plus an always-editable block list
 * (text, tasks, drawings, …). There is no separate read/edit mode — tap any block to
 * edit it, toggle tasks in place, tap drawings to open the whiteboard. The `/` command
 * menu and insert toolbar ride above the keyboard while a block is focused.
 */
@Composable
fun NotebookEditorScreen(
    store: NotebookStore,
    courseCode: String,
    notebookTitle: String,
    pageId: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    accessToken: String? = null,
) {
    var notebook by remember { mutableStateOf(store.load(courseCode)) }
    val page = notebook.pages.firstOrNull { it.id == pageId }

    var blocks by remember { mutableStateOf(NotebookMarkdown.editBlocks(page?.contentMd.orEmpty())) }
    var titleText by remember { mutableStateOf(page?.title?.takeIf { it != "Untitled" }.orEmpty()) }
    var focusedBlockId by remember { mutableStateOf<String?>(null) }
    var titleFocused by remember { mutableStateOf(false) }
    var pendingFocusId by remember { mutableStateOf<String?>(null) }
    var editingDrawing by remember { mutableStateOf<EditingDrawingTarget?>(null) }
    var dueTask by remember { mutableStateOf<ParsedNotebookTask?>(null) }
    var menuOpen by remember { mutableStateOf(false) }
    var confirmingDelete by remember { mutableStateOf(false) }
    var pushRevision by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current
    val titleFocusRequester = remember { FocusRequester() }

    fun persist() {
        val data = if (courseCode == NotebookStore.GLOBAL_KEY) notebook else notebook.copy(courseTitle = notebookTitle)
        store.save(courseCode, data)
        pushRevision++
    }

    fun saveDraft() {
        notebook = notebook.copy(
            pages = NotebookTree.updateContent(notebook.pages, pageId, NotebookMarkdown.markdownFromBlocks(blocks)),
        )
        persist()
    }

    fun syncTask(task: ParsedNotebookTask) {
        val token = accessToken ?: return
        scope.launch {
            runCatching {
                NotebookTasksApi.upsert(
                    NotebookTaskUpsert(
                        id = task.id,
                        courseCode = courseCode,
                        notebookPageId = pageId,
                        taskText = task.text,
                        completed = task.checked,
                        dueAt = task.dueAt,
                    ),
                    token,
                )
            }
        }
    }

    fun syncAllTasks() {
        NotebookMarkdown.parseTasks(NotebookMarkdown.markdownFromBlocks(blocks)).forEach { syncTask(it) }
    }

    fun leave() {
        saveDraft()
        syncAllTasks()
        onBack()
    }

    fun deletePage() {
        var pages = NotebookTree.delete(notebook.pages, pageId)
        var activeId = notebook.activePageId
        // Keep at least one real page in the notebook.
        if (pages.none { !NotebookTree.isGroup(it) }) {
            val (withPage, newId) = NotebookTree.addPage(pages, parentId = null)
            pages = withPage
            activeId = newId
        } else if (pages.none { it.id == activeId }) {
            activeId = pages.firstOrNull { !NotebookTree.isGroup(it) }?.id
        }
        notebook = notebook.copy(pages = pages, activePageId = activeId)
        persist()
        onBack()
    }

    // Opening a page makes it the notebook's active page (web sidebar parity); fresh
    // pages drop straight into the title so writing is one tap away.
    LaunchedEffect(pageId) {
        notebook = notebook.copy(activePageId = pageId)
        persist()
        if (page != null && page.contentMd.isBlank()) {
            delay(350)
            if (titleText.isEmpty()) {
                runCatching { titleFocusRequester.requestFocus() }
            } else {
                pendingFocusId = blocks.firstOrNull { it.isTextual }?.id
            }
        }
    }

    // Debounced server push so typing doesn't fire a request per keystroke; each persist
    // restarts the effect, which cancels the pending delay.
    LaunchedEffect(pushRevision) {
        if (pushRevision == 0) return@LaunchedEffect
        delay(1_500)
        NotebookSync.push(store, courseCode, accessToken)
    }

    BackHandler {
        if (focusedBlockId != null || titleFocused) {
            focusManager.clearFocus()
        } else {
            leave()
        }
    }

    Column(modifier = modifier.imePadding()) {
        // Top bar
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { leave() }) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = notebookTitle,
                fontSize = 15.sp,
                fontWeight = FontWeight.Medium,
                color = textSecondary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (focusedBlockId != null || titleFocused) {
                TextButton(onClick = {
                    saveDraft()
                    focusManager.clearFocus()
                }) {
                    Text("Done", fontWeight = FontWeight.SemiBold, color = accentColor())
                }
            } else {
                Box {
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(Icons.Default.MoreVert, contentDescription = "Page actions", tint = textPrimary())
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text("Delete page", color = LexturesColors.Error) },
                            leadingIcon = { Icon(Icons.Default.Delete, contentDescription = null, tint = LexturesColors.Error) },
                            onClick = {
                                menuOpen = false
                                confirmingDelete = true
                            },
                        )
                    }
                }
            }
        }

        BlockEditorPane(
            blocks = blocks,
            titleText = titleText,
            onTitleChange = { value ->
                // The title is a single line — a return moves into the page body.
                val flat = value.replace("\n", "")
                val returned = flat != value
                titleText = flat
                val name = flat.trim().ifEmpty { "Untitled" }
                notebook = notebook.copy(pages = NotebookTree.rename(notebook.pages, pageId, name))
                persist()
                if (returned) {
                    pendingFocusId = blocks.firstOrNull { it.isTextual }?.id
                }
            },
            titleFocusRequester = titleFocusRequester,
            onTitleFocusChanged = { titleFocused = it },
            pendingFocusId = pendingFocusId,
            onPendingFocusConsumed = { pendingFocusId = null },
            onFocusedChange = { focusedBlockId = it },
            onBlocksChange = { next, focusId ->
                blocks = next
                if (focusId != null) pendingFocusId = focusId
                notebook = notebook.copy(
                    pages = NotebookTree.updateContent(notebook.pages, pageId, NotebookMarkdown.markdownFromBlocks(next)),
                )
                persist()
            },
            onToggleTask = { blockId ->
                val idx = blocks.indexOfFirst { it.id == blockId }
                val kind = blocks.getOrNull(idx)?.kind as? NotebookEditBlock.Kind.Task ?: return@BlockEditorPane
                blocks = blocks.toMutableList().also {
                    it[idx] = it[idx].copy(kind = kind.copy(checked = !kind.checked))
                }
                saveDraft()
                syncTask(ParsedNotebookTask(kind.taskId, blocks[idx].text, !kind.checked, kind.dueAt))
            },
            onEditTaskDue = { blockId ->
                val block = blocks.firstOrNull { it.id == blockId } ?: return@BlockEditorPane
                val kind = block.kind as? NotebookEditBlock.Kind.Task ?: return@BlockEditorPane
                dueTask = ParsedNotebookTask(kind.taskId, block.text, kind.checked, kind.dueAt)
            },
            onEditDrawing = { blockId, elementsJson ->
                editingDrawing = EditingDrawingTarget(elementsJson, blockId)
            },
            accessToken = accessToken,
        )
    }

    dueTask?.let { task ->
        TaskDueDateDialog(
            task = task,
            onDismiss = { dueTask = null },
            onSave = { dueAt ->
                val idx = blocks.indexOfFirst { (it.kind as? NotebookEditBlock.Kind.Task)?.taskId == task.id }
                val kind = blocks.getOrNull(idx)?.kind as? NotebookEditBlock.Kind.Task
                if (kind != null) {
                    blocks = blocks.toMutableList().also {
                        it[idx] = it[idx].copy(kind = kind.copy(dueAt = dueAt))
                    }
                    saveDraft()
                    syncTask(ParsedNotebookTask(task.id, blocks[idx].text, kind.checked, dueAt))
                }
                dueTask = null
            },
        )
    }

    editingDrawing?.let { target ->
        NotebookDrawingEditor(
            initialElementsJson = target.elementsJson,
            onDismiss = { editingDrawing = null },
            onSave = { newJson ->
                val idx = blocks.indexOfFirst { it.id == target.blockId }
                if (idx >= 0) {
                    blocks = blocks.toMutableList().also {
                        it[idx] = it[idx].copy(kind = NotebookEditBlock.Kind.Drawing(newJson))
                    }
                    saveDraft()
                }
                editingDrawing = null
            },
        )
    }

    if (confirmingDelete) {
        AlertDialog(
            onDismissRequest = { confirmingDelete = false },
            title = { Text("Delete this page?") },
            text = { Text("The page and its notes will be deleted.") },
            confirmButton = {
                TextButton(onClick = {
                    confirmingDelete = false
                    deletePage()
                }) { Text("Delete", color = LexturesColors.Error) }
            },
            dismissButton = {
                TextButton(onClick = { confirmingDelete = false }) { Text("Cancel") }
            },
        )
    }
}

// MARK: Block editor (WYSIWYG, web parity)

/** Drawing being edited, identified by its block id. */
private data class EditingDrawingTarget(
    val elementsJson: String,
    val blockId: String,
)

@Composable
private fun ColumnScope.BlockEditorPane(
    blocks: List<NotebookEditBlock>,
    titleText: String,
    onTitleChange: (String) -> Unit,
    titleFocusRequester: FocusRequester,
    onTitleFocusChanged: (Boolean) -> Unit,
    pendingFocusId: String?,
    onPendingFocusConsumed: () -> Unit,
    onFocusedChange: (String?) -> Unit,
    onBlocksChange: (List<NotebookEditBlock>, focusId: String?) -> Unit,
    onToggleTask: (String) -> Unit,
    onEditTaskDue: (String) -> Unit,
    onEditDrawing: (blockId: String, elementsJson: String) -> Unit,
    accessToken: String?,
) {
    var focusedId by remember { mutableStateOf<String?>(null) }
    val focusRequesters = remember { mutableMapOf<String, FocusRequester>() }

    fun requesterFor(id: String): FocusRequester = focusRequesters.getOrPut(id) { FocusRequester() }

    LaunchedEffect(pendingFocusId, blocks.size) {
        val id = pendingFocusId ?: return@LaunchedEffect
        if (blocks.any { it.id == id }) {
            runCatching { requesterFor(id).requestFocus() }
            onPendingFocusConsumed()
        }
    }

    // `/query` typed at the start of the focused block opens the command menu (web parity).
    val slashQuery = blocks.firstOrNull { it.id == focusedId && it.isTextual }
        ?.text
        ?.takeIf { it.startsWith("/") }
        ?.drop(1)
        ?.takeIf { !it.contains(' ') && !it.contains('\n') && it.length <= 24 }
    val commands = slashQuery?.let { NotebookMarkdown.filterCommands(it) }.orEmpty()

    /** Newlines never live inside a text block (except code): a return splits the block. */
    fun handleTextChange(blockId: String, newText: String) {
        val idx = blocks.indexOfFirst { it.id == blockId }
        if (idx < 0) return
        val block = blocks[idx]
        if (block.kind is NotebookEditBlock.Kind.Code || !newText.contains('\n')) {
            onBlocksChange(blocks.toMutableList().also { it[idx] = block.copy(text = newText) }, null)
            return
        }
        val parts = newText.split('\n')
        val first = parts.first()
        val rest = parts.drop(1).joinToString(" ")

        // Return on an empty list/quote/task line exits the list back to a paragraph.
        if (first.isEmpty() && rest.isEmpty() && block.kind != NotebookEditBlock.Kind.Paragraph) {
            onBlocksChange(
                blocks.toMutableList().also {
                    it[idx] = block.copy(kind = NotebookEditBlock.Kind.Paragraph, text = "")
                },
                null,
            )
            return
        }

        val continuation = when (block.kind) {
            NotebookEditBlock.Kind.Bullet -> NotebookEditBlock.Kind.Bullet
            NotebookEditBlock.Kind.Ordered -> NotebookEditBlock.Kind.Ordered
            NotebookEditBlock.Kind.Quote -> NotebookEditBlock.Kind.Quote
            is NotebookEditBlock.Kind.Task ->
                NotebookEditBlock.Kind.Task(NotebookMarkdown.newTaskId(), checked = false, dueAt = null)
            else -> NotebookEditBlock.Kind.Paragraph
        }
        val newBlock = NotebookEditBlock(kind = continuation, text = rest)
        onBlocksChange(
            blocks.toMutableList().also {
                it[idx] = block.copy(text = first)
                it.add(idx + 1, newBlock)
            },
            newBlock.id,
        )
    }

    fun deleteBlock(blockId: String) {
        val idx = blocks.indexOfFirst { it.id == blockId }
        if (idx < 0) return
        val next = blocks.toMutableList().also { it.removeAt(idx) }
        if (next.isEmpty()) next.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph))
        onBlocksChange(next, next[(idx - 1).coerceIn(0, next.size - 1)].id)
    }

    /** Focus the trailing textual block (appending a paragraph if the page ends in a drawing). */
    fun focusTail() {
        val last = blocks.lastOrNull()
        if (last != null && last.isTextual) {
            runCatching { requesterFor(last.id).requestFocus() }
        } else {
            val paragraph = NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph)
            onBlocksChange(blocks + paragraph, paragraph.id)
        }
    }

    /** Convert the focused block (text kinds) or insert after it (divider / drawing). */
    fun applyCommand(command: NotebookSlashCommand) {
        val next = blocks.toMutableList()
        var idx = next.indexOfFirst { it.id == focusedId }
        if (idx < 0) {
            next.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph))
            idx = next.size - 1
        }
        if (next[idx].isTextual && next[idx].text.startsWith("/")) {
            next[idx] = next[idx].copy(text = "")
        }
        var focusId: String? = next[idx].id
        when (command.id) {
            "heading1" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Heading(1))
            "heading2" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Heading(2))
            "heading3" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Heading(3))
            "bulletList" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Bullet)
            "orderedList" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Ordered)
            "blockquote" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Quote)
            "codeBlock" -> next[idx] = next[idx].copy(kind = NotebookEditBlock.Kind.Code)
            "task" -> {
                if (next[idx].kind !is NotebookEditBlock.Kind.Task) {
                    next[idx] = next[idx].copy(
                        kind = NotebookEditBlock.Kind.Task(NotebookMarkdown.newTaskId(), checked = false, dueAt = null),
                    )
                }
            }
            "horizontalRule", "drawing" -> {
                val inserted = if (command.id == "drawing") {
                    NotebookEditBlock(kind = NotebookEditBlock.Kind.Drawing("[]"))
                } else {
                    NotebookEditBlock(kind = NotebookEditBlock.Kind.Divider)
                }
                val paragraph = NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph)
                next.add(idx + 1, inserted)
                next.add(idx + 2, paragraph)
                focusId = paragraph.id
            }
        }
        onBlocksChange(next, focusId)
    }

    Column(
        modifier = Modifier
            .weight(1f)
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp, vertical = 4.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        // Page title — Notion-style, edited in place.
        BasicTextField(
            value = titleText,
            onValueChange = onTitleChange,
            textStyle = LexturesType.display(26, FontWeight.Bold).copy(color = textPrimary()),
            cursorBrush = SolidColor(accentColor()),
            modifier = Modifier
                .fillMaxWidth()
                .focusRequester(titleFocusRequester)
                .onFocusChanged { onTitleFocusChanged(it.isFocused) },
            decorationBox = { inner ->
                Box {
                    if (titleText.isEmpty()) {
                        Text(
                            text = "Untitled",
                            style = LexturesType.display(26, FontWeight.Bold),
                            color = textSecondary().copy(alpha = 0.45f),
                        )
                    }
                    inner()
                }
            },
        )

        blocks.forEachIndexed { index, block ->
            // Ordered numbering: position within the current run of ordered blocks.
            var orderedNumber = 1
            if (block.kind is NotebookEditBlock.Kind.Ordered) {
                var i = index - 1
                while (i >= 0 && blocks[i].kind is NotebookEditBlock.Kind.Ordered) {
                    orderedNumber++
                    i--
                }
            }
            EditBlockRow(
                block = block,
                orderedNumber = orderedNumber,
                focusRequester = requesterFor(block.id),
                onFocused = { has ->
                    if (has) {
                        focusedId = block.id
                        onFocusedChange(block.id)
                    } else if (focusedId == block.id) {
                        focusedId = null
                        onFocusedChange(null)
                    }
                },
                onTextChange = { handleTextChange(block.id, it) },
                onToggleTask = { onToggleTask(block.id) },
                onEditTaskDue = { onEditTaskDue(block.id) },
                onEditDrawing = {
                    (block.kind as? NotebookEditBlock.Kind.Drawing)?.let { onEditDrawing(block.id, it.elementsJson) }
                },
                onDelete = { deleteBlock(block.id) },
                accessToken = accessToken,
            )
        }

        // Tap the empty space below the page to keep writing.
        Spacer(
            Modifier
                .fillMaxWidth()
                .height(220.dp)
                .clickable(
                    interactionSource = remember { MutableInteractionSource() },
                    indication = null,
                ) { focusTail() },
        )
    }

    // `/` command menu (anchored above the insert toolbar, parity with the web slash menu).
    if (commands.isNotEmpty()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp)
                .heightIn(max = 220.dp)
                .clip(RoundedCornerShape(14.dp))
                .background(cardBackground())
                .border(1.dp, fieldBorder(), RoundedCornerShape(14.dp))
                .verticalScroll(rememberScrollState()),
        ) {
            commands.forEach { command ->
                Row(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { applyCommand(command) }
                        .padding(horizontal = 14.dp, vertical = 8.dp),
                ) {
                    Icon(
                        imageVector = commandIcon(command.id),
                        contentDescription = null,
                        tint = accentColor(),
                        modifier = Modifier.size(18.dp),
                    )
                    Column {
                        Text(command.label, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary())
                        Text(command.detail, fontSize = 11.sp, color = textSecondary())
                    }
                }
            }
        }
    }

    // Insert toolbar — rides above the keyboard while a block is focused.
    if (focusedId != null) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .background(cardBackground())
                .border(1.dp, fieldBorder())
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 12.dp, vertical = 6.dp),
        ) {
            fun insert(commandId: String) {
                applyCommand(NotebookMarkdown.slashCommands.first { it.id == commandId })
            }
            ToolbarIconButton(Icons.Default.AddTask, "Task") { insert("task") }
            ToolbarIconButton(Icons.Default.Draw, "Drawing") { insert("drawing") }
            ToolbarDividerLine()
            ToolbarTextButton("H1") { insert("heading1") }
            ToolbarTextButton("H2") { insert("heading2") }
            ToolbarTextButton("H3") { insert("heading3") }
            ToolbarDividerLine()
            ToolbarIconButton(Icons.AutoMirrored.Filled.FormatListBulleted, "Bullet list") { insert("bulletList") }
            ToolbarIconButton(Icons.Default.FormatListNumbered, "Numbered list") { insert("orderedList") }
            ToolbarIconButton(Icons.Default.FormatQuote, "Quote") { insert("blockquote") }
            ToolbarIconButton(Icons.Default.Code, "Code") { insert("codeBlock") }
            ToolbarIconButton(Icons.Default.HorizontalRule, "Divider") { insert("horizontalRule") }
            ToolbarDividerLine()
            ToolbarIconButton(Icons.Default.Delete, "Delete block") {
                focusedId?.let { deleteBlock(it) }
            }
        }
    }
}

/** One block row in edit mode: styled like the reading view, but editable in place. */
@Composable
private fun EditBlockRow(
    block: NotebookEditBlock,
    orderedNumber: Int,
    focusRequester: FocusRequester,
    onFocused: (Boolean) -> Unit,
    onTextChange: (String) -> Unit,
    onToggleTask: () -> Unit,
    onEditTaskDue: () -> Unit,
    onEditDrawing: () -> Unit,
    onDelete: () -> Unit,
    accessToken: String?,
) {
    @Composable
    fun blockTextField(textStyle: TextStyle, modifier: Modifier = Modifier) {
        BasicTextField(
            value = block.text,
            onValueChange = onTextChange,
            textStyle = textStyle,
            cursorBrush = SolidColor(accentColor()),
            modifier = modifier
                .fillMaxWidth()
                .focusRequester(focusRequester)
                .onFocusChanged { onFocused(it.isFocused) },
        )
    }

    when (val kind = block.kind) {
        NotebookEditBlock.Kind.Paragraph ->
            blockTextField(TextStyle(fontSize = 14.sp, lineHeight = 21.sp, color = textPrimary()))

        is NotebookEditBlock.Kind.Heading -> {
            val size = if (kind.level == 1) 24 else if (kind.level == 2) 19 else 16
            blockTextField(LexturesType.display(size).copy(color = textPrimary()))
        }

        NotebookEditBlock.Kind.Bullet -> Row(
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
            blockTextField(TextStyle(fontSize = 14.sp, color = textPrimary()))
        }

        NotebookEditBlock.Kind.Ordered -> Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.padding(start = 4.dp),
        ) {
            Text(
                text = "$orderedNumber.",
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = accentColor(),
            )
            blockTextField(TextStyle(fontSize = 14.sp, color = textPrimary()))
        }

        NotebookEditBlock.Kind.Quote -> Row(
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
            blockTextField(TextStyle(fontSize = 14.sp, fontStyle = FontStyle.Italic, color = textSecondary()))
        }

        NotebookEditBlock.Kind.Code -> blockTextField(
            TextStyle(fontSize = 12.sp, fontFamily = FontFamily.Monospace, color = textPrimary()),
            modifier = Modifier
                .clip(RoundedCornerShape(10.dp))
                .background(if (isDarkTheme()) sceneBackground() else LexturesColors.SceneBackground)
                .border(1.dp, fieldBorder(), RoundedCornerShape(10.dp))
                .padding(12.dp),
        )

        NotebookEditBlock.Kind.Divider -> NonTextBlockRow(onDelete) {
            Box(
                Modifier
                    .weight(1f)
                    .padding(vertical = 8.dp)
                    .height(1.dp)
                    .background(fieldBorder()),
            )
        }

        is NotebookEditBlock.Kind.Task -> Row(
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .background(cardBackground())
                .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
                .padding(10.dp),
        ) {
            Icon(
                imageVector = if (kind.checked) Icons.Default.CheckBox else Icons.Default.CheckBoxOutlineBlank,
                contentDescription = if (kind.checked) "Mark task incomplete" else "Mark task complete",
                tint = if (kind.checked) accentColor() else textSecondary(),
                modifier = Modifier
                    .size(22.dp)
                    .clickable(onClick = onToggleTask),
            )
            Column(verticalArrangement = Arrangement.spacedBy(3.dp), modifier = Modifier.weight(1f)) {
                blockTextField(TextStyle(fontSize = 14.sp, color = textPrimary()))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.clickable(onClick = onEditTaskDue),
                ) {
                    Icon(
                        Icons.Default.CalendarMonth,
                        contentDescription = "Edit due date",
                        tint = textSecondary(),
                        modifier = Modifier.size(13.dp),
                    )
                    Text(
                        text = kind.dueAt?.let { "Due ${LmsDates.shortDate(it)}" } ?: "Add due date",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }

        is NotebookEditBlock.Kind.Image -> NonTextBlockRow(onDelete) {
            Box(Modifier.weight(1f)) {
                NotebookContentView(
                    markdown = "![${kind.alt}](${kind.url})",
                    onToggleTask = {},
                    onEditTaskDue = {},
                    accessToken = accessToken,
                )
            }
        }

        is NotebookEditBlock.Kind.Drawing -> NonTextBlockRow(onDelete) {
            Box(Modifier.weight(1f)) {
                NotebookContentView(
                    markdown = "```drawing\n${kind.elementsJson}\n```",
                    onToggleTask = {},
                    onEditTaskDue = {},
                    onEditDrawing = { _, _ -> onEditDrawing() },
                )
            }
        }
    }
}

/** Non-text blocks get a small delete affordance since they can't be backspaced away. */
@Composable
private fun NonTextBlockRow(onDelete: () -> Unit, content: @Composable RowScope.() -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.Top) {
        content()
        Icon(
            Icons.Default.Close,
            contentDescription = "Delete block",
            tint = textSecondary().copy(alpha = 0.7f),
            modifier = Modifier
                .size(20.dp)
                .clip(CircleShape)
                .clickable(onClick = onDelete),
        )
    }
}

private fun commandIcon(commandId: String): ImageVector = when (commandId) {
    "task" -> Icons.Default.AddTask
    "drawing" -> Icons.Default.Draw
    "bulletList" -> Icons.AutoMirrored.Filled.FormatListBulleted
    "orderedList" -> Icons.Default.FormatListNumbered
    "blockquote" -> Icons.Default.FormatQuote
    "codeBlock" -> Icons.Default.Code
    "horizontalRule" -> Icons.Default.HorizontalRule
    else -> Icons.Default.Edit
}

@Composable
private fun ToolbarTextButton(label: String, onClick: () -> Unit) {
    Text(
        text = label,
        fontSize = 13.sp,
        fontWeight = FontWeight.Bold,
        color = textPrimary(),
        modifier = Modifier
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 9.dp),
    )
}

@Composable
private fun ToolbarIconButton(icon: ImageVector, label: String, onClick: () -> Unit) {
    Icon(
        imageVector = icon,
        contentDescription = label,
        tint = textPrimary(),
        modifier = Modifier
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 8.dp)
            .size(20.dp),
    )
}

@Composable
private fun ToolbarDividerLine() {
    Box(
        Modifier
            .padding(horizontal = 4.dp)
            .width(1.dp)
            .heightIn(min = 22.dp)
            .background(fieldBorder()),
    )
}

// MARK: Due date

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TaskDueDateDialog(
    task: ParsedNotebookTask,
    onDismiss: () -> Unit,
    onSave: (String?) -> Unit,
) {
    val initialMillis = task.dueAt
        ?.let { runCatching { Instant.parse(it) }.getOrNull() }
        ?.atZone(ZoneId.systemDefault())
        ?.toLocalDate()
        ?.atStartOfDay(ZoneOffset.UTC)
        ?.toInstant()
        ?.toEpochMilli()
    val state = rememberDatePickerState(initialSelectedDateMillis = initialMillis)

    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = {
                val millis = state.selectedDateMillis
                if (millis == null) {
                    onSave(null)
                } else {
                    // Picker returns UTC midnight; due time is local end of day (web parity).
                    val localDate = Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate()
                    val dueAt = localDate.atTime(23, 59, 59).atZone(ZoneId.systemDefault()).toInstant().toString()
                    onSave(dueAt)
                }
            }) { Text("Save", fontWeight = FontWeight.SemiBold) }
        },
        dismissButton = {
            Row {
                if (task.dueAt != null) {
                    TextButton(onClick = { onSave(null) }) {
                        Text("Remove", color = LexturesColors.Error)
                    }
                }
                TextButton(onClick = onDismiss) { Text("Cancel") }
            }
        },
    ) {
        DatePicker(state = state, showModeToggle = false)
    }
}
