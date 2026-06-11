package com.lextures.android.features.notebooks

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.CheckBox
import androidx.compose.material.icons.filled.CheckBoxOutlineBlank
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.font.FontStyle
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.lms.LmsDates
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.FormatListBulleted
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.AddTask
import androidx.compose.material.icons.filled.Code
import androidx.compose.material.icons.filled.CreateNewFolder
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Draw
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.FormatListNumbered
import androidx.compose.material.icons.filled.FormatQuote
import androidx.compose.material.icons.filled.HorizontalRule
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.UnfoldMore
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.NotebookTaskUpsert
import com.lextures.android.core.lms.NotebookTasksApi
import com.lextures.android.core.notebook.NotebookEditBlock
import com.lextures.android.core.notebook.NotebookMarkdown
import com.lextures.android.core.notebook.NotebookPage
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
 * Notebook page editor with a page tree, a rendered reading view (interactive tasks),
 * and a markdown edit mode with `/` commands + insert toolbar (parity with web).
 */
@Composable
fun NotebookEditorScreen(
    store: NotebookStore,
    courseCode: String,
    title: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    accessToken: String? = null,
) {
    var notebook by remember {
        val loaded = store.load(courseCode)
        mutableStateOf(
            if (courseCode == NotebookStore.GLOBAL_KEY) loaded else loaded.copy(courseTitle = title),
        )
    }
    val activePage = notebook.pages.firstOrNull { it.id == notebook.activePageId }
        ?: notebook.pages.firstOrNull { !NotebookTree.isGroup(it) }
    val activeIsGroup = activePage != null && NotebookTree.isGroup(activePage)

    // WYSIWYG edit mode: the page is a list of rendered blocks (web block-editor parity);
    // markdown is only the storage format.
    var blocks by remember { mutableStateOf(NotebookMarkdown.editBlocks(activePage?.contentMd.orEmpty())) }
    var focusedBlockId by remember { mutableStateOf<String?>(null) }
    var pendingFocusId by remember { mutableStateOf<String?>(null) }
    var editingDrawing by remember { mutableStateOf<EditingDrawingTarget?>(null) }
    var editing by remember { mutableStateOf(activePage != null && !activeIsGroup && activePage.contentMd.isBlank()) }
    var showPages by remember { mutableStateOf(false) }
    var menuOpen by remember { mutableStateOf(false) }
    var renaming by remember { mutableStateOf(false) }
    var renameText by remember { mutableStateOf("") }
    var dueTask by remember { mutableStateOf<ParsedNotebookTask?>(null) }
    var pushRevision by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    fun persist() {
        store.save(courseCode, notebook)
        pushRevision++
    }

    // Rendered edit blocks from canonical markdown (fences never shown to the user).
    fun loadBlocks(contentMd: String) {
        blocks = NotebookMarkdown.editBlocks(contentMd)
    }

    // Canonical markdown rebuilt from the edit blocks.
    fun draftMarkdown(): String = NotebookMarkdown.markdownFromBlocks(blocks)

    // Merge any newer server copy (e.g. written on web) once on open.
    LaunchedEffect(courseCode) {
        if (NotebookSync.pull(store, accessToken)) {
            val loaded = store.load(courseCode)
            notebook = if (courseCode == NotebookStore.GLOBAL_KEY) loaded else loaded.copy(courseTitle = title)
            val page = loaded.pages.firstOrNull { it.id == loaded.activePageId }
                ?: loaded.pages.firstOrNull { !NotebookTree.isGroup(it) }
            loadBlocks(page?.contentMd.orEmpty())
            editing = page != null && !NotebookTree.isGroup(page) && page.contentMd.isBlank()
        }
    }

    // Debounced server push so typing doesn't fire a request per keystroke; each persist
    // restarts the effect, which cancels the pending delay.
    LaunchedEffect(pushRevision) {
        if (pushRevision == 0) return@LaunchedEffect
        delay(1_500)
        NotebookSync.push(store, courseCode, accessToken)
    }

    fun saveDraft() {
        val active = activePage ?: return
        if (editing && !NotebookTree.isGroup(active)) {
            notebook = notebook.copy(pages = NotebookTree.updateContent(notebook.pages, active.id, draftMarkdown()))
        }
        persist()
    }

    fun syncTask(task: ParsedNotebookTask) {
        val token = accessToken ?: return
        val pageId = activePage?.id ?: return
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

    fun updateActiveContent(contentMd: String) {
        val active = activePage ?: return
        notebook = notebook.copy(pages = NotebookTree.updateContent(notebook.pages, active.id, contentMd))
        loadBlocks(contentMd)
        persist()
    }

    fun selectPage(pageId: String) {
        saveDraft()
        editing = false
        val page = notebook.pages.firstOrNull { it.id == pageId }
        notebook = notebook.copy(activePageId = pageId)
        loadBlocks(page?.contentMd.orEmpty())
        if (page != null && !NotebookTree.isGroup(page) && page.contentMd.isBlank()) {
            editing = true
        }
        persist()
    }

    fun createPage(parentId: String?) {
        saveDraft()
        val (pages, newId) = NotebookTree.addPage(notebook.pages, parentId)
        notebook = notebook.copy(pages = pages, activePageId = newId)
        loadBlocks("")
        pendingFocusId = blocks.firstOrNull()?.id
        editing = true
        persist()
    }

    fun createGroup() {
        saveDraft()
        val (pages, _) = NotebookTree.addGroup(notebook.pages, parentId = null, title = "New group")
        notebook = notebook.copy(pages = pages)
        persist()
    }

    fun deletePage(pageId: String) {
        var pages = NotebookTree.delete(notebook.pages, pageId)
        var activeId = notebook.activePageId
        if (pages.none { !NotebookTree.isGroup(it) }) {
            val (withPage, newId) = NotebookTree.addPage(pages, parentId = null)
            pages = withPage
            activeId = newId
        }
        if (pages.none { it.id == activeId }) {
            activeId = pages.firstOrNull { !NotebookTree.isGroup(it) }?.id ?: pages.firstOrNull()?.id
        }
        notebook = notebook.copy(pages = pages, activePageId = activeId)
        loadBlocks(pages.firstOrNull { it.id == activeId }?.contentMd.orEmpty())
        editing = false
        persist()
    }

    fun finishEditing() {
        saveDraft()
        editing = false
        activePage?.let { page ->
            NotebookMarkdown.parseTasks(page.contentMd).forEach { syncTask(it) }
        }
    }

    BackHandler {
        if (editing) {
            finishEditing()
        } else {
            saveDraft()
            onBack()
        }
    }

    Column(modifier = modifier.imePadding()) {
        // Top bar
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = {
                saveDraft()
                onBack()
            }) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = title,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (editing) {
                TextButton(onClick = { finishEditing() }) {
                    Text("Done", fontWeight = FontWeight.SemiBold, color = accentColor())
                }
            } else {
                Box {
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(Icons.Default.MoreVert, contentDescription = "Page actions", tint = textPrimary())
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text("Rename page") },
                            leadingIcon = { Icon(Icons.Default.Edit, contentDescription = null) },
                            onClick = {
                                menuOpen = false
                                renameText = activePage?.title.orEmpty()
                                renaming = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text("New page") },
                            leadingIcon = { Icon(Icons.Default.Add, contentDescription = null) },
                            onClick = {
                                menuOpen = false
                                createPage(null)
                            },
                        )
                        DropdownMenuItem(
                            text = { Text("New group") },
                            leadingIcon = { Icon(Icons.Default.CreateNewFolder, contentDescription = null) },
                            onClick = {
                                menuOpen = false
                                createGroup()
                                showPages = true
                            },
                        )
                    }
                }
            }
        }

        // Page header: current page (opens the page tree) + Edit
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(cardBackground())
                    .border(1.dp, fieldBorder(), RoundedCornerShape(50))
                    .clickable {
                        saveDraft()
                        showPages = true
                    }
                    .padding(horizontal = 14.dp, vertical = 9.dp)
                    .weight(1f, fill = false),
            ) {
                Icon(
                    imageVector = if (activeIsGroup) Icons.Default.Folder else Icons.Default.Description,
                    contentDescription = null,
                    tint = if (activeIsGroup) LexturesColors.BrandAmber else accentColor(),
                    modifier = Modifier.size(14.dp),
                )
                Text(
                    text = activePage?.let { NotebookTree.pathLabel(notebook.pages, it.id) }?.ifBlank { "Untitled" } ?: "Untitled",
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Icon(Icons.Default.UnfoldMore, contentDescription = "Show pages", tint = textSecondary(), modifier = Modifier.size(14.dp))
            }

            Spacer(Modifier.weight(1f))

            if (!editing && !activeIsGroup) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(5.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .background(
                            Brush.horizontalGradient(
                                listOf(LexturesColors.Primary, Color(0xFF17897B)),
                            ),
                        )
                        .clickable {
                            loadBlocks(activePage?.contentMd.orEmpty())
                            editing = true
                            pendingFocusId = blocks.lastOrNull { it.isTextual }?.id
                        }
                        .padding(horizontal = 16.dp, vertical = 9.dp),
                ) {
                    Icon(Icons.Default.Edit, contentDescription = null, tint = Color.White, modifier = Modifier.size(14.dp))
                    Text("Edit", fontSize = 13.sp, fontWeight = FontWeight.SemiBold, color = Color.White)
                }
            }
        }

        when {
            activeIsGroup -> GroupPanel(
                notebookPages = notebook.pages,
                group = activePage,
                onSelect = { selectPage(it) },
                onCreatePage = { createPage(activePage?.id) },
            )

            editing -> BlockEditorPane(
                blocks = blocks,
                pendingFocusId = pendingFocusId,
                onPendingFocusConsumed = { pendingFocusId = null },
                onFocusedChange = { focusedBlockId = it },
                onBlocksChange = { next, focusId ->
                    blocks = next
                    if (focusId != null) pendingFocusId = focusId
                    val active = activePage
                    if (active != null) {
                        notebook = notebook.copy(pages = NotebookTree.updateContent(notebook.pages, active.id, draftMarkdown()))
                        persist()
                    }
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
                    editingDrawing = EditingDrawingTarget(elementsJson, readIndex = null, blockId = blockId)
                },
                accessToken = accessToken,
            )

            else -> ReadingPane(
                contentMd = activePage?.contentMd.orEmpty(),
                onStartEditing = {
                    loadBlocks(activePage?.contentMd.orEmpty())
                    editing = true
                    pendingFocusId = blocks.lastOrNull { it.isTextual }?.id
                },
                onToggleTask = { task ->
                    val active = activePage ?: return@ReadingPane
                    updateActiveContent(NotebookMarkdown.setTaskChecked(active.contentMd, task.id, !task.checked))
                    syncTask(task.copy(checked = !task.checked))
                },
                onEditTaskDue = { dueTask = it },
                accessToken = accessToken,
                onEditDrawing = { index, elementsJson ->
                    editingDrawing = EditingDrawingTarget(elementsJson, readIndex = index, blockId = null)
                },
            )
        }
    }

    if (showPages) {
        NotebookPagesSheet(
            pages = notebook.pages,
            activePageId = activePage?.id,
            onDismiss = { showPages = false },
            onSelect = { selectPage(it) },
            onCreatePage = { createPage(it) },
            onCreateGroup = { createGroup() },
            onRename = { pageId, name ->
                notebook = notebook.copy(pages = NotebookTree.rename(notebook.pages, pageId, name))
                persist()
            },
            onMove = { pageId, newParentId ->
                NotebookTree.moveToParent(notebook.pages, pageId, newParentId)?.let {
                    notebook = notebook.copy(pages = it)
                    persist()
                }
            },
            onDelete = { deletePage(it) },
        )
    }

    dueTask?.let { task ->
        TaskDueDateDialog(
            task = task,
            onDismiss = { dueTask = null },
            onSave = { dueAt ->
                saveDraft()
                val active = activePage
                if (active != null) {
                    val next = NotebookMarkdown.setTaskDueAt(active.contentMd, task.id, dueAt)
                    updateActiveContent(next)
                    NotebookMarkdown.parseTasks(next).firstOrNull { it.id == task.id }?.let { syncTask(it) }
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
                val blockId = target.blockId
                if (blockId != null) {
                    val idx = blocks.indexOfFirst { it.id == blockId }
                    if (idx >= 0) {
                        blocks = blocks.toMutableList().also {
                            it[idx] = it[idx].copy(kind = NotebookEditBlock.Kind.Drawing(newJson))
                        }
                        saveDraft()
                    }
                } else if (target.readIndex != null) {
                    activePage?.let { active ->
                        updateActiveContent(NotebookMarkdown.replaceDrawing(active.contentMd, target.readIndex, newJson))
                    }
                }
                editingDrawing = null
            },
        )
    }

    if (renaming) {
        AlertDialog(
            onDismissRequest = { renaming = false },
            title = { Text("Rename page") },
            text = {
                OutlinedTextField(value = renameText, onValueChange = { renameText = it }, singleLine = true)
            },
            confirmButton = {
                TextButton(onClick = {
                    val name = renameText.trim()
                    val active = activePage
                    if (name.isNotEmpty() && active != null) {
                        notebook = notebook.copy(pages = NotebookTree.rename(notebook.pages, active.id, name))
                        persist()
                    }
                    renaming = false
                }) { Text("Save") }
            },
            dismissButton = {
                TextButton(onClick = { renaming = false }) { Text("Cancel") }
            },
        )
    }
}

