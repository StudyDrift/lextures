package com.lextures.android.features.boards.layouts

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.boards.BoardPostCardSlot
import com.lextures.android.features.home.LmsCard

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun ColumnsLayout(
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    onCreateSection: ((String) -> Unit)?,
    onDeleteSection: ((String) -> Unit)?,
    modifier: Modifier = Modifier,
) {
    val ordered = remember(sections) { BoardsLogic.sortedSections(sections) }
    val pages = remember(ordered) { ordered + null } // null = unsorted
    val pagerState = rememberPagerState(pageCount = { maxOf(pages.size, 1) })
    var showAdd by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (canManage) {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                TextButton(onClick = { showAdd = true; newTitle = "" }) {
                    Icon(Icons.Default.Add, contentDescription = null)
                    Text(L.text(R.string.mobile_boards_section_add))
                }
            }
        }

        if (ordered.isEmpty() && posts.isEmpty()) {
            BoardPostsEmpty()
        } else {
            HorizontalPager(
                state = pagerState,
                contentPadding = PaddingValues(horizontal = 8.dp),
                pageSpacing = 12.dp,
                modifier = Modifier.fillMaxWidth().heightIn(min = 360.dp),
            ) { page ->
                val section = pages.getOrNull(page)
                val sectionId = section?.id
                val lanePosts = BoardsLogic.postsInSection(posts, sectionId)
                val title = section?.title ?: L.text(R.string.mobile_boards_section_unsorted)
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                title,
                                color = textPrimary(),
                                modifier = Modifier.weight(1f),
                            )
                            if (canManage && section != null && onDeleteSection != null) {
                                IconButton(onClick = { onDeleteSection(section.id) }) {
                                    Icon(
                                        Icons.Default.Delete,
                                        contentDescription = L.text(R.string.mobile_boards_section_delete),
                                    )
                                }
                            }
                        }
                        if (lanePosts.isEmpty()) {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .heightIn(min = 120.dp)
                                    .border(
                                        width = 1.dp,
                                        color = textSecondary().copy(alpha = 0.4f),
                                        shape = RoundedCornerShape(12.dp),
                                    )
                                    .padding(16.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(L.text(R.string.mobile_boards_section_dropHere), color = textSecondary())
                            }
                        } else {
                            LazyColumn(
                                verticalArrangement = Arrangement.spacedBy(10.dp),
                                modifier = Modifier.heightIn(max = 480.dp),
                                userScrollEnabled = true,
                            ) {
                                items(lanePosts, key = { it.id }) { post ->
                                    BoardPostCardSlot(
                                        post = post,
                                        siblings = lanePosts,
                                        sections = ordered,
                                        board = board,
                                        canManage = canManage,
                                        currentUserId = currentUserId,
                                        onEdit = onEdit,
                                        onDelete = onDelete,
                                        onArrange = onArrange,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showAdd && onCreateSection != null) {
        AlertDialog(
            onDismissRequest = { showAdd = false },
            title = { Text(L.text(R.string.mobile_boards_section_add)) },
            text = {
                OutlinedTextField(
                    value = newTitle,
                    onValueChange = { newTitle = it },
                    label = { Text(L.text(R.string.mobile_boards_section_titlePlaceholder)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val title = newTitle.trim()
                        if (title.isEmpty()) return@TextButton
                        onCreateSection(title)
                        showAdd = false
                    },
                    enabled = newTitle.trim().isNotEmpty(),
                ) { Text(L.text(R.string.mobile_common_save)) }
            },
            dismissButton = {
                TextButton(onClick = { showAdd = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}
