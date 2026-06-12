package com.lextures.android.features.notebooks

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CreateNewFolder
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
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
import com.lextures.android.core.notebook.CourseNotebook
import com.lextures.android.core.notebook.NotebookMarkdown
import com.lextures.android.core.notebook.NotebookPage
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.notebook.NotebookSync
import com.lextures.android.core.notebook.NotebookTree
import kotlinx.coroutines.launch

/**
 * Pages of one notebook: an Evernote-style list with collapsible groups, search, and a
 * floating new-page button. All page management (create / rename / move / delete) lives
 * here via per-row menus — replaces the old in-editor pages capsule + bottom sheet.
 */
@Composable
fun NotebookPagesScreen(
    store: NotebookStore,
    courseCode: String,
    title: String,
    accessToken: String?,
    onBack: () -> Unit,
    onOpenPage: (pageId: String) -> Unit,
    modifier: Modifier = Modifier,
) {
    var notebook by remember { mutableStateOf(store.load(courseCode)) }
    var collapsed by remember { mutableStateOf(setOf<String>()) }
    var searchText by remember { mutableStateOf("") }
    var renameTarget by remember { mutableStateOf<NotebookPage?>(null) }
    var renameText by remember { mutableStateOf("") }
    var deleteTarget by remember { mutableStateOf<NotebookPage?>(null) }
    val scope = rememberCoroutineScope()

    // Merge any newer server copy (e.g. written on web) once on open.
    LaunchedEffect(courseCode) {
        if (NotebookSync.pull(store, accessToken)) {
            notebook = store.load(courseCode)
        }
    }

    fun persist(next: CourseNotebook) {
        val data = if (courseCode == NotebookStore.GLOBAL_KEY) next else next.copy(courseTitle = title)
        store.save(courseCode, data)
        notebook = store.load(courseCode)
        scope.launch { NotebookSync.push(store, courseCode, accessToken) }
    }

    fun createPage(parentId: String?) {
        val (pages, newId) = NotebookTree.addPage(notebook.pages, parentId)
        if (parentId != null) collapsed = collapsed - parentId
        persist(notebook.copy(pages = pages, activePageId = newId))
        onOpenPage(newId)
    }

    fun createGroup() {
        val (pages, newId) = NotebookTree.addGroup(notebook.pages, parentId = null, title = "New group")
        persist(notebook.copy(pages = pages))
        renameTarget = notebook.pages.firstOrNull { it.id == newId }
        renameText = "New group"
    }

    fun deletePage(pageId: String) {
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
        persist(notebook.copy(pages = pages, activePageId = activeId))
    }

    BackHandler { onBack() }

    val searching = searchText.isNotBlank()
    val rows = if (searching) {
        val query = searchText.trim().lowercase()
        notebook.pages
            .filter { !NotebookTree.isGroup(it) }
            .filter {
                it.title.lowercase().contains(query) ||
                    NotebookMarkdown.previewText(it.contentMd).lowercase().contains(query)
            }
            .map { NotebookTree.FlatRow(it, 0) }
    } else {
        visibleRows(notebook.pages, collapsed)
    }

    Column(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = title,
                style = LexturesType.display(20),
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = { createGroup() }) {
                Icon(Icons.Default.CreateNewFolder, contentDescription = "New group", tint = textPrimary())
            }
        }

        OutlinedTextField(
            value = searchText,
            onValueChange = { searchText = it },
            placeholder = { Text("Search pages", fontSize = 14.sp, color = textSecondary()) },
            leadingIcon = { Icon(Icons.Default.Search, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(18.dp)) },
            singleLine = true,
            shape = RoundedCornerShape(12.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = accentColor(),
                unfocusedBorderColor = fieldBorder(),
                focusedContainerColor = cardBackground(),
                unfocusedContainerColor = cardBackground(),
            ),
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp),
        )

        Box(modifier = Modifier.weight(1f)) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 140.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (rows.isEmpty()) {
                    item {
                        Text(
                            text = if (searching) "No pages match your search." else "Tap + to create your first page.",
                            fontSize = 13.sp,
                            color = textSecondary(),
                            modifier = Modifier.fillMaxWidth().padding(vertical = 28.dp),
                        )
                    }
                }
                items(rows, key = { it.page.id }) { row ->
                    if (NotebookTree.isGroup(row.page)) {
                        GroupRow(
                            pages = notebook.pages,
                            row = row,
                            isCollapsed = row.page.id in collapsed,
                            onToggle = {
                                collapsed = if (row.page.id in collapsed) collapsed - row.page.id else collapsed + row.page.id
                            },
                            onCreateInside = { createPage(row.page.id) },
                            onRename = {
                                renameTarget = row.page
                                renameText = row.page.title
                            },
                            onMove = { newParent ->
                                NotebookTree.moveToParent(notebook.pages, row.page.id, newParent)?.let {
                                    persist(notebook.copy(pages = it))
                                }
                            },
                            onDelete = { deleteTarget = row.page },
                            canDelete = NotebookTree.delete(notebook.pages, row.page.id).any { !NotebookTree.isGroup(it) },
                        )
                    } else {
                        PageRow(
                            pages = notebook.pages,
                            row = row,
                            onOpen = { onOpenPage(row.page.id) },
                            onRename = {
                                renameTarget = row.page
                                renameText = row.page.title
                            },
                            onMove = { newParent ->
                                NotebookTree.moveToParent(notebook.pages, row.page.id, newParent)?.let {
                                    persist(notebook.copy(pages = it))
                                }
                            },
                            onDelete = { deleteTarget = row.page },
                            canDelete = NotebookTree.delete(notebook.pages, row.page.id).any { !NotebookTree.isGroup(it) },
                        )
                    }
                }
            }

            // Tab content is already inset for the floating bar; keep a small margin above that.
            FloatingActionButton(
                onClick = { createPage(null) },
                shape = CircleShape,
                containerColor = LexturesColors.Primary,
                contentColor = Color.White,
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(end = 20.dp, bottom = 20.dp),
            ) {
                Icon(Icons.Default.Add, contentDescription = "New page")
            }
        }
    }

    renameTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { renameTarget = null },
            title = { Text("Rename") },
            text = {
                OutlinedTextField(value = renameText, onValueChange = { renameText = it }, singleLine = true)
            },
            confirmButton = {
                TextButton(onClick = {
                    val name = renameText.trim()
                    if (name.isNotEmpty()) {
                        persist(notebook.copy(pages = NotebookTree.rename(notebook.pages, target.id, name)))
                    }
                    renameTarget = null
                }) { Text("Save") }
            },
            dismissButton = {
                TextButton(onClick = { renameTarget = null }) { Text("Cancel") }
            },
        )
    }

    deleteTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text("Delete \"${target.title.ifBlank { "Untitled" }}\"?") },
            text = {
                Text(
                    if (NotebookTree.isGroup(target)) {
                        "The group and everything inside it will be deleted."
                    } else {
                        "This page and its notes will be deleted."
                    },
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    deletePage(target.id)
                    deleteTarget = null
                }) { Text("Delete", color = LexturesColors.Error) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text("Cancel") }
            },
        )
    }
}

