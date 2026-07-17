package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.SwapVert
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CardArrangeMenu(
    post: BoardPost,
    sections: List<BoardSection>,
    siblings: List<BoardPost>,
    showTimeline: Boolean = false,
    showMap: Boolean = false,
    onMoveToSection: (String) -> Unit,
    onReorder: (Double) -> Unit,
    onSetEventDate: ((String?) -> Unit)? = null,
    onSetCoords: ((Double, Double) -> Unit)? = null,
) {
    var menuOpen by remember { mutableStateOf(false) }
    var showDate by remember { mutableStateOf(false) }
    var showCoords by remember { mutableStateOf(false) }
    var latText by remember { mutableStateOf("") }
    var lngText by remember { mutableStateOf("") }
    val moveUp = remember(post, siblings) { BoardsLogic.sortIndexMovingUp(post, siblings) }
    val moveDown = remember(post, siblings) { BoardsLogic.sortIndexMovingDown(post, siblings) }
    val orderedSections = remember(sections) { BoardsLogic.sortedSections(sections) }

    IconButton(onClick = { menuOpen = true }) {
        Icon(
            Icons.Default.SwapVert,
            contentDescription = L.text(R.string.mobile_boards_arrange_menuAria),
        )
    }
    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
        DropdownMenuItem(
            text = { Text(L.text(R.string.mobile_boards_arrange_moveUp)) },
            onClick = {
                moveUp?.let(onReorder)
                menuOpen = false
            },
            enabled = moveUp != null,
        )
        DropdownMenuItem(
            text = { Text(L.text(R.string.mobile_boards_arrange_moveDown)) },
            onClick = {
                moveDown?.let(onReorder)
                menuOpen = false
            },
            enabled = moveDown != null,
        )
        if (orderedSections.isNotEmpty()) {
            orderedSections.forEach { section ->
                DropdownMenuItem(
                    text = {
                        Text("${L.text(R.string.mobile_boards_arrange_moveToSection)}: ${section.title}")
                    },
                    onClick = {
                        onMoveToSection(section.id)
                        menuOpen = false
                    },
                    enabled = post.sectionId != section.id,
                )
            }
        }
        if (showTimeline && onSetEventDate != null) {
            DropdownMenuItem(
                text = { Text(L.text(R.string.mobile_boards_arrange_eventDate)) },
                onClick = {
                    menuOpen = false
                    showDate = true
                },
            )
            if (!post.eventDate.isNullOrBlank()) {
                DropdownMenuItem(
                    text = { Text(L.text(R.string.mobile_boards_arrange_clearEventDate)) },
                    onClick = {
                        onSetEventDate(null)
                        menuOpen = false
                    },
                )
            }
        }
        if (showMap && onSetCoords != null) {
            DropdownMenuItem(
                text = { Text(L.text(R.string.mobile_boards_arrange_editCoords)) },
                onClick = {
                    latText = post.lat?.toString().orEmpty()
                    lngText = post.lng?.toString().orEmpty()
                    menuOpen = false
                    showCoords = true
                },
            )
        }
    }

    if (showDate && onSetEventDate != null) {
        val initialMillis = post.eventDate
            ?.let { runCatching { Instant.parse(it).toEpochMilli() }.getOrNull() }
            ?: System.currentTimeMillis()
        val state = rememberDatePickerState(initialSelectedDateMillis = initialMillis)
        DatePickerDialog(
            onDismissRequest = { showDate = false },
            confirmButton = {
                TextButton(onClick = {
                    val millis = state.selectedDateMillis ?: return@TextButton
                    val iso = Instant.ofEpochMilli(millis)
                        .atZone(ZoneOffset.UTC)
                        .toLocalDate()
                        .format(DateTimeFormatter.ISO_LOCAL_DATE)
                    onSetEventDate(iso)
                    showDate = false
                }) { Text(L.text(R.string.mobile_common_save)) }
            },
            dismissButton = {
                TextButton(onClick = { showDate = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        ) {
            DatePicker(state = state)
        }
    }

    if (showCoords && onSetCoords != null) {
        AlertDialog(
            onDismissRequest = { showCoords = false },
            title = { Text(L.text(R.string.mobile_boards_arrange_setCoords)) },
            text = {
                Column {
                    OutlinedTextField(
                        value = latText,
                        onValueChange = { latText = it },
                        label = { Text(L.text(R.string.mobile_boards_arrange_latPrompt)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                    OutlinedTextField(
                        value = lngText,
                        onValueChange = { lngText = it },
                        label = { Text(L.text(R.string.mobile_boards_arrange_lngPrompt)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    val lat = latText.toDoubleOrNull()
                    val lng = lngText.toDoubleOrNull()
                    if (lat == null || lng == null || lat !in -90.0..90.0 || lng !in -180.0..180.0) {
                        return@TextButton
                    }
                    onSetCoords(lat, lng)
                    showCoords = false
                }) { Text(L.text(R.string.mobile_boards_arrange_saveCoords)) }
            },
            dismissButton = {
                TextButton(onClick = { showCoords = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}
