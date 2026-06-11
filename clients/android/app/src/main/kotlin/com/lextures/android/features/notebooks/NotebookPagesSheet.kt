package com.lextures.android.features.notebooks

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.CreateNewFolder
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.notebook.NotebookPage
import com.lextures.android.core.notebook.NotebookTree

/**
 * Page tree for a notebook: groups with nested pages, plus create / rename / move / delete
 * (parity with the web notebook sidebar).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotebookPagesSheet(
    pages: List<NotebookPage>,
    activePageId: String?,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
    onCreatePage: (parentId: String?) -> Unit,
    onCreateGroup: () -> Unit,
    onRename: (pageId: String, title: String) -> Unit,
    onMove: (pageId: String, newParentId: String?) -> Unit,
    onDelete: (pageId: String) -> Unit,
) {
    var renameTarget by remember { mutableStateOf<NotebookPage?>(null) }
    var renameText by remember { mutableStateOf("") }
    var deleteTarget by remember { mutableStateOf<NotebookPage?>(null) }

    ModalBottomSheet(onDismissRequest = onDismiss, containerColor = cardBackground()) {
        Text(
            text = "Pages",
            fontSize = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 4.dp),
        )
        Text(
            text = "Pages live on this device. Use groups to organize related pages.",
            fontSize = 12.sp,
            color = textSecondary(),
            modifier = Modifier.padding(horizontal = 20.dp),
        )

        LazyColumn(
            contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 10.dp),
        ) {
            items(NotebookTree.flatten(pages), key = { it.page.id }) { row ->
                PageRow(
                    pages = pages,
                    row = row,
                    isActive = row.page.id == activePageId,
                    onSelect = {
                        onSelect(row.page.id)
                        onDismiss()
                    },
                    onRename = {
                        renameTarget = row.page
                        renameText = row.page.title
                    },
                    onCreateInside = {
                        onCreatePage(row.page.id)
                        onDismiss()
                    },
                    onMove = { newParent -> onMove(row.page.id, newParent) },
                    onDelete = { deleteTarget = row.page },
                )
            }

            item {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            onCreatePage(null)
                            onDismiss()
                        }
                        .padding(horizontal = 20.dp, vertical = 12.dp),
                ) {
                    Icon(Icons.Default.Add, contentDescription = null, tint = accentColor(), modifier = Modifier.size(20.dp))
                    Text("New page", fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
                }
            }
            item {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { onCreateGroup() }
                        .padding(horizontal = 20.dp, vertical = 12.dp),
                ) {
                    Icon(Icons.Default.CreateNewFolder, contentDescription = null, tint = accentColor(), modifier = Modifier.size(20.dp))
                    Text("New group", fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
                }
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
                    if (name.isNotEmpty()) onRename(target.id, name)
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
                    onDelete(target.id)
                    deleteTarget = null
                }) { Text("Delete", color = LexturesColors.Error) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text("Cancel") }
            },
        )
    }
}

@Composable
private fun PageRow(
    pages: List<NotebookPage>,
    row: NotebookTree.FlatRow,
    isActive: Boolean,
    onSelect: () -> Unit,
    onRename: () -> Unit,
    onCreateInside: () -> Unit,
    onMove: (String?) -> Unit,
    onDelete: () -> Unit,
) {
    val page = row.page
    val isGroup = NotebookTree.isGroup(page)
    var menuOpen by remember { mutableStateOf(false) }
    var moveOpen by remember { mutableStateOf(false) }
    val canDelete = NotebookTree.delete(pages, page.id).any { !NotebookTree.isGroup(it) }
    val moveTargets = NotebookTree.groupMoveTargets(pages, page.id).filter { it.id != page.parentId }

    Box {
        Row(
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 2.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(if (isActive) accentColor().copy(alpha = 0.10f) else cardBackground())
                .clickable(onClick = onSelect)
                .padding(start = 8.dp + (row.depth * 20).dp, top = 10.dp, bottom = 10.dp, end = 8.dp),
        ) {
            Icon(
                imageVector = if (isGroup) Icons.Default.Folder else Icons.Default.Description,
                contentDescription = null,
                tint = if (isGroup) LexturesColors.BrandAmber else accentColor(),
                modifier = Modifier.size(18.dp),
            )
            Text(
                text = page.title.ifBlank { "Untitled" },
                fontSize = 14.sp,
                fontWeight = if (isActive) FontWeight.SemiBold else FontWeight.Normal,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (isActive) {
                Icon(Icons.Default.Check, contentDescription = "Active page", tint = accentColor(), modifier = Modifier.size(16.dp))
            }
            Icon(
                imageVector = Icons.Default.Edit,
                contentDescription = "Page actions",
                tint = textSecondary(),
                modifier = Modifier
                    .size(18.dp)
                    .clickable { menuOpen = true },
            )
        }

        DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
            DropdownMenuItem(
                text = { Text("Rename") },
                leadingIcon = { Icon(Icons.Default.Edit, contentDescription = null) },
                onClick = {
                    menuOpen = false
                    onRename()
                },
            )
            if (isGroup) {
                DropdownMenuItem(
                    text = { Text("New page inside") },
                    leadingIcon = { Icon(Icons.Default.Add, contentDescription = null) },
                    onClick = {
                        menuOpen = false
                        onCreateInside()
                    },
                )
            }
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