// MARK: Reading

@Composable
private fun ReadingPane(
    contentMd: String,
    onStartEditing: () -> Unit,
    onToggleTask: (ParsedNotebookTask) -> Unit,
    onEditTaskDue: (ParsedNotebookTask) -> Unit,
    accessToken: String? = null,
    onEditDrawing: ((Int, String) -> Unit)? = null,
) {
    if (contentMd.isBlank()) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(10.dp),
            modifier = Modifier.fillMaxWidth().padding(horizontal = 40.dp, vertical = 70.dp),
        ) {
            Icon(Icons.Default.Edit, contentDescription = null, tint = accentColor().copy(alpha = 0.7f), modifier = Modifier.size(34.dp))
            Text("This page is empty", style = LexturesType.display(19), color = textPrimary())
            Text(
                text = "Tap Edit to start writing. Type / while editing for headings, tasks, lists, and more.",
                fontSize = 13.sp,
                color = textSecondary(),
                textAlign = TextAlign.Center,
            )
            TextButton(onClick = onStartEditing) {
                Text("Start writing", fontWeight = FontWeight.SemiBold, color = accentColor())
            }
        }
        return
    }
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp)
            .padding(bottom = 24.dp),
    ) {
        NotebookContentView(
            markdown = contentMd,
            onToggleTask = onToggleTask,
            onEditTaskDue = onEditTaskDue,
            accessToken = accessToken,
            onEditDrawing = onEditDrawing,
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(cardBackground())
                .border(1.dp, fieldBorder().copy(alpha = 0.9f), RoundedCornerShape(16.dp))
                .padding(16.dp),
        )
    }
}

