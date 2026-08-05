package com.lextures.android.features.checklist

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ChecklistDismissReason
import com.lextures.android.core.lms.ChecklistItem
import com.lextures.android.core.lms.CourseChecklistLogic

@Composable
fun ChecklistDismissDialog(
    item: ChecklistItem,
    isOnline: Boolean,
    onDismissRequest: () -> Unit,
    onConfirm: (ChecklistDismissReason, String?) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var reason by remember { mutableStateOf(ChecklistDismissReason.NotApplicable) }
    var note by remember { mutableStateOf("") }
    var showNote by remember { mutableStateOf(false) }

    val reasonLabels = mapOf(
        ChecklistDismissReason.NotApplicable to R.string.mobile_checklist_dismissReason_not_applicable,
        ChecklistDismissReason.DoneElsewhere to R.string.mobile_checklist_dismissReason_done_elsewhere,
        ChecklistDismissReason.Disagree to R.string.mobile_checklist_dismissReason_disagree,
        ChecklistDismissReason.Later to R.string.mobile_checklist_dismissReason_later,
        ChecklistDismissReason.Other to R.string.mobile_checklist_dismissReason_other,
    )

    AlertDialog(
        onDismissRequest = onDismissRequest,
        title = { Text(L.text(context, localePrefs, R.string.mobile_checklist_dismissDialogTitle)) },
        text = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState()),
            ) {
                Text(item.title, modifier = Modifier.padding(bottom = 8.dp))
                Text(
                    L.text(context, localePrefs, R.string.mobile_checklist_dismissDialogHelp),
                    modifier = Modifier.padding(bottom = 12.dp),
                )
                ChecklistDismissReason.entries.forEach { r ->
                    androidx.compose.foundation.layout.Row(
                        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
                    ) {
                        RadioButton(selected = reason == r, onClick = { reason = r })
                        Text(L.text(context, localePrefs, reasonLabels.getValue(r)))
                    }
                }
                if (showNote) {
                    OutlinedTextField(
                        value = note,
                        onValueChange = {
                            note = if (it.length <= CourseChecklistLogic.MAX_DISMISS_NOTE_LENGTH) {
                                it
                            } else {
                                it.take(CourseChecklistLogic.MAX_DISMISS_NOTE_LENGTH)
                            }
                        },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_checklist_dismissNoteLabel)) },
                        placeholder = {
                            Text(L.text(context, localePrefs, R.string.mobile_checklist_dismissNotePlaceholder))
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 8.dp),
                        minLines = 2,
                    )
                } else {
                    TextButton(onClick = { showNote = true }) {
                        Text(L.text(context, localePrefs, R.string.mobile_checklist_addNote))
                    }
                }
                if (!isOnline) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_checklist_offlineMutations),
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = { onConfirm(reason, if (showNote) note else null) },
                enabled = isOnline,
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_checklist_dismissConfirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismissRequest) {
                Text(L.text(context, localePrefs, R.string.mobile_checklist_dismissCancel))
            }
        },
    )
}
