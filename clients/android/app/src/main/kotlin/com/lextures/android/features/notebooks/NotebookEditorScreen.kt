package com.lextures.android.features.notebooks

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.notebook.NotebookPage
import com.lextures.android.core.notebook.NotebookStore

/** Markdown notebook editor with a page picker; saves to device-local storage as you type. */
@Composable
fun NotebookEditorScreen(
    store: NotebookStore,
    courseCode: String,
    title: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var notebook by remember {
        val loaded = store.load(courseCode)
        mutableStateOf(
            if (courseCode == NotebookStore.GLOBAL_KEY) loaded else loaded.copy(courseTitle = title),
        )
    }
    val activePage = notebook.pages.firstOrNull { it.id == notebook.activePageId } ?: notebook.pages.firstOrNull()
    var draft by remember { mutableStateOf(activePage?.contentMd.orEmpty()) }
    var menuOpen by remember { mutableStateOf(false) }
    var renaming by remember { mutableStateOf(false) }
    var renameText by remember { mutableStateOf("") }

    fun commitDraft() {
        val active = activePage ?: return
        notebook = notebook.copy(
            pages = notebook.pages.map { if (it.id == active.id) it.copy(contentMd = draft) else it },
        )
    }

    fun save() {
        commitDraft()
        store.save(courseCode, notebook)
    }

    BackHandler {
        save()
        onBack()
    }

    Column(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = {
                save()
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
                            commitDraft()
                            val maxOrder = notebook.pages.maxOfOrNull { it.sortOrder } ?: 0
                            val page = NotebookPage.new(sortOrder = maxOrder + 1)
                            notebook = notebook.copy(
                                pages = notebook.pages + page,
                                activePageId = page.id,
                            )
                            draft = ""
                            store.save(courseCode, notebook)
                        },
                    )
                    if (notebook.pages.size > 1) {
                        DropdownMenuItem(
                            text = { Text("Delete page") },
                            leadingIcon = { Icon(Icons.Default.Delete, contentDescription = null) },
                            onClick = {
                                menuOpen = false
                                val active = activePage ?: return@DropdownMenuItem
                                val remaining = notebook.pages.filterNot { it.id == active.id }
                                notebook = notebook.copy(
                                    pages = remaining,
                                    activePageId = remaining.firstOrNull()?.id,
                                )
                                draft = remaining.firstOrNull()?.contentMd.orEmpty()
                                store.save(courseCode, notebook)
                            },
                        )
                    }
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            notebook.pages.sortedBy { it.sortOrder }.forEach { page ->
                val selected = page.id == activePage?.id
                Text(
                    text = page.title.ifBlank { "Untitled" },
                    fontSize = 13.sp,
                    fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                    color = if (selected) LexturesColors.Primary else textSecondary(),
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .background(
                            if (selected) LexturesColors.Primary.copy(alpha = 0.14f) else cardBackground(),
                        )
                        .border(
                            1.dp,
                            if (selected) LexturesColors.Primary.copy(alpha = 0.4f) else fieldBorder(),
                            RoundedCornerShape(50),
                        )
                        .clickable {
                            commitDraft()
                            notebook = notebook.copy(activePageId = page.id)
                            draft = notebook.pages.first { it.id == page.id }.contentMd
                            store.save(courseCode, notebook)
                        }
                        .padding(horizontal = 14.dp, vertical = 7.dp),
                )
            }
        }

        BasicTextField(
            value = draft,
            onValueChange = {
                draft = it
                save()
            },
            textStyle = TextStyle(
                fontSize = 14.sp,
                fontFamily = FontFamily.Monospace,
                color = textPrimary(),
            ),
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(cardBackground())
                .border(1.dp, fieldBorder().copy(alpha = 0.9f), RoundedCornerShape(12.dp))
                .padding(12.dp),
        )
    }

    if (renaming) {
        AlertDialog(
            onDismissRequest = { renaming = false },
            title = { Text("Rename page") },
            text = {
                OutlinedTextField(
                    value = renameText,
                    onValueChange = { renameText = it },
                    singleLine = true,
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    val name = renameText.trim()
                    val active = activePage
                    if (name.isNotEmpty() && active != null) {
                        notebook = notebook.copy(
                            pages = notebook.pages.map {
                                if (it.id == active.id) it.copy(title = name) else it
                            },
                        )
                        store.save(courseCode, notebook)
                    }
                    renaming = false
                }) {
                    Text("Save")
                }
            },
            dismissButton = {
                TextButton(onClick = { renaming = false }) { Text("Cancel") }
            },
        )
    }
}