// MARK: Group panel

@Composable
private fun GroupPanel(
    notebookPages: List<NotebookPage>,
    group: NotebookPage?,
    onSelect: (String) -> Unit,
    onCreatePage: () -> Unit,
) {
    val children = NotebookTree.sortedChildren(notebookPages, group?.id)
    Column(
        verticalArrangement = Arrangement.spacedBy(10.dp),
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        if (children.isEmpty()) {
            Text(
                text = "No pages in this group yet.",
                fontSize = 13.sp,
                color = textSecondary(),
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth().padding(top = 20.dp),
            )
        }
        children.forEach { child ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(cardBackground())
                    .border(1.dp, fieldBorder(), RoundedCornerShape(12.dp))
                    .clickable { onSelect(child.id) }
                    .padding(14.dp),
            ) {
                Icon(
                    imageVector = if (NotebookTree.isGroup(child)) Icons.Default.Folder else Icons.Default.Description,
                    contentDescription = null,
                    tint = if (NotebookTree.isGroup(child)) LexturesColors.BrandAmber else accentColor(),
                    modifier = Modifier.size(18.dp),
                )
                Text(
                    text = child.title.ifBlank { "Untitled" },
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                    modifier = Modifier.weight(1f),
                )
                Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(18.dp))
            }
        }
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.clickable(onClick = onCreatePage).padding(vertical = 8.dp),
        ) {
            Icon(Icons.Default.Add, contentDescription = null, tint = accentColor(), modifier = Modifier.size(18.dp))
            Text("New page in group", fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
        }
    }
}