/** Depth-first rows, skipping the contents of collapsed groups. */
private fun visibleRows(pages: List<NotebookPage>, collapsed: Set<String>): List<NotebookTree.FlatRow> {
    val rows = mutableListOf<NotebookTree.FlatRow>()
    fun walk(parentId: String?, depth: Int) {
        for (page in NotebookTree.sortedChildren(pages, parentId)) {
            rows.add(NotebookTree.FlatRow(page, depth))
            if (NotebookTree.isGroup(page) && page.id !in collapsed) {
                walk(page.id, depth + 1)
            }
        }
    }
    walk(null, 0)
    return rows
}

@Composable
private fun PageRow(
    pages: List<NotebookPage>,
    row: NotebookTree.FlatRow,
    onOpen: () -> Unit,
    onRename: () -> Unit,
    onMove: (String?) -> Unit,
    onDelete: () -> Unit,
    canDelete: Boolean,
) {
    val page = row.page
    val snippet = NotebookMarkdown.previewText(page.contentMd)

    Row(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = (row.depth * 18).dp)
            .clip(RoundedCornerShape(14.dp))
            .background(cardBackground())
            .border(1.dp, fieldBorder().copy(alpha = 0.9f), RoundedCornerShape(14.dp))
            .clickable(onClick = onOpen)
            .padding(start = 14.dp, top = 10.dp, bottom = 10.dp, end = 4.dp),
    ) {
        Icon(
            imageVector = Icons.Default.Description,
            contentDescription = null,
            tint = accentColor(),
            modifier = Modifier.size(18.dp),
        )
        Column(verticalArrangement = Arrangement.spacedBy(2.dp), modifier = Modifier.weight(1f)) {
            Text(
                text = page.title.ifBlank { "Untitled" },
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (snippet.isNotEmpty()) {
                Text(
                    text = snippet,
                    fontSize = 12.sp,
                    color = textSecondary(),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        PageActionsMenu(
            pages = pages,
            page = page,
            onRename = onRename,
            onMove = onMove,
            onDelete = onDelete,
            canDelete = canDelete,
            onCreateInside = null,
        )
    }
}

@Composable
private fun GroupRow(
    pages: List<NotebookPage>,
    row: NotebookTree.FlatRow,
    isCollapsed: Boolean,
    onToggle: () -> Unit,
    onCreateInside: () -> Unit,
    onRename: () -> Unit,
    onMove: (String?) -> Unit,
    onDelete: () -> Unit,
    canDelete: Boolean,
) {
    val group = row.page
    val count = NotebookTree.sortedChildren(pages, group.id).size

    Row(
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = (row.depth * 18).dp)
            .clip(RoundedCornerShape(10.dp))
            .clickable(onClick = onToggle)
            .padding(start = 8.dp, top = 6.dp, bottom = 6.dp, end = 4.dp),
    ) {
        Icon(
            imageVector = Icons.Default.KeyboardArrowDown,
            contentDescription = if (isCollapsed) "Expand group" else "Collapse group",
            tint = textSecondary(),
            modifier = Modifier.size(18.dp).rotate(if (isCollapsed) -90f else 0f),
        )
        Icon(
            imageVector = Icons.Default.Folder,
            contentDescription = null,
            tint = LexturesColors.BrandAmber,
            modifier = Modifier.size(18.dp),
        )
        Text(
            text = group.title.ifBlank { "Untitled group" },
            fontSize = 14.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        Text(
            text = if (count == 1) "1 item" else "$count items",
            fontSize = 11.sp,
            color = textSecondary(),
        )
        PageActionsMenu(
            pages = pages,
            page = group,
            onRename = onRename,
            onMove = onMove,
            onDelete = onDelete,
            canDelete = canDelete,
            onCreateInside = onCreateInside,
        )
    }
}

@Composable
private fun PageActionsMenu(
    pages: List<NotebookPage>,
    page: NotebookPage,
    onRename: () -> Unit,
    onMove: (String?) -> Unit,
    onDelete: () -> Unit,
    canDelete: Boolean,
    onCreateInside: (() -> Unit)?,
) {
    var menuOpen by remember { mutableStateOf(false) }
    var moveOpen by remember { mutableStateOf(false) }
    val moveTargets = NotebookTree.groupMoveTargets(pages, page.id).filter { it.id != page.parentId }

    Box {
        IconButton(onClick = { menuOpen = true }, modifier = Modifier.size(36.dp)) {
            Icon(
                imageVector = Icons.Default.MoreVert,
                contentDescription = "Page actions",
                tint = textSecondary(),
                modifier = Modifier.size(18.dp),
            )
        }

        DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
            if (onCreateInside != null) {
                DropdownMenuItem(
                    text = { Text("New page inside") },
                    leadingIcon = { Icon(Icons.Default.Add, contentDescription = null) },
                    onClick = {
                        menuOpen = false
                        onCreateInside()
                    },
                )
            }
            DropdownMenuItem(
                text = { Text("Rename") },
                leadingIcon = { Icon(Icons.Default.Edit, contentDescription = null) },
                onClick = {
                    menuOpen = false
                    onRename()
                },
            )
            if (moveTargets.isNotEmpty() || page.parentId != null) {
                DropdownMenuItem(
                    text = { Text("Move to…") },
                    leadingIcon = { Icon(Icons.AutoMirrored.Filled.ArrowForward, contentDescription = null) },
                    onClick = {
                        menuOpen = false
                        moveOpen = true
                    },
                )
            }
            if (canDelete) {
                DropdownMenuItem(
                    text = { Text("Delete", color = LexturesColors.Error) },
                    leadingIcon = { Icon(Icons.Default.Delete, contentDescription = null, tint = LexturesColors.Error) },
                    onClick = {
                        menuOpen = false
                        onDelete()
                    },
                )
            }
        }

        DropdownMenu(expanded = moveOpen, onDismissRequest = { moveOpen = false }) {
            if (page.parentId != null) {
                DropdownMenuItem(
                    text = { Text("Top level") },
                    onClick = {
                        moveOpen = false
                        onMove(null)
                    },
                )
            }
            moveTargets.forEach { group ->
                DropdownMenuItem(
                    text = { Text(NotebookTree.pathLabel(pages, group.id)) },
                    leadingIcon = { Icon(Icons.Default.Folder, contentDescription = null, tint = LexturesColors.BrandAmber) },
                    onClick = {
                        moveOpen = false
                        onMove(group.id)
                    },
                )
            }
        }
    }
}