// MARK: Block editor (WYSIWYG, web parity)

/** Drawing being edited — from the reading view (by document index) or the block editor (by block id). */
private data class EditingDrawingTarget(
    val elementsJson: String,
    val readIndex: Int?,
    val blockId: String?,
)

@Composable
private fun ColumnScope.BlockEditorPane(
    blocks: List<NotebookEditBlock>,
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

    /** Convert the focused block (text kinds) or insert after it (divider / drawing). */
    fun applyCommand(command: NotebookSlashCommand) {
        var next = blocks.toMutableList()
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
            .padding(horizontal = 16.dp)
            .clip(RoundedCornerShape(16.dp))
            .background(cardBackground())
            .border(1.dp, fieldBorder().copy(alpha = 0.9f), RoundedCornerShape(16.dp))
            .verticalScroll(rememberScrollState())
            .padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
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

    // Insert toolbar — converts the focused block, or inserts divider/drawing after it.
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
        ToolbarTextButton("H1") { insert("heading1") }
        ToolbarTextButton("H2") { insert("heading2") }
        ToolbarTextButton("H3") { insert("heading3") }
        ToolbarDividerLine()
        ToolbarIconButton(Icons.Default.AddTask, "Task") { insert("task") }
        ToolbarIconButton(Icons.Default.Draw, "Drawing") { insert("drawing") }
        ToolbarIconButton(Icons.AutoMirrored.Filled.FormatListBulleted, "Bullet list") { insert("bulletList") }
        ToolbarIconButton(Icons.Default.FormatListNumbered, "Numbered list") { insert("orderedList") }
        ToolbarDividerLine()
        ToolbarIconButton(Icons.Default.FormatQuote, "Quote") { insert("blockquote") }
        ToolbarIconButton(Icons.Default.Code, "Code") { insert("codeBlock") }
        ToolbarIconButton(Icons.Default.HorizontalRule, "Divider") { insert("horizontalRule") }
        ToolbarDividerLine()
        ToolbarIconButton(Icons.Default.Delete, "Delete block") {
            focusedId?.let { deleteBlock(it) }
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
